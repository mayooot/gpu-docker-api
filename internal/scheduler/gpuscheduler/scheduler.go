package gpuscheduler

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/mayooot/gpu-docker-api/internal/config"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/model"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

const (
	// 默认的可用GPU 数量
	defaultAvailableGpuNums = 8

	// gpuScheduler 存储在 etcd 中的 key
	gpuStatusMapKey = "gpuStatusMapKey"
)

var Scheduler *scheduler

type scheduler struct {
	sync.RWMutex

	AvailableGpuNums int
	GpuStatusMap     map[string]byte
}

func Init(cfg *config.Config) error {
	var err error
	Scheduler, err = initFormEtcd()
	if err != nil {
		return err
	}

	if Scheduler.AvailableGpuNums == 0 || len(Scheduler.GpuStatusMap) == 0 {
		// 如果没有初始化过
		Scheduler.AvailableGpuNums = defaultAvailableGpuNums
		if cfg.AvailableGpuNums >= 0 {
			Scheduler.AvailableGpuNums = cfg.AvailableGpuNums
		}

		gpus, err := getDetectGpus(cfg.DetectGPUAddr)
		if err != nil {
			return err
		}
		for i := 0; i < len(gpus); i++ {
			Scheduler.GpuStatusMap[gpus[i].UUID] = 0
		}
	}
	return nil
}

func Close() error {
	return etcd.Put(etcd.Gpus, gpuStatusMapKey, Scheduler.serialize())
}

// ApplyGpus 申请一定数量的 GPU
func (s *scheduler) ApplyGpus(num int) ([]string, error) {
	if num <= 0 || num > s.AvailableGpuNums {
		return nil, errors.New("num must be greater than 0 and less than " + strconv.Itoa(s.AvailableGpuNums))
	}

	s.Lock()
	defer s.Unlock()

	// 可用的 gpu
	var availableGpus []string
	for k, v := range s.GpuStatusMap {
		if v == 0 {
			s.GpuStatusMap[k] = 1
			availableGpus = append(availableGpus, k)
			if len(availableGpus) == num {
				break
			}
		}
	}

	// 小于用户申请的
	if len(availableGpus) < num {
		return nil, xerrors.NewGpuNotEnoughError()
	}

	return availableGpus, nil
}

// RestoreGpus 归还一定数量的 GPU
func (s *scheduler) RestoreGpus(gpus []string) {
	if len(gpus) <= 0 || len(gpus) > s.AvailableGpuNums {
		return
	}

	s.Lock()
	defer s.Unlock()

	for _, gpu := range gpus {
		s.GpuStatusMap[gpu] = 0
	}
}

// GetGpusStatus 获取 GPU 使用信息
func (s *scheduler) GetGpusStatus() map[string]byte {
	s.RLock()
	defer s.RUnlock()

	return s.GpuStatusMap
}

func (s *scheduler) serialize() *string {
	s.RLock()
	defer s.RUnlock()

	bytes, _ := json.Marshal(s)
	tmp := string(bytes)
	return &tmp
}

func initFormEtcd() (s *scheduler, err error) {
	bytes, err := etcd.Get(etcd.Gpus, gpuStatusMapKey)
	if err != nil {
		return s, err
	}

	s = &scheduler{
		GpuStatusMap: make(map[string]byte),
	}
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &s)
	}
	return s, err
}

func getDetectGpus(addr string) (gpus []model.GpuInfo, err error) {
	resp, err := http.Get(addr)
	if err != nil {
		return gpus, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return gpus, err
	}

	if err = json.Unmarshal(body, &gpus); err != nil {
		return gpus, err
	}
	return gpus, err
}
