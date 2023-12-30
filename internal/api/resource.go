package api

import (
	"github.com/gin-gonic/gin"

	"github.com/mayooot/gpu-docker-api/internal/scheduler/gpuscheduler"
	"github.com/mayooot/gpu-docker-api/internal/scheduler/portscheduler"
)

type Resource struct{}

func (gh *Resource) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/resources/gpus", gh.getGpus)
	g.GET("resources/usedPorts", gh.getUsedPorts)
}

func (gh *Resource) getGpus(c *gin.Context) {
	gpus := gpuscheduler.Scheduler.GetGpusStatus()
	ResponseSuccess(c, gin.H{
		"gpuStatus": gpus,
	})
}

func (gh *Resource) getUsedPorts(c *gin.Context) {
	ports := portscheduler.Scheduler.GetUsedPortSet()
	ResponseSuccess(c, gin.H{
		"usedPorts": ports,
	})
}
