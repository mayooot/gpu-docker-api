package utils

import (
	"context"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/docker"
)

// CopyDir copies everything under src into dest.
//
// Earlier versions ran `cp -rf -p src/* dest/` through a shell. The `*` glob skips
// dotfiles at the top level (e.g. /root/.bashrc, /root/.cache, /root/.conda), which are exactly
// the user state this project promises to keep. `cp -a src/. dest/` copies hidden files and
// also preserves symlinks, ownership, timestamps and xattrs. It is executed without a shell,
// so paths are not subject to word splitting or globbing.
func CopyDir(src, dest string) error {
	out, err := exec.Command("cp", "-a", src+"/.", dest+"/").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "cp -a failed, src: %s, dest: %s, output: %s", src, dest, out)
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
