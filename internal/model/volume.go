package model

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

func (b *Bind) Equal(other *Bind) bool {
	return b.Src == other.Src && b.Dest == other.Dest
}

func (b *Bind) Format() string {
	return fmt.Sprintf("%s:%s", b.Src, b.Dest)
}

type VolumeCreate struct {
	Name string `json:"name,omitempty"`
	Size string `json:"size,omitempty"`
}

type VolumeSize struct {
	Size string `json:"size"` // KB, MB, GB, TB
}

type VolumeDelete struct {
	Force                       bool `json:"force"`
	DelEtcdInfoAndVersionRecord bool `json:"delEtcdInfoAndVersionRecord"`
}
