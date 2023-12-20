package xerrors

import "github.com/pkg/errors"

const volumeExisted = "volume existed"

func NewVolumeExistedError() error {
	return errors.New(volumeExisted)
}

func IsVolumeExistedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == volumeExisted
}
