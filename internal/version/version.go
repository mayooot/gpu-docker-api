package version

import (
	"encoding/json"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

var (
	ContainerVersionMap *versionMap
	VolumeVersionMap    *versionMap
)

const (
	containerVersionMapKey = "containerVersionMapKey"
	volumeVersionMapKey    = "volumeVersionMapKey"
)

type (
	name    = string
	version = int64
)

type versionMap map[name]version

func InitVersionMap() error {
	var err error
	ContainerVersionMap, err = initVersionMapFormEtcd(containerVersionMapKey)
	if err != nil {
		return err
	}

	VolumeVersionMap, err = initVersionMapFormEtcd(volumeVersionMapKey)
	if err != nil {
		return err
	}

	return nil
}

func CloseVersionMap() error {
	if err := etcd.Put(etcd.Versions, containerVersionMapKey, ContainerVersionMap.serialize()); err != nil {
		return err
	}
	if err := etcd.Put(etcd.Versions, volumeVersionMapKey, VolumeVersionMap.serialize()); err != nil {
		return err
	}
	return nil
}

func (vm *versionMap) serialize() *string {
	bytes, _ := json.Marshal(vm)
	tmp := string(bytes)
	return &tmp
}

func (vm *versionMap) Set(key name, value version) {
	(*vm)[key] = value
}

func (vm *versionMap) Get(key name) (version, bool) {
	v, ok := (*vm)[key]
	return v, ok
}

func (vm *versionMap) Exist(key name) bool {
	_, ok := (*vm)[key]
	return ok
}

func (vm *versionMap) Remove(key name) {
	delete(*vm, key)
}

func initVersionMapFormEtcd(key string) (vm *versionMap, err error) {
	bytes, err := etcd.GetValue(etcd.Versions, key)
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

func newVersionMap() *versionMap {
	m := make(versionMap)
	return &m
}
