package service

import (
	"context"
	"sync"

	"github.com/mayooot/gpu-docker-api/internal/etcd"

	"github.com/ngaut/log"
)

const _maxContainerCount = 110

var WorkQueue chan interface{}
var cs ContainerService

func InitWorkQueue() {
	WorkQueue = make(chan interface{}, _maxContainerCount)
}

// SyncLoop 将容器的创建信息同步到etcd，当程序收到停止信号时，已经开始的 put 任务会继续执行
// 但是没有开始的任务，不会被执行，SyncLoop 会直接返回
func SyncLoop(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case v := <-WorkQueue:
			switch v := v.(type) {
			case etcd.KeyValue:
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := etcd.PutContainerInfo(ctx, v.Key, v.Value); err != nil {
						log.Error(err.Error())
						WorkQueue <- v
						return
					}
					log.Infof("put to etcd successfully, key: %s, value: %s", *v.Key, *v.Value)
				}()
			case *copyTask:
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := cs.copyMergedDirToContainer(v); err != nil {
						log.Error(err.Error())
						return
					}
					log.Infof("copy diff to container successfully, task: %+v", *v)
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
	close(WorkQueue)
}
