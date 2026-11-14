package schedulers

import (
	"os"
	"testing"

	"github.com/mayooot/gpu-docker-api/internal/workQueue"
)

// TestMain initialises the global work queue once and drains it,
// so that persisting snapshots never blocks in tests.
func TestMain(m *testing.M) {
	workQueue.InitWorkQueue()
	go func() {
		for range workQueue.Queue {
		}
	}()
	os.Exit(m.Run())
}

func TestGpuApplyFailureDoesNotLeak(t *testing.T) {
	gs := &gpuScheduler{
		AvailableGpuNums: 4,
		GpuStatusMap:     map[string]byte{"a": 0, "b": 0, "c": 1, "d": 1},
	}

	if _, err := gs.Apply(3); err == nil {
		t.Fatal("expected not enough gpus")
	}
	for k, v := range gs.GetGpuStatus() {
		if (k == "a" || k == "b") && v != 0 {
			t.Fatalf("gpu %s leaked after a failed Apply", k)
		}
	}

	got, err := gs.Apply(2)
	if err != nil || len(got) != 2 {
		t.Fatalf("expected to get the 2 free gpus, got %v, err %v", got, err)
	}
}

func TestPortApplyFailureDoesNotLeak(t *testing.T) {
	ps := &portScheduler{
		StartPort:      40000,
		EndPort:        40002,
		AvailableCount: 3,
		UsedPortSet:    map[string]struct{}{"40001": {}},
	}

	if _, err := ps.Apply(3); err == nil {
		t.Fatal("expected not enough ports")
	}
	if len(ps.UsedPortSet) != 1 {
		t.Fatalf("ports leaked after a failed Apply: %v", ps.UsedPortSet)
	}

	got, err := ps.Apply(2)
	if err != nil || len(got) != 2 {
		t.Fatalf("expected to get the 2 free ports, got %v, err %v", got, err)
	}
}
