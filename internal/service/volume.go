package service

import (
	"context"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/model"

	"github.com/docker/docker/api/types/volume"
	"github.com/ngaut/log"
	"github.com/pkg/errors"
)

type VolumeService struct{}

func (vs *VolumeService) CreateVolume(spec *model.VolumeCreate) (resp volume.Volume, err error) {
	var opt volume.CreateOptions
	if len(spec.Name) != 0 {
		opt.Name = spec.Name
	}
	if len(spec.Size) != 0 {
		opt.DriverOpts = map[string]string{"size": spec.Size}
	}

	ctx := context.Background()
	opt.Driver = "local"
	resp, err = docker.Cli.VolumeCreate(ctx, opt)
	if err != nil {
		return resp, errors.Wrapf(err, "failed to create volume, %+v", spec)
	}

	log.Infof("volume created successfully, name: %s, spec: %+v", resp.Name, spec)
	return resp, err
}

func (vs *VolumeService) DeleteVolume(name *string) error {
	ctx := context.Background()
	err := docker.Cli.VolumeRemove(ctx, *name, true)
	if err != nil {
		return errors.Wrapf(err, "failed to remove volume, name: %s", *name)
	}

	log.Infof("volume deleted successfully, name: %s", *name)
	return nil
}
