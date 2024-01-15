package main

import (
	"context"
	goflag "flag"
	"fmt"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/judwhite/go-svc"
	"github.com/ngaut/log"
	flag "github.com/spf13/pflag"

	"github.com/mayooot/gpu-docker-api/internal/api"
	"github.com/mayooot/gpu-docker-api/internal/config"
	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/scheduler/gpuscheduler"
	"github.com/mayooot/gpu-docker-api/internal/scheduler/portscheduler"
	"github.com/mayooot/gpu-docker-api/internal/version"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
)

var (
	BRANCH    string
	VERSION   string
	COMMIT    string
	GoVersion string
	BuildTime string
)

var configFile = flag.StringP("config", "c", "./etc/config.toml", "config file path")

type program struct {
	ctx context.Context
	wg  sync.WaitGroup

	cfg *config.Config
}

func main() {
	fmt.Printf("GPU-DOCKER-API\n BRANCH: %s\n Version: %s\n COMMIT: %s\n GoVersion: %s\n BuildTime: %s\n\n", BRANCH, VERSION, COMMIT, GoVersion, BuildTime)
	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(svc.Environment) error {
	var err error

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	p.ctx = context.Background()
	log.SetLevelByString("info")

	p.cfg, err = config.NewConfigWithFile(*configFile)
	if err != nil {
		return err
	}
	if err = docker.InitDockerClient(); err != nil {
		return err
	}

	if err = etcd.InitEtcdClient(p.cfg); err != nil {
		return err
	}

	workQueue.InitWorkQueue()

	if err = gpuscheduler.Init(); err != nil {
		return err
	}

	if err = portscheduler.Init(p.cfg); err != nil {
		return err
	}

	if err = version.Init(); err != nil {
		return err
	}

	return nil
}

func (p *program) Start() error {
	var (
		ch api.ContainerHandler
		vh api.VolumeHandler
		gh api.Resource
	)

	gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	apiv1 := r.Group("/api/v1")
	ch.RegisterRoute(apiv1)
	vh.RegisterRoute(apiv1)
	gh.RegisterRoute(apiv1)

	go func() {
		_ = r.Run(p.cfg.Port)
	}()

	go workQueue.SyncLoop(p.ctx, &p.wg)

	return nil
}

func (p *program) Stop() error {
	p.ctx.Done()
	p.wg.Wait()

	log.Info("stopping gpu-docker-api")

	workQueue.Close()
	docker.CloseDockerClient()
	gpuscheduler.Close()
	portscheduler.Close()
	version.Close()
	etcd.CloseEtcdClient()
	return nil
}
