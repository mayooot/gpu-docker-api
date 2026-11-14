package version

import (
	"encoding/json"
	"sync"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

var ContainerMergeMap *mergeMap

const containerMergeMapKey = "containerMergeMapKey"

type (
	// ctrVersionName is the versioned container name, e.g. "foo-3".
	// Earlier versions keyed this map by the bare version number, which made
	// "foo-1" and "bar-1" overwrite each other.
	ctrVersionName = string
	mergePath      = string
)

type mergeMap struct {
	mu sync.RWMutex
	m  map[ctrVersionName]mergePath
}

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
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.serializeLocked()
}

func (mm *mergeMap) serializeLocked() *string {
	bytes, _ := json.Marshal(mm.m)
	tmp := string(bytes)
	return &tmp
}

func (mm *mergeMap) persistLocked() {
	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Merges,
		Key:      containerMergeMapKey,
		Value:    mm.serializeLocked(),
	})
}

func (mm *mergeMap) Set(key ctrVersionName, value mergePath) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.m[key] = value
	mm.persistLocked()
}

func (mm *mergeMap) Get(key ctrVersionName) (mergePath, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	value, ok := mm.m[key]
	return value, ok
}

func (mm *mergeMap) Exist(key ctrVersionName) bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	_, ok := mm.m[key]
	return ok
}

func (mm *mergeMap) Remove(key ctrVersionName) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.m, key)
	mm.persistLocked()
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
		err = json.Unmarshal(bytes, &mm.m)
	}
	return mm, err
}

func newMergedMap() *mergeMap {
	return &mergeMap{m: make(map[ctrVersionName]mergePath)}
}
