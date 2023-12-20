package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func ToBytes(origin string) (int64, error) {
	sizeStr := origin[:len(origin)-2]
	unit := origin[len(origin)-2:]

	size, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return 0, err
	}

	var multiplier int64
	switch unit {
	case "KB":
		multiplier = 1 << 10
	case "MB":
		multiplier = 1 << 20
	case "GB":
		multiplier = 1 << 30
	case "TB":
		multiplier = 1 << 40
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}

	return int64(size * float64(multiplier)), nil
}
