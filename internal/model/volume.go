package model

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
