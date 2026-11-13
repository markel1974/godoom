package volumes

import (
	"math"
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

		x0, y0, z0 := face.GetPoint0()
		x1, y1, z1 := face.GetPoint1()
		x2, y2, z2 := face.GetPoint2()

		// Vertices relative to the reference point.
		ax, ay, az := x0-refX, y0-refY, z0-refZ
		bx, by, bz := x1-refX, y1-refY, z1-refZ
		dx, dy, dz := x2-refX, y2-refY, z2-refZ

		// Six times the signed tetrahedron volume.
		v6 := ax*(by*dz-bz*dy) +
			ay*(bz*dx-bx*dz) +
			az*(bx*dy-by*dx)

		volume6 += v6

		// Sum of the three non-reference vertices.
		sx := ax + bx + dx
		sy := ay + by + dy
		sz := az + bz + dz

		// First moments.
		mx += v6 * sx * 0.25
		my += v6 * sy * 0.25
		mz += v6 * sz * 0.25

		// Second moments:
		//
		// S = V/20 * (sum(v) * sum(v)^T + sum(v * v^T))
		//
		// Since V = v6/6, the coefficient is v6/120.

		k := v6 / 120.0

		sxx += k * (sx*sx + ax*ax + bx*bx + dx*dx)
		sxy += k * (sx*sy + ax*ay + bx*by + dx*dy)
		sxz += k * (sx*sz + ax*az + bx*bz + dx*dz)

		syy += k * (sy*sy + ay*ay + by*by + dy*dy)
		syz += k * (sy*sz + ay*az + by*bz + dy*dz)

		szz += k * (sz*sz + az*az + bz*bz + dz*dz)
	}

	if volume6 == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	// Centroid relative to the reference point.
	rx := mx / volume6
	ry := my / volume6
	rz := mz / volume6

	cx = refX + rx
	cy = refY + ry
	cz = refZ + rz

	// Signed volume.
	volume := volume6 / 6.0

	// Translate second moments to the centroid.
	sxx -= volume * rx * rx
	sxy -= volume * rx * ry
	sxz -= volume * rx * rz

	syy -= volume * ry * ry
	syz -= volume * ry * rz

	szz -= volume * rz * rz

	// Inertia tensor per unit mass:
	//
	// I = (trace(S) * Identity - S) / volume

	factor := 6.0 / volume6

	ixx = (syy + szz) * factor
	iyy = (sxx + szz) * factor
	izz = (sxx + syy) * factor

	ixy = -sxy * factor
	ixz = -sxz * factor
	iyz = -syz * factor

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
