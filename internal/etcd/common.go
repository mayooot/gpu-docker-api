package etcd

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
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
	_, err := cli.Put(ctx, resourcePrefix(resource, realName(key)), *value)
	if err != nil {
		return errors.Wrapf(err, "etcd.Put failed, resource %s, key: %s, value: %s", resource, key, *value)
	}
	return nil
}

func Get(resource EtcdResource, key string) (bytes []byte, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	resp, err := cli.Get(ctx, resourcePrefix(resource, realName(key)))
	if err != nil {
		return bytes, errors.Wrapf(err, "etcd.Get failed, key: %s", key)
	}

	if len(resp.Kvs) != 0 {
		bytes = resp.Kvs[0].Value
	}
	return bytes, err
}

func Del(resource EtcdResource, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationDuration)
	defer cancel()
	_, err := cli.Delete(ctx, resourcePrefix(resource, realName(key)))
	return err
}

func realName(key string) string {
	return strings.Split(key, "-")[0]
}

func resourcePrefix(prefix EtcdResource, name string) string {
	return path.Join(CommonPrefix, string(prefix), name)
}
