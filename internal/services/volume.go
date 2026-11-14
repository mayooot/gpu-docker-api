package services

import (
	"context"
	"encoding/json"
	"fmt"
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

	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Volumes,
		Key:      kv.Key,
		Value:    kv.Value,
	})
	return
}

// It will only be executed based on the `docker.client.ContainerCreate`
func (vs *VolumeService) createVolume(ctx context.Context, name string, info models.EtcdVolumeInfo) (resp volume.Volume, kv etcd.PutKeyValue, err error) {
	// set the version number
	version := vmap.VolumeVersionMap.Next(name)

	defer func() {
		// if create volume failed, roll back the version number
		if err != nil {
			vmap.VolumeVersionMap.Revert(name, version)
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
		return resp, errors.Errorf("volume: %s version: %d not found in VolumeVersionMap", name, version)
	}
	volVersionName := fmt.Sprintf("%s-%d", name, version)

	ctx := context.Background()
	infoBytes, err := etcd.GetValue(etcd.Volumes, name)
	if err != nil {
		return resp, errors.Wrapf(err, "etcd.GetValue failed, key: %s", etcd.ResourcePrefix(etcd.Volumes, name))
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

	// copy data from the old volume (e.g. foo-1) to the new one (e.g. foo-2).
	// Earlier versions passed resp.Name for both arguments, so the data was never copied.
	err = utils.CopyOldMountPointToContainerMountPoint(volVersionName, resp.Name)
	if err != nil {
		return resp, errors.WithMessage(err, "utils.CopyOldMountPointToContainerMountPoint failed")
	}

	// delete the old volume
	err = vs.DeleteVolume(volVersionName, false, false)
	if err != nil {
		return resp, errors.WithMessage(err, "services.DeleteVolume failed")
	}

	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Volumes,
		Key:      kv.Key,
		Value:    kv.Value,
	})

	log.Infof("services.PatchVolumeSize, volume size patched successfully, old name: %s, old size: %s, new name: %s, new size: %s",
		name, preSize, resp.Name, patchSize)
	return
}

// DeleteVolume deletes a specific version of volume or the latest version of volume.
// If deleteRecord is true, etcd info about this volume and VolumeVersionMap record are deleted.
func (vs *VolumeService) DeleteVolume(name string, isLatest, deleteRecord bool) error {
	// name is the volume name without version (e.g. "foo") when isLatest is true,
	// otherwise it is the versioned name (e.g. "foo-2").
	baseName := name
	if isLatest {
		// get the last version number
		version, ok := vmap.VolumeVersionMap.Get(name)
		if !ok {
			return errors.Errorf("volume: %s version: %d not found in VolumeVersionMap", name, version)
		}
		name = fmt.Sprintf("%s-%d", name, version)
	} else {
		baseName = trimVersionSuffix(name)
	}
	if deleteRecord {
		// Earlier versions used strings.Split(name, "-")[0], which breaks names containing "-",
		// and deleted the etcd key "foo-2" while the record is stored under "foo".
		log.Infof("services.DeleteVolume, volume: %s will be del etcd info and version record", baseName)
		vmap.VolumeVersionMap.Remove(baseName)
		workQueue.Enqueue(etcd.DelKey{
			Resource: etcd.Volumes,
			Key:      baseName,
		})
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
