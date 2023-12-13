package model

type Bind struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

type VolumeCreate struct {
	Name string `json:"name,omitempty"`
	Size string `json:"size,omitempty"`
}

type VolumeSize struct {
	Size string `json:"size"`
}

type VolumeDelete struct {
	Force       bool `json:"force"`
	DelEtcdInfo bool `json:"delEtcdInfo"`
}
