package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/model/sectors"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Volume represents a spatial model element that groups geometry, lighting, physics, and sector data.
type Volume struct {
	modelId    int
	id         string
	faces      *Faces
	tag        string
	light      *lights.Light
	entity     *rigid_body.Entity
	facesTree  *aabb.Tree
	thing      IThing
	sector     *sectors.Sector
	vertexTags []*articulated_dynamics.TagAxis
	marker     int
	dirty      bool
}

func NewVolume(modelId int, id string, tag string, mass, restitution, friction float64, restCombine, frictionCombine int, gForce float64) *Volume {
	v := &Volume{
		modelId:    modelId,
		id:         id,
		tag:        tag,
		faces:      NewFaces(128, false),
		entity:     rigid_body.NewEntity(mass, restitution, friction, restCombine, frictionCombine, gForce),
		facesTree:  aabb.NewTree(64, 0.0),
		vertexTags: make([]*articulated_dynamics.TagAxis, 0),
		dirty:      true,
		marker:     -1,
	}
	return v
}

func NewVolumeConcrete(modelId int, id string, tag string) *Volume {
	const concreteMass = 0.0
	const concreteRestitution = 0.0
	const concreteFriction = 0.2
	const concreteGForce = 9.8
	return NewVolume(modelId, id, tag, concreteMass, concreteRestitution, concreteFriction, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), concreteGForce)
}

// Rebuild updates all faces, refreshes the face tree and bounding entity, marks the volume dirty, and returns true.
func (v *Volume) Rebuild() bool {
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	v.facesTree.Clear()

	for x := 0; x < v.faces.Len(); x++ {
		face := v.faces.container[x]
		face.Rebuild()
		v.facesTree.InsertObject(face)
		minX, minY, minZ, maxX, maxY, maxZ = face.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
	}
	w := maxX - minX
	h := maxY - minY
	d := maxZ - minZ

	v.entity.Rebuild(minX, minY, minZ, w, h, d)
	//v.entity.GetAABB().Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
	v.dirty = true

	return true
}

// GetFacesTree returns the spatial tree containing the volume's faces.
func (v *Volume) GetFacesTree() *aabb.Tree {
	return v.facesTree
}

// IsDirty reports whether the volume has been marked dirty.
func (v *Volume) IsDirty() bool {
	return v.dirty
}

// SetDirty sets whether the volume is marked dirty.
func (v *Volume) SetDirty(dirty bool) {
	v.dirty = dirty
}

// ClearDirty clears the dirty flag.
func (v *Volume) ClearDirty() {
	v.dirty = false
}

// SetMarker sets the marker identifier used to track staged volume data.
func (v *Volume) SetMarker(marker int) {
	v.marker = marker
}

// GetMarker returns the marker identifier assigned to the volume.
func (v *Volume) GetMarker() int {
	return v.marker
}

// SetThing assigns the specified IThing instance to the volume.
func (v *Volume) SetThing(thing IThing) {
	v.thing = thing
}

// GetThing returns the thing associated with the volume.
func (v *Volume) GetThing() IThing {
	return v.thing
}

// GetEntity returns the physics entity associated with the volume.
func (v *Volume) GetEntity() *rigid_body.Entity {
	return v.entity
}

// GetAABB returns the axis-aligned bounding box associated with the volume's entity.
func (v *Volume) GetAABB() *aabb.AABB {
	return v.entity.GetAABB()
}

// GetModelId returns the model identifier assigned to the volume.
func (v *Volume) GetModelId() int {
	return v.modelId
}

func (v *Volume) GetId() string {
	return v.id
}

// GetLight returns the light associated with the volume.
func (v *Volume) GetLight() *lights.Light {
	return v.light
}

// AddFace adds face to the volume and sets its parent to the volume.
func (v *Volume) AddFace(face *Face) {
	face.SetParent(v)
	v.faces.Add(face)
}

// ClearFaces removes all faces from the volume.
func (v *Volume) ClearFaces() {
	v.faces.Clear()
}

// GetFace returns the face at the specified index.
func (v *Volume) GetFace(index int) *Face {
	return v.faces.At(index)
}

// GetFaceCount returns the number of faces in the volume.
func (v *Volume) GetFaceCount() int {
	return v.faces.Len()
}

// GetFaces returns the face collection associated with the volume.
func (v *Volume) GetFaces() *Faces {
	return v.faces
}

// SetLight assigns the light to the volume and sets the volume as the light's parent.
func (v *Volume) SetLight(light *lights.Light) {
	v.light = light
	v.light.SetParent(v)
}

// GetSector returns the sector associated with the volume.
func (v *Volume) GetSector() *sectors.Sector {
	return v.sector
}

// SetSector assigns the sector associated with the volume.
func (v *Volume) SetSector(s *sectors.Sector) {
	v.sector = s
}

// SetVertexTag sets the vertex tag at idx with the given origin and axis, growing storage as needed.
func (v *Volume) SetVertexTag(idx int, origin geometry.XYZ, axis [3]geometry.XYZ) {
	if idx >= len(v.vertexTags) {
		newTags := make([]*articulated_dynamics.TagAxis, idx+1)
		copy(newTags, v.vertexTags)
		v.vertexTags = newTags
	}
	v.vertexTags[idx] = articulated_dynamics.NewTagAxis(origin, axis)
}

// HasVertexTag reports whether idx is a valid vertex tag index.
func (v *Volume) HasVertexTag(idx int) bool {
	return idx >= 0 && idx < len(v.vertexTags)
}

// GetVertexTag returns the vertex tag at idx, or nil if idx is out of range.
func (v *Volume) GetVertexTag(idx int) *articulated_dynamics.TagAxis {
	if idx >= 0 && idx < len(v.vertexTags) {
		return v.vertexTags[idx]
	}
	return nil
}

// AddTag appends non-empty tags to the volume's tag list.
func (v *Volume) AddTag(tags string) {
	if len(tags) > 0 {
		v.tag += ";" + tags
	}
}

// GetTag returns the volume tag string.
func (v *Volume) GetTag() string {
	return v.tag
}

// GetCentroid returns the average position of all points across the volume's faces.
// GetCentroid returns a zero XYZ when the volume has no face points.
func (v *Volume) GetCentroid() (float64, float64, float64) {
	return v.faces.GetCentroid()
}

// QueryOverlaps searches the volume's face tree for objects whose AABB overlaps the given object's AABB.
func (v *Volume) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.facesTree.QueryOverlaps(object, callback)
}

// QueryPoint calls callback for each face whose bounding box contains the given 3D point.
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
