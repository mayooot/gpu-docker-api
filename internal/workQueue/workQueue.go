package workQueue

import (
	"context"
	"sync"
	"time"

	"github.com/ngaut/log"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
)

const (
	_maxContainerCount = 110

	_initialBackoff = 100 * time.Millisecond
	_maxBackoff     = 5 * time.Second
)

var Queue chan interface{}

func InitWorkQueue() {
	Queue = make(chan interface{}, _maxContainerCount)
}

// Enqueue adds a task to the queue. Tasks are applied to etcd strictly in enqueue order.
func Enqueue(v interface{}) {
	Queue <- v
}

// SyncLoop applies tasks to etcd with a single worker.
//
// Earlier versions spawned one goroutine per task, so two writes to the same key
// could land in etcd out of order (an older snapshot overwriting a newer one),
// and a failed task was pushed straight back to the queue, spinning hot while etcd was down.
// Now tasks are processed one by one, and a failed task is retried in place with backoff,
// which keeps the order intact.
//
// The caller must call wg.Add(1) before starting it in a goroutine.
func SyncLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case v := <-Queue:
			process(ctx, v)
		case <-ctx.Done():
			drain()
			return
		}
	}
}

// drain flushes the tasks that are still buffered when the loop is asked to stop,
// with a single attempt each, so that a graceful shutdown does not lose pending writes.
func drain() {
	for {
		select {
		case v := <-Queue:
			if err := apply(v); err != nil {
				log.Errorf("workQueue drain: %v", err)
			}
		default:
			return
		}
	}
}

func process(ctx context.Context, v interface{}) {
	backoff := _initialBackoff
	for {
		err := apply(v)
		if err == nil {
			return
		}
		log.Errorf("workQueue: %v, retry in %s", err, backoff)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			// give it one last try during shutdown
			if err := apply(v); err != nil {
				log.Errorf("workQueue: give up after shutdown: %v", err)
			}
			return
		}
		backoff *= 2
		if backoff > _maxBackoff {
			backoff = _maxBackoff
		}
	}
}

func apply(v interface{}) error {
	switch v := v.(type) {
	case etcd.PutKeyValue:
		if err := etcd.Put(v.Resource, v.Key, v.Value); err != nil {
			return err
		}
		log.Debugf("put to etcd successfully, resource %s, key: %s", v.Resource, v.Key)
	case etcd.DelKey:
		if err := etcd.Del(v.Resource, v.Key); err != nil {
			return err
		}
		log.Infof("delete etcd key successfully, resource %s, key: %s", v.Resource, v.Key)
	default:
		log.Warnf("workQueue: unknown task type %T, dropped", v)
	}
	return nil
}
