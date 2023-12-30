package portscheduler

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"sync"

	"github.com/mayooot/gpu-docker-api/internal/config"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

const (
	defaultStartPort   = 40000
	defaultEndPort     = 65535
	AvailablePortCount = defaultEndPort - defaultStartPort + 1
	usedPortSetKey     = "usedPortSetKey"
)

var Scheduler *scheduler

type scheduler struct {
	sync.RWMutex
	StartPort      int
	EndPort        int
	AvailableCount int
	UsedPortSet    map[int]struct{}
}

type Alias struct {
	StartPort      int
	EndPort        int
	AvailableCount int
	UsedPortSet    []int
}

// MarshalJSON 重载序列化结构体为 JSON 的方法
// 如果直接将 scheduler 序列化，UsedPortSet 字段以 map 的形式输出，value为 struct{}{}，而且是乱序的
func (s *scheduler) MarshalJSON() ([]byte, error) {
	onlyKeys := make([]int, 0, len(s.UsedPortSet))
	for k, _ := range s.UsedPortSet {
		onlyKeys = append(onlyKeys, k)
	}
	sort.Ints(onlyKeys)
	return json.Marshal(Alias{
		s.StartPort,
		s.EndPort,
		s.AvailableCount,
		onlyKeys,
	})
}

func Init(cfg *config.Config) error {
	var err error
	Scheduler, err = initFormEtcd()
	if err != nil {
		return err
	}

	if Scheduler.StartPort == 0 || Scheduler.EndPort == 0 || Scheduler.AvailableCount == 0 {
		// 如果没有初始化过
		Scheduler.StartPort = defaultStartPort
		Scheduler.EndPort = defaultEndPort
		Scheduler.AvailableCount = AvailablePortCount
		if cfg.StartPort >= 0 && cfg.EndPort >= 0 {
			Scheduler.StartPort = cfg.StartPort
			Scheduler.EndPort = cfg.EndPort
			Scheduler.AvailableCount = cfg.EndPort - cfg.StartPort + 1
		}
	}

	return nil
}

func Close() error {
	return etcd.Put(etcd.Ports, usedPortSetKey, Scheduler.serialize())
}

// ApplyPorts 申请一定数量的宿主机端口号
func (s *scheduler) ApplyPorts(num int) ([]int, error) {
	if num <= 0 || num > s.AvailableCount {
		return nil, errors.New("num must be greater than 0 and less than " + strconv.Itoa(s.AvailableCount))
	}

	s.Lock()
	defer s.Unlock()

	// 可用的端口
	var availablePorts []int
	for i := s.StartPort; i <= s.EndPort; i++ {
		if _, ok := s.UsedPortSet[i]; !ok {
			s.UsedPortSet[i] = struct{}{}
			availablePorts = append(availablePorts, i)
			if len(availablePorts) == num {
				break
			}
		}
	}

	// 小于用户申请的
	if len(availablePorts) < num {
		return nil, xerrors.NewPortNotEnoughError()
	}

	return availablePorts, nil
}

// RestorePorts 归还一定数量的宿主机端口号
func (s *scheduler) RestorePorts(ports []int) {
	if len(ports) <= 0 || len(ports) > s.AvailableCount {
		return
	}

	s.Lock()
	defer s.Unlock()

	for _, p := range ports {
		delete(s.UsedPortSet, p)
	}
}

// GetUsedPortSet 获取 GPU 使用信息
func (s *scheduler) GetUsedPortSet() map[int]struct{} {
	s.RLock()
	defer s.RUnlock()

	return s.UsedPortSet
}

func (s *scheduler) serialize() *string {
	s.RLock()
	defer s.RUnlock()

	bytes, _ := json.Marshal(s)
	tmp := string(bytes)
	return &tmp
}

func initFormEtcd() (s *scheduler, err error) {
	bytes, err := etcd.Get(etcd.Ports, usedPortSetKey)
	if err != nil {
		return s, err
	}
	var alias Alias
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &alias)
	}

	s = &scheduler{
		StartPort:      alias.StartPort,
		EndPort:        alias.EndPort,
		AvailableCount: alias.AvailableCount,
		UsedPortSet:    make(map[int]struct{}, len(alias.UsedPortSet)),
	}

	for _, port := range alias.UsedPortSet {
		s.UsedPortSet[port] = struct{}{}
	}
	return s, err
}
