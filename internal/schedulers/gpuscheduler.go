package schedulers

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
	allGpuUUIDCommand = "nvidia-smi --query-gpu=index,uuid --format=csv,noheader,nounits"

	gpuStatusMapKey = "gpuStatusMapKey"
)

var GpuScheduler *gpuScheduler

type gpu struct {
	Index int     `json:"index"`
	UUID  *string `json:"uuid"`
}

type gpuScheduler struct {
	sync.RWMutex

	AvailableGpuNums int             `json:"availableGpuNums"`
	GpuStatusMap     map[string]byte `json:"gpuStatusMap"`
}

func InitGPuScheduler() error {
	var err error
	GpuScheduler, err = initGpuFormEtcd()
	if err != nil {
		return errors.Wrap(err, "initFormEtcd failed")
	}

	if GpuScheduler.AvailableGpuNums == 0 || len(GpuScheduler.GpuStatusMap) == 0 {
		// if it has not been initialized
		gpus, err := getAllGpuUUID()
		if err != nil {
			return errors.Wrap(err, "getAllGpuUUID failed")
		}

		GpuScheduler.AvailableGpuNums = len(gpus)
		for i := 0; i < len(gpus); i++ {
			GpuScheduler.GpuStatusMap[*gpus[i].UUID] = 0
		}
	}
	return nil
}

func CloseGpuScheduler() error {
	return etcd.Put(etcd.Gpus, gpuStatusMapKey, GpuScheduler.serialize())
}

func initGpuFormEtcd() (s *gpuScheduler, err error) {
	bytes, err := etcd.GetValue(etcd.Gpus, gpuStatusMapKey)
	if err != nil {
		if xerrors.IsNotExistInEtcdError(err) {
			err = nil
		} else {
			return s, err
		}
	}

	s = &gpuScheduler{
		GpuStatusMap: make(map[string]byte),
	}
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &s)
	}
	return s, err
}

// Apply for a specified number of gpus
func (gs *gpuScheduler) Apply(num int) ([]string, error) {
	if num <= 0 || num > gs.AvailableGpuNums {
		return nil, errors.New("num must be greater than 0 and less than " + strconv.Itoa(gs.AvailableGpuNums))
	}

	gs.Lock()
	defer gs.Unlock()

	var availableGpus []string
	for k, v := range gs.GpuStatusMap {
		if v == 0 {
			gs.GpuStatusMap[k] = 1
			availableGpus = append(availableGpus, k)
			if len(availableGpus) == num {
				break
			}
		}
	}

	if len(availableGpus) < num {
		return nil, xerrors.NewGpuNotEnoughError()
	}

	return availableGpus, nil
}

// Restore a specified number of gpu
func (gs *gpuScheduler) Restore(gpus []string) {
	if len(gpus) <= 0 || len(gpus) > gs.AvailableGpuNums {
		return
	}

	gs.Lock()
	defer gs.Unlock()

	for _, gpu := range gpus {
		gs.GpuStatusMap[gpu] = 0
	}
}

func (gs *gpuScheduler) serialize() *string {
	gs.RLock()
	defer gs.RUnlock()

	bytes, _ := json.Marshal(gs)
	tmp := string(bytes)
	return &tmp
}

func (gs *gpuScheduler) GetGpuStatus() map[string]byte {
	gs.RLock()
	defer gs.RUnlock()

	copyMap := make(map[string]byte, len(gs.GpuStatusMap))
	for k, v := range gs.GpuStatusMap {
		copyMap[k] = v
	}

	return copyMap
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
				return gpuList, errors.Errorf("invaild index: %s, ", fields[0])
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
