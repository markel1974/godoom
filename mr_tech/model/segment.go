package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// edgePrecision defines the multiplier used to convert floating-point coordinates to integer-space for edge-key generation.
const edgePrecision = 1000.0

// EdgeKey uniquely represents a line segment in 2D space using quantized integer coordinates.
type EdgeKey struct {
	x1, y1, x2, y2 int64
}

// EdgeSegment represents an internal 2D segment structure with start and end points, sector reference, and index.
type EdgeSegment struct {
	start  geometry.XY
	end    geometry.XY
	sector *Sector
	np     int
}

// makeEdgeKey creates a unique EdgeKey by scaling and rounding the coordinates of the start and end points using precision.
func makeEdgeKey(precision float64, start geometry.XY, end geometry.XY) EdgeKey {
	return EdgeKey{
		x1: int64(math.Round(start.X * precision)),
		y1: int64(math.Round(start.Y * precision)),
		x2: int64(math.Round(end.X * precision)),
		y2: int64(math.Round(end.Y * precision)),
	}
}

// Segment represents a line segment with start and end points, a reference, type, sector association, and texture animations.
type Segment struct {
	Start  geometry.XY
	End    geometry.XY
	Ref    string
	Kind   int
	Sector *Sector
	Tag    string
	Upper  *textures.Animation
	Middle *textures.Animation
	Lower  *textures.Animation
	aabb   *physics.AABB
}

// NewSegment creates a new Segment instance with the provided start and end points, textures, and associated metadata.
func NewSegment(ref string, sector *Sector, kind int, start geometry.XY, end geometry.XY, tag string, tUpper, tMiddle, tLower *textures.Animation) *Segment {
	out := &Segment{
		Start:  start,
		End:    end,
		Ref:    ref,
		Kind:   kind,
		Sector: sector,
		Tag:    tag,
		Upper:  tUpper,
		Middle: tMiddle,
		Lower:  tLower,
		aabb:   nil,
	}
	out.ComputeAABB()
	return out
}

// ComputeAABB calculates the axis-aligned bounding box (AABB) for the segment and stores it in the `aabb` field.
func (k *Segment) ComputeAABB() {
	const eps = 0.001
	minX := math.Min(k.Start.X, k.End.X)
	maxX := math.Max(k.Start.X, k.End.X)
	minY := math.Min(k.Start.Y, k.End.Y)
	maxY := math.Max(k.Start.Y, k.End.Y)
	if minX == maxX {
		minX -= eps
		maxX += eps
	}
	if minY == maxY {
		minY -= eps
		maxY += eps
	}
	k.aabb = physics.NewAABB(minX, minY, 0, maxX, maxY, 0)
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the segment.
func (k *Segment) GetAABB() *physics.AABB {
	return k.aabb
}

// Copy creates and returns a deep copy of the current Segment instance, duplicating all its fields.
func (k *Segment) Copy() *Segment {
	out := &Segment{
		Start:  k.Start,
		End:    k.End,
		Ref:    k.Ref,
		Kind:   k.Kind,
		Sector: k.Sector,
		Tag:    k.Tag,
		Upper:  k.Upper,
		Middle: k.Middle,
		Lower:  k.Lower,
	}
	return out
}

// SetSector updates the segment's reference and associates it with a specified sector.
func (k *Segment) SetSector(ref string, sector *Sector) {
	k.Ref = ref
	k.Sector = sector
}

// MakeStraightEdgeKey generates a unique EdgeKey for the segment using its start and end points with a fixed precision.
func (k *Segment) MakeStraightEdgeKey() EdgeKey {
	return makeEdgeKey(edgePrecision, k.Start, k.End)
}

// MakeReverseEdgeKey creates a reversed EdgeKey for the segment, with the end point treated as the start.
func (k *Segment) MakeReverseEdgeKey() EdgeKey {
	return makeEdgeKey(edgePrecision, k.End, k.Start)
}
