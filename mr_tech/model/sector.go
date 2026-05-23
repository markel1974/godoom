package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Slope represents a 2D inclined plane characterized by its normal vector and gradient for geometric calculations.
type Slope struct {
	Nx       float64
	Ny       float64
	Gradient float64
	Start    geometry.XY
	End      geometry.XY
}

// Sector represents a 3D environment area with assigned properties such as geometry, materials, light, and physics data.
type Sector struct {
	modelId      int
	id           string
	minZ         float64
	maxZ         float64
	materials    []*textures.Material
	tag          string
	segments     []*Segment
	segmentCount int
	light        *Light
	entity       *physics.Entity
	segmentsTree *physics.AABBTree
	slopeF       *Slope
	slopeC       *Slope
}

// NewSector creates and returns a pointer to a new Sector with specified parameters such as id, bounds, and materials.
func NewSector(modelId int, id string, minZ float64, maxZ float64, materials []*textures.Material, tag string) *Sector {
	const concreteMass = 0.0
	const concreteRestitution = 0.0
	const concreteFriction = 0.2
	const concreteGForce = 9.8
	s := &Sector{
		modelId:      modelId,
		id:           id,
		minZ:         minZ,
		maxZ:         maxZ,
		materials:    []*textures.Material{nil},
		tag:          tag,
		segments:     make([]*Segment, 128),
		segmentCount: 0,
		light:        nil,
		entity:       physics.NewEntity(concreteMass, concreteRestitution, concreteFriction, concreteGForce),
		segmentsTree: physics.NewAABBTree(64, 0.0),
		slopeF:       nil,
		slopeC:       nil,
	}
	if len(materials) > 0 {
		s.materials = materials
	}
	return s
}

// GetModelId retrieves the model ID associated with the sector instance.
func (s *Sector) GetModelId() int {
	return s.modelId
}

// GetId retrieves the unique identifier of the Sector instance.
func (s *Sector) GetId() string {
	return s.id
}

// GetEntity returns the physics.Entity instance associated with the Sector.
func (s *Sector) GetEntity() *physics.Entity {
	return s.entity
}

// GetAABB retrieves the axis-aligned bounding box (AABB) of the Sector's associated physics entity.
func (s *Sector) GetAABB() *physics.AABB {
	return s.entity.GetAABB()
}

// GetTag retrieves the tag string associated with the sector.
func (s *Sector) GetTag() string {
	return s.tag
}

// GetMinZ returns the minimum Z value (floor height) for the sector.
func (s *Sector) GetMinZ() float64 {
	return s.minZ
}

// GetMaxZ returns the maximum Z-coordinate value of the sector.
func (s *Sector) GetMaxZ() float64 {
	return s.maxZ
}

// GetLight retrieves the Light instance associated with the Sector. Returns a pointer to the Light object.
func (s *Sector) GetLight() *Light {
	return s.light
}

// GetMaterialIndex returns the material at the specified index modulo the total number of materials in the sector.
func (s *Sector) GetMaterialIndex(m int) *textures.Material {
	//floor 0, ceil 1
	idx := m % len(s.materials)
	return s.materials[idx]
}

// GetSlopes returns the floor and ceiling slopes of the sector as two Slope pointers.
func (s *Sector) GetSlopes() (*Slope, *Slope) {
	return s.slopeF, s.slopeC
}

// SetSlopes sets the floor and ceiling slopes of the sector using the provided Slope objects.
func (s *Sector) SetSlopes(slopeF *Slope, slopeC *Slope) {
	s.slopeF = slopeF
	s.slopeC = slopeC
}

// AddSegment adds a new segment to the sector, setting its parent and expanding the internal storage if necessary.
func (s *Sector) AddSegment(segment *Segment) {
	segment.SetParent(s)
	if s.segmentCount >= len(s.segments) {
		newFaces := make([]*Segment, s.segmentCount*2)
		copy(newFaces, s.segments)
		s.segments = newFaces
	}
	s.segments[s.segmentCount] = segment
	s.segmentCount++
}

// AddTag appends a semicolon-delimited string to the sector's existing tag if the input is non-empty.
func (s *Sector) AddTag(tags string) {
	if len(tags) > 0 {
		s.tag += ";" + tags
	}
}

// SetLight assigns a Light object to the Sector, replacing any previously set Light instance.
func (s *Sector) SetLight(light *Light) {
	s.light = light
}

// GetSegments returns the list of segments in the sector and the total count of those segments.
func (s *Sector) GetSegments() ([]*Segment, int) {
	return s.segments, s.segmentCount
}

// GetCentroid computes and returns the centroid (geometry.XYZ) of the Sector, considering its 2D polygon and minimum Z value.
func (s *Sector) GetCentroid() geometry.XYZ {
	var signedArea, cx, cy float64
	for x := 0; x < s.segmentCount; x++ {
		start := s.segments[x].GetStart()
		end := s.segments[x].GetEnd()
		x0, y0 := start.X, start.Y
		x1, y1 := end.X, end.Y

		a := (x0 * y1) - (x1 * y0)
		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}
	floorY := s.GetMinZ()
	signedArea *= 0.5
	if signedArea == 0 {
		start := s.segments[0].GetStart()
		return geometry.XYZ{X: start.X, Y: start.Y, Z: floorY}
	}
	return geometry.XYZ{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
		Z: floorY,
	}
}

// Rebuild updates the sector's bounding box, recalculates segment bounds, and rebuilds the spatial data structure.
func (s *Sector) Rebuild() bool {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < s.segmentCount; x++ {
		face := s.segments[x]
		face.SetZ(s.minZ, s.maxZ)
		for _, p := range face.GetPoints() {
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
		}
	}
	s.entity.GetAABB().Rebuild(minX, minY, s.minZ, maxX, maxY, s.maxZ)
	s.segmentsTree.Clear()
	for x := 0; x < s.segmentCount; x++ {
		segment := s.segments[x]
		segment.Rebuild()
		s.segmentsTree.InsertObject(segment)
	}
	return true
}

// PointInLineSide determines if a point (px, py) lies on the valid side of all segments within the Sector. Returns false if not.
func (s *Sector) PointInLineSide(px, py float64) bool {
	for x := 0; x < s.segmentCount; x++ {
		face := s.segments[x]
		if !face.PointInLineSide(px, py) {
			return false
		}
	}
	return true
}
