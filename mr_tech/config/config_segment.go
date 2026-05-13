package config

import (
	"fmt"
	"math"

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
	Parent           string      `json:"parent"`
	Id               string      `json:"id"`
	Start            geometry.XY `json:"start"`
	End              geometry.XY `json:"end"`
	Kind             int         `json:"Kind"`
	Tag              string      `json:"tag"`
	Upper            *Material   `json:"upper"`
	Middle           *Material   `json:"middle"`
	Lower            *Material   `json:"lower"`
	SlopedCeilingRef bool        `json:"slopedCeilingRef"`
	SlopedFloorRef   bool        `json:"slopedFloorRef"`
}

// NewConfigSegment creates a new Segment instance with the specified parent, Kind, start, and end coordinates.
func NewConfigSegment(parent string, kind int, s geometry.XY, e geometry.XY) *Segment {
	is := &Segment{
		Parent:           parent,
		Id:               utils.NextUUId(),
		Start:            s,
		End:              e,
		Kind:             kind,
		Tag:              "",
		Upper:            nil,
		Lower:            nil,
		Middle:           nil,
		SlopedCeilingRef: false,
		SlopedFloorRef:   false,
	}
	return is
}

// ComputeNormal calculates the normalized normal vector of the segment based on its orientation.
// isCCW determines whether the normal vector should be counterclockwise or clockwise.
// Returns the x and y components of the normal vector and an error if the segment length is zero.
func (cs *Segment) ComputeNormal(isCCW bool) (float64, float64, error) {
	dx := cs.End.X - cs.Start.X
	dy := cs.End.Y - cs.Start.Y
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return 0, 0, fmt.Errorf("segment has zero length")
	}
	var nX, nY float64
	if isCCW {
		nX, nY = -dy/length, dx/length
	} else {
		nX, nY = dy/length, -dx/length
	}
	return nX, nY, nil
}
