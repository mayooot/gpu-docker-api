package version

import (
	"os"
	"sync"
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

func TestVersionMapNextIsAtomic(t *testing.T) {
	vm := newVersionMap("test")

	const n = 200
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = make(map[int64]bool, n)
	)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := vm.Next("foo")
			mu.Lock()
			seen[v] = true
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(seen) != n {
		t.Fatalf("expected %d distinct versions, got %d (duplicate versions handed out)", n, len(seen))
	}
	if v, _ := vm.Get("foo"); v != n {
		t.Fatalf("expected latest version %d, got %d", n, v)
	}
}

func TestVersionMapRevert(t *testing.T) {
	vm := newVersionMap("test")

	v := vm.Next("foo")
	vm.Revert("foo", v)
	if vm.Exist("foo") {
		t.Fatal("reverting the first version should remove the record")
	}

	vm.Next("foo")       // 1
	v2 := vm.Next("foo") // 2
	vm.Next("foo")       // 3, someone else moved forward
	vm.Revert("foo", v2)
	if got, _ := vm.Get("foo"); got != 3 {
		t.Fatalf("revert must not roll back a newer version, got %d", got)
	}
}

func TestMergeMapKeyedByContainerName(t *testing.T) {
	mm := newMergedMap()
	mm.Set("foo-1", "/merges/foo/foo-1")
	mm.Set("bar-1", "/merges/bar/bar-1")

	if p, _ := mm.Get("foo-1"); p != "/merges/foo/foo-1" {
		t.Fatalf("foo-1 was overwritten by another replicaSet with the same version, got %q", p)
	}
}
