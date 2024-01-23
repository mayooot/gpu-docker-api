package workQueue

import (
	"context"
	"sync"

	"github.com/ngaut/log"

	"github.com/mayooot/gpu-docker-api/internal/etcd"
)

const _maxContainerCount = 110

var Queue chan interface{}

func InitWorkQueue() {
	Queue = make(chan interface{}, _maxContainerCount)
}

func SyncLoop(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case v := <-Queue:
			switch v := v.(type) {
			case etcd.PutKeyValue:
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := etcd.Put(v.Resource, v.Key, v.Value); err != nil {
						log.Error(err.Error())
						Queue <- v
						return
					}
					log.Infof("put to etcd successfully, resource %s, key: %s, value: %s", v.Resource, v.Key, *v.Value)
				}()
			case etcd.DelKey:
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := etcd.Del(v.Resource, v.Key); err != nil {
						log.Error(err.Error())
						Queue <- v
						return
					}
					log.Infof("delete etcd key successfully, resource %s, key: %s", v.Resource, v.Key)
				}()
			default:
				//	nothing to do
			}
		case <-ctx.Done():
			return
		}
	}
}

func Close() {
	close(Queue)
}
