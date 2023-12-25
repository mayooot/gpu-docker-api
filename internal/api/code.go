package api

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
	CodeContainerContainerNameNotNull
	CodeContainerRunFailed
	CodeContainerIDNotNull
	CodeContainerDeleteFailed
	CodeContainerExecuteFailed
	CodeContainerPatchGpuInfoFailed
	CodeContainerExisted
	CodeContainerPatchVolumeInfoFailed
	CodeContainerStopFailed
	CodeContainerRestartFailed
	CodeContainerGpuCountMustGreaterThanZero
	CodeContainerGpuNotEnough
	CodeContainerGpuNoNeedPatch

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

	CodeEtcdDeleteFailed
)

var codeMsgMap = map[ResCode]string{

	CodeSuccess:       "success",
	CodeServeBusy:     "服务器繁忙",
	CodeInvalidParams: "POST 请求传递参数格式错误",

	CodeContainerMustPassedIDOrName:          "必须传递 ID 或 name",
	CodeContainerNameNotNull:                 "容器名称为空",
	CodeContainerImageNotNull:                "镜像不能为空",
	CodeContainerContainerNameNotNull:        "容器名称不能为空",
	CodeContainerNameNotContainsDash:         "容器名称不能包含-",
	CodeContainerNameMustContainVersion:      "容器名称必须包含版本号",
	CodeContainerRunFailed:                   "容器启动失败",
	CodeContainerIDNotNull:                   "容器 ID 为空",
	CodeContainerDeleteFailed:                "容器删除失败",
	CodeContainerExecuteFailed:               "容器执行失败",
	CodeContainerPatchGpuInfoFailed:          "更新容器 GPU 配置失败",
	CodeContainerExisted:                     "容器已存在",
	CodeContainerPatchVolumeInfoFailed:       "更新容器挂载卷配置失败",
	CodeContainerStopFailed:                  "容器停止失败",
	CodeContainerRestartFailed:               "容器重启动失败",
	CodeContainerGpuCountMustGreaterThanZero: "容器 GPU 数量必须大于 0",
	CodeContainerGpuNotEnough:                "没有足够的 GPU 资源",
	CodeContainerGpuNoNeedPatch:              "容器 GPU 资源不需要更新，因为更新前后数量一样",

	CodeVolumeCreateFailed:                 "卷创建失败",
	CodeVolumeNameNotNull:                  "卷名不能为空",
	CodeVolumeDeleteFailed:                 "卷删除失败",
	CodeVolumeExisted:                      "卷已存在",
	CodeVolumeNameMustContainVersion:       "卷名必须包含版本号",
	CodeVolumeSizeNoNeedPatch:              "卷大小不需要更新，因为更新前后大小一样",
	CodeVolumeSizeNotSupported:             "卷大小单位不支持，支持的单位：KB, MB, GB, TB",
	CodeVolumeSizeUsedGreaterThanReduce:    "卷大小更新失败，因为更新后的大小小于已使用的大小",
	CodeVolumeNameNotContainsDash:          "卷名称不能包含-",
	CodeVolumeNameNotBeginWithForwardSlash: "卷名称不能以 / 开头",

	CodeEtcdDeleteFailed: "删除 etcd 中的资源失败",
}

func (c ResCode) Msg() string {
	msg, ok := codeMsgMap[c]
	if !ok {
		msg = codeMsgMap[CodeServeBusy]
	}
	return msg
}
