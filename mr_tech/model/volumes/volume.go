package volumes

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Volume represents a spatial model element that groups geometry, lighting, physics, and sector data.
type Volume struct {
	modelId    int
	id         string
	faces      *Faces
	tag        string
	light      *lights.Light
	solver     *collisions.Solver
	aabb       *aabb.AABB
	object     any
	vertexTags []*articulated_dynamics.TagAxis
	marker     int
	dirty      bool
}

// NewVolume creates a Volume with identifiers, tags, physics settings, empty face storage, and a face tree.
func newVolume(dynamic bool, modelId int, id string, tag string, mass, restitution, friction float64, restCombine, frictionCombine int, gForce float64) *Volume {
	v := &Volume{
		modelId:    modelId,
		id:         id,
		tag:        tag,
		faces:      NewFaces(128, false),
		solver:     collisions.NewSolver(dynamic, mass, restitution, friction, restCombine, frictionCombine, gForce),
		vertexTags: make([]*articulated_dynamics.TagAxis, 0),
		dirty:      true,
		marker:     -1,
	}
	v.aabb = v.solver.GetEntity().GetAABB()
	return v
}

// NewVolumeDynamic creates a dynamic Volume with the given identifiers, tag, and physics settings.
func NewVolumeDynamic(modelId int, id string, tag string, mass, restitution, friction float64, restCombine, frictionCombine int, gForce float64) *Volume {
	return newVolume(true, modelId, id, tag, mass, restitution, friction, restCombine, frictionCombine, gForce)
}

// NewVolumeConcrete creates a static Volume with concrete material physics defaults and the given identifiers and tag.
func NewVolumeConcrete(modelId int, id string, tag string) *Volume {
	const concreteMass = 0.0
	const concreteRestitution = 0.0
	const concreteFriction = 0.2
	const concreteGForce = 9.8
	return newVolume(false, modelId, id, tag, concreteMass, concreteRestitution, concreteFriction, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), concreteGForce)
}

// Setup initializes the volume's collision solver and returns any setup error.
func (v *Volume) Setup(object collisions.IObject) error {
	if err := v.solver.Setup(object); err != nil {
		return err
	}
	return nil
}

// Rebuild updates all faces, refreshes the face tree and bounding entity, marks the volume dirty, and returns true.
func (v *Volume) Rebuild() bool {
	v.faces.Rebuild()

	minX, minY, minZ := v.faces.GetMin()
	w, h, d := v.faces.GetSize()
	cx, cy, cz := v.faces.GetCentroid()
	ixx, ixy, ixz, iyy, iyz, izz := v.faces.GetInertia()
	v.solver.Rebuild(minX, minY, minZ, w, h, d, cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz)

	v.dirty = true

	return true
}

// GetFacesTree returns the spatial tree containing the volume's faces.
func (v *Volume) GetFacesTree() *aabb.Tree {
	return v.faces.GetTree()
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

// GetSolver returns the collision solver associated with the volume.
func (v *Volume) GetSolver() *collisions.Solver {
	return v.solver
}

// GetAABB returns the axis-aligned bounding box of the volume's collision entity.
func (v *Volume) GetAABB() *aabb.AABB {
	return v.aabb
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

// GetObject returns the object associated with the volume.
func (v *Volume) GetObject() any {
	return v.object
}

// SetObject sets the sector object associated with the volume.
func (v *Volume) SetObject(s any) {
	v.object = s
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

// GetCentroid returns the volume centroid computed from the oriented triangular boundary
func (v *Volume) GetCentroid() (float64, float64, float64) {
	return v.faces.GetCentroid()
}

// QueryOverlaps searches the volume's face tree for objects whose AABB overlaps the given object's AABB.
func (v *Volume) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.faces.GetTree().QueryOverlaps(object, callback)
}

// QueryPoint calls callback for each face whose bounding box contains the given 3D point.
func (v *Volume) QueryPoint(px, py, pz float64, callback func(object aabb.IObject) bool) {
	v.faces.GetTree().QueryPoint3d(px, py, pz, callback)
}

// Deform applies a local plastic deformation to all volume faces within radius of impact.
func (v *Volume) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	deformed := false
	for i := 0; i < v.faces.Len(); i++ {
		face := v.faces.At(i)
		if face.Deform(impact, impulse, radius) {
			deformed = true
		}
	}
	if deformed {
		v.Rebuild()
	}
	return deformed
}

// DeformDent creates an indentation on the volume faces along the specified normal vector.
func (v *Volume) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	impulse := geometry.XYZ{
		X: -normal.X * depth,
		Y: -normal.Y * depth,
		Z: -normal.Z * depth,
	}
	status := v.Deform(impact, impulse, radius)
	if status {
		v.dirty = true
	}
	return status
}
