package lumps

import "github.com/markel1974/godoom/mr_tech/geometry"

// RawFace2 represents a 3D polygonal face with geometry points, UV mapping, texture name, and type classification.
// Points holds the vertices of the face as an array of 3D coordinates.
// UVs stores the UV mapping coordinates for texture placement.
// TexName specifies the name of the texture applied to the face.
// Kind denotes the type or category of the face as an integer.
type RawFace2 struct {
	Points  []geometry.XYZ
	UVs     [][2]float64
	TexName string
	Info    TexInfo2
}

// NewRawFace2 creates and returns a new RawFace2 instance with specified points, UV coordinates, texture name, and kind.
func NewRawFace2(pts []geometry.XYZ, uvs [][2]float64, tex string, info TexInfo2) *RawFace2 {
	return &RawFace2{Points: pts, UVs: uvs, TexName: tex, Info: info}
}
