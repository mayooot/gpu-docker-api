package model

type Memory struct {
	Total uint64
	Free  uint64
	Used  uint64
}

type ProcessInfo struct {
	Pid               uint32
	UsedGpuMemory     uint64
	GpuInstanceId     uint32
	ComputeInstanceId uint32
}

type GpuInfo struct {
	Index                       int           `json:"index"`
	UUID                        string        `json:"uuid"`
	Name                        string        `json:"name"`
	MemoryInfo                  Memory        `json:"memoryInfo"`
	PowerUsage                  uint32        `json:"powerUsage"`
	PowerState                  int32         `json:"powerState"`
	PowerManagementDefaultLimit uint32        `json:"powerManagementDefaultLimit"`
	InformImageVersion          string        `json:"informImageVersion"`
	DriverVersion               string        `json:"systemGetDriverVersion"`
	CUDADriverVersion           int           `json:"systemGetCudaDriverVersion"`
	GraphicsRunningProcesses    []ProcessInfo `json:"tGraphicsRunningProcesses"`
}
