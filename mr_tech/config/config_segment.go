package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

const (
	// SegmentUnknown represents an unspecified or undefined configuration type in the context of segment definition.
	SegmentUnknown = iota

	// SegmentWall represents a configuration segment identified as a wall within the level's geometry or structure.
	SegmentWall
)

// Segment represents a segment of input data with spatial coordinates, type, and associated metadata.
type Segment struct {
	Parent string      `json:"parent"`
	Id     string      `json:"id"`
	Start  geometry.XY `json:"start"`
	End    geometry.XY `json:"end"`
	Kind   int         `json:"Kind"`
	Tag    string      `json:"tag"`
	Upper  *Material   `json:"upper"`
	Middle *Material   `json:"middle"`
	Lower  *Material   `json:"lower"`
}

// NewConfigSegment creates a new Segment instance with the specified parent, Kind, start, and end coordinates.
func NewConfigSegment(parent string, kind int, s geometry.XY, e geometry.XY) *Segment {
	is := &Segment{
		Parent: parent,
		Id:     utils.NextUUId(),
		Start:  s,
		End:    e,
		Kind:   kind,
		Tag:    "",
		Upper:  nil,
		Lower:  nil,
		Middle: nil,
	}
	return is
}
