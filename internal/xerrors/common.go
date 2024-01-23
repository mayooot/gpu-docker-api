package xerrors

import (
	"github.com/pkg/errors"
)

const (
	noPatchRequired    = "no patch required"
	noRollbackRequired = "no rollback required"
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

func NewNoRollbackRequiredError() error {
	return errors.New(noRollbackRequired)
}

func IsNoRollbackRequiredError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == noRollbackRequired
}
