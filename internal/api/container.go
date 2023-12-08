package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngaut/log"

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
		log.Error("failed to create container, container name container '-'")
		ResponseError(c, CodeContainerImageNotNull)
		return
	}

	id, containerName, err := cs.RunGpuContainer(&spec)
	if err != nil {
		log.Error(err.Error())
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
		log.Error("failed to patch container gpu info, name must be in format: name-version")
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	err := cs.DeleteContainer(&name)
	if err != nil {
		log.Error(err.Error())
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
		log.Error("failed to patch container gpu info, name must be in format: name-version")
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	var spec model.ContainerExecute
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to execute container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	resp, err := cs.ExecuteContainer(&name, &spec)
	if err != nil {
		log.Error(err.Error())
		ResponseError(c, CodeContainerExecuteFailed)
		return
	}

	ResponseSuccess(c, resp)
}

// foo-0
func (ch *ContainerHandler) patchGpuInfo(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to patch container gpu info, name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Error("failed to patch container gpu info, name must be in format: name-version")
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
		log.Error(err.Error())
		ResponseError(c, CodeContainerPatchGpuInfoFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"id":   id,
		"name": containerName,
	})
}
