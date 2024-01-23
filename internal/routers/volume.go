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

type VolumeHandler struct{}

var vs services.VolumeService

func (vh *VolumeHandler) RegisterRoute(g *gin.RouterGroup) {
	g.POST("/volumes", vh.Create)
	g.PATCH("/volumes/:name/size", vh.Patch)
	g.DELETE("/volumes/:name", vh.Delete)
	g.GET("/volumes/:name", vh.Info)
	g.GET("/volumes/:name/history", vh.History)
}

// Create a volume, you can specify the size and name
func (vh *VolumeHandler) Create(c *gin.Context) {
	var spec models.VolumeCreate
	err := c.ShouldBindJSON(&spec)
	if err != nil {
		log.Error("failed to create volume, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}

	if strings.Contains(spec.Name, "-") {
		log.Errorf("failed to create volume, volume name: %s must contain '-'", spec.Name)
		ResponseError(c, CodeVolumeNameNotContainsDash)
		return
	}

	if strings.HasPrefix(spec.Name, "/") {
		log.Errorf("failed to create volume, volume name: %s not begin with '/'", spec.Name)
		ResponseError(c, CodeVolumeNameNotBeginWithForwardSlash)
		return
	}

	resp, err := vs.CreateVolume(&spec)
	if err != nil {
		log.Errorf("services.CreateVolume failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		if xerrors.IsVolumeExistedError(err) {
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

// Patch the size of the latest version of an existing volume via create a new volume and copy the old volume data to the new volume.
// Including expand and shrink of two operations, if the size is the same before and after the operation, it will be skipped.
// If the size already used is larger than the size after shrink, then shrink operation will fail.
func (vh *VolumeHandler) Patch(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to Patch volume size, name is empty")
		ResponseError(c, CodeVolumeNameCannotBeEmpty)
		return
	}

	var spec models.VolumeSize
	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Error("failed to Patch volume size, error:", err.Error())
		ResponseError(c, CodeInvalidParams)
		return
	}
	spec.Size = strings.ToUpper(spec.Size)
	unit := spec.Size[len(spec.Size)-2:]
	if _, ok := models.VolumeSizeMap[unit]; !ok {
		log.Errorf("failed to Patch volume size, size: %s is not supported", spec.Size)
		ResponseError(c, CodeVolumeSizeNotSupported)
		return
	}

	resp, err := vs.PatchVolumeSize(name, &spec)
	if err != nil {
		log.Errorf("services.PatchVolumeSize failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		if xerrors.IsNoPatchRequiredError(err) {
			ResponseError(c, CodeVolumeSizeNoNeedPatch)
			return
		}
		if xerrors.IsVolumeSizeUsedGreaterThanReduced(err) {
			ResponseError(c, CodeVolumePatchFailed)
			return
		}
		ResponseError(c, CodeVolumePatchFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"name": resp.Name,
		"size": resp.Options["size"],
	})
}

// Delete a volume
func (vh *VolumeHandler) Delete(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to Delete volume, name is empty")
		ResponseError(c, CodeVolumeNameCannotBeEmpty)
		return
	}

	if err := vs.DeleteVolume(name, true, true); err != nil {
		log.Errorf("services.DeleteVolume failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeVolumeDeleteFailed)
		return
	}

	ResponseSuccess(c, nil)
}

func (vh *VolumeHandler) Info(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to get volume Info, name is empty")
		ResponseError(c, CodeVolumeNameCannotBeEmpty)
		return
	}

	info, err := vs.GetVolumeInfo(name)
	if err != nil {
		log.Errorf("services.GetVolumeInfo failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeVolumeGetInfoFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"Info": info,
	})
}

func (vh *VolumeHandler) History(c *gin.Context) {
	name := c.Param("name")
	if len(name) == 0 {
		log.Error("failed to get volume Info, name is empty")
		ResponseError(c, CodeVolumeNameCannotBeEmpty)
		return
	}

	history, err := vs.GetVolumeHistory(name)
	if err != nil {
		log.Errorf("services.GetVolumeHistory failed, original error: %T %v", errors.Cause(err), err)
		log.Errorf("stack trace: \n%+v\n", err)
		ResponseError(c, CodeVolumeGetHistoryFailed)
		return
	}

	ResponseSuccess(c, gin.H{
		"history": history,
	})
}
