package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// ConfigFace represents a 3D face configuration with geometry, material, type, and neighbor information.
type ConfigFace struct {
	Id       string           `json:"id"`
	Points   []geometry.XYZ   `json:"points"`
	Material *ConfigAnimation `json:"material"`
	Kind     int              `json:"kind"`
	Neighbor string           `json:"neighbor"`
	Tag      string           `json:"tag"`
}

// NewConfigFace creates and returns a pointer to a ConfigFace instance with specified points, kind, neighbor, material, and tag.
func NewConfigFace(points []geometry.XYZ, kind int, neighbor string, material *ConfigAnimation, tag string) *ConfigFace {
	return &ConfigFace{
		Id:       utils.NextUUId(),
		Points:   points,
		Kind:     kind,
		Neighbor: neighbor,
		Material: material,
		Tag:      tag,
	}
}
