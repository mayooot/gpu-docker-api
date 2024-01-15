package gpuscheduler

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/commander-cli/cmd"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

const (
	// 执行命令获取 gpu 的 index 和 uuid
	allGpuUUIDCommand = "nvidia-smi --query-gpu=index,uuid --format=csv,noheader,nounits"

	// gpuScheduler 存储在 etcd 中的 key
	gpuStatusMapKey = "gpuStatusMapKey"
)

var Scheduler *scheduler

type gpu struct {
	Index int     `json:"index"`
	UUID  *string `json:"uuid"`
}

type scheduler struct {
	sync.RWMutex

	AvailableGpuNums int
	GpuStatusMap     map[string]byte
}

func Init() error {
	var err error
	Scheduler, err = initFormEtcd()
	if err != nil {
		return errors.Wrap(err, "initFormEtcd failed")
	}

	if Scheduler.AvailableGpuNums == 0 || len(Scheduler.GpuStatusMap) == 0 {
		// 如果没有初始化过
		gpus, err := getAllGpuUUID()
		if err != nil {
			return errors.Wrap(err, "getAllGpuUUID failed")
		}

		Scheduler.AvailableGpuNums = len(gpus)
		for i := 0; i < len(gpus); i++ {
			Scheduler.GpuStatusMap[*gpus[i].UUID] = 0
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
		if xerrors.IsNotExistInEtcdError(err) {
			err = nil
		} else {
			return s, err
		}
	}

	s = &scheduler{
		GpuStatusMap: make(map[string]byte),
	}
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &s)
	}
	return s, err
}

func getAllGpuUUID() ([]*gpu, error) {
	c := cmd.NewCommand(allGpuUUIDCommand)
	err := c.Execute()
	if err != nil {
		return nil, errors.Wrap(err, "cmd.Execute failed")
	}

	gpuList, err := parseOutput(c.Stdout())
	if err != nil {
		return nil, errors.Wrap(err, "parseOutput failed")
	}
	return gpuList, nil
}

func parseOutput(output string) (gpuList []*gpu, err error) {
	lines := strings.Split(output, "\n")
	gpuList = make([]*gpu, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) == 2 {
			index, err := strconv.Atoi(fields[0])
			if err != nil {
				return gpuList, errors.Wrapf(err, "strconv.Atoi failed, index: %s", fields[0])
			}
			uuid := fields[1]
			gpuList = append(gpuList, &gpu{
				Index: index,
				UUID:  &uuid,
			})
		}
	}
	return
}
