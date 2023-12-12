package model

import (
	"encoding/json"
	"github.com/docker/docker/api/types/volume"
)

type EtcdVolumeInfo struct {
	Opt     *volume.CreateOptions
	Version int64 `json:"Version"`
}

func (v *EtcdVolumeInfo) Serialize() *string {
	bytes, _ := json.Marshal(v)
	tmp := string(bytes)
	return &tmp
}
