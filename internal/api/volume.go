package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngaut/log"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/model"
	"github.com/mayooot/gpu-docker-api/internal/service"
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
		log.Errorf("failed to create volume, volume name: %s must contain '-'", spec.Name)
		ResponseError(c, CodeContainerNameNotContainsSpecialChar)
		return
	}

	resp, err := vs.CreateVolume(&spec)
	if err != nil {
		log.Errorf("service.CreateVolume failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
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
	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to delete volume, name: %s must be in format: name-version", name)
		ResponseError(c, CodeVolumeNameMustContainVersion)
	}

	var spec model.VolumeDelete
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to delete volume, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	if err := vs.DeleteVolume(name, &spec); err != nil {
		log.Errorf("service.DeleteVolume failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
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
		log.Errorf("failed to patch volume size, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
		return
	}

	var spec model.VolumeSize
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to patch volume size, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	resp, err := vs.PatchVolumeSize(name, &spec)
	if err != nil {
		log.Errorf("service.PatchVolumeSize failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerPatchGpuInfoFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"name": resp.Name,
		"size": resp.Options["size"],
	})
}
