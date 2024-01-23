package etcd

import (
	"github.com/pkg/errors"
)

type (
	ReplicaSet = []*combine
	Value      = []byte
)

type combine struct {
	Version  int64
	Revision int64
	Value    Value
}

func GetRevisionRange(resource Resource, key string) (ReplicaSet, error) {
	kvs, err := get(resource, key)
	if err != nil {
		return nil, err
	}

	set := make([]*combine, 0)
	modRev := kvs[0].ModRevision
	createRev := kvs[0].CreateRevision

	var pre int64
	for rev := modRev; rev >= createRev; rev-- {
		kvs, err := getWithRev(resource, key, rev)
		if err != nil {
			return nil, err
		}
		if kvs[0].Version != pre {
			set = append(set, &combine{
				Version:  kvs[0].Version,
				Revision: kvs[0].ModRevision,
				Value:    kvs[0].Value,
			})
			pre = kvs[0].Version
		}
	}
	return set, nil
}

func GetRevision(resource Resource, key string, version int64) (Value, error) {
	kvs, err := get(resource, key)
	if err != nil {
		return nil, err
	}

	modRev := kvs[0].ModRevision
	createRev := kvs[0].CreateRevision

	for rev := modRev; rev >= createRev; rev-- {
		kvs, err := getWithRev(resource, key, rev)
		if err != nil {
			return nil, err
		}
		if kvs[0].Version == version {
			return kvs[0].Value, nil
		}

	}
	return nil, errors.Errorf("not found version :%d", version)
}
