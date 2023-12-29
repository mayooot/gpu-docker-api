package model

import (
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

type ContainerRun struct {
	ImageName     string   `json:"imageName"`
	ContainerName string   `json:"containerName"`
	GpuCount      int      `json:"gpuCount,omitempty"`
	Binds         []Bind   `json:"binds,omitempty"`
	Env           []string `json:"env,omitempty"`
	Cmd           []string `json:"cmd,omitempty"`
	Ports         []Ports  `json:"ports,omitempty"`
}

type Ports struct {
	HostPort      uint16 `json:"hostPort"`
	ContainerPort uint16 `json:"containerPort"`
}

func (p Ports) Key() nat.Port {
	return nat.Port(fmt.Sprintf("%d/tcp", p.ContainerPort))
}

func (p Ports) Value() []nat.PortBinding {
	return []nat.PortBinding{{
		HostPort: fmt.Sprintf("%d", p.HostPort),
	}}
}

type ContainerExecute struct {
	WorkDir string   `json:"workDir,omitempty"`
	Cmd     []string `json:"cmd,omitempty"`
}

type ContainerGpuPatch struct {
	GpuCount int `json:"gpuCount"`
}

type ContainerVolumePatch struct {
	Type    mount.Type `json:"type"`
	OldBind *Bind      `json:"oldBind"`
	NewBind *Bind      `json:"newBind"`
}

type ContainerDelete struct {
	Force                       bool `json:"force"`
	DelEtcdInfoAndVersionRecord bool `json:"delEtcdInfoAndVersionRecord"`
}

type ContainerCommit struct {
	NewImageName string `json:"newImageName"`
}
