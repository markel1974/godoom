package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/model/sectors"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Tag represents a 3D attachment point with spatial origin and 3x3 orientation axes.
type Tag struct {
	Origin geometry.XYZ
	Axis   [3]geometry.XYZ
}

// Volume represents a 3D navigable space (a region, brush, or room), defined by geometric faces, materials, and associated properties.
type Volume struct {
	modelId    int
	id         string
	faces      []*Face
	facesPtr   *[]*Face
	faceCount  int
	tag        string
	light      *lights.Light
	entity     *physics.Entity
	facesTree  *aabb.Tree
	thing      IThing
	sector     *sectors.Sector
	vertexTags []Tag
}

// NewVolume creates a new 3D Volume instance with specified properties, including position, size, and physics attributes.
func NewVolume(modelId int, id string, tag string, mass, restitution, friction, gForce float64) *Volume {
	v := &Volume{
		modelId:    modelId,
		id:         id,
		tag:        tag,
		faces:      make([]*Face, 128),
		faceCount:  0,
		entity:     physics.NewEntity(mass, restitution, friction, gForce),
		facesTree:  aabb.NewTree(64, 0.0),
		vertexTags: make([]Tag, 0),
	}
	v.facesPtr = &v.faces
	return v
}

// NewVolumeConcrete creates a new Volume instance configured with concrete-specific physics properties like mass, friction, and restitution.
func NewVolumeConcrete(modelId int, id string, tag string) *Volume {
	const concreteMass = 0.0
	const concreteRestitution = 0.0
	const concreteFriction = 0.2
	const concreteGForce = 9.8
	return NewVolume(modelId, id, tag, concreteMass, concreteRestitution, concreteFriction, concreteGForce)
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
func (v *Volume) GetAABB() *aabb.AABB {
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
func (v *Volume) GetLight() *lights.Light {
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
func (v *Volume) SetLight(light *lights.Light) {
	v.light = light
	v.light.SetParent(v)
}

// GetSector retrieves the Sector associated with the Volume instance.
func (v *Volume) GetSector() *sectors.Sector {
	return v.sector
}

// SetSector assigns the specified Sector to the Volume instance.
func (v *Volume) SetSector(s *sectors.Sector) {
	v.sector = s
}

// SetVertexTag associates the given tag index with the specified origin and orientation axes in the Volume's vertexTags array.
func (v *Volume) SetVertexTag(idx int, origin geometry.XYZ, axis [3]geometry.XYZ) {
	if idx >= len(v.vertexTags) {
		newTags := make([]Tag, idx+1)
		copy(newTags, v.vertexTags)
		v.vertexTags = newTags
	}
	if axis[0].X == 0 && axis[0].Y == 0 && axis[0].Z == 0 &&
		axis[1].X == 0 && axis[1].Y == 0 && axis[1].Z == 0 &&
		axis[2].X == 0 && axis[2].Y == 0 && axis[2].Z == 0 {
		axis[0] = geometry.XYZ{X: 1, Y: 0, Z: 0}
		axis[1] = geometry.XYZ{X: 0, Y: 1, Z: 0}
		axis[2] = geometry.XYZ{X: 0, Y: 0, Z: 1}
	}
	v.vertexTags[idx] = Tag{Origin: origin, Axis: axis}
}

// GetVertexTag retrieves the tag origin and orientation axes associated with a given tag index from the Volume's vertexTags array.
func (v *Volume) GetVertexTag(idx int) (geometry.XYZ, [3]geometry.XYZ, bool) {
	if idx >= 0 && idx < len(v.vertexTags) {
		return v.vertexTags[idx].Origin, v.vertexTags[idx].Axis, true
	}
	identity := [3]geometry.XYZ{
		{X: 1, Y: 0, Z: 0},
		{X: 0, Y: 1, Z: 0},
		{X: 0, Y: 0, Z: 1},
	}
	return geometry.XYZ{}, identity, false
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

// BuildQuad constructs two triangular faces that form a quad, applies UV mapping, and adds them to the specified 3D volume.
func (v *Volume) BuildQuad(s, e geometry.XY, zBS, zBE, zTS, zTE float64, tag string, material *textures.Material, align *config.UVAlign, t1, t2 float64) {
	v0 := geometry.XYZ{X: s.X, Y: s.Y, Z: zBS}
	v1 := geometry.XYZ{X: e.X, Y: e.Y, Z: zBE}
	v2 := geometry.XYZ{X: e.X, Y: e.Y, Z: zTE}
	v3 := geometry.XYZ{X: s.X, Y: s.Y, Z: zTS}

	face1 := NewFace([3]geometry.XYZ{v0, v1, v2}, tag, material)
	face2 := NewFace([3]geometry.XYZ{v0, v2, v3}, tag, material)
	if align != nil {
		if material != nil {
			if tex := material.HeadFrame(); tex != nil {
				w, h := tex.GetSizeScaled()
				uStart, uEnd, vBS, vBE, vTS, vTE := align.ComputeWall(w, h, t1, t2, s, e, zBS, zBE, zTS, zTE)
				face1.SetUV(uStart, vBS, uEnd, vBE, uEnd, vTE)
				face1.LockUV(true)
				face2.SetUV(uStart, vBS, uEnd, vTE, uStart, vTS)
				face2.LockUV(true)
			}
		}
	}
	v.AddFace(face1)
	v.AddFace(face2)
}

// QueryOverlaps checks for overlaps between the Volume's AABB and the provided object, invoking a callback for each overlap.
func (v *Volume) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.facesTree.QueryOverlaps(object, callback)
}

func (v *Volume) QueryPoint(px, py, pz float64, callback func(object aabb.IObject) bool) {
	v.facesTree.QueryPoint3d(px, py, pz, callback)
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
	n := closestFace.GetNormal()
	dot := n.X*dirX + n.Y*dirY + n.Z*dirZ

	// Se il dot è > 0, raggio e normale vanno nella stessa direzione.
	// Significa che stai "sfondando" la parete da dentro verso fuori.
	return dot > 0.0
}
*/
