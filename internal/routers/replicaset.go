package routers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ngaut/log"
	"github.com/pkg/errors"

	"github.com/mayooot/gpu-docker-api/internal/models"
	"github.com/mayooot/gpu-docker-api/internal/services"
	"github.com/mayooot/gpu-docker-api/internal/xerrors"
)

// ReplicaSet is just an abstract concept, there is no concrete implementations,
// it just has to manage docker container, and save the container historical version information, that's all.

type ReplicaSetHandler struct{}

var cs services.ReplicaSetService

func (rh *ReplicaSetHandler) RegisterRoute(g *gin.RouterGroup) {
	// run a container via replicaSet
	g.POST("/replicaSet", rh.Run)
	// commit replicaSet the current version of the container as an image
	g.POST("/replicaSet/:name/commit", rh.Commit)
	// execute a command in the replicaSet current version of the container
	g.POST("/replicaSet/:name/execute", rh.Execute)

	// update the replicaSet, such as change gpu, volume
	// or replicating the container by create a new container.
	g.PATCH("/replicaSet/:name", rh.Patch)
	// rollback replicaSet the current version of the container toa specific version
	g.PATCH("/replicaSet/:name/rollback", rh.Rollback)

	// stop the current version of the replicaSet container,
	// gpu and port will be released
	g.PATCH("/replicaSet/:name/stop", rh.Stop)
	// restart the current version of the replicaSet container by recreate a container,
	// it will reapply gpu and port, and new container will be created.
	g.PATCH("/replicaSet/:name/restart", rh.Restart)

	// pause the current version of the replicaSet container,
	// gpu and port will not be release
	g.PATCH("/replicaSet/:name/pause", rh.Pause)
	// continue to run the current version of the replicaSet container,
	// it will call `docker restart`.
	g.PATCH("/replicaSet/:name/continue", rh.Continue)

	// get information about the current version of the replicaSet
	g.GET("/replicaSet/:name", rh.Info)
	// get information about all historical versions of the replicaSet
	g.GET("/replicaSet/:name/history", rh.History)

	// delete a replicaSet also delete the container and cannot be recovered.
	g.DELETE("/replicaSet/:name", rh.Delete)
}

func (rh *ReplicaSetHandler) Info(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to get container Info, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	info, err := cs.GetContainerInfo(name)
	if err != nil {
		log.Errorf("services.GetContainerInfo failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerGetInfoFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"Info": info,
	})
}

func (rh *ReplicaSetHandler) History(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to get container history, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	history, err := cs.GetContainerHistory(name)
	if err != nil {
		log.Errorf("services.GetContainerHistory failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerGetHistoryFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"history": history,
	})
}

// Run a container consists of two parts: create and start
func (rh *ReplicaSetHandler) Run(c *gin.Context) {
	var spec models.ContainerRun
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to create container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	if len(spec.ImageName) == 0 {
		log.Error("failed to create container, image name is empty")
		ResponseError(c, CodeImageNameCannotBeEmpty)
		return
	}

	if len(spec.ReplicaSetName) == 0 {
		log.Error("failed to create container, container name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	if spec.GpuCount < 0 {
		log.Error("failed to create container, gpu count must be greater than 0")
		ResponseError(c, CodeGpuCountMustBeGreaterThanOrEqualZero)
		return
	}

	if strings.Contains(spec.ReplicaSetName, "-") {
		log.Error("failed to create container, container name cannot contain dash")
		ResponseError(c, CodeContainerNameCannotContainDash)
		return
	}

	_, containerName, err := cs.RunGpuContainer(&spec)
	if err != nil {
		log.Errorf("services.RunGpuContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		if xerrors.IsContainerExistedError(err) {
			ResponseError(c, CodeContainerAlreadyExist)
			return
		}
		if xerrors.IsGpuNotEnoughError(err) {
			ResponseError(c, CodeContainerGpuNotEnough)
			return
		}
		if xerrors.IsPortNotEnoughError(err) {
			ResponseError(c, CodeContainerPortNotEnough)
			return
		}
		ResponseError(c, CodeContainerRunFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"name": containerName,
	})
}

// Commit the latest version of the container as image.
// The image name is the default image id, or you can specify a new image name.
func (rh *ReplicaSetHandler) Commit(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to commit container, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	var spec models.ContainerCommit
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to commit container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	imageName, err := cs.CommitContainer(name, spec)
	if err != nil {
		log.Errorf("services.RestartContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerCommitFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"imageName": imageName,
	})
}

// Execute a command in the latest version of the running container and return the output
func (rh *ReplicaSetHandler) Execute(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to execute container, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	var spec models.ContainerExecute
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to execute container, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	resp, err := cs.ExecuteContainer(name, &spec)
	if err != nil {
		log.Errorf("services.ExecuteContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerExecuteFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"stdout": resp,
	})
}

// Patch to change the configuration of the latest version of an existing container.
// You can change the gpu, volume.
// If you request body is empty(e.g. {}), it will recreate a container based on the existing configuration.
// Then the old container will be deleted.
func (rh *ReplicaSetHandler) Patch(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to patch container, container name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	var spec models.PatchRequest
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Errorf("failed to patch container, error: %v", err)
		ResponseError(c, CodeInvalidParams)
		return
	}

	if spec.GpuPatch != nil && spec.GpuPatch.GpuCount < 0 {
		log.Errorf("failed to patch container, gpucount: %d must be greater than or equal to 0", spec.GpuPatch.GpuCount)
		ResponseError(c, CodeGpuCountMustBeGreaterThanOrEqualZero)
		return
	}

	if spec.VolumePatch != nil && (spec.VolumePatch.OldBind.Format() == "" ||
		spec.VolumePatch.NewBind.Format() == "") {
		log.Errorf("failed to patch container,volume Patch Info is invalid: %v", spec.VolumePatch)
		ResponseError(c, CodeInvalidParams)
		return
	}

	_, containerName, err := cs.PatchContainer(name, &spec)
	if err != nil {
		log.Errorf("services.PatchContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerPatchFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"containerName": containerName,
	})
}

// Rollback a container to a specific version
func (rh *ReplicaSetHandler) Rollback(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to rollback container, container name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	var spec models.RollbackRequest
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Errorf("failed to rollback container, error: %v", err)
		ResponseError(c, CodeInvalidParams)
		return
	}

	if spec.Version < 0 {
		log.Errorf("failed to rollback container, version: %d must be greater than or equal to 0", spec.Version)
		ResponseError(c, CodeContainerVersionMustBeGreaterThanOrEqualZero)
		return
	}

	containerName, err := cs.RollbackContainer(name, &spec)

	if err != nil {
		log.Errorf("services.RollbackContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		if xerrors.IsNoRollbackRequiredError(err) {
			ResponseError(c, CodeContainerNoNeedRollback)
			return
		}
		ResponseError(c, CodeContainerRollbackFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"containerName": containerName,
	})
}

// Pause the latest version of the container,
// gpu and port will not be released
func (rh *ReplicaSetHandler) Pause(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to shut down container, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	if err := cs.StopContainer(name, false, false, true); err != nil {
		log.Errorf("services.StopContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerShutDownFailed)
		return
	}

	ResponseSuccess(c, nil)
}

// Continue the  latest version of the container
// just call `docker restart`
func (rh *ReplicaSetHandler) Continue(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to startup container, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	if err := cs.StartupContainer(name); err != nil {
		log.Errorf("services.StartupContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerRestartFailed)
		return
	}

	ResponseSuccess(c, nil)
}

// Stop the latest version of the container
// gpu and port will be released
func (rh *ReplicaSetHandler) Stop(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to stop container, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	if err := cs.StopContainer(name, true, true, true); err != nil {
		log.Errorf("services.StopContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerStopFailed)
		return
	}

	ResponseSuccess(c, nil)
}

// Restart the latest version of the container.
// It may fail because restart require apply for gpu
func (rh *ReplicaSetHandler) Restart(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to restart container, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	_, containerName, err := cs.RestartContainer(name)
	if err != nil {
		log.Errorf("services.RestartContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerRestartFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"containerName": containerName,
	})
}

// Delete containers, including historical versions
func (rh *ReplicaSetHandler) Delete(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to delete container, name is empty")
		ResponseError(c, CodeContainerNameCannotBeEmpty)
		return
	}

	if err := cs.DeleteContainer(name); err != nil {
		log.Errorf("services.DeleteContainer failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeContainerDeleteFailed)
		return
	}

	ResponseSuccess(c, nil)
}
