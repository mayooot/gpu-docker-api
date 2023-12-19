package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/commander-cli/cmd"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/ngaut/log"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/pkg/errors"
	"github.com/siddontang/go/sync2"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/gpuscheduler"
	"github.com/mayooot/gpu-docker-api/internal/model"
	xerrors "github.com/mayooot/gpu-docker-api/internal/xerrors"
)

var containerVersionMap = cmap.New[sync2.AtomicInt64]()

type ContainerService struct{}

func (cs *ContainerService) RunGpuContainer(spec *model.ContainerRun) (id, containerName string, err error) {
	var (
		config           container.Config
		hostConfig       container.HostConfig
		networkingConfig network.NetworkingConfig
		platform         ocispec.Platform
	)

	ctx := context.Background()
	if cs.existContainer(spec.ContainerName) {
		return id, containerName, errors.Wrapf(xerrors.NewContainerExistedError(), "container %s", spec.ContainerName)
	}

	config = container.Config{
		Image:     spec.ImageName,
		Cmd:       spec.Cmd,
		Env:       spec.Env,
		OpenStdin: true,
		Tty:       true,
	}

	hostConfig.PortBindings = make(nat.PortMap, len(spec.Ports))
	for _, port := range spec.Ports {
		hostConfig.PortBindings[nat.Port(fmt.Sprintf("%d/tcp", port.ContainerPort))] = []nat.PortBinding{{
			HostPort: fmt.Sprintf("%d", port.HostPort),
		}}
	}

	if !spec.Cardless {
		// 有卡模式启动容器
		uuids, err := gpuscheduler.Scheduler.ApplyGpus(spec.GpuCount)
		if err != nil {
			return id, containerName, errors.Wrapf(err, "gpuscheduler.ApplyGpus failed, spec: %+v", spec)
		}

		hostConfig.Resources = container.Resources{DeviceRequests: []container.DeviceRequest{{
			Driver:       "nvidia",
			DeviceIDs:    uuids,
			Capabilities: [][]string{{"gpu"}},
			Options:      nil,
		}}}
	}

	// 卷挂载
	hostConfig.Mounts = make([]mount.Mount, 0, len(spec.Binds))
	for i := range spec.Binds {
		src := spec.Binds[i].Src
		m := mount.Mount{
			Source: src,
			Target: spec.Binds[i].Dest,
		}
		if strings.HasPrefix(src, "/") {
			// host dir
			m.Type = mount.TypeBind
		} else {
			// docker volume
			m.Type = mount.TypeVolume
		}
		hostConfig.Mounts = append(hostConfig.Mounts, m)
	}

	id, containerName, err = cs.runContainer(ctx, spec.ContainerName, model.EtcdContainerInfo{
		Config:           &config,
		HostConfig:       &hostConfig,
		NetworkingConfig: &networkingConfig,
		Platform:         &platform,
	})
	if err != nil {
		return id, containerName, errors.Wrapf(err, "serivce.runContainer failed, spec: %+v", spec)
	}
	return id, containerName, err
}

func (cs *ContainerService) DeleteContainer(name string, spec *model.ContainerDelete) error {
	var err error
	ctx := context.Background()
	if err = docker.Cli.ContainerRemove(ctx, name, types.ContainerRemoveOptions{Force: spec.Force}); err != nil {
		return errors.Wrapf(err, "docker.ContainerRemove failed, name: %s", name)
	}

	uuids, err := cs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return errors.WithMessage(err, "service.containerDeviceRequestsDeviceIDs failed")
	}
	gpuscheduler.Scheduler.RestoreGpus(uuids)

	if spec.DelEtcdInfo {
		WorkQueue <- etcd.DelKey{
			Resource: etcd.ContainerPrefix,
			Key:      name,
		}
	}
	log.Info("container deleted successfully, name:", name)
	return err
}

func (cs *ContainerService) ExecuteContainer(name string, exec *model.ContainerExecute) (resp *string, err error) {
	workDir := "/"
	var cmd []string
	if len(exec.WorkDir) != 0 {
		workDir = exec.WorkDir
	}
	if len(exec.Cmd) != 0 {
		cmd = exec.Cmd
	}

	ctx := context.Background()
	execCreate, err := docker.Cli.ContainerExecCreate(ctx, name, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Detach:       true,
		DetachKeys:   "ctrl-p,q",
		WorkingDir:   workDir,
		Cmd:          cmd,
	})
	if err != nil {
		return resp, errors.Wrapf(err, "docker.ContainerExecCreate failed, name: %s, spec: %+v", name, exec)
	}

	hijackedResp, err := docker.Cli.ContainerExecAttach(ctx, execCreate.ID, types.ExecStartCheck{})
	defer hijackedResp.Close()
	if err != nil {
		return resp, errors.Wrapf(err, "docker.ContainerExecAttach failed, name: %s, spec: %+v", name, exec)
	}

	var buf bytes.Buffer
	_, _ = stdcopy.StdCopy(&buf, &buf, hijackedResp.Reader)
	str := buf.String()
	resp = &str

	return resp, err
}

func (cs *ContainerService) PatchContainerGpuInfo(name string, spec *model.ContainerGpuPatch) (id, newContainerName string, err error) {
	ctx := context.Background()
	infoBytes, err := etcd.Get(etcd.ContainerPrefix, name)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "etcd.Get failed")
	}

	var info model.EtcdContainerInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return id, newContainerName, errors.WithMessage(err, "json.Unmarshal failed")
	}

	uuids, err := gpuscheduler.Scheduler.ApplyGpus(spec.GpuCount)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "gpuscheduler.Scheduler.ApplyGpus failed")
	}

	// 更改 gpu 配置
	info.HostConfig.Resources.DeviceRequests[0].DeviceIDs = uuids
	id, newContainerName, err = cs.runContainer(ctx, strings.Split(name, "-")[0], info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "service.runContainer failed")
	}

	// 异步拷贝旧容器的系统盘到新的容器
	WorkQueue <- &copyTask{
		Resource:    etcd.ContainerPrefix,
		OldResource: info.ContainerName,
		NewResource: newContainerName,
	}

	return id, newContainerName, err
}

func (cs *ContainerService) PatchContainerVolumeInfo(name string, spec *model.ContainerVolumePatch) (id, newContainerName string, err error) {
	ctx := context.Background()
	infoBytes, err := etcd.Get(etcd.ContainerPrefix, name)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "etcd.Get failed")
	}

	var info model.EtcdContainerInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return id, newContainerName, errors.WithMessage(err, "json.Unmarshal failed")
	}

	for i := range info.HostConfig.Mounts {
		if info.HostConfig.Mounts[i].Type == spec.Type && info.HostConfig.Mounts[i].Source == spec.OldVolumeName {
			info.HostConfig.Mounts[i].Source = spec.NewVolumeName
			break
		}
	}
	id, newContainerName, err = cs.runContainer(ctx, strings.Split(name, "-")[0], info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "service.runContainer failed")
	}

	// 异步拷贝旧容器的系统盘到新的容器
	WorkQueue <- &copyTask{
		Resource:    etcd.ContainerPrefix,
		OldResource: info.ContainerName,
		NewResource: newContainerName,
	}

	return id, newContainerName, err
}

func (cs *ContainerService) StopContainer(name string) error {
	ctx := context.Background()
	if err := docker.Cli.ContainerStop(ctx, name, container.StopOptions{}); err != nil {
		return errors.WithMessage(err, "docker.ContainerStop failed")
	}

	uuids, err := cs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return errors.WithMessage(err, "service.containerDeviceRequestsDeviceIDs failed")
	}
	gpuscheduler.Scheduler.RestoreGpus(uuids)
	return nil
}

func (cs *ContainerService) RestartContainer(name string) error {
	ctx := context.Background()
	if err := docker.Cli.ContainerRestart(ctx, name, container.StopOptions{}); err != nil {
		return errors.WithMessage(err, "docker.ContainerRestart failed")
	}
	uuids, err := cs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return errors.WithMessage(err, "service.containerDeviceRequestsDeviceIDs failed")
	}
	gpuscheduler.Scheduler.RestoreGpus(uuids)
	return nil
}

func (cs *ContainerService) CommitContainer(name string) (string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.ContainerCommit(ctx, name, types.ContainerCommitOptions{
		Comment: fmt.Sprintf("container name %s, commit time: %s", name, time.Now().Format("2006-01-02 15:04:05")),
	})
	if err != nil {
		return "", errors.WithMessage(err, "docker.ContainerRestart failed")
	}

	if err = docker.Cli.ImageTag(ctx, resp.ID, name); err != nil {
		return "", errors.WithMessage(err, "docker.ImageTag failed")
	}

	return resp.ID, err
}

func (cs *ContainerService) runContainer(ctx context.Context, name string, info model.EtcdContainerInfo) (id, containerName string, err error) {
	// 容器的版本号
	version, ok := containerVersionMap.Get(name)
	if !ok {
		containerVersionMap.Set(name, 0)
	} else {
		containerVersionMap.Set(name, sync2.AtomicInt64(version.Add(1)))
	}

	// 容器名称
	containerName = fmt.Sprintf("%s-%d", name, version)
	resp, err := docker.Cli.ContainerCreate(ctx, info.Config, info.HostConfig, info.NetworkingConfig, info.Platform, containerName)
	if err != nil {
		return id, containerName, errors.Wrapf(err, "docker.ContainerCreate failed, name: %s", containerName)
	}
	id = resp.ID

	// 启动容器
	if err = docker.Cli.ContainerStart(ctx, id, types.ContainerStartOptions{}); err != nil {
		_ = docker.Cli.ContainerRemove(ctx,
			resp.ID,
			types.ContainerRemoveOptions{Force: true})
		return id, containerName, errors.Wrapf(err, "docker.ContainerStart failed, id: %s, name: %s", id, containerName)
	}

	// 经过 docker create 校验后的容器配置，放入到 etcd 中
	val := &model.EtcdContainerInfo{
		Config:           info.Config,
		HostConfig:       info.HostConfig,
		NetworkingConfig: info.NetworkingConfig,
		Platform:         info.Platform,
		ContainerName:    containerName,
		Version:          version.Get(),
	}
	// 异步添加到 etcd 中
	WorkQueue <- etcd.PutKeyValue{
		Key:      containerName,
		Value:    val.Serialize(),
		Resource: etcd.ContainerPrefix,
	}

	log.Infof("container started successfully, id: %s, name: %s", id, containerName)
	return id, containerName, err
}

func (cs *ContainerService) existContainer(name string) bool {
	ctx := context.Background()
	list, err := docker.Cli.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: fmt.Sprintf("^%s-", name)}),
	})
	if err != nil || len(list) == 0 {
		return false
	}

	return len(list) > 0
}

func (cs *ContainerService) copyMergedDirToContainer(task *copyTask) error {
	oldMerged, err := cs.containerGraphDriverDataMergedDir(task.OldResource)
	if err != nil {
		return errors.WithMessage(err, "service.containerGraphDriverDataMergedDir failed")
	}
	newMerged, err := cs.containerGraphDriverDataMergedDir(task.NewResource)
	if err != nil {
		return errors.WithMessage(err, "service.containerGraphDriverDataMergedDir failed")
	}

	if err = cs.copyMergedDirFromOldVersion(oldMerged, newMerged); err != nil {
		return errors.WithMessage(err, "service.containerGraphDriverDataMergedDir failed")
	}

	return nil
}

func (cs *ContainerService) containerGraphDriverDataMergedDir(name string) (string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.ContainerInspect(ctx, name)
	if err != nil || len(resp.GraphDriver.Data["MergedDir"]) == 0 {
		return "", errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
	}
	return resp.GraphDriver.Data["MergedDir"], nil
}

func (cs *ContainerService) copyMergedDirFromOldVersion(src, dest string) error {
	startT := time.Now()
	command := fmt.Sprintf(cpRFPOption, src, dest)
	if err := cmd.NewCommand(command).Execute(); err != nil {
		return errors.Wrapf(err, "cmd.Execute failed, command %s, src:%s, dest: %s", command, src, dest)
	}
	log.Infof("service.copyDiffFromOldVersion copy merged successfully, src: %s, dest: %s, time cost: %v", src, dest, time.Since(startT))
	return nil
}

// 获取容器使用的 GPU 列表 （UUID）
func (cs *ContainerService) containerDeviceRequestsDeviceIDs(name string) ([]string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.ContainerInspect(ctx, name)
	if err != nil || len(resp.HostConfig.DeviceRequests[0].DeviceIDs) == 0 {
		return nil, errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
	}

	return resp.HostConfig.DeviceRequests[0].DeviceIDs, nil
}
