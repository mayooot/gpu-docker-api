package xerrors

import "github.com/pkg/errors"

const (
	volumeExisted                    = "volume existed"
	volumeSizeUsedGreaterThanReduced = "volume The used size is greater than the reduced size"
)

func NewVolumeExistedError() error {
	return errors.New(volumeExisted)
}

func IsVolumeExistedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == volumeExisted
}

func NewVolumeSizeUsedGreaterThanReduced() error {
	return errors.New(volumeSizeUsedGreaterThanReduced)
}

func IsVolumeSizeUsedGreaterThanReduced(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == volumeSizeUsedGreaterThanReduced
}
