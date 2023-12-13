package main

import (
	"context"
	"fmt"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/judwhite/go-svc"
	"github.com/ngaut/log"

	"github.com/mayooot/gpu-docker-api/internal/api"
	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/service"
)

var (
	BRANCH     string
	VERSION    string
	COMMIT     string
	GO_VERSION string
	BUILD_TIME string
)

type program struct {
	ctx context.Context
	wg  sync.WaitGroup
}

func main() {
	fmt.Printf("GPU-DOCKER-API\n BRANCH: %s\n Version: %s\n COMMIT: %s\n GO_VERSION: %s\n BUILD_TIME: %s\n\n", BRANCH, VERSION, COMMIT, GO_VERSION, BUILD_TIME)
	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	p.ctx = context.Background()
	log.SetLevelByString("info")

	err := docker.InitDockerClient()
	if err != nil {
		return err
	}

	err = etcd.InitEtcdClient()
	if err != nil {
		return err
	}

	service.InitWorkQueue()

	return nil
}

func (p *program) Start() error {
	var (
		ch api.ContainerHandler
		vh api.VolumeHandler
	)

	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	apiv1 := r.Group("/api/v1")
	ch.RegisterRoute(apiv1)
	vh.RegisterRoute(apiv1)

	go func() {
		_ = r.Run(":2378")
	}()

	go service.SyncLoop(p.ctx, &p.wg)

	return nil
}

func (p *program) Stop() error {
	p.wg.Wait()
	p.ctx.Done()

	log.Info("stopping gpu-docker-api")
	docker.CloseDockerClient()
	etcd.CloseEtcdClient()
	service.Close()
	return nil
}
