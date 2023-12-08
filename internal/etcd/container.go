package etcd

import (
	"context"
	"strings"

	cmap "github.com/orcaman/concurrent-map/v2"

	"github.com/pkg/errors"
)

// 保存容器版本和 etcd 中的 mod_revision对应关系，用于回滚历史版本的容器创建信息
// key: 容器名称，格式：name-version
// value：etcd 中的 mod_revision
var containerModRevisionMap = cmap.New[int64]()

type KeyValue struct {
	Key   *string
	Value *string
}

func PutContainerInfo(ctx context.Context, key, value *string) error {
	resp, err := cli.Put(ctx, containerRealName(*key), *value)
	if err != nil {
		return errors.Wrapf(err, "failed to put container info to etcd, key: %s", *key)
	}
	containerModRevisionMap.Set(*key, resp.Header.Revision)
	return nil
}

func GetContainerInfo(ctx context.Context, key string) (bytes []byte, err error) {
	resp, err := cli.Get(ctx, containerRealName(key))
	if err != nil {
		return bytes, errors.Wrapf(err, "failed to get container info from etcd, key: %s", key)
	}

	bytes = resp.Kvs[0].Value
	return bytes, err
}

func containerRealName(key string) string {
	return strings.Split(key, "-")[0]
}
