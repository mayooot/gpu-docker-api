package etcd

import (
	"github.com/pkg/errors"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"

	"github.com/mayooot/gpu-docker-api/internal/config"
)

var cli *clientv3.Client

func InitEtcdClient(cfg *config.Config) error {
	var err error
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.EtcdAddr},
		DialTimeout: 2 * time.Second,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		return errors.Wrap(err, "failed to connect etcd")
	}
	return nil
}

func CloseEtcdClient() error {
	return cli.Close()
}
