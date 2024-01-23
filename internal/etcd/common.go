package etcd

import (
	"context"
	"path"
	"time"

	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

const (
	// CommonPrefix is the common prefix for all keys.
	CommonPrefix = "/gpu-docker-api/apis/v1"
)

type Resource = string

const (
	Containers Resource = "containers"
	Volumes    Resource = "volumes"
	Versions   Resource = "versions"
	Merges     Resource = "merges"
	Gpus       Resource = "gpus"
	Ports      Resource = "ports"

	operationDuration = 1 * time.Second
)

type PutKeyValue struct {
	Key      string
	Value    *string
	Resource Resource
}

type DelKey struct {
	Resource Resource
	Key      string
}

func Put(resource Resource, key string, value *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	_, err := cli.Put(ctx, ResourcePrefix(resource, key), *value)
	if err != nil {
		return errors.Wrapf(err, "etcd.Put failed, resource %s, key: %s, value: %s", resource, key, *value)
	}
	return nil
}

func GetValue(resource Resource, key string) ([]byte, error) {
	kvs, err := get(resource, key)
	if err != nil {
		return nil, err
	}
	return kvs[0].Value, nil
}

func Del(resource Resource, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	_, err := cli.Delete(ctx, ResourcePrefix(resource, key))
	return err
}

func get(resource Resource, key string) ([]*mvccpb.KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	resp, err := cli.Get(ctx, ResourcePrefix(resource, key))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, xerrors.NewNotExistInEtcdError()
	}
	return resp.Kvs, nil
}

func getWithRev(resource Resource, key string, rev int64) ([]*mvccpb.KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	resp, err := cli.Get(ctx, ResourcePrefix(resource, key), clientv3.WithRev(rev))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, xerrors.NewNotExistInEtcdError()
	}
	return resp.Kvs, nil
}

func ResourcePrefix(prefix Resource, name string) string {
	return path.Join(CommonPrefix, prefix, name)
}
