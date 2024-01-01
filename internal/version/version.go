package version

import (
	"encoding/json"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/siddontang/go/sync2"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

var (
	// ContainerVersionMap 用于追踪容器的版本信息
	ContainerVersionMap *versionMap
	// VolumeVersionMap 用于跟踪 Volume 的版本信息
	VolumeVersionMap *versionMap
)

const (
	// 存储在 etcd 中的 key
	containerVersionMapKey = "containerVersionMapKey"
	volumeVersionMapKey    = "volumeVersionMapKey"
)

type versionMap struct {
	cmap.ConcurrentMap[string, sync2.AtomicInt64]
}

func newVersionMap() *versionMap {
	return &versionMap{cmap.New[sync2.AtomicInt64]()}
}

func (vm *versionMap) serialize() *string {
	bytes, _ := json.Marshal(vm)
	tmp := string(bytes)
	return &tmp
}

func Init() error {
	var err error
	ContainerVersionMap, err = initVersionMap(containerVersionMapKey)
	if err != nil {
		return err
	}

	VolumeVersionMap, err = initVersionMap(volumeVersionMapKey)
	if err != nil {
		return err
	}

	return nil
}

func Close() error {
	if err := etcd.Put(etcd.Versions, containerVersionMapKey, ContainerVersionMap.serialize()); err != nil {
		return err
	}
	if err := etcd.Put(etcd.Versions, volumeVersionMapKey, VolumeVersionMap.serialize()); err != nil {
		return err
	}
	return nil
}

func initVersionMap(key string) (vm *versionMap, err error) {
	bytes, err := etcd.Get(etcd.Versions, key)
	if err != nil {
		if xerrors.IsNotExistInEtcdError(err) {
			err = nil
		} else {
			return vm, err
		}
	}

	vm = newVersionMap()
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &vm)
	}
	return vm, err
}
