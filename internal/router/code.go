package router

type ResCode int64

const (
	CodeSuccess       ResCode = 200
	CodeServeBusy     ResCode = 500
	CodeInvalidParams         = 1000 + iota

	CodeContainerImageNotNull

	CodeContainerMustPassedIDOrName
	CodeContainerNameNotNull
	CodeContainerNameNotContainsDash
	CodeContainerNameMustContainVersion
	CodeContainerRunFailed
	CodeContainerDeleteFailed
	CodeContainerExecuteFailed
	CodeContainerPatchGpuInfoFailed
	CodeContainerExisted
	CodeContainerNoNeedPatch
	CodeContainerPatchVolumeInfoFailed
	CodeContainerPatchVolumeInfoExistNull
	CodeContainerStopFailed
	CodeContainerRestartFailed
	CodeContainerGpuCountMustGreaterThanZero
	CodeContainerGpuNotEnough
	CodeContainerCommitFailed
	CodeContainerGetInfoFailed

	CodeVolumeCreateFailed
	CodeVolumeNameNotNull
	CodeVolumeDeleteFailed
	CodeVolumeExisted
	CodeVolumeNameMustContainVersion
	CodeVolumeSizeNoNeedPatch
	CodeVolumeSizeNotSupported
	CodeVolumeSizeUsedGreaterThanReduce
	CodeVolumeNameNotContainsDash
	CodeVolumeNameNotBeginWithForwardSlash
	CodeVolumeGetInfoFailed

	CodeVersionNotMatch
)

var codeMsgMap = map[ResCode]string{

	CodeSuccess:       "success",
	CodeServeBusy:     "服务器繁忙",
	CodeInvalidParams: "POST 请求传递参数格式错误",

	CodeContainerMustPassedIDOrName:          "必须传递 ID 或 name",
	CodeContainerNameNotNull:                 "容器名称为空",
	CodeContainerImageNotNull:                "镜像不能为空",
	CodeContainerNameNotContainsDash:         "容器名称不能包含-",
	CodeContainerNameMustContainVersion:      "容器名称必须包含版本号",
	CodeContainerRunFailed:                   "容器启动失败",
	CodeContainerDeleteFailed:                "容器删除失败",
	CodeContainerExecuteFailed:               "容器执行失败",
	CodeContainerPatchGpuInfoFailed:          "更新容器 GPU 配置失败",
	CodeContainerExisted:                     "容器已存在",
	CodeContainerNoNeedPatch:                 "容器不需要更新",
	CodeContainerPatchVolumeInfoFailed:       "更新容器挂载卷配置失败",
	CodeContainerPatchVolumeInfoExistNull:    "Patch Volume 字段存在空字符串",
	CodeContainerStopFailed:                  "容器停止失败",
	CodeContainerRestartFailed:               "容器重启动失败",
	CodeContainerGpuCountMustGreaterThanZero: "容器 GPU 数量必须大于 0",
	CodeContainerGpuNotEnough:                "没有足够的 GPU 资源",
	CodeContainerCommitFailed:                "容器提交为镜像失败",
	CodeContainerGetInfoFailed:               "获取容器信息失败",

	CodeVolumeCreateFailed:                 "Volume 创建失败",
	CodeVolumeNameNotNull:                  "Volume name 不能为空",
	CodeVolumeDeleteFailed:                 "Volume 删除失败",
	CodeVolumeExisted:                      "Volume 已存在",
	CodeVolumeNameMustContainVersion:       "Volume 名必须包含版本号",
	CodeVolumeSizeNoNeedPatch:              "Volume 大小不需要更新，因为更新前后大小一样",
	CodeVolumeSizeNotSupported:             "Volume 大小单位不支持，支持的单位：KB, MB, GB, TB",
	CodeVolumeSizeUsedGreaterThanReduce:    "Volume 大小更新失败，因为更新后的大小小于已使用的大小",
	CodeVolumeNameNotContainsDash:          "Volume 名称不能包含-",
	CodeVolumeNameNotBeginWithForwardSlash: "Volume 名称不能以 / 开头",
	CodeVolumeGetInfoFailed:                "获取 Volume 信息失败",

	CodeVersionNotMatch: "要更新的资源的版本号和存储在 ETCD 中最新的版本号不匹配",
}

func (c ResCode) Msg() string {
	msg, ok := codeMsgMap[c]
	if !ok {
		msg = codeMsgMap[CodeServeBusy]
	}
	return msg
}
