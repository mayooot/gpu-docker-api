package etcd

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

const (
	CommonPrefix = "/apis/v1"
	// ContainerPrefix 容器信息存储在 etcd 中的前缀，同时也用于判断 workQueue 中的 CopyTask 类型
	//ContainerPrefix = "/apis/v1/containers"
	// VolumePrefix Volume 信息存储在 etcd 中的前缀，同时也用于判断 workQueue 中的 CopyTask 类型
	//VolumePrefix = "/apis/v1/volumes"
)

type EtcdResource string

const (
	Containers EtcdResource = "containers"
	Volumes    EtcdResource = "volumes"
	Versions   EtcdResource = "versions"
	Gpus       EtcdResource = "gpus"
	Ports      EtcdResource = "ports"

	operationDuration = 1 * time.Second
)

type PutKeyValue struct {
	Key      string
	Value    *string
	Resource EtcdResource
}

type DelKey struct {
	Resource EtcdResource
	Key      string
}

func Put(resource EtcdResource, key string, value *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	_, err := cli.Put(ctx, ResourcePrefix(resource, realName(key)), *value)
	if err != nil {
		return errors.Wrapf(err, "etcd.Put failed, resource %s, key: %s, value: %s", resource, key, *value)
	}
	return nil
}

func Get(resource EtcdResource, key string) (bytes []byte, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	resp, err := cli.Get(ctx, ResourcePrefix(resource, realName(key)))
	if err != nil {
		return bytes, err
	}
	if len(resp.Kvs) == 0 {
		return bytes, xerrors.NewNotExistInEtcdError()
	}
	return resp.Kvs[0].Value, err
}

func Del(resource EtcdResource, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	_, err := cli.Delete(ctx, ResourcePrefix(resource, realName(key)))
	return err
}

func realName(key string) string {
	return strings.Split(key, "-")[0]
}

func ResourcePrefix(prefix EtcdResource, name string) string {
	return path.Join(CommonPrefix, string(prefix), name)
}
