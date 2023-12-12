package service

import (
	"context"
	"sync"

	"github.com/mayooot/gpu-docker-api/internal/etcd"

	"github.com/ngaut/log"
)

const _maxContainerCount = 110

var WorkQueue chan interface{}
var (
	cs ContainerService
	vs VolumeService
)

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
			case etcd.PutKeyValue:
				switch v.Resource {
				case etcd.ContainerPrefix:
					go func() {
						wg.Add(1)
						defer wg.Done()
						if err := etcd.PutContainerInfo(ctx, v.Key, v.Value); err != nil {
							log.Error(err.Error())
							WorkQueue <- v
							return
						}
						log.Infof("put to etcd successfully, key: %s, value: %s", *v.Key, *v.Value)
					}()
				case etcd.VolumePrefix:
					go func() {
						wg.Add(1)
						defer wg.Done()
						if err := etcd.PutVolumeInfo(ctx, v.Key, v.Value); err != nil {
							log.Error(err.Error())
							WorkQueue <- v
							return
						}
						log.Infof("put to etcd successfully, key: %s, value: %s", *v.Key, *v.Value)
					}()
				default:
					// do nothing
				}
			case *copyTask:
				switch v.Resource {
				case etcd.ContainerPrefix:
					go func() {
						wg.Add(1)
						defer wg.Done()
						if err := cs.copyMergedDirToContainer(v); err != nil {
							log.Error(err.Error())
							return
						}
						log.Infof("copy merged to volume successfully, task: %+v", *v)
					}()
				case etcd.VolumePrefix:
					go func() {
						wg.Add(1)
						defer wg.Done()
						if err := vs.copyMountpointToContainer(v); err != nil {
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
	close(WorkQueue)
}
