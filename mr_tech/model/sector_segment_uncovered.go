package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// SegmentUncovered represents a linear segment in 3D space with associated properties like material, tags, and alignment.
type SegmentUncovered struct {
	start    geometry.XY
	end      geometry.XY
	zBS      float64
	zBE      float64
	zTS      float64
	zTE      float64
	tag      string
	mat      *textures.Material
	csId     string
	csTag    string
	csLight  *config.Light
	modelId  int
	align    *config.UVAlign
	centroid geometry.XY
	lightPos geometry.XYZ
}

// NewSegmentUncovered creates a new SegmentUncovered object with given start/end points, z-values, material, and configuration.
// Parameters:
// - start: The starting point of the wall as a geometry.XY structure.
// - end: The ending point of the wall as a geometry.XY structure.
// - zBS: The bottom start Z-coordinate of the wall.
// - zBE: The bottom end Z-coordinate of the wall.
// - zTS: The top start Z-coordinate of the wall.
// - zTE: The top end Z-coordinate of the wall.
// - tag: A string identifier for the wall.
// - mat: A pointer to a textures.Material defining the wall's appearance.
// - csId: A string identifier for the corresponding construction system.
// - csTag: A string tag for the construction system.
// - csLight: A pointer to a config.Light representing the wall's lighting configuration.
// - modelId: An integer ID for the model associated with the wall.
// - align: A pointer to a config.UVAlign for texture alignment.
// Returns: A pointer to the newly created SegmentUncovered instance.
func NewSegmentUncovered(start, end geometry.XY, zBS float64, zBE float64, zTS float64, zTE float64, tag string, mat *textures.Material, csId string, csTag string, csLight *config.Light, modelId int, align *config.UVAlign) *SegmentUncovered {
	uw := &SegmentUncovered{
		start:   start,
		end:     end,
		zBS:     zBS,
		zBE:     zBE,
		zTS:     zTS,
		zTE:     zTE,
		tag:     tag,
		mat:     mat,
		csId:    csId,
		csTag:   csTag,
		csLight: csLight,
		modelId: modelId,
		align:   align,
	}
	uw.centroid = geometry.XY{X: (uw.start.X + uw.end.X) * 0.5, Y: (uw.start.Y + uw.end.Y) * 0.5}
	uw.lightPos = geometry.XYZ{X: uw.centroid.X, Y: uw.centroid.Y, Z: (uw.zBS + uw.zBE + uw.zTS + uw.zTE) * 0.25}
	return uw
}

// CreateQuad returns four geometry.XYZ points representing a quadrilateral in 3D space based on the wall's properties.
func (uw *SegmentUncovered) CreateQuad() (geometry.XYZ, geometry.XYZ, geometry.XYZ, geometry.XYZ) {
	v0 := geometry.XYZ{X: uw.start.X, Y: uw.start.Y, Z: uw.zBS}
	v1 := geometry.XYZ{X: uw.end.X, Y: uw.end.Y, Z: uw.zBE}
	v2 := geometry.XYZ{X: uw.end.X, Y: uw.end.Y, Z: uw.zTE}
	v3 := geometry.XYZ{X: uw.start.X, Y: uw.start.Y, Z: uw.zTS}
	return v0, v1, v2, v3
}

// CreateFaces generates two triangular faces representing a wall surface based on the SegmentUncovered's geometry and properties.
func (uw *SegmentUncovered) CreateFaces() []*Face {
	v0, v1, v2, v3 := uw.CreateQuad()
	face1 := NewFace([3]geometry.XYZ{v0, v1, v2}, uw.tag, uw.mat)
	face2 := NewFace([3]geometry.XYZ{v0, v2, v3}, uw.tag, uw.mat)
	face1.ApplyWallUV(uw.start, uw.end, uw.align, 0.0, 1.0, uw.zBS, uw.zBE, uw.zTS, uw.zTE, true)
	face2.ApplyWallUV(uw.start, uw.end, uw.align, 0.0, 1.0, uw.zBS, uw.zBE, uw.zTS, uw.zTE, false)
	return []*Face{face1, face2}
}
