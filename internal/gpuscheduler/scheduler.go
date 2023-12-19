package gpuscheduler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/config"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

var _defaultAvailableGpuNums = 8

var Scheduler *scheduler

type scheduler struct {
	sync.RWMutex

	availableGpuNums int
	gpuStatusMap     map[string]byte
}

type memory struct {
	Total uint64
	Free  uint64
	Used  uint64
}

type processInfo struct {
	Pid               uint32
	UsedGpuMemory     uint64
	GpuInstanceId     uint32
	ComputeInstanceId uint32
}

type gpuInfo struct {
	UUID                        string        `json:"UUID"`
	Name                        string        `json:"name"`
	MemoryInfo                  memory        `json:"memoryInfo"`
	PowerUsage                  uint32        `json:"powerUsage"`
	PowerState                  int32         `json:"powerState"`
	PowerManagementDefaultLimit uint32        `json:"powerManagementDefaultLimit"`
	InfoImageVersion            string        `json:"infoImageVersion"`
	InforomImageVersion         string        `json:"inforomImageVersion"`
	DriverVersion               string        `json:"systemGetDriverVersion"`
	CUDADriverVersion           int           `json:"systemGetCudaDriverVersion"`
	GraphicsRunningProcesses    []processInfo `json:"tGraphicsRunningProcesses"`
}

func InitScheduler(cfg *config.Config) error {
	Scheduler = &scheduler{
		gpuStatusMap: make(map[string]byte),
	}

	Scheduler.availableGpuNums = _defaultAvailableGpuNums
	if cfg.AvailableGpuNums != 0 {
		Scheduler.availableGpuNums = cfg.AvailableGpuNums
	}

	resp, err := http.Get(cfg.DetectGPUAddr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var gpuInfos []gpuInfo
	err = json.Unmarshal(body, &gpuInfos)
	if err != nil {
		return err
	}

	for i := 0; i < len(gpuInfos); i++ {
		Scheduler.gpuStatusMap[gpuInfos[i].UUID] = 0
	}
	return nil
}

func Close() {
	fmt.Println("写入 etcd....")
}

// ApplyGpus 申请一定数量的 GPU
func (s *scheduler) ApplyGpus(num int) ([]string, error) {
	if num <= 0 || num > s.availableGpuNums {
		return nil, errors.New("num must be greater than 0 and less than " + strconv.Itoa(s.availableGpuNums))
	}

	s.Lock()
	defer s.Unlock()

	// 可用的 gpu
	var availableGpus []string
	for k, v := range s.gpuStatusMap {
		if v == 0 {
			availableGpus = append(availableGpus, k)
		}
	}

	// 小于用户申请的
	if len(availableGpus) < num {
		return nil, xerrors.NewGpuNotEnoughError()
	}

	needGpus := availableGpus[:num]
	for _, v := range needGpus {
		s.gpuStatusMap[v] = 1
	}
	return needGpus, nil
}

// RestoreGpus 归还一定数量的 GPU
func (s *scheduler) RestoreGpus(gpus []string) {
	if len(gpus) <= 0 || len(gpus) > s.availableGpuNums {
		return
	}

	s.Lock()
	defer s.Unlock()

	for _, gpu := range gpus {
		s.gpuStatusMap[gpu] = 0
	}
}

// GetGpuStatus 获取 GPU 使用信息
func (s *scheduler) GetGpuStatus() map[string]byte {
	s.RLock()
	defer s.RUnlock()

	return s.gpuStatusMap
}
