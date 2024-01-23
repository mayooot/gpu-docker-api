package main

import (
	"context"
	goflag "flag"
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/judwhite/go-svc"
	"github.com/ngaut/log"
	flag "github.com/spf13/pflag"

	"github.com/mayooot/gpu-docker-api/internal/docker"
	"github.com/mayooot/gpu-docker-api/internal/etcd"
	"github.com/mayooot/gpu-docker-api/internal/routers"
	"github.com/mayooot/gpu-docker-api/internal/schedulers"
	"github.com/mayooot/gpu-docker-api/internal/version"
	"github.com/mayooot/gpu-docker-api/internal/workQueue"
	"github.com/mayooot/gpu-docker-api/utils"
)

var (
	BRANCH    string
	VERSION   string
	COMMIT    string
	GoVersion string
	BuildTime string
)

var (
	addr      = flag.StringP("addr", "a", "0.0.0.0:2378", "Address of gpu-docker-routers server,format: ip:port")
	etcdAddr  = flag.StringP("etcd", "e", "0.0.0.0:2379", "Address of etcd server,format: ip:port")
	portRange = flag.StringP("portRange", "p", "40000-65535", "Port range of docker container,format: startPort-endPort")
	logLevel  = flag.StringP("logLevel", "l", "debug", "Log level, optional: release")
)

type program struct {
	ctx context.Context
	wg  sync.WaitGroup
}

func main() {
	fmt.Printf("GPU-DOCKER-API\n BRANCH: %s\n Version: %s\n COMMIT: %s\n GoVersion: %s\n BuildTime: %s\n\n", BRANCH, VERSION, COMMIT, GoVersion, BuildTime)
	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		log.Fatal(err.Error())
	}
}

func (p *program) Init(svc.Environment) (err error) {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
	p.ctx = context.Background()
	log.SetLevelByString(*logLevel)

	if err = docker.InitDockerClient(); err != nil {
		return
	}

	if err = etcd.InitEtcdClient(*etcdAddr); err != nil {
		return
	}

	workQueue.InitWorkQueue()

	if err = schedulers.InitGPuScheduler(); err != nil {
		return
	}

	if err = schedulers.InitPortScheduler(*portRange); err != nil {
		return
	}

	if err = version.InitVersionMap(); err != nil {
		return
	}

	if err = version.InitMergedMap(); err != nil {
		return
	}

	//  create merges dir, that used to store container merged layer
	layer := "merges"
	if err := utils.IsDir(layer); err != nil {
		_ = os.Mkdir(layer, 0755)
		err = nil
	}

	return
}

func (p *program) Start() error {
	var (
		ch routers.ReplicaSetHandler
		vh routers.VolumeHandler
		gh routers.Resource
	)

	fmt.Printf("CONFIG\n addr: %s\n etcdAddr: %s\n portRange: %s\n logLevel: %s\n\n", *addr, *etcdAddr, *portRange, *logLevel)
	log.Info("gpu-docker-routers started successfully!")

	gin.SetMode(*logLevel)
	r := gin.New()
	r.Use(routers.Cors())
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
		_ = r.Run(*addr)
	}()

	go workQueue.SyncLoop(p.ctx, &p.wg)

	return nil
}

func (p *program) Stop() error {
	log.Info("gpu-docker-routers is stopping...")
	p.ctx.Done()
	p.wg.Wait()

	workQueue.Close()
	docker.CloseDockerClient()
	_ = schedulers.CloseGpuScheduler()
	_ = schedulers.ClosePortScheduler()
	_ = version.CloseVersionMap()
	_ = version.CloseMergedMap()
	_ = etcd.CloseEtcdClient()
	log.Info("gpu-docker-routers stopped successfully!")
	return nil
}
