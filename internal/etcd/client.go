package etcd

import (
	"time"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

var cli *clientv3.Client

func InitEtcdClient(addr string) error {
	var err error
	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{addr},
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
