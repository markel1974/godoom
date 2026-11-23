package config

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Face represents a 3D face configuration with geometry, material, type, and neighbor information.
type Face struct {
	Id                string         `json:"id"`
	Points            []geometry.XYZ `json:"points"`
	UVs               [][2]float64   `json:"uvs"`
	Normals           []geometry.XYZ `json:"normals,omitempty"`
	ExpectedNormal    geometry.XYZ   `json:"expectedNormal,omitempty"`
	HasExpectedNormal bool           `json:"hasExpectedNormal,omitempty"`
	LmUVs             [][2]float64   `json:"lmUvs,omitempty"`
	LightmapID        int32          `json:"lightmapId"`
	Material          *Material      `json:"material"`
	Tag               string         `json:"tag"`
}

// NewConfigFace creates and returns a pointer to a Face instance with specified points, kind, neighbor, material, and tag.
func NewConfigFace(points []geometry.XYZ, uvs [][2]float64, material *Material, tag string) *Face {
	return &Face{
		Id:         utils.NextUUId(),
		Points:     points,
		UVs:        uvs,
		LightmapID: -1,
		Material:   material,
		Tag:        tag,
	}
}
