package etcd

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var cli *clientv3.Client

func InitEtcdClient() error {
	var err error
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	return err
}

func CloseEtcdClient() {
	cli.Close()
}
