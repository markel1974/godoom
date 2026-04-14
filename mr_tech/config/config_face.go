package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Face represents a 3D face configuration with geometry, material, type, and neighbor information.
type Face struct {
	Id       string         `json:"id"`
	Points   []geometry.XYZ `json:"points"`
	Material *Animation     `json:"material"`
	Tag      string         `json:"tag"`
}

// NewConfigFace creates and returns a pointer to a Face instance with specified points, kind, neighbor, material, and tag.
func NewConfigFace(points []geometry.XYZ, material *Animation, tag string) *Face {
	return &Face{
		Id:       utils.NextUUId(),
		Points:   points,
		Material: material,
		Tag:      tag,
	}
}
