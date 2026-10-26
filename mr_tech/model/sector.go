package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

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
	segmentsTree *aabb.Tree
	slopeF       *Slope
	slopeC       *Slope
	floorAlign   *config.UVAlign
	ceilAlign    *config.UVAlign
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
		segmentsTree: aabb.NewTree(64, 0.0),
		slopeF:       nil,
		slopeC:       nil,
	}
	if len(materials) > 0 {
		s.materials = materials
	}
	return s
}

// GetModelId retrieves the model ID associated with the sector instance.
func (sr *Sector) GetModelId() int {
	return sr.modelId
}

// GetId retrieves the unique identifier of the Sector instance.
func (sr *Sector) GetId() string {
	return sr.id
}

// GetEntity returns the physics.Entity instance associated with the Sector.
func (sr *Sector) GetEntity() *physics.Entity {
	return sr.entity
}

// GetAABB retrieves the axis-aligned bounding box (AABB) of the Sector's associated physics entity.
func (sr *Sector) GetAABB() *aabb.AABB {
	return sr.entity.GetAABB()
}

// GetTag retrieves the tag string associated with the sector.
func (sr *Sector) GetTag() string {
	return sr.tag
}

// GetMinZ returns the minimum Z value (floor height) for the sector.
func (sr *Sector) GetMinZ() float64 {
	return sr.minZ
}

// GetMaxZ returns the maximum Z-coordinate value of the sector.
func (sr *Sector) GetMaxZ() float64 {
	return sr.maxZ
}

// GetLight retrieves the Light instance associated with the Sector. Returns a pointer to the Light object.
func (sr *Sector) GetLight() *Light {
	return sr.light
}

// GetMaterialIndex returns the material at the specified index modulo the total number of materials in the sector.
func (sr *Sector) GetMaterialIndex(m int) *textures.Material {
	//floor 0, ceil 1
	idx := m % len(sr.materials)
	return sr.materials[idx]
}

// GetSlopes returns the floor and ceiling slopes of the sector as two Slope pointers.
func (sr *Sector) GetSlopes() (*Slope, *Slope) {
	return sr.slopeF, sr.slopeC
}

// SetSlopes sets the floor and ceiling slopes of the sector using the provided Slope objects.
func (sr *Sector) SetSlopes(slopeF *Slope, slopeC *Slope) {
	sr.slopeF = slopeF
	sr.slopeC = slopeC
}

// SetAlign sets the UV alignment parameters for floor and ceiling.
func (sr *Sector) SetAlign(floorAlign, ceilAlign *config.UVAlign) {
	sr.floorAlign = floorAlign
	sr.ceilAlign = ceilAlign
}

// GetFloorAlign returns the UV alignment configuration for the floor.
func (sr *Sector) GetFloorAlign() *config.UVAlign {
	return sr.floorAlign
}

// GetCeilAlign returns the UV alignment configuration for the ceiling.
func (sr *Sector) GetCeilAlign() *config.UVAlign {
	return sr.ceilAlign
}

// AddSegment adds a new segment to the sector, setting its parent and expanding the internal storage if necessary.
func (sr *Sector) AddSegment(segment *Segment) {
	segment.SetParent(sr)
	if sr.segmentCount >= len(sr.segments) {
		newFaces := make([]*Segment, sr.segmentCount*2)
		copy(newFaces, sr.segments)
		sr.segments = newFaces
	}
	sr.segments[sr.segmentCount] = segment
	sr.segmentCount++
}

// AddTag appends a semicolon-delimited string to the sector's existing tag if the input is non-empty.
func (sr *Sector) AddTag(tags string) {
	if len(tags) > 0 {
		sr.tag += ";" + tags
	}
}

// GetSegments returns the list of segments in the sector and the total count of those segments.
func (sr *Sector) GetSegments() ([]*Segment, int) {
	return sr.segments, sr.segmentCount
}

func (sr *Sector) GetLightPos(z float64) geometry.XYZ {
	centroid := sr.GetCentroid()
	lightPos := geometry.XYZ{X: centroid.X, Y: centroid.Y, Z: z}
	return lightPos
}

// CreateLight initializes a light source for the Sector at a specified Z-coordinate using the provided light configuration.
func (sr *Sector) CreateLight(light *config.Light, z float64) {
	if light == nil {
		return
	}
	centroid := sr.GetCentroid()
	lightPos := geometry.XYZ{X: centroid.X, Y: centroid.Y, Z: z}
	sr.light = NewLight()
	sr.light.Setup(light, lightPos)
}

// GetCentroid computes and returns the centroid (geometry.XYZ) of the Sector, considering its 2D polygon and minimum Z value.
func (sr *Sector) GetCentroid() geometry.XYZ {
	var signedArea, cx, cy float64
	for x := 0; x < sr.segmentCount; x++ {
		start := sr.segments[x].GetStart()
		end := sr.segments[x].GetEnd()
		x0, y0 := start.X, start.Y
		x1, y1 := end.X, end.Y

		a := (x0 * y1) - (x1 * y0)
		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}
	floorY := sr.GetMinZ()
	signedArea *= 0.5
	if signedArea == 0 {
		start := sr.segments[0].GetStart()
		return geometry.XYZ{X: start.X, Y: start.Y, Z: floorY}
	}
	return geometry.XYZ{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
		Z: floorY,
	}
}

// Rebuild updates the sector's bounding box, recalculates segment bounds, and rebuilds the spatial data structure.
func (sr *Sector) Rebuild() bool {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < sr.segmentCount; x++ {
		face := sr.segments[x]
		face.SetZ(sr.minZ, sr.maxZ)
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
	sr.entity.GetAABB().Rebuild(minX, minY, sr.minZ, maxX, maxY, sr.maxZ)
	sr.segmentsTree.Clear()
	for x := 0; x < sr.segmentCount; x++ {
		segment := sr.segments[x]
		segment.Rebuild()
		sr.segmentsTree.InsertObject(segment)
	}
	return true
}

// PointInLineSide determines if a point (px, py) lies on the valid side of all segments within the Sector. Returns false if not.
func (sr *Sector) PointInLineSide(px, py float64) bool {
	for x := 0; x < sr.segmentCount; x++ {
		face := sr.segments[x]
		if !face.PointInLineSide(px, py) {
			return false
		}
	}
	return true
}

// ResolveZ calculates the adjusted floor (zF) and ceiling (zC) heights at a given point p using base values and slope data.
func (sr *Sector) ResolveZ(p geometry.XYZ, baseF, baseC float64, clamp bool) (float64, float64) {
	zF := baseF
	if sr.slopeF != nil {
		zF = sr.slopeF.ComputeZ(baseF, p.X, p.Y)
	}
	zC := baseC
	if sr.slopeC != nil {
		zC = sr.slopeC.ComputeZ(baseC, p.X, p.Y)
	}
	if clamp && zF > zC {
		mid := (zF + zC) * 0.5
		return mid, mid
	}
	return zF, zC
}
