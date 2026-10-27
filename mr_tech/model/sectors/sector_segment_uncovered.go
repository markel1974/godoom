package sectors

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// SegmentUncovered represents a linear segment in 3D space with associated material, alignment, and light properties.
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

// NewSegmentUncovered creates and initializes a new SegmentUncovered object with the given geometry, material, and properties.
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

// CreateQuad returns four geometry.XYZ points representing the corners of a quadrilateral defined by the segment's parameters.
func (uw *SegmentUncovered) CreateQuad() (geometry.XYZ, geometry.XYZ, geometry.XYZ, geometry.XYZ) {
	v0 := geometry.XYZ{X: uw.start.X, Y: uw.start.Y, Z: uw.zBS}
	v1 := geometry.XYZ{X: uw.end.X, Y: uw.end.Y, Z: uw.zBE}
	v2 := geometry.XYZ{X: uw.end.X, Y: uw.end.Y, Z: uw.zTE}
	v3 := geometry.XYZ{X: uw.start.X, Y: uw.start.Y, Z: uw.zTS}
	return v0, v1, v2, v3
}

// GetStart retrieves the starting 2D point of the segment as a geometry.XY instance.
func (uw *SegmentUncovered) GetStart() geometry.XY {
	return uw.start
}

// GetEnd returns the end point of the segment as a geometry.XY instance.
func (uw *SegmentUncovered) GetEnd() geometry.XY {
	return uw.end
}

// GetZBS returns the starting Z-coordinate of the bottom surface (zBS) for the segment.
func (uw *SegmentUncovered) GetZBS() float64 {
	return uw.zBS
}

// GetZBE returns the ending Z-coordinate value (zBE) of the segment.
func (uw *SegmentUncovered) GetZBE() float64 {
	return uw.zBE
}

// GetZTS returns the Z-coordinate of the top start position (zTS) for the SegmentUncovered instance.
func (uw *SegmentUncovered) GetZTS() float64 {
	return uw.zTS
}

// GetZTE retrieves the Z-coordinate of the top endpoint for the segment.
func (uw *SegmentUncovered) GetZTE() float64 {
	return uw.zTE
}

// GetTag returns the tag associated with the SegmentUncovered instance.
func (uw *SegmentUncovered) GetTag() string {
	return uw.tag
}

// GetMat retrieves the pointer to the Material associated with the SegmentUncovered instance.
func (uw *SegmentUncovered) GetMat() *textures.Material {
	return uw.mat
}

// GetCsId returns the coordinate system ID associated with the SegmentUncovered instance.
func (uw *SegmentUncovered) GetCsId() string {
	return uw.csId
}

// GetCsTag retrieves the csTag value associated with the SegmentUncovered instance.
func (uw *SegmentUncovered) GetCsTag() string {
	return uw.csTag
}

// GetCsLight returns the associated light source configuration of the SegmentUncovered instance.
func (uw *SegmentUncovered) GetCsLight() *config.Light {
	return uw.csLight
}

// GetModelId returns the model identifier associated with the SegmentUncovered instance.
func (uw *SegmentUncovered) GetModelId() int {
	return uw.modelId
}

// GetAlign retrieves the UV alignment and transformation parameters associated with the segment.
func (uw *SegmentUncovered) GetAlign() *config.UVAlign {
	return uw.align
}

// GetCentroid returns the centroid of the segment as a geometry.XY instance.
func (uw *SegmentUncovered) GetCentroid() geometry.XY {
	return uw.centroid
}

// GetLightPos returns the XYZ coordinates of the light position associated with the SegmentUncovered instance.
func (uw *SegmentUncovered) GetLightPos() geometry.XYZ {
	return uw.lightPos
}
