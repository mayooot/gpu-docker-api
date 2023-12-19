package etcd

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/mayooot/gpu-docker-api/internal/config"
)

var cli *clientv3.Client

func InitEtcdClient(cfg *config.Config) error {
	var err error
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.EtcdAddr},
		DialTimeout: 5 * time.Second,
	})
	return err
}

func CloseEtcdClient() {
	cli.Close()
}
