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

// NewAABB creates a new axis-aligned bounding box (AABB) with the specified min and max coordinates.
func NewAABB(minX float64, minY float64, minZ float64, maxX float64, maxY float64, maxZ float64) *AABB {
	a := &AABB{
		minX: minX,
		minY: minY,
		minZ: minZ,
		maxX: maxX,
		maxY: maxY,
		maxZ: maxZ,
	}
	a.surfaceArea = a.CalculateSurfaceArea()
	return a
}

// Expand increases the size of the AABB by the given margin in all directions and returns a new expanded AABB.
func (a *AABB) Expand(margin float64) *AABB {
	return NewAABB(
		a.minX-margin, a.minY-margin, a.minZ-margin,
		a.maxX+margin, a.maxY+margin, a.maxZ+margin,
	)
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

// QueryPoint checks if the given point (pMinX, pMinY) lies within the bounds of the AABB.
func (a *AABB) QueryPoint(pMinX, pMinY float64) bool {
	return a.maxX >= pMinX &&
		a.minX <= pMinX &&
		a.maxY >= pMinY &&
		a.minY <= pMinY
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

// Merge combines the current AABB with another AABB to produce a new AABB that encapsulates both.
func (a *AABB) Merge(other *AABB) *AABB {
	b := NewAABB(math.Min(a.minX, other.minX), math.Min(a.minY, other.minY), math.Min(a.minZ, other.minZ),
		math.Max(a.maxX, other.maxX), math.Max(a.maxY, other.maxY), math.Max(a.maxZ, other.maxZ))
	return b
}

// Intersection returns a new AABB representing the overlapping region of the two AABBs or nil if there is no intersection.
func (a *AABB) Intersection(other *AABB) *AABB {
	b := NewAABB(math.Max(a.minX, other.minX), math.Max(a.minY, other.minY), math.Max(a.minZ, other.minZ),
		math.Min(a.maxX, other.maxX), math.Min(a.maxY, other.maxY), math.Min(a.maxZ, other.maxZ))
	return b
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

// GetDepth calculates and returns the depth of the axis-aligned bounding box (AABB) along the Z-axis.
func (a *AABB) GetDepth() float64 {
	return a.maxZ - a.minZ
}

// CalculateSurfaceArea computes and returns the surface area of the axis-aligned bounding box (AABB).
func (a *AABB) CalculateSurfaceArea() float64 {
	s := 2.0 * (a.GetWidth()*a.GetHeight() + a.GetWidth()*a.GetDepth() + a.GetHeight()*a.GetDepth())
	return s
}
