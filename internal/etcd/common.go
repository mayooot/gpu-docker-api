package etcd

import (
	"context"
	"path"
	"strings"

	"github.com/pkg/errors"
)

const (
	// ContainerPrefix 容器信息存储在 etcd 中的前缀，同时也用于判断 workQueue 中的 CopyTask 类型
	ContainerPrefix = "/apis/v1/containers"
	// VolumePrefix Volume 信息存储在 etcd 中的前缀，同时也用于判断 workQueue 中的 CopyTask 类型
	VolumePrefix = "/apis/v1/volumes"
)

type PutKeyValue struct {
	Key      string
	Value    *string
	Resource string
}

type DelKey struct {
	Resource string
	Key      string
}

func Put(resource, key string, value *string) error {
	ctx := context.Background()
	_, err := cli.Put(ctx, resourcePrefix(resource, realName(key)), *value)
	if err != nil {
		return errors.Wrapf(err, "etcd.Put failed, resource %s, key: %s, value: %s", resource, key, *value)
	}
	return nil
}

func Get(resource, key string) (bytes []byte, err error) {
	ctx := context.Background()
	resp, err := cli.Get(ctx, resourcePrefix(resource, realName(key)))
	if err != nil {
		return bytes, errors.Wrapf(err, "etcd.Get failed, key: %s", key)
	}

	bytes = resp.Kvs[0].Value
	return bytes, err
}

func Del(resource, key string) error {
	ctx := context.Background()
	_, err := cli.Delete(ctx, resourcePrefix(resource, realName(key)))
	return err
}

func realName(key string) string {
	return strings.Split(key, "-")[0]
}

func resourcePrefix(prefix, name string) string {
	return path.Join(prefix, name)
}
