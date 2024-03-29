package models

import (
	"fmt"
)

var VolumeSizeMap = map[string]struct{}{
	"KB": {},
	"MB": {},
	"GB": {},
	"TB": {},
}

type Bind struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

func (b *Bind) Format() string {
	if len(b.Src) == 0 || len(b.Dest) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%s", b.Src, b.Dest)
}

type VolumeCreate struct {
	Name string `json:"name,omitempty"`
	Size string `json:"size,omitempty"`
}

type VolumeSize struct {
	Size string `json:"size"` // KB, MB, GB, TB
}

type VolumeHistoryItem struct {
	Version    int64          `json:"version"`
	CreateTime string         `json:"createTime"`
	Status     EtcdVolumeInfo `json:"status"`
}
