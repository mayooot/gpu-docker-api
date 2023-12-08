package api

import (
	"github.com/mayooot/gpu-docker-api/internal/model"
	"github.com/mayooot/gpu-docker-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/ngaut/log"
)

var volumeService service.VolumeService

type VolumeHandler struct{}

func (vh *VolumeHandler) RegisterRoute(g *gin.RouterGroup) {
	g.POST("/volumes", vh.create)
	g.DELETE("/volumes/:name", vh.delete)
}

func (vh *VolumeHandler) create(c *gin.Context) {
	var spec model.VolumeCreate
	err := c.ShouldBindJSON(&spec)
	if err != nil {
		log.Error("failed to create volume, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	resp, err := volumeService.CreateVolume(&spec)
	if err != nil {
		log.Error(err.Error())
		ResponseError(c, CodeVolumeCreateFailed)
		return
	}

	ResponseSuccess(c, resp)
}

func (vh *VolumeHandler) delete(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to delete volume, name is empty")
		ResponseError(c, CodeVolumeNameNotNull)
		return
	}

	err := volumeService.DeleteVolume(&name)
	if err != nil {
		log.Error(err.Error())
		ResponseError(c, CodeVolumeDeleteFailed)
		return
	}

	ResponseSuccess(c, nil)
}
