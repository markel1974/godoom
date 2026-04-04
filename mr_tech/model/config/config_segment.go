package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// SegmentDataWall represents the wall segment data identifier.
// SegmentDataTexture represents the texture segment data identifier.
// SegmentDataNeighbor represents the neighboring segment data identifier.
const (
	SegmentDataWall     = 0
	SegmentDataTexture  = iota
	SegmentDataNeighbor = iota
)

// segmentData represents detailed information about a line segment in the system, including its coordinates and properties.
type segmentData struct {
	id            string
	point         geometry.XY
	kind          int
	neighbor      string
	textureUpper  *ConfigAnimation
	textureMiddle *ConfigAnimation
	textureLower  *ConfigAnimation
	distance      float64
	high          bool
}

// ConfigSegment represents a segment of input data with spatial coordinates, type, and associated metadata.
type ConfigSegment struct {
	Parent   string           `json:"parent"`
	Id       string           `json:"id"`
	Start    geometry.XY      `json:"start"`
	End      geometry.XY      `json:"end"`
	Kind     int              `json:"Kind"`
	Neighbor string           `json:"neighbor"`
	Tag      string           `json:"tag"`
	Upper    *ConfigAnimation `json:"upper"`
	Middle   *ConfigAnimation `json:"middle"`
	Lower    *ConfigAnimation `json:"lower"`
}

// NewConfigSegment creates a new ConfigSegment instance with the specified parent, Kind, start, and end coordinates.
func NewConfigSegment(parent string, kind int, s geometry.XY, e geometry.XY, neighbor string) *ConfigSegment {
	is := &ConfigSegment{
		Parent:   parent,
		Id:       utils.NextUUId(),
		Start:    s,
		End:      e,
		Kind:     kind,
		Neighbor: neighbor,
		Tag:      "",
		Upper:    nil,
		Lower:    nil,
		Middle:   nil,
	}
	return is
}
