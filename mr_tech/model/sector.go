package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

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
}

func NewSector(modelId int, id string, minZ float64, maxZ float64, materials []*textures.Material, tag string) *Sector {
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
		entity:       physics.NewEntity(0, 0, 0, 0, 0, 0, 0, solidRestitution, solidFriction, solidGForce),
		segmentsTree: physics.NewAABBTree(64, 0.0),
	}
	if len(materials) > 0 {
		s.materials = materials
	}
	return s
}

func (s *Sector) GetModelId() int {
	return s.modelId
}

func (s *Sector) GetId() string {
	return s.id
}

func (s *Sector) GetEntity() *physics.Entity {
	return s.entity
}

func (s *Sector) GetAABB() *physics.AABB {
	return s.entity.GetAABB()
}

func (s *Sector) GetTag() string {
	return s.tag
}

func (s *Sector) GetMinZ() float64 {
	return s.minZ
}

func (s *Sector) GetMaxZ() float64 {
	return s.maxZ
}

func (s *Sector) GetLight() *Light {
	return s.light
}

// GetMaterialIndex retrieves a material material from the location's materials list based on the provided index modulo the list size.
func (s *Sector) GetMaterialIndex(m int) *textures.Material {
	//floor 0, ceil 1
	idx := m % len(s.materials)
	return s.materials[idx]
}

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

func (s *Sector) AddTag(tags string) {
	if len(tags) > 0 {
		s.tag += ";" + tags
	}
}

func (s *Sector) SetLight(light *Light) {
	s.light = light
}

func (s *Sector) GetSegments() ([]*Segment, int) {
	return s.segments, s.segmentCount
}

func (s *Sector) GetCentroid2d() geometry.XYZ {
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

// PointInLineSide checks if the point (px, py) lies on the inner side of all faces' lines within the location.
func (s *Sector) PointInLineSide(px, py float64) bool {
	for x := 0; x < s.segmentCount; x++ {
		face := s.segments[x]
		if !face.PointInLineSide(px, py) {
			return false
		}
	}
	return true
}
