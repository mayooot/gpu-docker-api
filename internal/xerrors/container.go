package xerrors

import (
	"github.com/pkg/errors"
)

const containerExisted = "container existed"

const noPatchRequired = "no patch required"

func NewContainerExistedError() error {
	return errors.New(containerExisted)
}

func IsContainerExistedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == containerExisted
}

func NewNoPatchRequiredError() error {
	return errors.New(noPatchRequired)
}

func IsNoPatchRequiredError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == noPatchRequired
}
