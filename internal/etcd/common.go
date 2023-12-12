package etcd

import (
	"path"
	"strings"
)

type PutKeyValue struct {
	Key      *string
	Value    *string
	Resource string
}

func realName(key string) string {
	return strings.Split(key, "-")[0]
}

func resourcePrefix(prefix, name string) string {
	return path.Join(prefix, name)
}
