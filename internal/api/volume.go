package api

import (
	"github.com/mayooot/gpu-docker-api/internal/model"
	"github.com/mayooot/gpu-docker-api/internal/service"
	"github.com/pkg/errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngaut/log"
)

var vs service.VolumeService

type VolumeHandler struct{}

func (vh *VolumeHandler) RegisterRoute(g *gin.RouterGroup) {
	g.POST("/volumes", vh.create)
	g.DELETE("/volumes/:name", vh.delete)
	g.PATCH("/volumes/:name/size", vh.patchSize)
}

func (vh *VolumeHandler) create(c *gin.Context) {
	var spec model.VolumeCreate
	err := c.ShouldBindJSON(&spec)
	if err != nil {
		log.Error("failed to create volume, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	if strings.Contains(spec.Name, "-") {
		log.Error("failed to create volume, volume name contain '-'")
		ResponseError(c, CodeContainerNameNotContainsSpecialChar)
		return
	}

	resp, err := vs.CreateVolume(&spec)
	if err != nil {
		log.Error(err.Error())
		if errors.Is(err, service.ErrorVolumeExisted) {
			ResponseError(c, CodeVolumeExisted)
			return
		}
		ResponseError(c, CodeVolumeCreateFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"name": resp.Name,
		"size": resp.Options["size"],
	})
}

func (vh *VolumeHandler) delete(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to delete volume, name is empty")
		ResponseError(c, CodeVolumeNameNotNull)
		return
	}

	err := vs.DeleteVolume(&name)
	if err != nil {
		log.Error(err.Error())
		ResponseError(c, CodeVolumeDeleteFailed)
		return
	}

	ResponseSuccess(c, nil)
}

func (vh *VolumeHandler) patchSize(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to patch volume size, name is empty")
		ResponseError(c, CodeVolumeNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Error("failed to patch container gpu info, name must be in format: name-version")
		ResponseError(c, CodeContainerNameMustContainVersion)
		return
	}

	var spec model.VolumeSize
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to patch container gpu info, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	resp, err := vs.PatchVolumeSize(name, &spec)
	if err != nil {
		log.Error(err.Error())
		ResponseError(c, CodeContainerPatchGpuInfoFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"name": resp.Name,
		"size": resp.Options["size"],
	})
}
