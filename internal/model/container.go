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

type GpuPatch struct {
	GpuCount int `json:"gpuCount"`
}

type VolumePatch struct {
	Type    mount.Type `json:"type"`
	OldBind *Bind      `json:"oldBind"`
	NewBind *Bind      `json:"newBind"`
}

type PatchRequest struct {
	GpuPatch    *GpuPatch    `json:"gpuPatch"`
	VolumePatch *VolumePatch `json:"volumePatch"`
}

type ContainerDelete struct {
	Force                       bool `json:"force,omitempty"`
	DelEtcdInfoAndVersionRecord bool `json:"delEtcdInfoAndVersionRecord,omitempty"`
}

type ContainerCommit struct {
	NewImageName string `json:"newImageName"`
}

type ContainerStop struct {
	RestoreGpus  bool `json:"restoreGpus,omitempty"`
	RestorePorts bool `json:"restorePorts,omitempty"`
}
