package xerrors

import (
	"github.com/pkg/errors"
)

const containerExisted = "container existed"

func NewContainerExistedError() error {
	return errors.New(containerExisted)
}

func IsContainerExistedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == containerExisted
}
