package xerrors

import (
	"github.com/pkg/errors"
)

const (
	noPatchRequired = "no patch required"
	versionNotMatch = "version not match"
)

func NewNoPatchRequiredError() error {
	return errors.New(noPatchRequired)
}

func IsNoPatchRequiredError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == noPatchRequired
}

func NewVersionNotMatchError() error {
	return errors.New(versionNotMatch)
}

func IsVersionNotMatchError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == versionNotMatch
}
