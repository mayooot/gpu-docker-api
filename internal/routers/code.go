package routers

type ResCode int64

const (
	CodeSuccess                                      ResCode = 200
	CodeServeBusy                                    ResCode = 500
	CodeInvalidParams                                ResCode = 1000
	CodeImageNameCannotBeEmpty                       ResCode = 1001
	CodeContainerNameCannotBeEmpty                   ResCode = 1002
	CodeContainerNameCannotContainDash               ResCode = 1003
	CodeContainerRunFailed                           ResCode = 1004
	CodeContainerDeleteFailed                        ResCode = 1005
	CodeContainerExecuteFailed                       ResCode = 1006
	CodeContainerPatchFailed                         ResCode = 1007
	CodeContainerAlreadyExist                        ResCode = 1008
	CodeContainerNoNeedPatch                         ResCode = 1009
	CodeContainerStopFailed                          ResCode = 1010
	CodeContainerRestartFailed                       ResCode = 1011
	CodeGpuCountMustBeGreaterThanOrEqualZero         ResCode = 1012
	CodeContainerGpuNotEnough                        ResCode = 1013
	CodeContainerPortNotEnough                       ResCode = 1014
	CodeContainerCommitFailed                        ResCode = 1015
	CodeContainerGetInfoFailed                       ResCode = 1016
	CodeContainerGetHistoryFailed                    ResCode = 1017
	CodeContainerShutDownFailed                      ResCode = 1018
	CodeContainerStartUpFailed                       ResCode = 1019
	CodeContainerVersionMustBeGreaterThanOrEqualZero ResCode = 1020
	CodeContainerRollbackFailed                      ResCode = 1021
	CodeContainerNoNeedRollback                      ResCode = 1022
	CodeVolumeCreateFailed                           ResCode = 1023
	CodeVolumeNameCannotBeEmpty                      ResCode = 1024
	CodeVolumeDeleteFailed                           ResCode = 1025
	CodeVolumeExisted                                ResCode = 1026
	CodeVolumeNameMustContainVersion                 ResCode = 1027
	CodeVolumeSizeNoNeedPatch                        ResCode = 1028
	CodeVolumeSizeNotSupported                       ResCode = 1029
	CodeVolumeSizeUsedGreaterThanReduce              ResCode = 1030
	CodeVolumeNameNotContainsDash                    ResCode = 1031
	CodeVolumeNameNotBeginWithForwardSlash           ResCode = 1032
	CodeVolumeGetInfoFailed                          ResCode = 1033
	CodeVolumeGetHistoryFailed                       ResCode = 1034
	CodeVolumePatchFailed                            ResCode = 1035
)

var codeMsgMap = map[ResCode]string{
	CodeSuccess:                                      "Success",
	CodeServeBusy:                                    "Server busy",
	CodeInvalidParams:                                "Failed to parse body",
	CodeImageNameCannotBeEmpty:                       "Image name cannot be empty",
	CodeContainerNameCannotBeEmpty:                   "Container name cannot be empty",
	CodeContainerNameCannotContainDash:               "Container name cannot contain dash",
	CodeContainerRunFailed:                           "Failed to start container",
	CodeContainerDeleteFailed:                        "Failed to delete container",
	CodeContainerExecuteFailed:                       "Failed to execute a command",
	CodeContainerPatchFailed:                         "Failed to patch container",
	CodeContainerAlreadyExist:                        "Container already exists",
	CodeContainerNoNeedPatch:                         "Container doesn't need patch",
	CodeContainerStopFailed:                          "Failed to stop container",
	CodeContainerRestartFailed:                       "Failed to restart container",
	CodeGpuCountMustBeGreaterThanOrEqualZero:         "GPU count must be greater than or equal to 0",
	CodeContainerGpuNotEnough:                        "Not enough GPU resources",
	CodeContainerPortNotEnough:                       "Not enough port resources",
	CodeContainerCommitFailed:                        "Failed to commit image",
	CodeContainerGetInfoFailed:                       "Failed to get container info, container not found",
	CodeContainerGetHistoryFailed:                    "Failed to get container history, container not found",
	CodeContainerShutDownFailed:                      "Failed to shut down container",
	CodeContainerStartUpFailed:                       "Failed to start up container",
	CodeContainerVersionMustBeGreaterThanOrEqualZero: "Container version must be greater than or equal to 0",
	CodeContainerRollbackFailed:                      "Failed to rollback container",
	CodeContainerNoNeedRollback:                      "Container doesn't need rollback, the current version is the same as the requested version",
	CodeVolumeCreateFailed:                           "Failed to create volume",
	CodeVolumeNameCannotBeEmpty:                      "Volume name cannot be empty",
	CodeVolumeDeleteFailed:                           "Failed to delete volume",
	CodeVolumeExisted:                                "Volume already exists",
	CodeVolumeNameMustContainVersion:                 "Volume name must contain the version number",
	CodeVolumeSizeNoNeedPatch:                        "Volume doesn't need patch, as it is the same size before and after the update",
	CodeVolumeSizeNotSupported:                       "Volume size units are not supported, supported units: KB, MB, GB, TB",
	CodeVolumeSizeUsedGreaterThanReduce:              "Failed to patch volume size, the patch size is smaller than the used size",
	CodeVolumeNameNotContainsDash:                    "Volume name cannot contain dash",
	CodeVolumeNameNotBeginWithForwardSlash:           "Volume name must not begin with /",
	CodeVolumeGetInfoFailed:                          "Failed to get volume info",
	CodeVolumeGetHistoryFailed:                       "Failed to get volume history",
	CodeVolumePatchFailed:                            "Failed to patch volume",
}

func (c ResCode) Msg() string {
	msg, ok := codeMsgMap[c]
	if !ok {
		msg = codeMsgMap[CodeServeBusy]
	}
	return msg
}
