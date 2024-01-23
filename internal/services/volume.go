package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/ngaut/log"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/models"
	vmap "github.com/mayooot/gpu-docker-api/internal/version"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
	"github.com/mayooot/gpu-docker-api/utils"
)

type VolumeService struct{}

func (vs *VolumeService) CreateVolume(spec *models.VolumeCreate) (resp volume.Volume, err error) {
	ctx := context.Background()
	if vs.existVolume(spec.Name) {
		return resp, errors.Wrapf(xerrors.NewVolumeExistedError(), "volume %s", spec.Name)
	}

	opt := volume.CreateOptions{Driver: "local"}
	if len(spec.Name) != 0 {
		opt.Name = spec.Name
	}
	if len(spec.Size) != 0 {
		opt.DriverOpts = map[string]string{"size": spec.Size}
	}

	resp, kv, err := vs.createVolume(ctx, spec.Name, models.EtcdVolumeInfo{Opt: &opt})
	if err != nil {
		return resp, errors.WithMessage(err, "services.createVolume failed")
	}

	workQueue.Queue <- etcd.PutKeyValue{
		Resource: etcd.Volumes,
		Key:      kv.Key,
		Value:    kv.Value,
	}
	return
}

// It will only be executed based on the `docker.client.ContainerCreate`
func (vs *VolumeService) createVolume(ctx context.Context, name string, info models.EtcdVolumeInfo) (resp volume.Volume, kv etcd.PutKeyValue, err error) {
	// set the version number
	version, _ := vmap.VolumeVersionMap.Get(name)
	version = version + 1
	vmap.VolumeVersionMap.Set(name, version)

	defer func() {
		// if run container failed, clear the version number
		if err != nil {
			if version == 1 {
				vmap.VolumeVersionMap.Remove(name)
			} else {
				vmap.VolumeVersionMap.Set(name, version-1)
			}
		}
	}()

	// generate name and save creation time
	info.Opt.Name = fmt.Sprintf("%s-%d", name, version)
	info.CreateTime = time.Now().Format("2006-01-02 15:04:05")

	// create volume
	resp, err = docker.Cli.VolumeCreate(ctx, *info.Opt)
	if err != nil {
		return resp, kv, errors.Wrapf(err, "docker.VolumeCreate failed, opt: %+v", info)
	}

	// creation info is added to etcd asynchronously
	val := &models.EtcdVolumeInfo{
		Opt:        info.Opt,
		Version:    version,
		CreateTime: info.CreateTime,
	}
	kv = etcd.PutKeyValue{
		Resource: etcd.Volumes,
		Key:      name,
		Value:    val.Serialize(),
	}

	log.Infof("serivce.createVolume, volume created successfully, name: %s, opt: %+v, version: %d", resp.Name, *info.Opt, info.Version)
	return
}

func (vs *VolumeService) PatchVolumeSize(name string, spec *models.VolumeSize) (resp volume.Volume, err error) {
	// get the latest version number
	version, ok := vmap.VolumeVersionMap.Get(name)
	if !ok {
		return resp, errors.Errorf("container: %s version: %d not found in ContainerVersionMap", name, version)
	}
	volVersionName := fmt.Sprintf("%s-%d", name, version)

	ctx := context.Background()
	infoBytes, err := etcd.GetValue(etcd.Volumes, name)
	if err != nil {
		return resp, errors.Wrapf(err, "etcd.GetValue failed, key: %s", etcd.ResourcePrefix(etcd.Containers, name))
	}
	var info models.EtcdVolumeInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return resp, errors.WithMessage(err, "json.Unmarshal failed")
	}

	preSize := info.Opt.DriverOpts["size"]
	preSizeBytes, _ := utils.ToBytes(preSize)
	patchSize := spec.Size
	patchSizeBytes, _ := utils.ToBytes(patchSize)

	if patchSize == preSize {
		return resp, errors.Wrapf(xerrors.NewNoPatchRequiredError(), "volume: %s", volVersionName)
	}

	// check whether the size after shrink is larger than used size
	if patchSizeBytes < preSizeBytes {
		mountpoint, err := utils.GetVolumeMountPoint(volVersionName)
		if err != nil {
			return resp, errors.WithMessage(err, "services.volumeMountpoint failed")
		}
		usedSize, err := utils.DirSize(mountpoint)
		if err != nil {
			return resp, errors.Wrapf(err, "utils.DirSize failed, volume: %s, mountpoint: %s", volVersionName, mountpoint)
		}

		if usedSize > patchSizeBytes {
			return resp, errors.Wrapf(xerrors.NewVolumeSizeUsedGreaterThanReduced(),
				"volume: %s, usedSize: %d, patchSize: %d", volVersionName, usedSize, patchSizeBytes)
		}
	}

	info.Opt.DriverOpts["size"] = patchSize

	// create a new volume to replace the old one
	resp, kv, err := vs.createVolume(ctx, name, info)
	if err != nil {
		return resp, errors.WithMessage(err, "services.createVolume failed")
	}

	err = utils.CopyOldMountPointToContainerMountPoint(resp.Name, resp.Name)
	if err != nil {
		return resp, errors.WithMessage(err, "utils.CopyOldMergedToNewContainerMerged failed")
	}

	// delete the old volume
	err = vs.DeleteVolume(volVersionName, false, false)
	if err != nil {
		return resp, errors.WithMessage(err, "services.DeleteVolume failed")
	}

	workQueue.Queue <- etcd.PutKeyValue{
		Resource: etcd.Volumes,
		Key:      kv.Key,
		Value:    kv.Value,
	}

	log.Infof("services.PatchVolumeSize, volume size patched successfully, old name: %s, old size: %s, new name: %s, new size: %s",
		name, preSize, resp.Name, patchSize)
	return
}

// DeleteVolume deletes a specific version of volume or the latest version of volume.
// If deleteRecord is true, etcd info about this volume and VolumeVersionMap record are deleted.
func (vs *VolumeService) DeleteVolume(name string, isLatest, deleteRecord bool) error {
	if isLatest {
		// get the last version number
		version, ok := vmap.VolumeVersionMap.Get(name)
		if !ok {
			return errors.Errorf("volume: %s version: %d not found in VolumeVersionMap", name, version)
		}
		name = fmt.Sprintf("%s-%d", name, version)
	}
	if deleteRecord {
		log.Infof("services.DeleteVolume, volume: %s will be del etcd info and version record", name)
		vmap.VolumeVersionMap.Remove(strings.Split(name, "-")[0])
		workQueue.Queue <- etcd.DelKey{
			Resource: etcd.Volumes,
			Key:      name,
		}
	}

	err := docker.Cli.VolumeRemove(context.TODO(), name, true)
	if err != nil {
		return errors.WithMessage(err, "docker.VolumeRemove failed")
	}

	log.Infof("services.DeleteVolume, volume deleted successfully, name: %s", name)
	return nil
}

func (vs *VolumeService) GetVolumeInfo(name string) (info models.EtcdVolumeInfo, err error) {
	infoBytes, err := etcd.GetValue(etcd.Volumes, name)
	if err != nil {
		return info, errors.Wrapf(err, "etcd.GetValue failed, key: %s", etcd.ResourcePrefix(etcd.Containers, name))
	}

	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return info, errors.WithMessage(err, "json.Unmarshal failed")
	}
	return
}

func (vs *VolumeService) GetVolumeHistory(name string) ([]*models.VolumeHistoryItem, error) {
	replicaSet, err := etcd.GetRevisionRange(etcd.Volumes, name)
	if err != nil {
		return nil, errors.Wrapf(err, "etcd.GetRevisionRange failed, key: %s",
			etcd.ResourcePrefix(etcd.Volumes, name))
	}

	resp := make([]*models.VolumeHistoryItem, 0, len(replicaSet))
	for _, combine := range replicaSet {
		var info models.EtcdVolumeInfo
		err := json.Unmarshal(combine.Value, &info)
		if err != nil {
			return nil, errors.Wrapf(err, "json.Unmarshal failed, value: %s", combine.Value)
		}
		resp = append(resp, &models.VolumeHistoryItem{
			Version:    combine.Version,
			CreateTime: info.CreateTime,
			Status:     info,
		})
	}

	return resp, nil
}

func (vs *VolumeService) existVolume(name string) bool {
	ctx := context.Background()
	list, err := docker.Cli.VolumeList(ctx, volume.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: fmt.Sprintf("^%s-", name)}),
	})
	if err != nil || len(list.Volumes) == 0 {
		return false
	}

	return len(list.Volumes) > 0
}
