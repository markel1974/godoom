package volumes

import (
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

var globalVolumeDirtyEpoch atomic.Uint64

// GlobalDirtyEpoch returns a monotonic counter incremented whenever any volume geometry rebuilds or becomes dirty.
func GlobalDirtyEpoch() uint64 {
	return globalVolumeDirtyEpoch.Load()
}

const (
	treeInitSize = 1
)

// Volume represents a spatial model element that groups geometry, lighting, physics, and sector data.
type Volume struct {
	dynamic    bool
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
	deformable bool
	dirty      bool
}

func newVolumeWithFaces(dynamic bool, modelId int, id string, tag string, faces *Faces, solver *collisions.Solver) *Volume {
	v := &Volume{
		dynamic:    dynamic,
		modelId:    modelId,
		id:         id,
		tag:        tag,
		faces:      faces,
		solver:     solver,
		vertexTags: make([]*articulated_dynamics.TagAxis, 0),
		dirty:      true,
		marker:     -1,
		deformable: true,
	}
	v.aabb = solver.GetEntity().GetAABB()
	return v
}

// newVolume creates and returns a pointer to a new Volume instance with the provided parameters initialized.
// The Volume instance is configured with properties such as dynamic state, model ID, identifier, tag, and associated AABB tree.
// It initializes collision handling via the provided solver and sets default values for various internal properties.
func newVolume(dynamic bool, modelId int, id string, tag string, tree *aabb.Tree, solver *collisions.Solver) *Volume {
	return newVolumeWithFaces(dynamic, modelId, id, tag, NewFaces(tree, false), solver)
}

// NewVolumeDynamic creates a dynamic Volume with the given identifiers, tag, and physics settings.
func NewVolumeDynamic(modelId int, id string, tag string, mass, restitution, friction float64, restCombine, frictionCombine int, gForce float64) *Volume {
	tree := aabb.NewTree(uint(treeInitSize), 0.0)
	solver := collisions.NewSolver(true, mass, restitution, friction, restCombine, frictionCombine, gForce)
	v := newVolume(true, modelId, id, tag, tree, solver)
	v.SetAllowTessellate(false)
	return v
}

// NewVolumeDynamicShared creates a dynamic volume instance without attaching a collision solver. It disables tessellation.
func NewVolumeDynamicShared(modelId int, id string, tag string, tree *aabb.Tree, solver *collisions.Solver) *Volume {
	v := newVolume(true, modelId, id, tag, tree, solver)
	v.SetAllowTessellate(false)
	return v
}

// NewVolumeDynamicSharedSized creates a dynamic volume backed by a pre-sized Faces container without duplicating capacity.
func NewVolumeDynamicSharedSized(modelId int, id string, tag string, tree *aabb.Tree, solver *collisions.Solver, faceCap int) *Volume {
	v := newVolumeWithFaces(true, modelId, id, tag, NewFacesSized(tree, faceCap, false), solver)
	v.SetAllowTessellate(false)
	return v
}

// NewVolumeConcrete creates a static Volume with concrete material physics defaults and the given identifiers and tag.
func NewVolumeConcrete(modelId int, id string, tag string) *Volume {
	const concreteMass = 0.0
	const concreteRestitution = 0.0
	const concreteFriction = 0.2
	const concreteGForce = 9.8
	tree := aabb.NewTree(uint(treeInitSize), 0.0)
	solver := collisions.NewSolver(false, concreteMass, concreteRestitution, concreteFriction, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), concreteGForce)
	v := newVolume(false, modelId, id, tag, tree, solver)
	return v
}

// Setup initializes the volume's collision solver and returns any setup error.
func (v *Volume) Setup(object aabb.IObject) error {
	if err := v.solver.Setup(object); err != nil {
		return err
	}
	return nil
}

// Rebuild updates all faces, refreshes the face tree and bounding entity, marks the volume dirty, and returns true.
func (v *Volume) Rebuild() bool {
	v.faces.Rebuild()
	v.Sync()
	return true
}

// RebuildGeometry updates face geometry, cached bounds, and physics properties without modifying the spatial tree.
func (v *Volume) RebuildGeometry() bool {
	v.faces.RebuildGeometry()
	v.Sync()
	return true
}

// RebuildTree rebuilds the volume's face acceleration tree and reports success.
func (v *Volume) RebuildTree() bool {
	v.faces.RebuildTree()
	return true
}

// Sync updates the collision solver from face data and marks the volume as dirty.
func (v *Volume) Sync() {
	if v.solver != nil {
		minX, minY, minZ := v.faces.GetMin()
		w, h, d := v.faces.GetSize()
		cx, cy, cz := v.faces.GetCentroid()
		ixx, ixy, ixz, iyy, iyz, izz := v.faces.GetInertia()
		v.solver.Rebuild(minX, minY, minZ, w, h, d, cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz)
	}
	v.dirty = true
	globalVolumeDirtyEpoch.Add(1)
}

// IsDeformable reports whether the volume can undergo deformation.
func (v *Volume) IsDeformable() bool {
	return v.deformable
}

// IsDynamic reports whether the volume is configured for dynamic physics.
func (v *Volume) IsDynamic() bool {
	return v.dynamic
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
	if dirty {
		globalVolumeDirtyEpoch.Add(1)
	}
}

// ClearDirty clears the dirty flag.
func (v *Volume) ClearDirty() {
	v.dirty = false
}

// GetHash returns a deterministic state hash owned by this volume, or 0 if empty.
func (v *Volume) GetHash() uint64 {
	if v == nil || v.faces == nil || v.faces.Len() == 0 {
		return 0
	}
	h := v.faces.GetRenderHash()
	if h == 0 {
		return 0
	}
	if v.dirty {
		h = geometry.MixU64(h, 0x9e3779b97f4a7c15)
		if h == 0 {
			return 1
		}
	}
	return h
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

// SetSolver attaches solver to the volume and syncs its AABB pointer.
func (v *Volume) SetSolver(solver *collisions.Solver) {
	if solver == nil {
		panic("solver can't be nil")
	}
	v.solver = solver
	v.aabb = solver.GetEntity().GetAABB()
}

// GetAABB returns the axis-aligned bounding box of the volume's collision entity.
func (v *Volume) GetAABB() *aabb.AABB {
	return v.aabb
}

// GetModelId returns the model identifier assigned to the volume.
func (v *Volume) GetModelId() int {
	return v.modelId
}

// SetModelId updates the model frame identifier assigned to the volume.
func (v *Volume) SetModelId(modelId int) {
	v.modelId = modelId
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

// SetAllowTessellate enables or disables dynamic tessellation on the volume's faces.
func (v *Volume) SetAllowTessellate(allow bool) {
	v.faces.SetAllowTessellate(allow)
}

// IsAllowTessellate reports whether dynamic tessellation is enabled on the volume's faces.
func (v *Volume) IsAllowTessellate() bool {
	return v.faces.IsAllowTessellate()
}

// QueryPoint calls callback for each face whose bounding box contains the given 3D point.
func (v *Volume) QueryPoint(px, py, pz float64, callback func(object aabb.IObject) bool) {
	v.faces.GetTree().QueryPoint3d(px, py, pz, callback)
}

// Deform applies an impulse around impact within radius, syncs the volume, and reports whether it changed.
func (v *Volume) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	if radius <= 0 {
		return false
	}
	if !v.faces.Deform(impact, impulse, radius) {
		return false
	}
	v.Sync()
	return true
}

// SyncDeformedTree updates the face acceleration tree after deformation changes.
func (v *Volume) SyncDeformedTree() {
	v.faces.SyncDeformedTree()
}
