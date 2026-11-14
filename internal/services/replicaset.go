package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/models"
	"github.com/mayooot/gpu-docker-api/internal/schedulers"
	vmap "github.com/mayooot/gpu-docker-api/internal/version"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
	"github.com/mayooot/gpu-docker-api/utils"
)

type ReplicaSetService struct{}

// RunGpuContainer just sets the parameters, the real run a container is in the `runContainer`
func (rs *ReplicaSetService) RunGpuContainer(spec *models.ContainerRun) (id, containerName string, err error) {
	var (
		config           container.Config
		hostConfig       container.HostConfig
		networkingConfig network.NetworkingConfig
		platform         ocispec.Platform
	)
	ctx := context.Background()

	if rs.existContainer(spec.ReplicaSetName) {
		return id, containerName, errors.Wrapf(xerrors.NewContainerExistedError(), "container %s", spec.ReplicaSetName)
	}

	config = container.Config{
		Image:     spec.ImageName,
		Cmd:       spec.Cmd,
		Env:       spec.Env,
		OpenStdin: true,
		Tty:       true,
	}

	// bind port
	if len(spec.ContainerPorts) > 0 {
		hostConfig.PortBindings = make(nat.PortMap, len(spec.ContainerPorts))
		config.ExposedPorts = make(nat.PortSet, len(spec.ContainerPorts))
		for _, port := range spec.ContainerPorts {
			config.ExposedPorts[nat.Port(port+"/tcp")] = struct{}{}
			hostConfig.PortBindings[nat.Port(port+"/tcp")] = nil
		}
	}

	// bind gpu resource
	if spec.GpuCount > 0 {
		uuids, err := schedulers.GpuScheduler.Apply(spec.GpuCount)
		if err != nil {
			return id, containerName, errors.Wrapf(err, "GpuScheduler.Apply failed, spec: %+v", spec)
		}
		hostConfig.Resources = rs.newContainerResource(uuids)
		log.Infof("services.RunGpuContainer, container: %s apply %d gpus, uuids: %+v", spec.ReplicaSetName+"-0", len(uuids), uuids)
	}

	// bind volume
	hostConfig.Binds = make([]string, 0, len(spec.Binds))
	for i := range spec.Binds {
		// Binds
		hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", spec.Binds[i].Src, spec.Binds[i].Dest))
	}

	// create and start
	id, containerName, kv, err := rs.runContainer(ctx, spec.ReplicaSetName, &models.EtcdContainerInfo{
		Config:           &config,
		HostConfig:       &hostConfig,
		NetworkingConfig: &networkingConfig,
		Platform:         &platform,
	})
	if err != nil {
		return id, containerName, errors.Wrapf(err, "serivce.runContainer failed, spec: %+v", spec)
	}

	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Containers,
		Key:      kv.Key,
		Value:    kv.Value,
	})
	return
}

func (rs *ReplicaSetService) DeleteContainer(name string) error {
	// get the latest version number
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		return errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}

	ctrVersionName := fmt.Sprintf("%s-%d", name, version)

	uuids, err := rs.containerDeviceRequestsDeviceIDs(ctrVersionName)
	if err != nil {
		return errors.WithMessage(err, "services.containerDeviceRequestsDeviceIDs failed")
	}
	schedulers.GpuScheduler.Restore(uuids)

	ports, err := rs.containerPortBindings(ctrVersionName)
	if err != nil {
		return errors.WithMessage(err, "services.containerPortBindings failed")
	}
	schedulers.PortScheduler.Restore(ports)

	// delete the version number and asynchronously delete the container info in etcd.
	// name is already the replicaSet name, splitting it on "-" broke names like "my-model".
	vmap.ContainerVersionMap.Remove(name)
	workQueue.Enqueue(etcd.DelKey{
		Resource: etcd.Containers,
		Key:      name,
	})

	err = docker.Cli.ContainerRemove(context.TODO(),
		fmt.Sprintf("%s-%d", name, version),
		types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return errors.WithMessage(err, "docker.Cli.ContainerRemove failed")
	}

	log.Infof("services.DeleteContainer, container: %s delete successfully", fmt.Sprintf("%s-%d", name, version))
	log.Infof("services.DeleteContainer, container: %s will be del etcd info and version record", name)
	return nil
}

func (rs *ReplicaSetService) ExecuteContainer(name string, exec *models.ContainerExecute) (resp *string, err error) {
	// get the latest version number
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		return nil, errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}

	workDir := "/"
	var cmd []string
	if len(exec.WorkDir) != 0 {
		workDir = exec.WorkDir
	}
	if len(exec.Cmd) != 0 {
		cmd = exec.Cmd
	}

	ctx := context.Background()
	execCreate, err := docker.Cli.ContainerExecCreate(ctx, fmt.Sprintf("%s-%d", name, version), types.ExecConfig{
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
	if err != nil {
		return resp, errors.Wrapf(err, "docker.ContainerExecAttach failed, name: %s, spec: %+v", name, exec)
	}
	// Close must be deferred after the error check: on failure hijackedResp.Conn is nil
	// and Close would panic.
	defer hijackedResp.Close()

	var buf bytes.Buffer
	_, _ = stdcopy.StdCopy(&buf, &buf, hijackedResp.Reader)
	str := buf.String()
	resp = &str
	log.Infof("services.ExecuteContainer, container: %s execute successfully, exec: %+v", name, exec)
	return
}

func (rs *ReplicaSetService) PatchContainer(name string, spec *models.PatchRequest) (id, newContainerName string, err error) {
	// get the latest version number
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		return id, newContainerName, errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}
	ctrVersionName := fmt.Sprintf("%s-%d", name, version)

	// get the container info
	ctx := context.Background()
	infoBytes, err := etcd.GetValue(etcd.Containers, name)
	if err != nil {
		return id, newContainerName, errors.Wrapf(err, "etcd.GetValue failed, key: %s", etcd.ResourcePrefix(etcd.Containers, name))
	}
	info := &models.EtcdContainerInfo{}
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return id, newContainerName, errors.WithMessage(err, "json.Unmarshal failed")
	}

	// update gpu info
	info, err = rs.patchGpu(ctrVersionName, spec.GpuPatch, info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "patchGpu failed")
	}

	// update volume info
	info, err = rs.patchVolume(spec.VolumePatch, info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "patchVolume failed")
	}

	// create a new container to replace the old one
	id, newContainerName, kv, err := rs.runContainer(ctx, name, info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "runContainer failed")
	}

	// copy the old container's merged files to the new container
	err = utils.CopyOldMergedToNewContainerMerged(info.ContainerName, newContainerName)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "utils.CopyOldMergedToNewContainerMerged failed")
	}

	// delete the old container
	// no gpu resources are returned because they are already returned when the gpu is lowered
	// or when upgrading the gpu, the original gpu will be used.
	err = setToMergeMap(ctrVersionName, version)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "setToMergeMap failed")
	}
	err = rs.DeleteContainerForUpdate(ctrVersionName)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "DeleteContainerForUpdate failed")
	}

	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Containers,
		Key:      kv.Key,
		Value:    kv.Value,
	})

	log.Infof("services.PatchContainer, container: %s patch configuration successfully", name)
	return
}

func (rs *ReplicaSetService) RollbackContainer(name string, spec *models.RollbackRequest) (string, error) {
	// check that the version to be rolled back is the same as the current version
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		return "", errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}
	if spec.Version == version {
		return "", xerrors.NewNoRollbackRequiredError()
	}

	// get revision info form etcd
	value, err := etcd.GetRevision(etcd.Containers, name, spec.Version)
	if err != nil {
		return "", errors.WithMessage(err, "etcd.GetRevisionRange failed")
	}
	info := &models.EtcdContainerInfo{}
	if err = json.Unmarshal(value, &info); err != nil {
		return "", errors.WithMessage(err, "json.Unmarshal failed")
	}

	// compare gpu info
	ctrVersionName := fmt.Sprintf("%s-%d", name, version)
	targetGpuCount := 0
	if len(info.HostConfig.Resources.DeviceRequests) > 0 {
		// the target version may be a cardless container, indexing [0] directly would panic
		targetGpuCount = len(info.HostConfig.Resources.DeviceRequests[0].DeviceIDs)
	}
	// The DeviceIDs recorded in the target version are historical and may already be
	// allocated to other containers. Start from the gpus currently held by this replicaSet,
	// then grow or shrink to the target count.
	currentUUIDs, err := rs.containerDeviceRequestsDeviceIDs(ctrVersionName)
	if err != nil {
		return "", errors.WithMessage(err, "services.containerDeviceRequestsDeviceIDs failed")
	}
	if len(currentUUIDs) > 0 {
		info.HostConfig.Resources = rs.newContainerResource(currentUUIDs)
	} else {
		info.HostConfig.Resources = container.Resources{}
	}
	info, err = rs.patchGpu(ctrVersionName, &models.GpuPatch{
		GpuCount: targetGpuCount,
	}, info)
	if err != nil {
		return "", errors.WithMessage(err, "patchGpu failed")
	}

	// create a new container to replace the old one
	_, newContainerName, kv, err := rs.runContainer(context.TODO(), name, info)
	if err != nil {
		return "", errors.WithMessage(err, "runContainer failed")
	}

	// copy the target version's merged files to the new container.
	// info.ContainerName is the versioned name of the target (e.g. "foo-2"), which is the key
	// used by setToMergeMap. Keying by the bare version number mixed up different replicaSets.
	src, ok := vmap.ContainerMergeMap.Get(info.ContainerName)
	if !ok {
		return "", errors.Errorf("container: %s version: %d merge path not found in ContainerMergeMap", info.ContainerName, info.Version)
	}
	dest, err := utils.GetContainerMergedLayer(newContainerName)
	if err != nil {
		return "", errors.WithMessage(err, "utils.GetContainerMergedLayer failed")
	}

	err = utils.CopyDir(src, dest)
	if err != nil {
		return "", errors.WithMessage(err, "utils.CopyOldMergedToNewContainerMerged failed")
	}

	// delete the old container
	// no gpu resources are returned because they are already returned when the gpu is lowered
	// or when upgrading the gpu, the original gpu will be used.
	err = setToMergeMap(ctrVersionName, version)
	if err != nil {
		return "", errors.WithMessage(err, "setToMergeMap failed")
	}
	err = rs.DeleteContainerForUpdate(ctrVersionName)
	if err != nil {
		return "", errors.WithMessage(err, "DeleteContainerForUpdate failed")
	}

	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Containers,
		Key:      kv.Key,
		Value:    kv.Value,
	})

	log.Infof("services.RollbackContainer, container: %s patch configuration successfully", ctrVersionName)
	return newContainerName, nil
}

func (rs *ReplicaSetService) patchGpu(name string, spec *models.GpuPatch, info *models.EtcdContainerInfo) (*models.EtcdContainerInfo, error) {
	if spec == nil {
		return info, nil
	}
	uuids, err := rs.containerDeviceRequestsDeviceIDs(name)
	if err != nil {
		return info, errors.WithMessage(err, "services.containerDeviceRequestsDeviceIDs failed")
	}

	if len(uuids) == spec.GpuCount {
		return info, nil
	}

	if spec.GpuCount > len(uuids) {
		// lift gpu configuration
		applyGpus := spec.GpuCount - len(uuids)
		uuids, err := schedulers.GpuScheduler.Apply(applyGpus)
		log.Infof("services.PatchContainerGpuInfo, container: %s apply %d gpus, uuids: %+v", name, applyGpus, uuids)
		if err != nil {
			return info, errors.WithMessage(err, "GpuScheduler.Apply failed")
		}
		if applyGpus == spec.GpuCount {
			// no gpu was used before.
			info.HostConfig.Resources = rs.newContainerResource(uuids)
			log.Infof("services.PatchContainerGpuInfo, container: %s change to card container, now use %d gpus, uuids: %s",
				name, len(info.HostConfig.Resources.DeviceRequests[0].DeviceIDs), info.HostConfig.Resources.DeviceRequests[0].DeviceIDs)
		} else {
			// before using gpu
			info.HostConfig.Resources.DeviceRequests[0].DeviceIDs = append(info.HostConfig.Resources.DeviceRequests[0].DeviceIDs, uuids...)
			log.Infof("services.PatchContainerGpuInfo, container: %s upgrad %d gpu configuration, now use %d gpus, uuids: %+v",
				name, applyGpus, len(info.HostConfig.Resources.DeviceRequests[0].DeviceIDs), info.HostConfig.Resources.DeviceRequests[0].DeviceIDs)
		}
	} else {
		restoreGpus := len(uuids) - spec.GpuCount
		schedulers.GpuScheduler.Restore(uuids[:restoreGpus])
		log.Infof("services.PatchContainerGpuInfo, container: %s restore %d gpus, uuids: %+v",
			name, len(uuids[:restoreGpus]), uuids[:restoreGpus])
		if len(uuids[:spec.GpuCount]) == 0 {
			// change to no using gpu
			info.HostConfig.Resources = container.Resources{}
			log.Infof("services.PatchContainerGpuInfo, container: %s change to cardless container", name)
		} else {
			// lower gpu configuration
			info.HostConfig.Resources.DeviceRequests[0].DeviceIDs = uuids[restoreGpus:]
			log.Infof("services.PatchContainerGpuInfo, container: %s reduce %d gpu configuration, now use %d gpus, uuids: %+v",
				name, restoreGpus, len(uuids[:restoreGpus]), uuids[:restoreGpus])
		}
	}

	return info, nil
}

func (rs *ReplicaSetService) patchVolume(spec *models.VolumePatch, info *models.EtcdContainerInfo) (*models.EtcdContainerInfo, error) {
	if spec == nil {
		return info, nil
	}

	if spec.OldBind.Format() == spec.NewBind.Format() {
		return info, nil
	}

	for i := range info.HostConfig.Binds {
		if info.HostConfig.Binds[i] == spec.OldBind.Format() {
			info.HostConfig.Binds[i] = spec.NewBind.Format()
			break
		}
	}
	return info, nil
}

func (rs *ReplicaSetService) StopContainer(name string, restoreGpu, restorePort, isLatest bool) error {
	if isLatest {
		// get the latest version number
		version, ok := vmap.ContainerVersionMap.Get(name)
		if !ok {
			return errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
		}
		name = fmt.Sprintf("%s-%d", name, version)
	}

	// whether to restore gpu resources
	if restoreGpu {
		uuids, err := rs.containerDeviceRequestsDeviceIDs(name)
		if err != nil {
			return errors.WithMessage(err, "services.containerDeviceRequestsDeviceIDs failed")
		}
		schedulers.GpuScheduler.Restore(uuids)
		log.Infof("services.StopContainer, container: %s restore %d gpus, uuids: %+v",
			name, len(uuids), uuids)
	}

	// whether to restore port resources
	if restorePort {
		ports, err := rs.containerPortBindings(name)
		if err != nil {
			return errors.WithMessage(err, "services.containerPortBindings failed")
		}
		schedulers.PortScheduler.Restore(ports)
		log.Infof("services.StopContainer, container: %s restore %d ports: %+v",
			name, len(ports), ports)
	}

	// stop container
	ctx := context.Background()
	if err := docker.Cli.ContainerStop(ctx, name, container.StopOptions{}); err != nil {
		return errors.WithMessage(err, "docker.ContainerStop failed")
	}

	log.Infof("services.StopContainer, container: %s stop successfully", name)
	return nil
}

func (rs *ReplicaSetService) DeleteContainerForUpdate(name string) error {
	// restore port resources
	ports, err := rs.containerPortBindings(name)
	if err != nil {
		return errors.WithMessage(err, "services.containerPortBindings failed")
	}
	schedulers.PortScheduler.Restore(ports)
	log.Infof("services.DeleteContainerForUpdate, container: %s restore %d ports: %+v",
		name, len(ports), ports)

	// delete container
	err = docker.Cli.ContainerRemove(context.TODO(),
		name,
		types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return errors.WithMessage(err, "docker.ContainerRemove failed")
	}

	return nil
}

func setToMergeMap(name string, version int64) error {
	var err error
	defer func() {
		if err != nil {
			vmap.ContainerMergeMap.Remove(name)
		}
	}()

	mergedDir, err := utils.GetContainerMergedLayer(name)
	if err != nil {
		return errors.WithMessagef(err, "utils.GetContainerMergedLayer failed, container: %s", name)
	}
	layer := "merges"
	dir, _ := os.Getwd()
	path := filepath.Join(dir, layer, trimVersionSuffix(name), name)
	_ = os.MkdirAll(path, 0755)

	err = utils.CopyDir(mergedDir, path)
	if err != nil {
		return errors.WithMessagef(err, "utils.CopyDir failed, container: %s, version: %d", name, version)
	}
	vmap.ContainerMergeMap.Set(name, path)
	return nil
}

func (rs *ReplicaSetService) StartupContainer(name string) error {
	// get the latest version number
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		return errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}

	err := docker.Cli.ContainerRestart(context.TODO(),
		fmt.Sprintf("%s-%d", name, version),
		container.StopOptions{})
	if err != nil {
		return errors.WithMessagef(err, "docker.ContainerRestart failed, name: %s", name)
	}

	return nil
}

// RestartContainer will reapply gpu and port,
// but the logic for applying port is in the runContainer function
func (rs *ReplicaSetService) RestartContainer(name string) (id, newContainerName string, err error) {
	// get the latest version number
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		return id, newContainerName, errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}
	ctrVersionName := fmt.Sprintf("%s-%d", name, version)

	// get info about used gpus
	ctx := context.Background()
	uuids, err := rs.containerDeviceRequestsDeviceIDs(ctrVersionName)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "services.containerDeviceRequestsDeviceIDs failed")
	}

	// get creation info from etcd
	infoBytes, err := etcd.GetValue(etcd.Containers, name)
	if err != nil {
		return id, newContainerName, errors.Wrapf(err, "etcd.GetValue failed, key: %s", etcd.ResourcePrefix(etcd.Containers, name))
	}
	info := &models.EtcdContainerInfo{}
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return id, newContainerName, errors.WithMessage(err, "json.Unmarshal failed")
	}

	// check whether the container is using gpu
	if len(uuids) != 0 {
		// apply for gpu
		availableGpus, err := schedulers.GpuScheduler.Apply(len(uuids))
		if err != nil {
			return id, newContainerName, errors.WithMessage(err, "GpuScheduler.Apply failed")
		}
		log.Infof("services.RestartContainer, container: %s apply %d gpus, uuids: %+v", ctrVersionName, len(availableGpus), availableGpus)
		info.HostConfig.Resources.DeviceRequests[0].DeviceIDs = availableGpus
	}

	//  create a container to replace the old one
	id, newContainerName, kv, err := rs.runContainer(ctx, name, info)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "services.runContainer failed")
	}

	// copy the old container's merged files to the new container
	err = utils.CopyOldMergedToNewContainerMerged(info.ContainerName, newContainerName)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "utils.CopyOldMergedToNewContainerMerged failed")
	}

	// delete the old container
	// no gpu resources are returned because they are already returned when the gpu is lowered
	// or when upgrading the gpu, the original gpu will be used.
	err = setToMergeMap(ctrVersionName, version)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "setToMergeMap failed")
	}
	err = rs.DeleteContainerForUpdate(ctrVersionName)
	if err != nil {
		return id, newContainerName, errors.WithMessage(err, "DeleteContainerForUpdate failed")
	}

	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Containers,
		Key:      kv.Key,
		Value:    kv.Value,
	})

	log.Infof("services.RestartContainer, container restart successfully, "+
		"old container name: %s, new container name: %s, "+
		ctrVersionName, newContainerName)
	return
}

func (rs *ReplicaSetService) CommitContainer(name string, spec models.ContainerCommit) (imageName string, err error) {
	// get the latest version number
	version, ok := vmap.ContainerVersionMap.Get(name)
	if !ok {
		return imageName, errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}

	// commit image
	ctx := context.Background()
	resp, err := docker.Cli.ContainerCommit(ctx, fmt.Sprintf("%s-%d", name, version), types.ContainerCommitOptions{
		Comment: fmt.Sprintf("container name %s, commit time: %s", fmt.Sprintf("%s-%d", name, version), time.Now().Format("2006-01-02 15:04:05")),
	})
	if err != nil {
		return imageName, errors.WithMessage(err, "docker.ContainerRestart failed")
	}

	// tag
	if len(spec.NewImageName) != 0 {
		imageName = spec.NewImageName
	}
	if err = docker.Cli.ImageTag(ctx, resp.ID, imageName); err != nil {
		return imageName, errors.WithMessage(err, "docker.ImageTag failed")
	}
	log.Infof("services.CommitContainer, container: %s commit successfully", fmt.Sprintf("%s-%d", name, version))
	return imageName, err
}

func (rs *ReplicaSetService) GetContainerInfo(name string) (info models.EtcdContainerInfo, err error) {
	infoBytes, err := etcd.GetValue(etcd.Containers, name)
	if err != nil {
		return info, errors.Wrapf(err, "etcd.GetValue failed, key: %s", etcd.ResourcePrefix(etcd.Containers, name))
	}

	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return info, errors.WithMessage(err, "json.Unmarshal failed")
	}
	return
}

func (rs *ReplicaSetService) GetContainerHistory(name string) ([]*models.ContainerHistoryItem, error) {
	replicaSet, err := etcd.GetRevisionRange(etcd.Containers, name)
	if err != nil {
		return nil, errors.Wrapf(err, "etcd.GetRevisionRange failed, key: %s",
			etcd.ResourcePrefix(etcd.Containers, name))
	}

	resp := make([]*models.ContainerHistoryItem, 0, len(replicaSet))
	for _, combine := range replicaSet {
		var info models.EtcdContainerInfo
		err := json.Unmarshal(combine.Value, &info)
		if err != nil {
			return nil, errors.Wrapf(err, "json.Unmarshal failed, value: %s", combine.Value)
		}
		resp = append(resp, &models.ContainerHistoryItem{
			Version:    combine.Version,
			CreateTime: info.CreateTime,
			Status:     info,
		})
	}
	return resp, nil
}

// It will only be executed based on the `docker.client.ContainerCreate`
func (rs *ReplicaSetService) runContainer(ctx context.Context, name string, info *models.EtcdContainerInfo) (string, string, etcd.PutKeyValue, error) {
	// set the version number
	version := vmap.ContainerVersionMap.Next(name)

	// add the version number to the env
	isExist := false
	for i := range info.Config.Env {
		if strings.HasPrefix(info.Config.Env[i], "CONTAINER_VERSION=") {
			isExist = true
			info.Config.Env[i] = fmt.Sprintf("CONTAINER_VERSION=%d", version)
			break
		}
	}
	if !isExist {
		info.Config.Env = append(info.Config.Env, fmt.Sprintf("CONTAINER_VERSION=%d", version))
	}

	var (
		err          error
		appliedPorts []string
	)
	defer func() {
		// if run container failed, roll back the version number and release the ports.
		// Earlier versions declared err with := inside the port block, so a port failure
		// skipped this rollback, and ports applied before a create/start failure were leaked.
		if err != nil {
			vmap.ContainerVersionMap.Revert(name, version)
			if len(appliedPorts) > 0 {
				schedulers.PortScheduler.Restore(appliedPorts)
			}
		}
	}()

	// apply for some host port
	if len(info.HostConfig.PortBindings) > 0 {
		appliedPorts, err = schedulers.PortScheduler.Apply(len(info.HostConfig.PortBindings))
		if err != nil {
			return "", "", etcd.PutKeyValue{}, errors.Wrapf(err, "Portscheduler.Apply failed, info: %+v", info)
		}
		var index int
		for k := range info.HostConfig.PortBindings {
			info.HostConfig.PortBindings[k] = []nat.PortBinding{{
				HostPort: appliedPorts[index],
			}}
			index++
		}
	}

	// generate container name with version and save creation time
	ctrVersionName := fmt.Sprintf("%s-%d", name, version)
	info.CreateTime = time.Now().Format("2006-01-02 15:04:05")

	// create container
	resp, err := docker.Cli.ContainerCreate(ctx, info.Config, info.HostConfig, info.NetworkingConfig, info.Platform, ctrVersionName)
	if err != nil {
		return "", "", etcd.PutKeyValue{}, errors.Wrapf(err, "docker.ContainerCreate failed, name: %s", ctrVersionName)
	}

	// start container
	if err = docker.Cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		_ = docker.Cli.ContainerRemove(ctx,
			resp.ID,
			types.ContainerRemoveOptions{Force: true})
		return "", "", etcd.PutKeyValue{}, errors.Wrapf(err, "docker.ContainerStart failed, id: %s, name: %s", resp.ID, ctrVersionName)
	}

	// creation info is added to etcd asynchronously
	val := &models.EtcdContainerInfo{
		Config:           info.Config,
		HostConfig:       info.HostConfig,
		NetworkingConfig: info.NetworkingConfig,
		Platform:         info.Platform,
		ContainerName:    ctrVersionName,
		Version:          version,
		CreateTime:       info.CreateTime,
	}

	log.Infof("services.runContainer, container: %s run successfully", ctrVersionName)
	return resp.ID,
		ctrVersionName,
		etcd.PutKeyValue{
			Resource: etcd.Containers,
			Key:      name,
			Value:    val.Serialize(),
		},
		nil
}

// Check whether the container exists
func (rs *ReplicaSetService) existContainer(name string) bool {
	ctx := context.Background()
	list, err := docker.Cli.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: fmt.Sprintf("^%s-", name)}),
	})
	if err != nil || len(list) == 0 {
		return false
	}

	return len(list) > 0
}

func (rs *ReplicaSetService) containerDeviceRequestsDeviceIDs(name string) ([]string, error) {
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

func (rs *ReplicaSetService) containerPortBindings(name string) ([]string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.ContainerInspect(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
	}
	if resp.HostConfig.PortBindings == nil {
		return []string{}, nil
	}
	var ports []string
	for _, v := range resp.HostConfig.PortBindings {
		ports = append(ports, v[0].HostPort)
	}
	return ports, nil
}

func (rs *ReplicaSetService) newContainerResource(uuids []string) container.Resources {
	return container.Resources{DeviceRequests: []container.DeviceRequest{{
		Driver:       "nvidia",
		DeviceIDs:    uuids,
		Capabilities: [][]string{{"gpu"}},
		Options:      nil,
	}}}
}
