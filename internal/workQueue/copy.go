package workQueue

import (
	"context"
	"fmt"

	"github.com/commander-cli/cmd"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
)

var (
	// 拷贝容器的 MergedDir 和卷的 MountPoint 的命令
	cpRFPOption = "cp -rf -p %s/* %s/"
)

type CopyTask struct {
	Resource    etcd.EtcdResource
	OldResource string
	NewResource string
}

func copyDir(src, dest string) error {
	command := fmt.Sprintf(cpRFPOption, src, dest)
	if err := cmd.NewCommand(command).Execute(); err != nil {
		return errors.Wrapf(err, "cmd.Execute failed, command %s, src:%s, dest: %s", command, src, dest)
	}
	return nil
}

// 拷贝容器的 Merged 层到新容器中，该方法用于 workQueue 异步调用
func copyMergedDirToContainer(task *CopyTask) error {
	oldMerged, err := containerGraphDriverDataMergedDir(task.OldResource)
	if err != nil {
		return errors.WithMessage(err, "workQueue.containerGraphDriverDataMergedDir failed")
	}
	newMerged, err := containerGraphDriverDataMergedDir(task.NewResource)
	if err != nil {
		return errors.WithMessage(err, "workQueue.containerGraphDriverDataMergedDir failed")
	}

	if err = copyDir(oldMerged, newMerged); err != nil {
		return errors.WithMessage(err, "workQueue.copyDir failed")
	}
	return nil
}

// 获取容器的 Merged 层的实际存储位置
func containerGraphDriverDataMergedDir(name string) (string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.ContainerInspect(ctx, name)
	if err != nil || len(resp.GraphDriver.Data["MergedDir"]) == 0 {
		return "", errors.Wrapf(err, "docker.ContainerInspect failed, name: %s", name)
	}
	return resp.GraphDriver.Data["MergedDir"], nil
}

// 拷贝 Volume 的 MountPoint 到新的 Volume，该方法用于 workQueue 异步调用
func copyMountPointToContainer(task *CopyTask) error {
	oldMountPoint, err := VolumeMountPoint(task.OldResource)
	if err != nil {
		return errors.WithMessage(err, "workQueue.volumeMountpoint failed")
	}
	newMountPoint, err := VolumeMountPoint(task.NewResource)
	if err != nil {
		return errors.WithMessage(err, "workQueue.volumeMountpoint failed")
	}

	if err = copyDir(oldMountPoint, newMountPoint); err != nil {
		return errors.WithMessage(err, "workQueue.copyDir failed")
	}
	return nil
}

// VolumeMountPoint 获取 Volume 的 MountPoint
func VolumeMountPoint(name string) (string, error) {
	ctx := context.Background()
	resp, err := docker.Cli.VolumeInspect(ctx, name)
	if err != nil || len(resp.Mountpoint) == 0 {
		return "", errors.Wrapf(err, "docker.VolumeInspect failed, name: %s", name)
	}
	return resp.Mountpoint, nil
}
