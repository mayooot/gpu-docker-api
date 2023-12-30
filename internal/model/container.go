package model

import (
	"github.com/docker/docker/api/types/mount"
)

type ContainerRun struct {
	ImageName      string   `json:"imageName"`
	ContainerName  string   `json:"containerName"`
	GpuCount       int      `json:"gpuCount,omitempty"`
	Binds          []Bind   `json:"binds,omitempty"`
	Env            []string `json:"env,omitempty"`
	Cmd            []string `json:"cmd,omitempty"`
	ContainerPorts []string `json:"containerPorts,omitempty"`
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
