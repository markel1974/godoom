package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// edgePrecision defines the precision factor for converting geometric coordinates to integer-based EdgeKey values.
const edgePrecision = 1000.0

// EdgeKey represents a unique key for an edge defined by its start and end coordinates with integer precision.
type EdgeKey struct {
	x1, y1, x2, y2 int64
}

// EdgeSegment represents a line segment defined by a start and end point, associated with a sector and a numerical property.
type EdgeSegment struct {
	start  geometry.XY
	end    geometry.XY
	sector *Sector
	np     int
}

// makeEdgeKey generates a unique EdgeKey by scaling and rounding the coordinates of the start and end points.
func makeEdgeKey(precision float64, start geometry.XY, end geometry.XY) EdgeKey {
	return EdgeKey{
		x1: int64(math.Round(start.X * precision)),
		y1: int64(math.Round(start.Y * precision)),
		x2: int64(math.Round(end.X * precision)),
		y2: int64(math.Round(end.Y * precision)),
	}
}

// Segment represents a line segment with start and end coordinates, type, neighbor references, and texture animations.
type Segment struct {
	Parent   *Sector
	Start    geometry.XY
	End      geometry.XY
	Kind     int
	Neighbor *Sector
	Tag      string
	Upper    *textures.Animation
	Middle   *textures.Animation
	Lower    *textures.Animation
	aabb     *physics.AABB
}

// NewSegment creates and initializes a new Segment with the specified parameters and computes its axis-aligned bounding box (AABB).
func NewSegment(neighbor *Sector, kind int, start geometry.XY, end geometry.XY, tag string, tUpper, tMiddle, tLower *textures.Animation) *Segment {
	out := &Segment{
		Start:    start,
		End:      end,
		Kind:     kind,
		Neighbor: neighbor,
		Tag:      tag,
		Upper:    tUpper,
		Middle:   tMiddle,
		Lower:    tLower,
		aabb:     nil,
	}
	out.ComputeAABB()
	return out
}

// ComputeAABB calculates the axis-aligned bounding box (AABB) for the segment and updates its internal `aabb` field.
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

// GetAABB returns the axis-aligned bounding box (AABB) associated with the Segment instance.
func (k *Segment) GetAABB() *physics.AABB {
	return k.aabb
}

// MakeStraightEdgeKey generates a unique EdgeKey for the segment using its start and end points with predefined precision.
func (k *Segment) MakeStraightEdgeKey() EdgeKey {
	return makeEdgeKey(edgePrecision, k.Start, k.End)
}

// MakeReverseEdgeKey generates an EdgeKey by reversing the Start and End points of the Segment.
func (k *Segment) MakeReverseEdgeKey() EdgeKey {
	return makeEdgeKey(edgePrecision, k.End, k.Start)
}
