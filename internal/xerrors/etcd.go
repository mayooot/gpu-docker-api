package xerrors

import (
	"github.com/pkg/errors"
)

const (
	notExistInEtcd = "not exist in etcd"
)

func NewNotExistInEtcdError() error {
	return errors.New(notExistInEtcd)
}

func IsNotExistInEtcdError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == notExistInEtcd
}
