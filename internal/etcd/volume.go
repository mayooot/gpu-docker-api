package etcd

import (
	"context"
	cmap "github.com/orcaman/concurrent-map/v2"

	"github.com/pkg/errors"
)

// 保存容器版本和 etcd 中的 mod_revision对应关系，用于回滚历史版本的容器创建信息
// key: 容器名称，格式：name-version
// value：etcd 中的 mod_revision
var volumeModRevisionMap = cmap.New[int64]()

const VolumePrefix = "/apis/v1/volumes"

func PutVolumeInfo(ctx context.Context, key, value *string) error {
	resp, err := cli.Put(ctx, resourcePrefix(VolumePrefix, realName(*key)), *value)
	if err != nil {
		return errors.Wrapf(err, "etcd.PutVolumeInfo key: %s", *key)
	}
	volumeModRevisionMap.Set(*key, resp.Header.Revision)
	return nil
}

func GetVolumeInfo(ctx context.Context, key string) (bytes []byte, err error) {
	resp, err := cli.Get(ctx, resourcePrefix(VolumePrefix, realName(key)))
	if err != nil {
		return bytes, errors.Wrapf(err, "etcd.PutVolumeInfo key: %s", key)
	}

	bytes = resp.Kvs[0].Value
	return bytes, err
}
