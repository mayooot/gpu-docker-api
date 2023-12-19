package xerrors

import (
	"github.com/pkg/errors"
)

const gpuNotEnough = "gpu not enough"

func NewGpuNotEnoughError() error {
	return errors.New(gpuNotEnough)
}

func IsGpuNotEnoughError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Cause(err).Error() == gpuNotEnough
}
