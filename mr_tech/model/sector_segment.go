package model

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type Segment struct {
	points    [3]geometry.XYZ
	minZ      float64
	maxZ      float64
	materials []*textures.Material
	tag       string
	parent    *Sector
	neighbor  *Sector
	aabb      *physics.AABB
}

func NewSegment(neighbor *Sector, start geometry.XY, end geometry.XY, tag string, materials []*textures.Material) *Segment {
	out := &Segment{
		neighbor:  neighbor,
		tag:       tag,
		minZ:      0,
		maxZ:      0,
		aabb:      physics.NewAABB(),
		materials: []*textures.Material{nil},
	}
	if len(materials) > 0 {
		out.materials = materials
	}
	out.points[0] = geometry.XYZ{X: start.X, Y: start.Y, Z: 0}
	out.points[1] = geometry.XYZ{X: (start.X + end.X) * 0.5, Y: (start.Y + end.Y) * 0.5, Z: 0}
	out.points[2] = geometry.XYZ{X: end.X, Y: end.Y, Z: 0}
	out.Rebuild()
	return out
}

func (s *Segment) GetTag() string {
	return s.tag
}

func (s *Segment) GetParent() *Sector {
	return s.parent
}

func (s *Segment) SetParent(parent *Sector) {
	s.parent = parent
}

func (s *Segment) GetAABB() *physics.AABB {
	return s.aabb
}

func (s *Segment) GetNeighbor() *Sector {
	return s.neighbor
}

func (s *Segment) SetNeighbor(neighbor *Sector) {
	s.neighbor = neighbor
}

// GetStart returns the first point of the segment as a geometry.XYZ value.
func (s *Segment) GetStart() geometry.XYZ {
	return s.points[0]
}

// GetMiddle retrieves the middle point (geometry.XYZ) of the Face from its predefined points array.
func (s *Segment) GetMiddle() geometry.XYZ {
	return s.points[1]
}

// GetEnd returns the last point of the segment as a geometry.XYZ value.
func (s *Segment) GetEnd() geometry.XYZ {
	return s.points[2]
}

func (s *Segment) SetZ(minZ, maxZ float64) {
	s.minZ = minZ
	s.maxZ = maxZ
	s.points[0].Z = minZ
	s.points[1].Z = minZ
	s.points[2].Z = minZ
	s.Rebuild()
}

// Rebuild calculates the axis-aligned bounding box (AABB) for the segment, considering both 2D and 3D cases.
func (s *Segment) Rebuild() {
	s.computeAABB()
}

func (s *Segment) GetPoints() [3]geometry.XYZ {
	return s.points
}

func (s *Segment) GetMaterialIndex(m int) *textures.Material {
	//0 Upper, 1 Middle, 2 Lower
	idx := m % len(s.materials)
	return s.materials[idx]
}

func (s *Segment) PointInLineSide(px, py float64) bool {
	start := s.GetStart()
	end := s.GetEnd()
	dir := mathematic.PointInLineDirectionF(px, py, start.X, start.Y, end.X, end.Y)
	if dir < 0 {
		return false
	}
	return true
}

// computeAABB calculates the axis-aligned bounding box (AABB) for the Face using its points and optional Z bounds.
func (s *Segment) computeAABB() {
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
	minZ = s.minZ
	maxZ = s.maxZ
	s.aabb.Rebuild(minX-eps, minY-eps, minZ, maxX+eps, maxY+eps, maxZ)
}
