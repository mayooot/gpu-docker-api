package model

type ContainerRun struct {
	ImageName     string   `json:"imageName"`
	ContainerName string   `json:"containerName"`
	GpuCount      int      `json:"gpuCount,omitempty"`
	GpuNumbers    []string `json:"gpuNumbers,omitempty"`
	Cardless      bool     `json:"cardless,omitempty"`
	Binds         []Bind   `json:"binds,omitempty"`
}

type ContainerExecute struct {
	WorkDir string   `json:"workDir,omitempty"`
	Cmd     []string `json:"cmd,omitempty"`
}

type ContainerRename struct {
	NewName string `json:"newName"`
}

type ContainerGpuPatch struct {
	GpuCount int `json:"gpuCount"`
}
