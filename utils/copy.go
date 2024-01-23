package utils

import (
	"context"
	"fmt"

	"github.com/commander-cli/cmd"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/docker"
)

var (
	cpRFPOption = "cp -rf -p %s/* %s/"
)

func CopyDir(src, dest string) error {
	command := fmt.Sprintf(cpRFPOption, src, dest)
	if err := cmd.NewCommand(command).Execute(); err != nil {
		return errors.Wrapf(err, "cmd.Execute failed, command %s, src:%s, dest: %s", command, src, dest)
	}
	return nil
}

// CopyOldMergedToNewContainerMerged is used to copy the merged layer from the old container
// to the new container during patch operations.
func CopyOldMergedToNewContainerMerged(oldContainer, newContainer string) error {
	oldMerged, err := GetContainerMergedLayer(oldContainer)
	if err != nil {
		return errors.WithMessage(err, "GetContainerMergedLayer failed")
	}
	newMerged, err := GetContainerMergedLayer(newContainer)
	if err != nil {
		return errors.WithMessage(err, "GetContainerMergedLayer failed")
	}

	if err = CopyDir(oldMerged, newMerged); err != nil {
		return errors.WithMessage(err, "copyDir failed")
	}
	return nil
}

func GetContainerMergedLayer(name string) (string, error) {
	resp, err := docker.Cli.ContainerInspect(context.TODO(), name)
	if err != nil || len(resp.GraphDriver.Data["MergedDir"]) == 0 {
		return "", errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
	}
	return resp.GraphDriver.Data["MergedDir"], nil
}

// CopyOldMountPointToContainerMountPoint is used to copy the volume data from the old container
// to the new container during patch operations.
func CopyOldMountPointToContainerMountPoint(oldVolume, newVolume string) error {
	oldMountPoint, err := GetVolumeMountPoint(oldVolume)
	if err != nil {
		return errors.WithMessage(err, "GetVolumeMountPoint failed")
	}
	newMountPoint, err := GetVolumeMountPoint(newVolume)
	if err != nil {
		return errors.WithMessage(err, "GetVolumeMountPoint failed")
	}

	if err = CopyDir(oldMountPoint, newMountPoint); err != nil {
		return errors.WithMessage(err, "copyDir failed")
	}
	return nil
}

func GetVolumeMountPoint(name string) (string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.VolumeInspect(ctx, name)
	if err != nil || len(resp.Mountpoint) == 0 {
		return "", errors.Wrapf(err, "docker.VolumeInspect failed, name: %s", name)
	}
	return resp.Mountpoint, nil
}
