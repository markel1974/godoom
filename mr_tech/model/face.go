package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// edgePrecision defines the scaling factor used to convert floating-point coordinates into integer-based EdgeKey components.
const edgePrecision = 1000.0

// EdgeKey represents a unique identifier for an edge in 2D space, defined by the rounded coordinates of its endpoints.
type EdgeKey struct {
	x1, y1, x2, y2 int64
}

// makeEdgeKey generates an EdgeKey by scaling and rounding the start and end coordinates using the given precision.
func makeEdgeKey(precision float64, start geometry.XYZ, end geometry.XYZ) EdgeKey {
	return EdgeKey{
		x1: int64(math.Round(start.X * precision)),
		y1: int64(math.Round(start.Y * precision)),
		x2: int64(math.Round(end.X * precision)),
		y2: int64(math.Round(end.Y * precision)),
	}
}

// Face represents a boundary or edge of a Sector, defined by its geometry, connectivity, and optional metadata.
type Face struct {
	parent   *Volume
	kind     int
	neighbor *Volume
	tag      string
	aabb     *physics.AABB
	points   []geometry.XYZ
	pSize    int
	normal   geometry.XYZ
	material [3]*textures.Animation
	minZ     float64
	maxZ     float64
	hasZ     bool
}

// NewFaceSegment creates a new Face with specified geometry, type, associated neighbor, tag, and texture animations.
func NewFaceSegment(neighbor *Volume, kind int, start geometry.XYZ, end geometry.XYZ, tag string, tUpper, tMiddle, tLower *textures.Animation) *Face {
	out := &Face{
		hasZ:     true,
		points:   make([]geometry.XYZ, 2),
		kind:     kind,
		neighbor: neighbor,
		tag:      tag,
		minZ:     0,
		maxZ:     0,
	}
	out.material[0] = tUpper
	out.material[1] = tMiddle
	out.material[2] = tLower
	out.points[0] = start
	out.points[1] = end
	out.pSize = len(out.points) - 1
	out.Rebuild()
	return out
}

// NewFace creates a new 3D segment with specified neighbor, kind, points, tag, and material, and computes its normal and AABB.
func NewFace(neighbor *Volume, kind int, points []geometry.XYZ, tag string, material *textures.Animation) *Face {
	out := &Face{
		hasZ:     false,
		points:   points,
		kind:     kind,
		neighbor: neighbor,
		tag:      tag,
	}
	out.material[0] = material
	out.material[1] = material
	out.material[2] = material
	for i := len(points); i < 3; i++ {
		points[i] = geometry.XYZ{X: 0, Y: 0, Z: 0}
	}
	out.pSize = len(out.points) - 1
	out.ComputeNormal()
	out.Rebuild()
	return out
}

// SetZ sets the minimum and maximum Z coordinates for the volume, marks it as having custom Z bounds, and rebuilds its AABB.
func (s *Face) SetZ(minZ, maxZ float64) {
	s.minZ = minZ
	s.maxZ = maxZ
	s.hasZ = true
	s.Rebuild()
}

// ClearZ resets the Z-coordinate bounds of the volume, marks it as lacking custom Z bounds, and triggers a rebuild.
func (s *Face) ClearZ() {
	s.minZ = 0
	s.maxZ = 0
	s.hasZ = false
	s.Rebuild()
}

// GetParent retrieves the parent Sector of the Face instance. Returns nil if no parent is set.
func (s *Face) GetParent() *Volume {
	return s.parent
}

// SetParent assigns a parent sector to the segment.
func (s *Face) SetParent(parent *Volume) {
	s.parent = parent
}

// GetKind retrieves the integer value representing the kind or type of the segment.
func (s *Face) GetKind() int {
	return s.kind
}

// SetKind sets the type or category of the segment by assigning a specific integer value to its kind field.
func (s *Face) SetKind(kind int) {
	s.kind = kind
}

// GetNeighbor returns the neighboring Sector associated with the Face.
func (s *Face) GetNeighbor() *Volume {
	return s.neighbor
}

// SetNeighbor sets the neighbor sector of the segment. It establishes a link to an adjacent sector.
func (s *Face) SetNeighbor(neighbor *Volume) {
	s.neighbor = neighbor
}

// GetTag returns the tag associated with the segment.
func (s *Face) GetTag() string {
	return s.tag
}

// SetTag sets the tag field of the Face to the specified string value.
func (s *Face) SetTag(tag string) {
	s.tag = tag
}

// GetNormal returns the precomputed normal vector (geometry.XYZ) of the Face.
func (s *Face) GetNormal() geometry.XYZ {
	return s.normal
}

// GetStart returns the first point of the segment as a geometry.XYZ value.
func (s *Face) GetStart() geometry.XYZ {
	return s.points[0]
}

// GetEnd returns the last point of the segment as a geometry.XYZ value.
func (s *Face) GetEnd() geometry.XYZ {
	return s.points[s.pSize]
}

// GetMaterialUpper retrieves the upper texture animation for the segment, typically used for rendering upper walls.
func (s *Face) GetMaterialUpper() *textures.Animation {
	return s.material[0]
}

// GetMaterialMiddle retrieves the middle texture animation of the segment.
func (s *Face) GetMaterialMiddle() *textures.Animation {
	return s.material[1]
}

// GetMaterialLower returns the lower animation material of the segment located at index 2 in the material list.
func (s *Face) GetMaterialLower() *textures.Animation {
	return s.material[2]
}

// GetPoints returns the list of 3D points (geometry.XYZ) that define the segment's shape or path.
func (s *Face) GetPoints() []geometry.XYZ {
	return s.points
}

// GetMaterial retrieves the first material (upper texture) of the segment from the material slice.
func (s *Face) GetMaterial() *textures.Animation {
	return s.material[0]
}

// Scale2D scales the starting and ending points of the segment by applying the given scale factor.
func (s *Face) Scale2D(scale float64) {
	for idx := range s.points {
		s.points[idx].Scale(scale)
	}
}

// ComputeNormal calculates and sets the unit normal vector for the segment based on its first three points.
func (s *Face) ComputeNormal() {
	if len(s.points) < 3 {
		s.normal = geometry.XYZ{X: 0, Y: 0, Z: 1}
		return
	}
	p0, p1, p2 := s.points[0], s.points[1], s.points[2]
	v1x, v1y, v1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	v2x, v2y, v2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
	nx := v1y*v2z - v1z*v2y
	ny := v1z*v2x - v1x*v2z
	nz := v1x*v2y - v1y*v2x
	l := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if l > 0 {
		s.normal = geometry.XYZ{X: nx / l, Y: ny / l, Z: nz / l}
	}
}

// Rebuild calculates the axis-aligned bounding box (AABB) for the segment, considering both 2D and 3D cases.
func (s *Face) Rebuild() {
	const eps = 0.001
	minX, minY, minZ := s.points[0].X, s.points[0].Y, s.points[0].Z
	maxX, maxY, maxZ := minX, minY, minZ
	for _, p := range s.points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
		if p.Z < minZ {
			minZ = p.Z
		}
		if p.Z > maxZ {
			maxZ = p.Z
		}
	}
	if s.hasZ {
		minZ = s.minZ
		maxZ = s.maxZ
	} else {
		if minZ == maxZ {
			minZ -= eps
			maxZ += eps
		}
	}
	s.aabb = physics.NewAABB(minX-eps, minY-eps, minZ, maxX+eps, maxY+eps, maxZ)
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the segment.
func (s *Face) GetAABB() *physics.AABB {
	return s.aabb
}

// MakeStraightEdgeKey generates an EdgeKey for the segment using its start and end points, based on a fixed precision.
func (s *Face) MakeStraightEdgeKey() EdgeKey {
	return makeEdgeKey(edgePrecision, s.GetStart(), s.GetEnd())
}

// MakeReverseEdgeKey generates an EdgeKey by reversing the start and end points of the segment with a fixed precision.
func (s *Face) MakeReverseEdgeKey() EdgeKey {
	return makeEdgeKey(edgePrecision, s.GetEnd(), s.GetStart())
}
