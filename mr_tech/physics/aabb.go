package physics

import (
	"math"
)

// IAABB represents an interface defining objects that can provide an Axis-Aligned Bounding Box (AABB).
// GetAABB retrieves the AABB associated with the object implementing the IAABB interface.
type IAABB interface {
	GetAABB() *AABB
}

// AABB represents an axis-aligned bounding box in 3D space, defined by its minimum and maximum coordinates.
type AABB struct {
	minX float64
	minY float64
	minZ float64
	maxX float64
	maxY float64
	maxZ float64
}

// NewAABB creates and returns a new axis-aligned bounding box (AABB) with default uninitialized properties.
func NewAABB() *AABB {
	a := &AABB{}
	return a
}

// Rebuild recalculates and sets the bounds of the AABB using the specified minimum and maximum coordinates.
func (a *AABB) Rebuild(minX float64, minY float64, minZ float64, maxX float64, maxY float64, maxZ float64) {
	a.minX = minX
	a.minY = minY
	a.minZ = minZ
	a.maxX = maxX
	a.maxY = maxY
	a.maxZ = maxZ
}

// MergeInPlace updates the current AABB to encompass the union of the specified source and other AABBs.
func (a *AABB) MergeInPlace(src, other *AABB) {
	minX := math.Min(src.minX, other.minX)
	minY := math.Min(src.minY, other.minY)
	minZ := math.Min(src.minZ, other.minZ)
	maxX := math.Max(src.maxX, other.maxX)
	maxY := math.Max(src.maxY, other.maxY)
	maxZ := math.Max(src.maxZ, other.maxZ)
	a.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
}

// ExpandInPlace recalculates and updates the AABB by expanding it in all directions by the specified margin.
func (a *AABB) ExpandInPlace(src *AABB, margin float64) {
	minX := src.minX - margin
	minY := src.minY - margin
	minZ := src.minZ - margin
	maxX := src.maxX + margin
	maxY := src.maxY + margin
	maxZ := src.maxZ + margin
	a.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
}

// IntersectInPlace updates the AABB in place to represent the intersection volume of the source and other AABB.
func (a *AABB) IntersectInPlace(src, other *AABB) {
	minX := math.Max(src.minX, other.minX)
	minY := math.Max(src.minY, other.minY)
	minZ := math.Max(src.minZ, other.minZ)
	maxX := math.Min(src.maxX, other.maxX)
	maxY := math.Min(src.maxY, other.maxY)
	maxZ := math.Min(src.maxZ, other.maxZ)
	a.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
}

// Overlaps checks if the current AABB intersects with another AABB and returns true if an overlap exists.
func (a *AABB) Overlaps(other *AABB) bool {
	// y is deliberately first in the list of checks below as it is seen as more likely than things
	// collide on x,z but not on y than they do on y, thus we drop sooner on a y fail
	return a.maxX > other.minX &&
		a.minX < other.maxX &&
		a.maxY > other.minY &&
		a.minY < other.maxY &&
		a.maxZ > other.minZ &&
		a.minZ < other.maxZ
}

// ContainsPoint2d checks if the given 2D point (px, py) lies within the bounds of the AABB instance.
func (a *AABB) ContainsPoint2d(px, py float64) bool {
	return a.maxX >= px &&
		a.minX <= px &&
		a.maxY >= py &&
		a.minY <= py
}

// ContainsPoint3d checks if the AABB contains the specified 3D point (px, py, pz) and returns true if it does.
func (a *AABB) ContainsPoint3d(px, py, pz float64) bool {
	return px >= a.minX && px <= a.maxX &&
		py >= a.minY && py <= a.maxY &&
		pz >= a.minZ && pz <= a.maxZ
}

// Contains checks if the given AABB is fully enclosed within the boundaries of the current AABB.
func (a *AABB) Contains(other *AABB) bool {
	return other.minX >= a.minX &&
		other.maxX <= a.maxX &&
		other.minY >= a.minY &&
		other.maxY <= a.maxY &&
		other.minZ >= a.minZ &&
		other.maxZ <= a.maxZ
}

// GetWidth calculates and returns the width of the AABB by subtracting minX from maxX.
func (a *AABB) GetWidth() float64 {
	return a.maxX - a.minX
}

// GetHeight calculates and returns the height of the AABB by subtracting minY from maxY.
func (a *AABB) GetHeight() float64 {
	return a.maxY - a.minY
}

// GetMinY returns the minimum Y-coordinate (minY) of the AABB.
func (a *AABB) GetMinY() float64 {
	return a.minY
}

// GetMaxY returns the maximum Y-coordinate value (maxY) of the AABB instance.
func (a *AABB) GetMaxY() float64 {
	return a.maxY
}

// GetMinX returns the minimum X coordinate of the AABB.
func (a *AABB) GetMinX() float64 {
	return a.minX
}

// GetMaxX returns the maximum x-coordinate (maxX) of the AABB.
func (a *AABB) GetMaxX() float64 {
	return a.maxX
}

// GetSurfaceArea calculates and returns the total surface area of the axis-aligned bounding box (AABB).
func (a *AABB) GetSurfaceArea() float64 {
	w := a.maxX - a.minX
	h := a.maxY - a.minY
	d := a.maxZ - a.minZ
	out := 2.0 * ((w * h) + (w * d) + (h * d))
	return out
}

// GetSurfaceAreaMerged calculates the surface area of the bounding box formed by merging two AABBs.
func (a *AABB) GetSurfaceAreaMerged(other *AABB) float64 {
	minX := math.Min(a.minX, other.minX)
	minY := math.Min(a.minY, other.minY)
	minZ := math.Min(a.minZ, other.minZ)
	maxX := math.Max(a.maxX, other.maxX)
	maxY := math.Max(a.maxY, other.maxY)
	maxZ := math.Max(a.maxZ, other.maxZ)
	w := maxX - minX
	h := maxY - minY
	d := maxZ - minZ
	out := 2.0 * ((w * h) + (w * d) + (h * d))
	return out
}

// GetMinZ returns the minimum Z-coordinate of the AABB.
func (a *AABB) GetMinZ() float64 {
	return a.minZ
}

// GetMaxZ returns the maximum Z-coordinate of the axis-aligned bounding box.
func (a *AABB) GetMaxZ() float64 {
	return a.maxZ
}

// GetDepth calculates and returns the depth of the AABB as the difference between maxZ and minZ.
func (a *AABB) GetDepth() float64 {
	return a.maxZ - a.minZ
}

// IntersectRay checks if a ray intersects the AABB and calculates the intersection distance if applicable.
func (a *AABB) IntersectRay(oX, oY, oZ, invDirX, invDirY, invDirZ float64) (float64, bool) {
	t1 := (a.minX - oX) * invDirX
	t2 := (a.maxX - oX) * invDirX
	tMin := math.Min(t1, t2)
	tMax := math.Max(t1, t2)

	t1 = (a.minY - oY) * invDirY
	t2 = (a.maxY - oY) * invDirY
	tMin = math.Max(tMin, math.Min(t1, t2))
	tMax = math.Min(tMax, math.Max(t1, t2))

	t1 = (a.minZ - oZ) * invDirZ
	t2 = (a.maxZ - oZ) * invDirZ
	tMin = math.Max(tMin, math.Min(t1, t2))
	tMax = math.Min(tMax, math.Max(t1, t2))

	// Intersezione valida se tMax >= tMin e il volume è davanti al raggio (tMax >= 0)
	if tMax >= math.Max(tMin, 0.0) {
		return tMin, true
	}
	return 0.0, false
}

// IntersectFrustum determines whether the AABB intersects with or is partially contained within the given Frustum.
func (a *AABB) IntersectFrustum(f *Frustum) bool {
	for i := 0; i < 6; i++ {
		plane := f.Planes[i]
		// Troviamo il "Positive Vertex" (il vertice dell'AABB più allineato con la normale del piano)
		pX, pY, pZ := a.minX, a.minY, a.minZ
		if plane.NormalX > 0 {
			pX = a.maxX
		}
		if plane.NormalY > 0 {
			pY = a.maxY
		}
		if plane.NormalZ > 0 {
			pZ = a.maxZ
		}
		// Calcolo della distanza del P-Vertex dal piano (Dot Product)
		// Se la distanza è minore di 0, tutto l'AABB si trova nel semispazio negativo (fuori dal frustum)
		if (plane.NormalX*pX + plane.NormalY*pY + plane.NormalZ*pZ + plane.D) < 0 {
			return false // Scartato!
		}
	}
	// Se nessun piano lo ha scartato, l'AABB è visibile (almeno parzialmente)
	return true
}
