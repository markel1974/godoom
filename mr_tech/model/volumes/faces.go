package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Faces stores a collection of Face values and accumulated geometric properties.
type Faces struct {
	tree             *aabb.Tree
	container        []*Face
	count            int
	cx, cy, cz       float64
	ixx, ixy, ixz    float64
	iyy, iyz, izz    float64
	minX, minY, minZ float64
	maxX, maxY, maxZ float64
	w, h, d          float64
}

// NewFaces creates a Faces collection with the given initial capacity.
// If fill is true, the collection length is initialized to initSize.
func NewFaces(initSize int, fill bool) *Faces {
	count := 0
	if fill {
		count = initSize
	}
	return &Faces{
		tree:      aabb.NewTree(64, 0.0),
		container: make([]*Face, initSize),
		count:     count,
	}
}

// Rebuild recomputes cached centroid and inertia values from the current faces.
func (f *Faces) Rebuild() {
	f.tree.Clear()
	if f.count == 0 {
		f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ = 0, 0, 0, 0, 0, 0
		f.w, f.h, f.d = 0, 0, 0
		f.cx, f.cy, f.cz = 0, 0, 0
		f.ixy, f.ixz, f.iyy, f.iyz, f.izz = 0, 0, 0, 0, 0
		return
	}
	f.minX, f.minY, f.minZ = math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	f.maxX, f.maxY, f.maxZ = -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < f.count; x++ {
		face := f.container[x]
		face.Rebuild()
		f.tree.InsertObject(face)
		f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ = face.GetMinMax(f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ)
	}
	f.cx, f.cy, f.cz, f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz = f.ComputeMassProperties()
	f.w = f.maxX - f.minX
	f.h = f.maxY - f.minY
	f.d = f.maxZ - f.minZ
}

// GetTree returns the AABB tree containing the faces.
func (f *Faces) GetTree() *aabb.Tree {
	return f.tree
}

// GetMin returns the cached minimum X, Y, and Z coordinates of the faces' bounding box.
func (f *Faces) GetMin() (float64, float64, float64) {
	return f.minX, f.minY, f.minZ
}

// GetMax returns the cached maximum X, Y, and Z coordinates of the faces' bounding box.
func (f *Faces) GetMax() (float64, float64, float64) {
	return f.maxX, f.maxY, f.maxZ
}

// GetMinMax returns the cached minimum and maximum X, Y, and Z coordinates.
func (f *Faces) GetMinMax() (float64, float64, float64, float64, float64, float64) {
	return f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ
}

// GetSize returns the cached width, height, and depth of the faces' bounding box.
func (f *Faces) GetSize() (float64, float64, float64) {
	return f.w, f.h, f.d
}

// GetInertia returns the stored inertia tensor components for the faces.
func (f *Faces) GetInertia() (ixx, ixy, ixz, iyy, iyz, izz float64) {
	return f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz
}

// GetCentroid returns the cached centroid coordinates of the faces.
func (f *Faces) GetCentroid() (float64, float64, float64) {
	return f.cx, f.cy, f.cz
}

// Len returns the number of faces in the collection.
func (f *Faces) Len() int {
	return f.count
}

// Add appends a face to the collection, growing the internal storage when needed.
func (f *Faces) Add(face *Face) {
	if f.count >= len(f.container) {
		newFaces := make([]*Face, f.count*2)
		copy(newFaces, f.container)
		f.container = newFaces
	}
	f.container[f.count] = face
	f.count++
}

// ComputeMassProperties computes the centroid and unit-mass inertia tensor for the stored faces.
// It returns zero values when there are no faces, no valid face, or the signed volume is zero.
func (f *Faces) ComputeMassProperties() (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	if f.count == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	// Use a vertex as reference to improve numerical stability.
	refX, refY, refZ := f.container[0].GetPoint0()
	var volume6 float64
	// First moments.
	var mx, my, mz float64
	// Second moments matrix.
	var sxx, sxy, sxz float64
	var syy, syz, szz float64
	for i := 0; i < f.count; i++ {
		face := f.container[i]
		volume6, mx, my, mz, sxx, sxy, sxz, syy, syz, szz = rigid_body.CompileMassAccumulate(refX, refY, refZ, face, volume6, mx, my, mz, sxx, sxy, sxz, syy, syz, szz)
	}
	if volume6 == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz := rigid_body.CompileMassFinalize(volume6, refX, refY, refZ, mx, my, mz, sxx, sxy, sxz, syy, syz, szz)
	return cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz
}

// Set assigns face to the element at idx.
func (f *Faces) Set(idx int, face *Face) {
	f.container[idx] = face
}

// At returns the face stored at idx.
func (f *Faces) At(idx int) *Face {
	return f.container[idx]
}

// Clear removes all faces from the collection.
func (f *Faces) Clear() {
	f.count = 0
}
