package lumps

import "github.com/markel1974/godoom/mr_tech/geometry"

// RawFace represents a geometric face with associated vertex points, texture coordinates, and surface properties.
type RawFace struct {
	Points  []geometry.XYZ
	UVs     [][2]float64
	TexName string
	IsSky   bool
}

// NewRawFace creates a new RawFace with the provided points, UV coordinates, texture name, and sky flag.
func NewRawFace(pts []geometry.XYZ, uvs [][2]float64, tex string, isSky bool) *RawFace {
	return &RawFace{Points: pts, UVs: uvs, TexName: tex, IsSky: isSky}
}
