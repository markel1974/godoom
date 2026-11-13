package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Faces stores a collection of Face values and accumulated geometric properties.
type Faces struct {
	container     []*Face
	count         int
	cx, cy, cz    float64
	ixx, ixy, ixz float64
	iyy, iyz, izz float64
}

// NewFaces creates a Faces collection with the given initial capacity.
// If fill is true, the collection length is initialized to initSize.
func NewFaces(initSize int, fill bool) *Faces {
	count := 0
	if fill {
		count = initSize
	}
	return &Faces{
		container: make([]*Face, initSize),
		count:     count,
	}
}

// Rebuild recomputes cached centroid and inertia values from the current faces.
func (f *Faces) Rebuild() {
	f.cx, f.cy, f.cz, f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz = f.compileMassProperties()
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

// Len returns the number of faces in the collection.
func (f *Faces) Len() int {
	return f.count
}

// GetInertia returns the stored inertia tensor components for the faces.
func (f *Faces) GetInertia() (ixx, ixy, ixz, iyy, iyz, izz float64) {
	return f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz
}

// GetCentroid returns the cached centroid coordinates of the faces.
func (f *Faces) GetCentroid() (float64, float64, float64) {
	return f.cx, f.cy, f.cz
}

// compileMassProperties computes the centroid and unit-mass inertia tensor for the stored faces.
// It returns zero values when there are no faces, no valid face, or the signed volume is zero.
func (f *Faces) compileMassProperties() (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	if f.count == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	var cx, cy, cz float64
	var ixx, ixy, ixz float64
	var iyy, iyz, izz float64

	// Use a vertex as reference to improve numerical stability.
	var refX, refY, refZ float64
	found := false

	for i := 0; i < f.count; i++ {
		if face := f.container[i]; face != nil {
			refX, refY, refZ = face.GetPoint0()
			found = true
			break
		}
	}

	if !found {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	var volume6 float64
	// First moments.
	var mx, my, mz float64
	// Second moments matrix.
	var sxx, sxy, sxz float64
	var syy, syz, szz float64
	for i := 0; i < f.count; i++ {
		face := f.container[i]
		if face == nil {
			continue
		}
		volume6, mx, my, mz, sxx, sxy, sxz, syy, syz, szz = rigid_body.CompileMassAccumulate(refX, refY, refZ, face, volume6, mx, my, mz, sxx, sxy, sxz, syy, syz, szz)
	}
	if volume6 == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz = rigid_body.CompileMassFinalize(volume6, refX, refY, refZ, mx, my, mz, sxx, sxy, sxz, syy, syz, szz)
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

// GetMinMax returns the minimum and maximum X, Y, and Z coordinates across all non-nil faces.
func (f *Faces) GetMinMax() (float64, float64, float64, float64, float64, float64) {
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < f.count; x++ {
		face := f.container[x]
		if face == nil {
			continue
		}
		minX, minY, minZ, maxX, maxY, maxZ = face.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
	}
	return minX, minY, minZ, maxX, maxY, maxZ
}
