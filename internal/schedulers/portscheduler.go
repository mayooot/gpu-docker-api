package schedulers

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

const usedPortSetKey = "usedPortSetKey"

var PortScheduler *portScheduler

type portScheduler struct {
	sync.RWMutex

	StartPort      int
	EndPort        int
	AvailableCount int
	UsedPortSet    map[string]struct{}
}

func InitPortScheduler(portRange string) error {
	var err error
	PortScheduler, err = initPortFormEtcd()
	if err != nil {
		return errors.Wrap(err, "initFormEtcd failed")
	}

	if PortScheduler.StartPort == 0 || PortScheduler.EndPort == 0 || PortScheduler.AvailableCount == 0 {
		// if it has not been initialized
		startPort, endPort, err := splitPortRange(portRange)
		if err != nil {
			return errors.Wrap(err, "splitPortRange failed")
		}
		PortScheduler.StartPort = startPort
		PortScheduler.EndPort = endPort
		PortScheduler.AvailableCount = PortScheduler.EndPort - PortScheduler.StartPort + 1
	}

	return nil
}

func ClosePortScheduler() error {
	return etcd.Put(etcd.Ports, usedPortSetKey, PortScheduler.serialize())
}

func initPortFormEtcd() (s *portScheduler, err error) {
	bytes, err := etcd.GetValue(etcd.Ports, usedPortSetKey)
	if err != nil {
		if xerrors.IsNotExistInEtcdError(err) {
			err = nil
		} else {
			return s, err
		}
	}

	s = &portScheduler{
		UsedPortSet: make(map[string]struct{}),
	}
	if len(bytes) != 0 {
		err = json.Unmarshal(bytes, &s)
	}
	return s, err
}

// Apply for a specified number of ports
func (ps *portScheduler) Apply(num int) ([]string, error) {
	if num <= 0 || num > ps.AvailableCount {
		return nil, errors.New("num must be greater than 0 and less than " + strconv.Itoa(ps.AvailableCount))
	}

	ps.Lock()
	defer ps.Unlock()

	var availablePorts []string
	for i := ps.StartPort; i <= ps.EndPort; i++ {
		if _, ok := ps.UsedPortSet[strconv.Itoa(i)]; !ok {
			ps.UsedPortSet[strconv.Itoa(i)] = struct{}{}
			availablePorts = append(availablePorts, strconv.Itoa(i))
			if len(availablePorts) == num {
				break
			}
		}
	}

	if len(availablePorts) < num {
		return nil, xerrors.NewPortNotEnoughError()
	}

	return availablePorts, nil
}

// Restore a specified number of ports
func (ps *portScheduler) Restore(ports []string) {
	if len(ports) <= 0 || len(ports) > ps.AvailableCount {
		return
	}

	ps.Lock()
	defer ps.Unlock()

	for _, port := range ports {
		delete(ps.UsedPortSet, port)
	}
}

func (ps *portScheduler) serialize() *string {
	ps.RLock()
	defer ps.RUnlock()

	bytes, _ := json.Marshal(ps)
	tmp := string(bytes)
	return &tmp
}

// GetPortStatus get all ports status
func (ps *portScheduler) GetPortStatus() *portScheduler {
	ps.RLock()
	defer ps.RUnlock()

	copyPS := &portScheduler{
		StartPort:      ps.StartPort,
		EndPort:        ps.EndPort,
		AvailableCount: ps.AvailableCount,
	}

	// sort
	keys := make([]string, 0, len(ps.UsedPortSet))
	for k := range ps.UsedPortSet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// reset
	copyPS.UsedPortSet = make(map[string]struct{}, len(ps.UsedPortSet))
	for _, k := range keys {
		copyPS.UsedPortSet[k] = struct{}{}
	}

	return copyPS
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
