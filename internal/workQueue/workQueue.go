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

// SyncLoop 将容器的创建信息同步到etcd，当程序收到停止信号时，已经开始的 put 任务会继续执行
// 但是没有开始的任务，不会被执行，SyncLoop 会直接返回
func SyncLoop(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case v := <-Queue:
			switch v := v.(type) {
			case etcd.PutKeyValue:
				go func() {
					wg.Add(1)
					defer wg.Done()
					if err := etcd.Put(v.Resource, v.Key, v.Value); err != nil {
						log.Error(err.Error())
						Queue <- v
						return
					}
					log.Infof("put to etcd successfully, resource %s, key: %s, value: %s", v.Resource, v.Key, *v.Value)
				}()
			case etcd.DelKey:
				go func() {
					wg.Add(1)
					defer wg.Done()
					if err := etcd.Del(v.Resource, v.Key); err != nil {
						log.Error(err.Error())
						Queue <- v
						return
					}
					log.Infof("delete etcd key successfully, resource %s, key: %s", v.Resource, v.Key)
				}()
			case *CopyTask:
				switch v.Resource {
				case etcd.Containers:
					go func() {
						wg.Add(1)
						defer wg.Done()
						if err := copyMergedDirToContainer(v); err != nil {
							log.Error(err.Error())
							return
						}
						log.Infof("copy merged to volume successfully, task: %+v", *v)
					}()
				case etcd.Volumes:
					go func() {
						wg.Add(1)
						defer wg.Done()
						if err := copyMountPointToContainer(v); err != nil {
							log.Error(err.Error())
							return
						}
						log.Infof("copy mountpoint to volume successfully, task: %+v", *v)
					}()
				}
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
