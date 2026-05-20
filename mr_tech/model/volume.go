package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Volume represents a 3D navigable space (a region, brush, or room), defined by geometric faces, materials, and associated properties.
type Volume struct {
	modelId   int
	id        string
	faces     []*Face
	facesPtr  *[]*Face
	faceCount int
	tag       string
	light     *Light
	entity    *physics.Entity
	facesTree *physics.AABBTree
	thing     IThing
	sector    *Sector
}

const solidRestitution = 0.0
const solidFriction = 0.2
const solidGForce = 9.8

// NewVolume creates a new 3D Volume instance with specified properties, including position, size, and physics attributes.
func NewVolume(modelId int, id string, tag string, mass, restitution, friction, gForce float64) *Volume {
	v := &Volume{
		modelId:   modelId,
		id:        id,
		tag:       tag,
		faces:     make([]*Face, 128),
		faceCount: 0,
		entity:    physics.NewEntity(0, 0, 0, 0, 0, 0, mass, restitution, friction, gForce),
		facesTree: physics.NewAABBTree(64, 0.0),
	}
	v.facesPtr = &v.faces
	return v
}

// Rebuild recalculates the axis-aligned bounding box (AABB) for the location based on its faces and dimensions.
func (v *Volume) Rebuild() bool {
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	v.facesTree.Clear()

	for x := 0; x < v.faceCount; x++ {
		face := v.faces[x]
		face.Rebuild()
		v.facesTree.InsertObject(face)

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
			if p.Z < minZ {
				minZ = p.Z
			}
			if p.Z > maxZ {
				maxZ = p.Z
			}
		}
	}
	w := maxX - minX
	h := maxY - minY
	d := maxZ - minZ

	v.entity.Rebuild(minX, minY, minZ, w, h, d)
	//v.entity.GetAABB().Rebuild(minX, minY, minZ, maxX, maxY, maxZ)

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

// GetModelId retrieves the model ID associated with the Volume instance.
func (v *Volume) GetModelId() int {
	return v.modelId
}

// GetId retrieves the unique identifier of the location.
func (v *Volume) GetId() string {
	return v.id
}

// GetLight retrieves the Light object associated with the Volume, or nil if no Light is assigned.
func (v *Volume) GetLight() *Light {
	return v.light
}

// AddFace adds a new face to the location and sets the location as the parent of the face.
func (v *Volume) AddFace(face *Face) {
	face.SetParent(v)
	if v.faceCount >= len(v.faces) {
		newFaces := make([]*Face, v.faceCount*2)
		copy(newFaces, v.faces)
		v.faces = newFaces
		v.facesPtr = &v.faces
	}
	v.faces[v.faceCount] = face
	v.faceCount++
}

// ClearFaces resets the face count of the Volume to zero, effectively removing all associated faces.
func (v *Volume) ClearFaces() {
	v.faceCount = 0
}

// GetFace returns the Face at the specified index within the Volume's faces array.
func (v *Volume) GetFace(index int) *Face {
	return v.faces[index]
}

// GetFaceCount returns the number of faces currently associated with the Volume instance.
func (v *Volume) GetFaceCount() int {
	return v.faceCount
}

// GetFaces returns the list of all faces and the total count of faces associated with the Volume.
func (v *Volume) GetFaces() (*[]*Face, int) {
	return v.facesPtr, v.faceCount
}

// SetLight assigns a Light object to the Volume and establishes the Volume as the parent of the Light instance.
func (v *Volume) SetLight(light *Light) {
	v.light = light
	v.light.SetParent(v)
}

// GetSector retrieves the Sector associated with the Volume instance.
func (v *Volume) GetSector() *Sector {
	return v.sector
}

// SetSector assigns the specified Sector to the Volume instance.
func (v *Volume) SetSector(s *Sector) {
	v.sector = s
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

// GetCentroid calculates and returns the geometric centroid of the location based on its faces and 3D mode.
func (v *Volume) GetCentroid() geometry.XYZ {
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
