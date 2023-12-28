package api

import (
	"github.com/gin-gonic/gin"

	"github.com/mayooot/gpu-docker-api/internal/gpuscheduler"
)

type GpuHandler struct{}

func (gh *GpuHandler) RegisterRoute(g *gin.RouterGroup) {
	g.GET("/gpus", gh.getGpu)
}

func (gh *GpuHandler) getGpu(c *gin.Context) {
	gpus := gpuscheduler.Scheduler.GetGpuStatus()
	ResponseSuccess(c, gin.H{
		"gpuStatus": gpus,
	})
}
