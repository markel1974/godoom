package lumps

import "github.com/markel1974/godoom/mr_tech/geometry"

// RawFace3 represents a 3D face with its vertices, UV mappings, texture name, and associated texture information.
type RawFace3 struct {
	Points  []geometry.XYZ
	UVs     [][2]float64
	TexName string
	Info    Texture3
}

// NewRawFace3 creates and returns a new RawFace3 with the given 3D points, UV coordinates, texture name, and texture info.
func NewRawFace3(pts []geometry.XYZ, uvs [][2]float64, tex string, t Texture3) *RawFace3 {
	return &RawFace3{Points: pts, UVs: uvs, TexName: tex, Info: t}
}
