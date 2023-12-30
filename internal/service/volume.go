package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/ngaut/log"
	"github.com/pkg/errors"
	"github.com/siddontang/go/sync2"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/model"
	vmap "github.com/mayooot/gpu-docker-api/internal/version"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
	"github.com/mayooot/gpu-docker-api/utils"
)

type VolumeService struct{}

// CreateVolume 创建一个可指定名称和大小的 Volume
func (vs *VolumeService) CreateVolume(spec *model.VolumeCreate) (resp volume.Volume, err error) {
	ctx := context.Background()
	// 如果 Volume 已存在，则返回错误
	if vs.existVolume(spec.Name) {
		return resp, errors.Wrapf(xerrors.NewVolumeExistedError(), "volume %s", spec.Name)
	}

	opt := volume.CreateOptions{Driver: "local"}
	// 设置 Volume 的名称
	if len(spec.Name) != 0 {
		opt.Name = spec.Name
	}
	// 设置 Volume 的大小
	if len(spec.Size) != 0 {
		opt.DriverOpts = map[string]string{"size": spec.Size}
	}

	// 创建 Volume
	resp, err = vs.createVolume(ctx, model.EtcdVolumeInfo{
		Opt: &opt,
	})
	if err != nil {
		return resp, errors.WithMessage(err, "service.createVolume failed")
	}
	return
}

// 真正创建 Volume 的方法，参考 runContainer 方法
func (vs *VolumeService) createVolume(ctx context.Context, info model.EtcdVolumeInfo) (resp volume.Volume, err error) {
	// 获取卷的版本信息
	version, ok := vmap.VolumeVersionMap.Get(info.Opt.Name)
	if !ok {
		vmap.VolumeVersionMap.Set(info.Opt.Name, 0)
	} else {
		vmap.VolumeVersionMap.Set(info.Opt.Name, sync2.AtomicInt64(version.Add(1)))
	}

	defer func() {
		if err != nil {
			vmap.VolumeVersionMap.Set(info.Opt.Name, sync2.AtomicInt64(version.Add(-1)))
		}
	}()

	// 生成此次要创建的 Volume 的名称
	info.Opt.Name = fmt.Sprintf("%s-%d", info.Opt.Name, version)
	resp, err = docker.Cli.VolumeCreate(ctx, *info.Opt)
	if err != nil {
		return resp, errors.Wrapf(err, "docker.VolumeCreate failed, opt: %+v", info)
	}

	// 经过 docker volume create 校验后的容器配置，放入到 etcd 中
	val := &model.EtcdVolumeInfo{
		Opt:     info.Opt,
		Version: version.Get(),
	}
	// 异步添加到 etcd 中
	workQueue.Queue <- etcd.PutKeyValue{
		Resource: etcd.Volumes,
		Key:      info.Opt.Name,
		Value:    val.Serialize(),
	}
	log.Infof("serivce.createVolume, volume created successfully, name: %s, opt: %+v, version: %d", resp.Name, *info.Opt, info.Version)
	return
}

// DeleteVolume 删除一个 Volume
func (vs *VolumeService) DeleteVolume(name string, spec *model.VolumeDelete) error {
	var err error
	ctx := context.Background()
	if err = docker.Cli.VolumeRemove(ctx, name, spec.Force); err != nil {
		return errors.Wrapf(err, "docker.VolumeRemove failed, name: %s", name)
	}

	// 是否需要异步删除 etcd 中关于容器的描述和版本号记录
	if spec.DelEtcdInfoAndVersionRecord {
		vmap.VolumeVersionMap.Remove(strings.Split(name, "-")[0])
		workQueue.Queue <- etcd.DelKey{
			Resource: etcd.Volumes,
			Key:      name,
		}
		log.Infof("service.DeleteVolume, volume: %s will be del etcd info and version record", name)
	}
	log.Infof("service.DeleteVolume, volume deleted successfully, name: %s", name)
	return err
}

// PatchVolumeSize 变更 Volume 的大小
// 包括扩容和缩容两个操作，如果操作前后的大小相同，则直接返回，不做任何操作
// 需要注意的是，缩容操作会判断已经使用的 Volume 大小是否大于缩容之后的 Volume 大小，如果大于，则返回错误
// 例如：缩容前的 Volume 大小为 10G，缩容后的 Volume 大小为 5G，但是已经使用了 6G，则返回错误
func (vs *VolumeService) PatchVolumeSize(name string, spec *model.VolumeSize) (resp volume.Volume, err error) {
	// 从 etcd 中获取 Volume 的描述
	ctx := context.Background()
	infoBytes, err := etcd.Get(etcd.Volumes, name)
	if err != nil {
		return resp, errors.WithMessage(err, "etcd.Get failed")
	}
	var info model.EtcdVolumeInfo
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return resp, errors.WithMessage(err, "json.Unmarshal failed")
	}

	// 只有 etcd 中的 Volume 对象的版本和要修改的 Volume 版本一致时，才能更新
	if strconv.FormatInt(info.Version, 10) != strings.Split(name, "-")[1] {
		return resp, errors.Wrapf(xerrors.NewVersionNotMatchError(),
			"volume: %s, etcd version: %d, patch version: %s", name, info.Version, strings.Split(name, "-")[1])
	}

	// 获取修改前的卷大小和修改后的卷大小
	preSize := info.Opt.DriverOpts["size"]
	preSizeBytes, _ := utils.ToBytes(preSize)
	patchSize := spec.Size
	patchSizeBytes, _ := utils.ToBytes(patchSize)

	// 如果修改前后的大小相同，则直接返回
	if patchSize == preSize {
		return resp, errors.Wrapf(xerrors.NewNoPatchRequiredError(), "volume: %s", name)
	}

	if patchSizeBytes < preSizeBytes {
		// 如果是缩容操作，需要判断已经使用的卷容量是否大于缩容后的容量
		mountpoint, err := workQueue.VolumeMountPoint(name)
		if err != nil {
			return resp, errors.WithMessage(err, "service.volumeMountpoint failed")
		}
		usedSize, err := utils.DirSize(mountpoint)
		if err != nil {
			return resp, errors.Wrapf(err, "utils.DirSize failed, volume: %s, mountpoint: %s", name, mountpoint)
		}

		if usedSize > patchSizeBytes {
			return resp, errors.Wrapf(xerrors.NewVolumeSizeUsedGreaterThanReduced(),
				"volume: %s, usedSize: %d, patchSize: %d", name, usedSize, patchSizeBytes)
		}
	}

	// 更改卷的大小
	info.Opt.DriverOpts["size"] = patchSize
	info.Opt.Name = strings.Split(name, "-")[0]

	// 创建一个新的 Volume，替换旧的 Volume
	resp, err = vs.createVolume(ctx, info)
	if err != nil {
		return resp, errors.WithMessage(err, "service.createVolume failed")
	}

	// 异步拷贝旧 Volume 的数据到新的 Volume
	workQueue.Queue <- &workQueue.CopyTask{
		Resource:    etcd.Volumes,
		OldResource: name,
		NewResource: resp.Name,
	}
	log.Infof("service.PatchVolumeSize, volume size patched successfully, old name: %s, old size: %s, new name: %s, new size: %s",
		name, preSize, resp.Name, patchSize)
	return
}

// 判断 Volume 是否存在
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
