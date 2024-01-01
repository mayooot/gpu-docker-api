package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngaut/log"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/model"
	"github.com/mayooot/gpu-docker-api/internal/service"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

type ContainerHandler struct{}

var cs service.ContainerService

func (ch *ContainerHandler) RegisterRoute(g *gin.RouterGroup) {
	// 创建容器
	g.POST("/containers", ch.run)
	// 删除容器
	g.DELETE("/containers/:name", ch.delete)
	// 执行容器
	g.POST("/containers/:name/execute", ch.execute)
	// 变更已存在容器的 GPU 资源
	g.PATCH("/containers/:name/gpu", ch.patchGpuInfo)
	// 变更已存在容器的 Volume 资源
	g.PATCH("/containers/:name/volume", ch.pathVolumeInfo)
	// 停止容器
	g.PATCH("/containers/:name/stop", ch.stop)
	// 重启容器
	g.PATCH("/containers/:name/restart", ch.restart)
	// 提交容器为镜像
	g.POST("/containers/:name/commit", ch.commit)
	// 查看容器创建信息
	g.GET("/containers/:name/info", ch.info)
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

	if spec.GpuCount < 0 {
		log.Error("failed to create container, gpu count must be greater than 0")
		ResponseError(c, CodeContainerGpuCountMustGreaterThanZero)
		return
	}

	if strings.Contains(spec.ContainerName, "-") {
		log.Error("failed to create container, container name: %s must container '-'", spec.ContainerName)
		ResponseError(c, CodeContainerNameNotContainsDash)
		return
	}

	id, containerName, err := cs.RunGpuContainer(&spec)
	if err != nil {
		log.Errorf("service.RunGpuContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		if xerrors.IsContainerExistedError(err) {
			ResponseError(c, CodeContainerExisted)
			return
		}
		if xerrors.IsGpuNotEnoughError(err) {
			ResponseError(c, CodeContainerGpuNotEnough)
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
		log.Errorf("failed to delete container, name %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
		return
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

	ResponseSuccess(c, gin.H{
		"stdout": resp,
	})
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
		if xerrors.IsVersionNotMatchError(err) {
			ResponseError(c, CodeVersionNotMatch)
			return
		}
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
		if xerrors.IsNoPatchRequiredError(err) {
			ResponseError(c, CodeContainerVolumeNoNeedPatch)
			return
		}
		if xerrors.IsVersionNotMatchError(err) {
			ResponseError(c, CodeVersionNotMatch)
			return
		}
		ResponseError(c, CodeContainerPatchVolumeInfoFailed)
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

	var spec model.ContainerStop
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to stop container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	if err := cs.StopContainer(name, &spec); err != nil {
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

	id, containerName, err := cs.RestartContainer(name)
	if err != nil {
		log.Errorf("service.RestartContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerRestartFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"id":   id,
		"name": containerName,
	})
}

func (ch *ContainerHandler) commit(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to commit container, name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to commit container, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	var spec model.ContainerCommit
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to commit container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	imageName, err := cs.CommitContainer(name, spec)
	if err != nil {
		log.Errorf("service.RestartContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerCommitFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"imageName": imageName,
		"container": name,
	})
}

func (ch *ContainerHandler) info(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to get container info, name is empty")
		ResponseError(c, CodeContainerNameNotNull)
		return
	}

	if !strings.Contains(name, "-") || len(strings.Split(name, "-")[1]) == 0 {
		log.Errorf("failed to get container info, name: %s must be in format: name-version", name)
		ResponseError(c, CodeContainerNameMustContainVersion)
	}

	info, err := cs.GetContainerInfo(name)
	if err != nil {
		log.Errorf("service.GetContainerInfo failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerCommitFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"info": info,
	})
}
