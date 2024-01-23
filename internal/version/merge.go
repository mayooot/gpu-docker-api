package version

import (
	"encoding/json"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

var ContainerMergeMap *mergeMap

const containerMergeMapKey = "containerMergeMapKey"

type mergePath = string

type mergeMap map[version]mergePath

func InitMergedMap() error {
	var err error
	ContainerMergeMap, err = initMergeMapFormEtcd()
	if err != nil {
		return err
	}

	return nil
}

func CloseMergedMap() error {
	if err := etcd.Put(etcd.Merges, containerMergeMapKey, ContainerMergeMap.serialize()); err != nil {
		return err
	}
	return nil
}

func (mm *mergeMap) serialize() *string {
	bytes, _ := json.Marshal(mm)
	tmp := string(bytes)
	return &tmp
}

func (mm *mergeMap) Set(key version, value mergePath) {
	(*mm)[key] = value
}

func (mm *mergeMap) Get(key version) (mergePath, bool) {
	value, ok := (*mm)[key]
	return value, ok
}

func (mm *mergeMap) Exist(key version) bool {
	_, ok := (*mm)[key]
	return ok
}

func (mm *mergeMap) Remove(key version) {
	delete(*mm, key)
}

func initMergeMapFormEtcd() (mm *mergeMap, err error) {
	bytes, err := etcd.GetValue(etcd.Merges, containerMergeMapKey)
	if err != nil {
		if xerrors.IsNotExistInEtcdError(err) {
			err = nil
		} else {
			return mm, err
		}
	}

	mm = newMergedMap()
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &mm)
	}
	return mm, err
}

func newMergedMap() *mergeMap {
	m := make(mergeMap)
	return &m
}
