package portscheduler

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

// portScheduler 存储在 etcd 中的 key
const usedPortSetKey = "usedPortSetKey"

var Scheduler *scheduler

type portParams struct {
	StartPort      int
	EndPort        int
	AvailableCount int
}

type scheduler struct {
	sync.RWMutex

	portParams
	UsedPortSet map[int]struct{}
}

type alias struct {
	portParams
	UsedPortSet []int
}

// MarshalJSON 重载序列化结构体为 JSON 的方法
// 如果直接将 scheduler 序列化，UsedPortSet 字段以 map 的形式输出，value为 struct{}{}，而且是乱序
func (s *scheduler) MarshalJSON() ([]byte, error) {
	onlyKeys := make([]int, 0, len(s.UsedPortSet))
	for k := range s.UsedPortSet {
		onlyKeys = append(onlyKeys, k)
	}
	sort.Ints(onlyKeys)
	return json.Marshal(alias{
		portParams:  s.portParams,
		UsedPortSet: onlyKeys})
}

func Init(portRange string) error {
	var err error
	Scheduler, err = initFormEtcd()
	if err != nil {
		return errors.Wrap(err, "initFormEtcd failed")
	}

	if Scheduler.StartPort == 0 || Scheduler.EndPort == 0 || Scheduler.AvailableCount == 0 {
		// 如果没有初始化过
		startPort, endPort, err := splitPortRange(portRange)
		if err != nil {
			return errors.Wrap(err, "splitPortRange failed")
		}
		Scheduler.StartPort = startPort
		Scheduler.EndPort = endPort
		Scheduler.AvailableCount = Scheduler.EndPort - Scheduler.StartPort + 1
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

// GetPortStatus 获取 Port 使用情况
func (s *scheduler) GetPortStatus() *scheduler {
	s.RLock()
	defer s.RUnlock()
	return s
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
		if xerrors.IsNotExistInEtcdError(err) {
			err = nil
		} else {
			return s, err
		}
	}

	var alias alias
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &alias)
	}

	s = &scheduler{
		portParams:  alias.portParams,
		UsedPortSet: make(map[int]struct{}, len(alias.UsedPortSet)),
	}
	for _, port := range alias.UsedPortSet {
		s.UsedPortSet[port] = struct{}{}
	}
	return s, err
}

func splitPortRange(portRange string) (startPort, endPort int, err error) {
	parts := strings.Split(portRange, "-")
	if len(parts) != 2 {
		return 0, 0, errors.Errorf("invalid port range format, portRange: %s", portRange)
	}

	startPort, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, errors.Errorf("invalid start port: %s", parts[0])
	}

	endPort, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, errors.Errorf("invalid end port: %s", parts[1])
	}

	if startPort < 0 || startPort > 65535 || endPort < 0 || endPort > 65535 || startPort > endPort {
		return 0, 0, errors.Errorf("invalid port range values, startPort: %d, endPort: %d", startPort, endPort)
	}

	return
}
