package volumes

import (
	"math"
)

// Faces represents a collection of face entries.
type Faces struct {
	container  []*Face
	count      int
	cx, cy, cz float64
}

// NewFaces creates a Faces collection with storage preallocated to initSize.
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

func (f *Faces) Rebuild() {
	f.cx, f.cy, f.cz = f.compileCentroid()
}

// Add appends face to the collection, growing the internal storage when needed.
func (f *Faces) Add(face *Face) {
	if f.count >= len(f.container) {
		newFaces := make([]*Face, f.count*2)
		copy(newFaces, f.container)
		f.container = newFaces
	}
	f.container[f.count] = face
	f.count++
}

// Len returns the number of faces currently stored.
func (f *Faces) Len() int {
	return f.count
}

func (f *Faces) GetCentroid() (float64, float64, float64) {
	return f.cx, f.cy, f.cz
}

// GetCentroid returns the volume centroid computed from the oriented triangular boundary
func (f *Faces) compileCentroid() (float64, float64, float64) {
	if f.count == 0 {
		return 0, 0, 0
	}
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
		return 0, 0, 0
	}
	var volume6 float64
	var cx, cy, cz float64
	for i := 0; i < f.count; i++ {
		face := f.container[i]
		if face == nil {
			continue
		}
		point0X, point0Y, point0Z := face.GetPoint0()
		ax := point0X - refX
		ay := point0Y - refY
		az := point0Z - refZ
		point1X, point1Y, point1Z := face.GetPoint1()
		bx := point1X - refX
		by := point1Y - refY
		bz := point1Z - refZ
		point2X, point2Y, point2Z := face.GetPoint2()
		cxv := point2X - refX
		cyv := point2Y - refY
		czv := point2Z - refZ
		// Six times the signed tetrahedron volume.
		v6 := ax*(by*czv-bz*cyv) + ay*(bz*cxv-bx*czv) + az*(bx*cyv-by*cxv)
		volume6 += v6
		// Tetrahedron centroid relative to the reference point.
		cx += v6 * (ax + bx + cxv) * 0.25
		cy += v6 * (ay + by + cyv) * 0.25
		cz += v6 * (az + bz + czv) * 0.25
	}

	if volume6 == 0 {
		return 0, 0, 0
	}

	return refX + cx/volume6, refY + cy/volume6, refZ + cz/volume6
}

// Set assigns face to the element at idx.
func (f *Faces) Set(idx int, face *Face) {
	f.container[idx] = face
}

// At returns the face stored at the specified index.
func (f *Faces) At(idx int) *Face {
	return f.container[idx]
}

// Clear resets the face count to zero, leaving allocated storage available for reuse.
func (f *Faces) Clear() {
	f.count = 0
}

// GetMinMax returns the minimum and maximum x, y, and z coordinates across all non-nil faces.
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
