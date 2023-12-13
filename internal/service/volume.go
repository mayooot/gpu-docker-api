package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/commander-cli/cmd"
	"github.com/docker/docker/api/types/filters"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/siddontang/go/sync2"
	"strings"
	"time"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/model"

	"github.com/docker/docker/api/types/volume"
	"github.com/ngaut/log"
	"github.com/pkg/errors"
)

var volumeVersionMap = cmap.New[sync2.AtomicInt64]()
var ErrorVolumeExisted = errors.New("volume already exist")

type VolumeService struct{}

func (vs *VolumeService) CreateVolume(spec *model.VolumeCreate) (resp volume.Volume, err error) {
	ctx := context.Background()
	if vs.existVolume(spec.Name) {
		return resp, errors.Wrapf(ErrorVolumeExisted, "serivce.CreateVolume failed, volume %s", spec.Name)
	}

	var opt volume.CreateOptions
	if len(spec.Name) != 0 {
		opt.Name = spec.Name
	}
	if len(spec.Size) != 0 {
		opt.DriverOpts = map[string]string{"size": spec.Size}
	}

	opt.Driver = "local"
	resp, err = vs.createVolume(ctx, model.EtcdVolumeInfo{
		Opt: &opt,
	})
	if err != nil {
		return resp, errors.WithMessage(err, "service.CreateVolume failed")
	}

	log.Infof("volume created successfully, name: %s, spec: %+v", resp.Name, spec)
	return resp, err
}

func (vs *VolumeService) createVolume(ctx context.Context, info model.EtcdVolumeInfo) (resp volume.Volume, err error) {
	version, ok := volumeVersionMap.Get(info.Opt.Name)
	if !ok {
		volumeVersionMap.Set(info.Opt.Name, 0)
	} else {
		volumeVersionMap.Set(info.Opt.Name, sync2.AtomicInt64(version.Add(1)))
	}

	info.Opt.Name = fmt.Sprintf("%s-%d", info.Opt.Name, version)
	resp, err = docker.Cli.VolumeCreate(ctx, *info.Opt)
	if err != nil {
		return resp, errors.Wrapf(err, "service.createVolume failed, opt: %+v", info)
	}

	val := &model.EtcdVolumeInfo{
		Opt:     info.Opt,
		Version: version.Get(),
	}
	WorkQueue <- etcd.PutKeyValue{
		Key:      &info.Opt.Name,
		Value:    val.Serialize(),
		Resource: etcd.VolumePrefix,
	}

	return resp, err
}

func (vs *VolumeService) DeleteVolume(name *string) error {
	ctx := context.Background()
	err := docker.Cli.VolumeRemove(ctx, *name, true)
	if err != nil {
		return errors.Wrapf(err, "service.DeleteVolume failed, name: %s", *name)
	}

	log.Infof("volume deleted successfully, name: %s", *name)
	return nil
}

func (vs *VolumeService) PatchVolumeSize(name string, spec *model.VolumeSize) (resp volume.Volume, err error) {
	ctx := context.Background()
	infoBytes, err := etcd.GetVolumeInfo(ctx, name)
	if err != nil {
		return resp, errors.WithMessage(err, "service.PatchVolumeSize failed")
	}

	var info model.EtcdVolumeInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return resp, errors.WithMessage(err, "service.PatchVolumeSize failed")
	}

	// 更改 volume 的 size
	info.Opt.DriverOpts["size"] = spec.Size
	info.Opt.Name = strings.Split(name, "-")[0]
	resp, err = vs.createVolume(ctx, info)
	if err != nil {
		return resp, errors.WithMessage(err, "service.PatchVolumeSize failed")
	}

	// 将旧的Volume 里的数据移到新的 Volume 中
	WorkQueue <- &copyTask{
		Resource:    etcd.VolumePrefix,
		OldResource: name,
		NewResource: resp.Name,
	}

	return resp, err
}

func (vs *VolumeService) volumeMountpoint(name string) (string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.VolumeInspect(ctx, name)
	if err != nil || len(resp.Mountpoint) == 0 {
		return "", errors.Wrapf(err, "service.volumeMountpoint failed, name: %s", name)
	}

	return resp.Mountpoint, nil
}

func (vs *VolumeService) copyMountpointToContainer(task *copyTask) error {
	oldMountpoint, err := vs.volumeMountpoint(task.OldResource)
	if err != nil {
		return errors.WithMessage(err, "service.copyMountpointToContainer failed")
	}
	newMountpoint, err := vs.volumeMountpoint(task.NewResource)
	if err != nil {
		return errors.WithMessage(err, "service.copyMountpointToContainer failed")
	}

	if err = vs.copyMountpointFromOldVersion(oldMountpoint, newMountpoint); err != nil {
		return errors.WithMessage(err, "service.copyMountpointToContainer failed")
	}

	return nil
}

func (vs *VolumeService) copyMountpointFromOldVersion(src, dest string) error {
	startT := time.Now()
	command := fmt.Sprintf(cpRFPOption, src, dest)
	if err := cmd.NewCommand(command).Execute(); err != nil {
		return errors.Wrapf(err, "service.copyMountpointFromOldVersion failed, src:%s, dest: %s", src, dest)
	}
	log.Infof("service.copyMountpointFromOldVersion copy mountpoint successfully, src: %s, dest: %s, time cost: %v", src, dest, time.Since(startT))
	return nil
}

// 以 name- 为前缀的 volume 是否存在
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
