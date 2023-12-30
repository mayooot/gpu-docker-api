package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/ngaut/log"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/siddontang/go/sync2"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/model"
	"github.com/mayooot/gpu-docker-api/internal/scheduler/gpuscheduler"
	"github.com/mayooot/gpu-docker-api/internal/scheduler/portscheduler"
	vmap "github.com/mayooot/gpu-docker-api/internal/version"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

type ContainerService struct{}

// RunGpuContainer 创建并启动一个 GPU 容器
func (cs *ContainerService) RunGpuContainer(spec *model.ContainerRun) (id, containerName string, err error) {
	var (
		config           container.Config
		hostConfig       container.HostConfig
		networkingConfig network.NetworkingConfig
		platform         ocispec.Platform
	)

	// 判断容器是否存在
	ctx := context.Background()
	if cs.existContainer(spec.ContainerName) {
		return id, containerName, errors.Wrapf(xerrors.NewContainerExistedError(), "container %s", spec.ContainerName)
	}

	// 设置镜像，启动命令，环境变量等
	config = container.Config{
		Image:     spec.ImageName,
		Cmd:       spec.Cmd,
		Env:       spec.Env,
		OpenStdin: true,
		Tty:       true,
	}

	// 只想容器要暴露的端口，添加到创建容器的信息中
	// 具体这个端口要映射到宿主机的哪个端口，交给 runContainer 方法
	// 这样做的好处就是，不管是创建容器、变更容器 GPU/Volume、重启动容器，都无需关心端口的配置
	hostConfig.PortBindings = make(nat.PortMap, len(spec.ContainerPorts))
	config.ExposedPorts = make(nat.PortSet, len(spec.ContainerPorts))
	for _, port := range spec.ContainerPorts {
		config.ExposedPorts[nat.Port(port+"/tcp")] = struct{}{}
		hostConfig.PortBindings[nat.Port(port+"/tcp")] = nil
	}

	// 绑定 GPU 资源信息
	if spec.GpuCount > 0 {
		// 有卡模式启动容器
		uuids, err := gpuscheduler.Scheduler.ApplyGpus(spec.GpuCount)
		if err != nil {
			return id, containerName, errors.Wrapf(err, "gpuscheduler.ApplyGpus failed, spec: %+v", spec)
		}
		hostConfig.Resources = cs.newContainerResource(uuids)
		log.Infof("service.RunGpuContainer, container: %s apply %d gpus, uuids: %+v", spec.ContainerName+"-0", len(uuids), uuids)
	}

	// 卷挂载
	hostConfig.Binds = make([]string, 0, len(spec.Binds))
	for i := range spec.Binds {
		// Binds
		hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", spec.Binds[i].Src, spec.Binds[i].Dest))
	}

	// 创建并启动容器
	id, containerName, err = cs.runContainer(ctx, spec.ContainerName, model.EtcdContainerInfo{
		Config:           &config,
		HostConfig:       &hostConfig,
		NetworkingConfig: &networkingConfig,
		Platform:         &platform,
	})
	if err != nil {
		return id, containerName, errors.Wrapf(err, "serivce.runContainer failed, spec: %+v", spec)
	}
	return
}

// DeleteContainer 删除一个容器，归还端口资源，如果是 GPU 容器，会归还使用的 GPU 资源
// 根据入参选择是否要删除 etcd 中关于容器的描述，以及版本号记录
func (cs *ContainerService) DeleteContainer(name string, spec *model.ContainerDelete) error {
	var err error
	// 归还 gpu 资源
	uuids, err := cs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return errors.WithMessage(err, "service.containerDeviceRequestsDeviceIDs failed")
	}
	gpuscheduler.Scheduler.RestoreGpus(uuids)

	// 归还端口资源
	ports, err := cs.containerPortBindings(name)
	if err != nil {
		return errors.WithMessage(err, "service.containerPortBindings failed")
	}
	portscheduler.Scheduler.RestorePorts(ports)

	// 删除容器
	ctx := context.Background()
	if err = docker.Cli.ContainerRemove(ctx, name, types.ContainerRemoveOptions{Force: spec.Force}); err != nil {
		return errors.Wrapf(err, "docker.ContainerRemove failed, name: %s", name)
	}

	// 是否需要异步删除 etcd 中关于容器的描述和版本号记录
	if spec.DelEtcdInfoAndVersionRecord {
		vmap.ContainerVersionMap.Remove(strings.Split(name, "-")[0])
		workQueue.Queue <- etcd.DelKey{
			Resource: etcd.Containers,
			Key:      name,
		}
		log.Infof("service.DeleteContainer, container: %s will be del etcd info and version record", name)
	}
	log.Infof("service.DeleteContainer, container: %s delete successfully", name)
	return err
}

// ExecuteContainer 执行容器中的命令并返回输出
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
	log.Infof("service.ExecuteContainer, container: %s execute successfully, exec: %+v", name, exec)
	return
}

// PatchContainerGpuInfo 变更容器的 GPU 资源
// 关于容器即将变为无卡容器，还是变为有卡容器，全由入参中的 GpuCount 决定，你只需要告诉我你想要的 GPU 资源数量
// 例如 GPUCount 为 0，就是要将旧容器变为无卡容器，GPUCount 为 1，就是要将旧容器变为有卡容器
// 对于状态未发生变化，如：无卡变无卡，有卡容器 GPU 数量变更前后一致，会直接跳过，以提高效率
func (cs *ContainerService) PatchContainerGpuInfo(name string, spec *model.ContainerGpuPatch) (id, newContainerName string, err error) {
	// 从 etcd 中获取容器的描述
	ctx := context.Background()
	infoBytes, err := etcd.Get(etcd.Containers, name)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "etcd.Get failed")
	}
	var info model.EtcdContainerInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return id, newContainerName, errors.WithMessage(err, "json.Unmarshal failed")
	}

	// 只有 etcd 中的 Volume 对象的版本和要修改的 Volume 版本一致时，才能更新
	if strconv.FormatInt(info.Version, 10) != strings.Split(name, "-")[1] {
		return id, newContainerName, errors.Wrapf(xerrors.NewVersionNotMatchError(),
			"container: %s, etcd version: %d, patch version: %s", name, info.Version, strings.Split(name, "-")[1])
	}

	// 获取容器的 gpu 资源
	uuids, err := cs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "service.containerDeviceRequestsDeviceIDs failed")
	}

	// 当前容器使用的 gpu 资源和要 patch 的 gpu 资源相同
	if len(uuids) == spec.GpuCount {
		return id, newContainerName, errors.Wrapf(xerrors.NewNoPatchRequiredError(), "container: %s", name)
	}

	if spec.GpuCount > len(uuids) {
		// 升级配置
		applyGpus := spec.GpuCount - len(uuids)
		uuids, err := gpuscheduler.Scheduler.ApplyGpus(applyGpus)
		log.Infof("service.PatchContainerGpuInfo, container: %s apply %d gpus, uuids: %+v", name, applyGpus, uuids)
		if err != nil {
			return id, newContainerName, errors.WithMessage(err, "gpuscheduler.Scheduler.ApplyGpus failed")
		}
		if applyGpus == spec.GpuCount {
			// 之前是无卡容器，所以实际申请的 gpu 资源和 要升级的 gpu 资源相同
			info.HostConfig.Resources = cs.newContainerResource(uuids)
			log.Infof("service.PatchContainerGpuInfo, container: %s change to card container, now use %d gpus, uuids: %s",
				name, len(info.HostConfig.Resources.DeviceRequests[0].DeviceIDs), info.HostConfig.Resources.DeviceRequests[0].DeviceIDs)
		} else {
			// 之前不是无卡容器
			info.HostConfig.Resources.DeviceRequests[0].DeviceIDs = append(info.HostConfig.Resources.DeviceRequests[0].DeviceIDs, uuids...)
			log.Infof("service.PatchContainerGpuInfo, container: %s upgrad %d gpu configuration, now use %d gpus, uuids: %+v",
				name, applyGpus, len(info.HostConfig.Resources.DeviceRequests[0].DeviceIDs), info.HostConfig.Resources.DeviceRequests[0].DeviceIDs)
		}
	} else {
		// 降低配置或变为无卡容器
		restoreGpus := len(uuids) - spec.GpuCount
		gpuscheduler.Scheduler.RestoreGpus(uuids[:restoreGpus])
		log.Infof("service.PatchContainerGpuInfo, container: %s restore %d gpus, uuids: %+v",
			name, len(uuids[:restoreGpus]), uuids[:restoreGpus])
		if len(uuids[:spec.GpuCount]) == 0 {
			// 变为无卡容器
			info.HostConfig.Resources = container.Resources{}
			log.Infof("service.PatchContainerGpuInfo, container: %s change to cardless container", name)
		} else {
			// 降低配置
			info.HostConfig.Resources.DeviceRequests[0].DeviceIDs = uuids[restoreGpus:]
			log.Infof("service.PatchContainerGpuInfo, container: %s reduce %d gpu configuration, now use %d gpus, uuids: %+v",
				name, restoreGpus, len(uuids[:restoreGpus]), uuids[:restoreGpus])
		}
	}

	// 创建一个新的容器，用来替换旧的容器
	id, newContainerName, err = cs.runContainer(ctx, strings.Split(name, "-")[0], info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "service.runContainer failed")
	}

	// 异步拷贝旧容器的系统盘到新的容器
	workQueue.Queue <- &workQueue.CopyTask{
		Resource:    etcd.Containers,
		OldResource: info.ContainerName,
		NewResource: newContainerName,
	}
	log.Infof("service.PatchContainerGpuInfo, container: %s patch gpu info successfully", name)
	return
}

// PatchContainerVolumeInfo 变更容器的 Volume 资源
// 需要注意的是，这个方法只是会替换新容器绑定的 Volume 资源
// 例如 foo-0 容器绑定了 volume-0 到 /root/example 目录，现在要使用的是 volume-1，那么就会将 volume-0 替换成 volume-1，然后创建新容器
func (cs *ContainerService) PatchContainerVolumeInfo(name string, spec *model.ContainerVolumePatch) (id, newContainerName string, err error) {
	if spec.OldBind.Format() == spec.NewBind.Format() {
		return id, newContainerName, errors.Wrapf(xerrors.NewNoPatchRequiredError(), "container: %s", name)
	}

	// 从 etcd 中获取容器的描述
	ctx := context.Background()
	infoBytes, err := etcd.Get(etcd.Containers, name)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "etcd.Get failed")
	}

	var info model.EtcdContainerInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return id, newContainerName, errors.WithMessage(err, "json.Unmarshal failed")
	}

	// 只有 etcd 中的 Volume 对象的版本和要修改的 Volume 版本一致时，才能更新
	if strconv.FormatInt(info.Version, 10) != strings.Split(name, "-")[1] {
		return id, newContainerName, errors.Wrapf(xerrors.NewVersionNotMatchError(),
			"container: %s, etcd version: %d, patch version: %s", name, info.Version, strings.Split(name, "-")[1])
	}

	// 变更容器的绑定信息
	for i := range info.HostConfig.Binds {
		if info.HostConfig.Binds[i] == spec.OldBind.Format() {
			info.HostConfig.Binds[i] = spec.NewBind.Format()
			break
		}
	}

	// 创建一个新的容器，用来替换旧的容器
	id, newContainerName, err = cs.runContainer(ctx, strings.Split(name, "-")[0], info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "service.runContainer failed")
	}

	// 异步拷贝旧容器的系统盘到新的容器
	workQueue.Queue <- &workQueue.CopyTask{
		Resource:    etcd.Containers,
		OldResource: info.ContainerName,
		NewResource: newContainerName,
	}

	log.Infof("service.PatchContainerVolumeInfo, container: %s patch volume info successfully", name)
	return
}

// StopContainer 停止容器，会归还端口资源，如果是 GPU 容器，会归还使用的资源
func (cs *ContainerService) StopContainer(name string) error {
	// 归还 gpu 资源
	uuids, err := cs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return errors.WithMessage(err, "service.containerDeviceRequestsDeviceIDs failed")
	}
	gpuscheduler.Scheduler.RestoreGpus(uuids)
	log.Infof("service.StopContainer, container: %s restore %d gpus, uuids: %+v",
		name, len(uuids), uuids)

	// 归还端口资源
	ports, err := cs.containerPortBindings(name)
	if err != nil {
		return errors.WithMessage(err, "service.containerPortBindings failed")
	}
	portscheduler.Scheduler.RestorePorts(ports)

	// 停止容器
	ctx := context.Background()
	if err := docker.Cli.ContainerStop(ctx, name, container.StopOptions{}); err != nil {
		return errors.WithMessage(err, "docker.ContainerStop failed")
	}
	log.Infof("service.StopContainer, container: %s stop successfully", name)
	return nil
}

// RestartContainer 重启动容器
// 无卡容器会直接 docker restart
// 有卡容器会重新申请 GPU 资源，然后创建一个新的容器，用来替换旧的容器
func (cs *ContainerService) RestartContainer(name string) (id, newContainerName string, err error) {
	ctx := context.Background()
	// 获取上次启动时使用的 gpu 资源
	uuids, err := cs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "service.containerDeviceRequestsDeviceIDs failed")
	}
	if len(uuids) == 0 {
		// 停止的时候是无卡启动的，直接使用 docker restart 重启
		if err = docker.Cli.ContainerRestart(ctx, name, container.StopOptions{}); err != nil {
			return id, newContainerName, errors.Wrapf(err, "docker.ContainerRestart failed, name: %s", name)
		}
		resp, err := docker.Cli.ContainerInspect(ctx, name)
		if err != nil {
			return id, newContainerName, errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
		}

		id = resp.ID
		newContainerName = name
		log.Infof("service.RestartContainer, cardless container: %s restart successfully", name)
		return id, newContainerName, err
	}

	// 停止的时候是有卡启动的
	// 获取 etcd 中关于容器启动的描述
	infoBytes, err := etcd.Get(etcd.Containers, name)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "etcd.Get failed")
	}
	var info model.EtcdContainerInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return id, newContainerName, errors.WithMessage(err, "json.Unmarshal failed")
	}

	// 申请 gpu 资源
	availableGpus, err := gpuscheduler.Scheduler.ApplyGpus(len(uuids))
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "gpuscheduler.Scheduler.ApplyGpus failed")
	}
	log.Infof("service.RestartContainer, card container: %s apply %d gpus, uuids: %+v", name, len(availableGpus), availableGpus)

	info.HostConfig.Resources.DeviceRequests[0].DeviceIDs = availableGpus
	// 重启一个容器
	id, newContainerName, err = cs.runContainer(ctx, strings.Split(name, "-")[0], info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "service.runContainer failed")
	}
	// 异步拷贝旧容器的系统盘到新的容器
	workQueue.Queue <- &workQueue.CopyTask{
		Resource:    etcd.Containers,
		OldResource: info.ContainerName,
		NewResource: newContainerName,
	}

	log.Infof("service.RestartContainer, card container restart successfully, "+
		"old container name: %s, new container name: %s, "+
		"old gpu resources: %+v, new gpu resources: %+v",
		name, newContainerName,
		uuids, availableGpus)
	return
}

// CommitContainer 提交容器为镜像，镜像名称默认为镜像的 ID
func (cs *ContainerService) CommitContainer(name string, spec model.ContainerCommit) (imageName string, err error) {
	// 提交为镜像
	ctx := context.Background()
	resp, err := docker.Cli.ContainerCommit(ctx, name, types.ContainerCommitOptions{
		Comment: fmt.Sprintf("container name %s, commit time: %s", name, time.Now().Format("2006-01-02 15:04:05")),
	})
	if err != nil {
		return imageName, errors.WithMessage(err, "docker.ContainerRestart failed")
	}

	// 为镜像打标签
	if len(spec.NewImageName) != 0 {
		imageName = spec.NewImageName
	}
	if err = docker.Cli.ImageTag(ctx, resp.ID, imageName); err != nil {
		return imageName, errors.WithMessage(err, "docker.ImageTag failed")
	}
	log.Infof("service.CommitContainer, container: %s commit successfully", name)
	return imageName, err
}

// 真正创建容器和启动容器的方法，这个方法不区分是用来创建 GPU 容器还是普通容器，因为它只会根据入参来创建容器
// 用于创建容器、变更容器的 GPU 信息、变更容器的 Volume 信息、重启动 GPU 容器等
func (cs *ContainerService) runContainer(ctx context.Context, name string, info model.EtcdContainerInfo) (id, containerName string, err error) {
	// 传递到这个方法的容器名称也就是 name，会被去掉版本号
	// 例如调用创建容器接口，name 不应该携带 -，例如：foo-0 会报错并返回，应该是 foo
	// 变更容器的 GPU 信息时，传递的是 bar-0，那么会被去掉版本号，name 变成 bar
	// 所以下面代码的作用就是，判断 name 在 map 中是否存在，如果不存在，则加入到 map 中，如果存在，则版本号加 1
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		vmap.ContainerVersionMap.Set(name, 0)
	} else {
		vmap.ContainerVersionMap.Set(name, sync2.AtomicInt64(version.Add(1)))
	}

	defer func() {
		if err != nil {
			vmap.ContainerVersionMap.Set(name, sync2.AtomicInt64(version.Add(-1)))
		}
	}()

	// 生成此次要创建的容器的名称
	containerName = fmt.Sprintf("%s-%d", name, version)

	availableOSPorts, err := portscheduler.Scheduler.ApplyPorts(len(info.HostConfig.PortBindings))
	if err != nil {
		return id, containerName, errors.Wrapf(err, "portscheduler.ApplyPorts failed, info: %+v", info)
	}

	var index int
	for k := range info.HostConfig.PortBindings {
		info.HostConfig.PortBindings[k] = []nat.PortBinding{{
			HostPort: strconv.Itoa(availableOSPorts[index]),
		}}
		index++
	}
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
	workQueue.Queue <- etcd.PutKeyValue{
		Resource: etcd.Containers,
		Key:      containerName,
		Value:    val.Serialize(),
	}
	log.Infof("service.runContainer, container: %s run successfully", containerName)
	return
}

// 判断容器是否存在
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

// 获取容器使用的 GPU 列表 （UUID）
func (cs *ContainerService) containerDeviceRequestsDeviceIDs(name string) ([]string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.ContainerInspect(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
	}
	if resp.HostConfig.DeviceRequests == nil {
		return []string{}, nil
	}
	return resp.HostConfig.DeviceRequests[0].DeviceIDs, nil
}

func (cs *ContainerService) containerPortBindings(name string) ([]int, error) {
	ctx := context.Background()
	resp, err := docker.Cli.ContainerInspect(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
	}
	if resp.HostConfig.PortBindings == nil {
		return []int{}, nil
	}
	var ports []int
	for _, v := range resp.HostConfig.PortBindings {
		port, _ := strconv.Atoi(v[0].HostPort)
		ports = append(ports, port)
	}
	return ports, nil
}

func (cs *ContainerService) newContainerResource(uuids []string) container.Resources {
	return container.Resources{DeviceRequests: []container.DeviceRequest{{
		Driver:       "nvidia",
		DeviceIDs:    uuids,
		Capabilities: [][]string{{"gpu"}},
		Options:      nil,
	}}}
}
