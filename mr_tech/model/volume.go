package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Volume represents a 3D navigable space (a region, brush, or room), defined by geometric faces, materials, and associated properties.
type Volume struct {
	modelId   int
	id        string
	faces     []*Face
	faceCount int
	tag       string
	materials []*textures.Material
	Light     *Light
	entity    *physics.Entity
	minZ      float64
	maxZ      float64
	hasFixedZ bool
	facesTree *physics.AABBTree
	thing     IThing
}

const solidRestitution = 0.0
const solidFriction = 0.2
const solidGForce = 9.8

// NewVolume2d creates a new 2.5D Volume instance with the specified attributes, mimicking legacy extruded world.
func NewVolume2d(modelId int, id string, minZ float64, maxZ float64, materials []*textures.Material, tag string) *Volume {
	v := NewVolumeDetails3d(modelId, id, tag, 0, 0, 0, 0, 0, 0, 0, solidRestitution, solidFriction, solidGForce)
	v.hasFixedZ = true
	v.minZ = minZ
	v.maxZ = maxZ
	if len(materials) > 0 {
		v.materials = materials
	}
	return v
}

// NewVolume3d creates and returns a new true 3D Volume instance (convex polyhedron) with the specified model ID, ID, and tag.
func NewVolume3d(modelId int, id string, tag string) *Volume {
	v := NewVolumeDetails3d(modelId, id, tag, 0, 0, 0, 0, 0, 0, 0, solidRestitution, solidFriction, solidGForce)
	return v
}

// NewVolumeDetails3d creates a new 3D Volume instance with specified properties, including position, size, and physics attributes.
func NewVolumeDetails3d(modelId int, id string, tag string, x, y, z, w, h, d, mass, restitution, friction, gForce float64) *Volume {
	v := &Volume{
		modelId:   modelId,
		id:        id,
		tag:       tag,
		minZ:      0,
		maxZ:      0,
		hasFixedZ: false,
		materials: []*textures.Material{nil},
		faces:     make([]*Face, 128),
		faceCount: 0,
		entity:    physics.NewEntity(x, y, z, w, h, d, mass, restitution, friction, gForce),
		facesTree: physics.NewAABBTree(64, 0.0),
	}
	return v
}

// Rebuild recalculates the axis-aligned bounding box (AABB) for the location based on its faces and dimensions.
func (v *Volume) Rebuild() bool {
	minX, minY, calcMinZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, calcMaxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < v.faceCount; x++ {
		face := v.faces[x]
		if v.hasFixedZ {
			face.SetZ(v.minZ, v.maxZ)
		}
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
			if p.Z < calcMinZ {
				calcMinZ = p.Z
			}
			if p.Z > calcMaxZ {
				calcMaxZ = p.Z
			}
		}
	}
	if v.hasFixedZ {
		calcMinZ = v.minZ
		calcMaxZ = v.maxZ
	} else {
		v.minZ = calcMinZ
		v.maxZ = calcMaxZ
	}
	v.entity.GetAABB().Rebuild(minX, minY, calcMinZ, maxX, maxY, calcMaxZ)

	v.facesTree.Clear()
	for x := 0; x < v.faceCount; x++ {
		face := v.faces[x]
		face.Rebuild()
		v.facesTree.InsertObject(face)
	}
	return true
}

// SetThing sets the IThing instance associated with the Volume.
func (v *Volume) SetThing(thing IThing) {
	v.thing = thing
}

// GetThing retrieves the IThing object associated with the Volume instance.
func (v *Volume) GetThing() IThing {
	return v.thing
}

func (v *Volume) GetEntity() *physics.Entity {
	return v.entity
}

// GetAABB returns the Axis-Aligned Bounding Box (AABB) of the location, representing its 3D bounds.
func (v *Volume) GetAABB() *physics.AABB {
	return v.entity.GetAABB()
}

// SetZ sets the minimum and maximum Z coordinates for the location, marks it as having custom Z bounds, and rebuilds its AABB.
func (v *Volume) SetZ(minZ, maxZ float64) {
	v.minZ = minZ
	v.maxZ = maxZ
	v.hasFixedZ = true
	v.Rebuild()
}

// ClearZ resets the Z-coordinate bounds of the location, marks it as lacking custom Z bounds, and triggers a rebuild.
func (v *Volume) ClearZ() {
	v.minZ = 0
	v.maxZ = 0
	v.hasFixedZ = false
	v.Rebuild()
}

// GetModelId retrieves the model ID associated with the Volume instance.
func (v *Volume) GetModelId() int {
	return v.modelId
}

// GetId retrieves the unique identifier of the location.
func (v *Volume) GetId() string {
	return v.id
}

// GetMinZ retrieves the minimum Z-coordinate of the location's axis-aligned bounding box (AABB).
func (v *Volume) GetMinZ() float64 {
	return v.entity.GetAABB().GetMinZ()
}

// GetMaxZ retrieves the maximum Z-coordinate of the Volume's axis-aligned bounding box (AABB).
func (v *Volume) GetMaxZ() float64 {
	return v.entity.GetAABB().GetMaxZ()
}

// GetMaterialIndex retrieves a material material from the location's materials list based on the provided index modulo the list size.
func (v *Volume) GetMaterialIndex(m int) *textures.Material {
	//floor 0, ceil 1
	idx := m % len(v.materials)
	return v.materials[idx]
}

// AddFace adds a new face to the location and sets the location as the parent of the face.
func (v *Volume) AddFace(face *Face) {
	face.SetParent(v)
	if v.faceCount >= len(v.faces) {
		newFaces := make([]*Face, v.faceCount*2)
		copy(newFaces, v.faces)
		v.faces = newFaces
	}
	v.faces[v.faceCount] = face
	v.faceCount++
}

// ClearFace resets the face count of the Volume to zero, effectively removing all associated faces.
func (v *Volume) ClearFace() {
	v.faceCount = 0
}

// GetFaces retrieves the list of face objects associated with the location.
func (v *Volume) GetFaces() ([]*Face, int) {
	return v.faces, v.faceCount
}

// AddTag appends the specified tags to the location's existing tags, separated by a semicolon.
func (v *Volume) AddTag(tags string) {
	if len(tags) > 0 {
		v.tag += ";" + tags
	}
}

// GetTag retrieves the tag string associated with the Volume instance.
func (v *Volume) GetTag() string {
	return v.tag
}

// Neighbor2d returns the neighboring location that contains the specified point (px, py, pz), or nil if no such location exists.
func (v *Volume) Neighbor2d(px, py, pz float64) *Volume {
	if v.hasFixedZ {
		if v.PointInLineSide(px, py) {
			return v
		}
		faces, faceCount := v.GetFaces()
		for x := 0; x < faceCount; x++ {
			face := faces[x]
			if neighbor := face.GetNeighbor(); neighbor != nil {
				if neighbor.PointInLineSide(px, py) {
					return neighbor
				}
			}
		}
		return nil
	}
	if v.PointInside2d(px, py, pz) {
		return v
	}

	faces, faceCount := v.GetFaces()
	for x := 0; x < faceCount; x++ {
		face := faces[x]
		if neighbor := face.GetNeighbor(); neighbor != nil {
			if neighbor.PointInside2d(px, py, pz) {
				return neighbor
			}
		}
	}
	return nil
}

// PointInside2d checks if a 3D point (px, py, pz) lies inside the 2D bounds of the Volume, accounting for fixed Z bounds.
func (v *Volume) PointInside2d(px, py, pz float64) bool {
	if v.hasFixedZ {
		const epsilon = 0.01
		if pz < (v.minZ-epsilon) || pz > (v.maxZ+epsilon) {
			return false
		}
		return v.PointInLineSide(px, py)
	}
	return false
}

// PointInLineSide checks if the point (px, py) lies on the inner side of all faces' lines within the location.
func (v *Volume) PointInLineSide(px, py float64) bool {
	for x := 0; x < v.faceCount; x++ {
		face := v.faces[x]
		if !face.PointInLineSide(px, py) {
			return false
		}
	}
	return true
}

// GetCentroid3d calculates and returns the geometric centroid of the location based on its faces and 3D mode.
func (v *Volume) GetCentroid3d() geometry.XYZ {
	var cx, cy, cz, count float64
	for x := 0; x < v.faceCount; x++ {
		face := v.faces[x]
		for _, p := range face.GetPoints() {
			cx += p.X
			cy += p.Y
			cz += p.Z
			count++
		}
	}
	if count > 0 {
		return geometry.XYZ{X: cx / count, Y: cy / count, Z: cz / count}
	}
	return geometry.XYZ{}
}

// GetCentroid2d calculates and returns the 2D centroid of the location projected onto the XY plane.
func (v *Volume) GetCentroid2d() geometry.XYZ {
	var signedArea, cx, cy float64
	for x := 0; x < v.faceCount; x++ {
		start := v.faces[x].GetStart()
		end := v.faces[x].GetEnd()
		x0, y0 := start.X, start.Y
		x1, y1 := end.X, end.Y

		a := (x0 * y1) - (x1 * y0)
		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}
	floorY := v.GetMinZ()
	signedArea *= 0.5
	if signedArea == 0 {
		start := v.faces[0].GetStart()
		return geometry.XYZ{X: start.X, Y: start.Y, Z: floorY}
	}
	return geometry.XYZ{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
		Z: floorY,
	}
}

// QueryOverlaps checks for overlaps between the Volume's AABB and the provided object, invoking a callback for each overlap.
func (v *Volume) QueryOverlaps(object physics.IAABB, callback func(object physics.IAABB) bool) {
	v.facesTree.QueryOverlaps(object, callback)
}

/*
// PointInside3d determines if the point (px, py, pz) lies inside the 3D location, considering optional fixed Z bounds.
func (v *Volume) PointInside3d(px, py, pz float64) bool {
	if v.hasFixedZ {
		const epsilon = 0.01
		if pz < (v.minZ-epsilon) || pz > (v.maxZ+epsilon) {
			return false
		}
		return v.PointInLineSide(px, py)
	}

	// Spara UN SOLO raggio (una direzione asimmetrica per evitare parallelismi perfetti)
	dirX, dirY, dirZ := 0.312, 0.945, 0.111

	minT := math.MaxFloat64
	var closestFace *Face // Sostituisci col tuo tipo Faccia esatto

	// 1. Trova l'intersezione più vicina in assoluto
	for _, face := range v.faces {
		hit, t := face.RayIntersectDist(px, py, pz, dirX, dirY, dirZ)

		// t > 0.0001 evita l'auto-intersezione se il punto è esattamente sul bordo
		if hit && t > 0.0001 && t < minT {
			minT = t
			closestFace = face
		}
	}

	// 2. Se non colpiamo nulla verso l'infinito, siamo chiaramente fuori
	if closestFace == nil {
		return false
	}

	// 3. Risoluzione tramite Dot Product
	// closestFace.nx, ny, nz devono essere la NORMALE USCENTE (Outward Normal) del triangolo
	normal := closestFace.GetNormal()
	dot := normal.X*dirX + normal.Y*dirY + normal.Z*dirZ

	// Se il dot è > 0, raggio e normale vanno nella stessa direzione.
	// Significa che stai "sfondando" la parete da dentro verso fuori.
	return dot > 0.0
}
*/
