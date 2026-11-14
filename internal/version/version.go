package version

import (
	"encoding/json"
	"sync"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
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

// versionMap records the latest version of each replicaSet / volume.
// It is accessed concurrently by HTTP handlers, so all access goes through the mutex.
// Every mutation enqueues a snapshot to etcd, so the state survives an unexpected crash.
type versionMap struct {
	mu  sync.RWMutex
	key string
	m   map[name]version
}

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
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.serializeLocked()
}

func (vm *versionMap) serializeLocked() *string {
	bytes, _ := json.Marshal(vm.m)
	tmp := string(bytes)
	return &tmp
}

// persistLocked must be called with vm.mu held, so snapshots are enqueued in mutation order.
func (vm *versionMap) persistLocked() {
	workQueue.Enqueue(etcd.PutKeyValue{
		Resource: etcd.Versions,
		Key:      vm.key,
		Value:    vm.serializeLocked(),
	})
}

func (vm *versionMap) Set(key name, value version) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.m[key] = value
	vm.persistLocked()
}

func (vm *versionMap) Get(key name) (version, bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	v, ok := vm.m[key]
	return v, ok
}

func (vm *versionMap) Exist(key name) bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	_, ok := vm.m[key]
	return ok
}

func (vm *versionMap) Remove(key name) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	delete(vm.m, key)
	vm.persistLocked()
}

// Next atomically increments and returns the version of key.
// It avoids the read-modify-write race of Get + Set when two requests touch the same name.
func (vm *versionMap) Next(key name) version {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.m[key]++
	vm.persistLocked()
	return vm.m[key]
}

// Revert undoes a Next call that was not followed by a successful creation.
func (vm *versionMap) Revert(key name, failed version) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	if vm.m[key] != failed {
		// someone else has moved the version forward, leave it alone
		return
	}
	if failed <= 1 {
		delete(vm.m, key)
	} else {
		vm.m[key] = failed - 1
	}
	vm.persistLocked()
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

	vm = newVersionMap(key)
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &vm.m)
	}
	return vm, err
}

func newVersionMap(key string) *versionMap {
	return &versionMap{key: key, m: make(map[name]version)}
}
