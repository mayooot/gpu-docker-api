package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngaut/log"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/model"
	"github.com/mayooot/gpu-docker-api/internal/service"
)

var cs service.ContainerService

type ContainerHandler struct{}

func (ch *ContainerHandler) RegisterRoute(g *gin.RouterGroup) {
	g.POST("/containers", ch.run)
	g.DELETE("/containers/:name", ch.delete)
	g.POST("/containers/:name/execute", ch.execute)
	g.PATCH("/containers/:name/gpu", ch.patchGpuInfo)
	g.PATCH("/containers/:name/volume", ch.pathVolumeInfo)
	g.PATCH("/containers/:name/stop", ch.stop)
	g.PATCH("/containers/:name/restart", ch.restart)
}

func (ch *ContainerHandler) run(c *gin.Context) {
	var spec model.ContainerRun
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to create container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	if len(spec.ImageName) == 0 {
		log.Error("failed to create container, image name is empty")
		ResponseError(c, CodeContainerImageNotNull)
		return
	}

	if len(spec.ContainerName) == 0 {
		log.Error("failed to create container, container name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if strings.Contains(spec.ContainerName, "-") {
		log.Error("failed to create container, container name: %s must container '-'", spec.ContainerName)
		ResponseError(c, CodeContainerNameNotContainsSpecialChar)
		return
	}

	id, containerName, err := cs.RunGpuContainer(&spec)
	if err != nil {
		log.Errorf("service.RunGpuContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		if errors.Is(err, service.ErrorContainerExisted) {
			ResponseError(c, CodeContainerExisted)
			return
		}
		ResponseError(c, CodeContainerRunFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"id":   id,
		"name": containerName,
	})
}

func (ch *ContainerHandler) delete(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to delete container, name is empty")
		ResponseError(c, CodeContainerMustPassedIDOrName)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to delete container, name must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	var spec model.ContainerDelete
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to delete container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	if err := cs.DeleteContainer(name, &spec); err != nil {
		log.Errorf("service.DeleteContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerDeleteFailed)
		return
	}

	ResponseSuccess(c, nil)
}

func (ch *ContainerHandler) execute(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to execute container, name is empty")
		ResponseError(c, CodeContainerMustPassedIDOrName)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to execute container, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	var spec model.ContainerExecute
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to execute container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	resp, err := cs.ExecuteContainer(name, &spec)
	if err != nil {
		log.Errorf("service.ExecuteContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerExecuteFailed)
		return
	}

	ResponseSuccess(c, resp)
}

func (ch *ContainerHandler) patchGpuInfo(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to patch container gpu info, name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to patch container gpu info, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	var spec model.ContainerGpuPatch
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to patch container gpu info, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	id, containerName, err := cs.PatchContainerGpuInfo(name, &spec)
	if err != nil {
		log.Errorf("service.PatchContainerGpuInfo failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerPatchGpuInfoFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"id":   id,
		"name": containerName,
	})
}

func (ch *ContainerHandler) pathVolumeInfo(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to patch container volume info, name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to patch container volume info, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	var spec model.ContainerVolumePatch
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to patch container volume info, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	id, containerName, err := cs.PatchContainerVolumeInfo(name, &spec)
	if err != nil {
		log.Errorf("service.PatchContainerVolumeInfo failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerPatchGpuInfoFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"id":   id,
		"name": containerName,
	})
}

func (ch *ContainerHandler) stop(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to stop container, name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to stop container, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	if err := cs.StopContainer(name); err != nil {
		log.Errorf("service.StopContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerStopFailed)
		return
	}

	ResponseSuccess(c, nil)
}

func (ch *ContainerHandler) restart(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to restart container, name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to restart container, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	if err := cs.RestartContainer(name); err != nil {
		log.Errorf("service.RestartContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerRestartFailed)
		return
	}

	ResponseSuccess(c, nil)
}
