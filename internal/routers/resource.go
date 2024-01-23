package routers

import (
	"github.com/gin-gonic/gin"

	"github.com/mayooot/gpu-docker-api/internal/schedulers"
)

type Resource struct{}

func (gh *Resource) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/resources/gpus", gh.GetGpus)
	g.GET("resources/ports", gh.GetPorts)
}

// GetGpus 0 means not used, 1 means used.
func (gh *Resource) GetGpus(c *gin.Context) {
	gpus := schedulers.GpuScheduler.GetGpuStatus()
	ResponseSuccess(c, gin.H{
		"gpus": gpus,
	})
}

func (gh *Resource) GetPorts(c *gin.Context) {
	status := schedulers.PortScheduler.GetPortStatus()
	status.AvailableCount = status.AvailableCount - len(status.UsedPortSet)
	ResponseSuccess(c, gin.H{
		"ports": status,
	})
}
