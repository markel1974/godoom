package physics

import (
	"math"
)

// IAABB represents an interface for objects that can provide their Axis-Aligned Bounding Box (AABB).
type IAABB interface {
	GetAABB() *AABB
}

// AABB represents an axis-aligned bounding box in 3D space, defined by its minimum and maximum coordinates.
type AABB struct {
	minX        float64
	minY        float64
	minZ        float64
	maxX        float64
	maxY        float64
	maxZ        float64
	surfaceArea float64
}

// NewAABB creates and initializes a new AABB instance with the provided minimum and maximum coordinates.
func NewAABB() *AABB {
	a := &AABB{}
	return a
}

// Rebuild updates the AABB's bounds and surface area using the provided minimum and maximum coordinates.
func (a *AABB) Rebuild(minX float64, minY float64, minZ float64, maxX float64, maxY float64, maxZ float64) {
	a.minX = minX
	a.minY = minY
	a.minZ = minZ
	a.maxX = maxX
	a.maxY = maxY
	a.maxZ = maxZ
	a.surfaceArea = 2.0 * (a.GetWidth()*a.GetHeight() + a.GetWidth()*a.GetDepth() + a.GetHeight()*a.GetDepth())
}

// Expand increases the size of the AABB by the given margin in all directions and returns a new expanded AABB.
func (a *AABB) Expand(margin float64) *AABB {
	minX, minY, minZ := a.minX-margin, a.minY-margin, a.minZ-margin
	maxX, maxY, maxZ := a.maxX+margin, a.maxY+margin, a.maxZ+margin
	out := NewAABB()
	out.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
	return out
}

// Merge combines the current AABB with another AABB to produce a new AABB that encapsulates both.
func (a *AABB) Merge(other *AABB) *AABB {
	minX := math.Min(a.minX, other.minX)
	minY := math.Min(a.minY, other.minY)
	minZ := math.Min(a.minZ, other.minZ)
	maxX := math.Max(a.maxX, other.maxX)
	maxY := math.Max(a.maxY, other.maxY)
	maxZ := math.Max(a.maxZ, other.maxZ)
	out := NewAABB()
	out.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
	return out
}

// Intersection returns a new AABB representing the overlapping region of the two AABBs or nil if there is no intersection.
func (a *AABB) Intersection(other *AABB) *AABB {
	minX := math.Max(a.minX, other.minX)
	minY := math.Max(a.minY, other.minY)
	minZ := math.Max(a.minZ, other.minZ)
	maxX := math.Min(a.maxX, other.maxX)
	maxY := math.Min(a.maxY, other.maxY)
	maxZ := math.Min(a.maxZ, other.maxZ)
	out := NewAABB()
	out.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
	return out
}

// Overlaps checks if the current AABB intersects with another AABB by comparing their bounds on all axes.
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

// ContainsPoint2d checks if the given 2D point (px, py) is inside the horizontal bounds of the AABB.
func (a *AABB) ContainsPoint2d(px, py float64) bool {
	return a.maxX >= px &&
		a.minX <= px &&
		a.maxY >= py &&
		a.minY <= py
}

// ContainsPoint3d checks if the 3D point (px, py, pz) lies within the bounds of the AABB.
func (a *AABB) ContainsPoint3d(px, py, pz float64) bool {
	return px >= a.minX && px <= a.maxX &&
		py >= a.minY && py <= a.maxY &&
		pz >= a.minZ && pz <= a.maxZ
}

// Contains checks if the other AABB is entirely contained within the current AABB.
func (a *AABB) Contains(other *AABB) bool {
	return other.minX >= a.minX &&
		other.maxX <= a.maxX &&
		other.minY >= a.minY &&
		other.maxY <= a.maxY &&
		other.minZ >= a.minZ &&
		other.maxZ <= a.maxZ
}

// GetWidth calculates and returns the width of the AABB along the x-axis.
func (a *AABB) GetWidth() float64 {
	return a.maxX - a.minX
}

// GetHeight returns the height of the AABB by calculating the difference between maxY and minY.
func (a *AABB) GetHeight() float64 {
	return a.maxY - a.minY
}

// GetMinY returns the minimum Y-coordinate of the axis-aligned bounding box (AABB).
func (a *AABB) GetMinY() float64 {
	return a.minY
}

// GetMaxY retrieves the maximum Y-coordinate (maxY) of the axis-aligned bounding box (AABB).
func (a *AABB) GetMaxY() float64 {
	return a.maxY
}

// GetMinX returns the minimum X-coordinate of the axis-aligned bounding box (AABB).
func (a *AABB) GetMinX() float64 {
	return a.minX
}

// GetMaxX returns the maximum X-coordinate (maxX) of the axis-aligned bounding box (AABB).
func (a *AABB) GetMaxX() float64 {
	return a.maxX
}

// GetMinZ returns the minimum Z-coordinate of the axis-aligned bounding box (AABB).
func (a *AABB) GetMinZ() float64 {
	return a.minZ
}

// GetMaxZ returns the maximum Z-coordinate (maxZ) of the axis-aligned bounding box (AABB).
func (a *AABB) GetMaxZ() float64 {
	return a.maxZ
}

// GetDepth calculates and returns the depth of the axis-aligned bounding box (AABB) along the Z-axis.
func (a *AABB) GetDepth() float64 {
	return a.maxZ - a.minZ
}

// IntersectRay tests if a ray intersects the AABB and returns the hit distance and a boolean indicating intersection success.
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

// IntersectFrustum controlla se l'AABB si trova all'interno (o interseca) il Frustum fornito.
// Restituisce true se l'AABB è visibile.
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
