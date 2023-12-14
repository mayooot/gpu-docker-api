package model

import (
	"encoding/json"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type EtcdContainerInfo struct {
	Config           *container.Config         `json:"Config"`
	HostConfig       *container.HostConfig     `json:"HostConfig"`
	NetworkingConfig *network.NetworkingConfig `json:"NetworkingConfig"`
	Platform         *ocispec.Platform         `json:"Platform"`
	ContainerName    string                    `json:"ContainerName"`
	Version          int64                     `json:"Version"`
}

func (i *EtcdContainerInfo) Serialize() *string {
	bytes, _ := json.Marshal(i)
	tmp := string(bytes)
	return &tmp
}

type EtcdVolumeInfo struct {
	Opt     *volume.CreateOptions
	Version int64 `json:"Version"`
}

func (i *EtcdVolumeInfo) Serialize() *string {
	bytes, _ := json.Marshal(i)
	tmp := string(bytes)
	return &tmp
}
