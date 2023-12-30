package etcd

import (
	"context"
	"time"

	"github.com/pkg/errors"
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
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if _, err = cli.Put(ctx, "/ping", "pong"); err != nil {
		return errors.Wrap(err, "etcd client init failed")
	}
	return nil
}

func CloseEtcdClient() error {
	return cli.Close()
}
