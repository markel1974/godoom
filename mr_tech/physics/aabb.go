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
	return px >= a.minX &&
		px <= a.maxX &&
		py >= a.minY &&
		py <= a.maxY &&
		pz >= a.minZ &&
		pz <= a.maxZ
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
		// Find the "Positive Vertex" (the AABB vertex most aligned with the plane normal)
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
		// Calculate the distance of the P-Vertex from the plane (Dot Product)
		// If the distance is less than 0, the entire AABB is in the negative half-space (outside the frustum)
		if (plane.NormalX*pX + plane.NormalY*pY + plane.NormalZ*pZ + plane.D) < 0 {
			return false // Rejected!
		}
	}
	// If no plane has rejected it, the AABB is visible (at least partially)
	return true
}

// SweepAABB performs a Continuous Collision Detection (CCD) sweep of a moving AABB against a 3D triangle.
// It uses the Minkowski sum approach to expand the triangle by the AABB's half-extents.
// Returns:
// - tHit: Time of impact in range [0.0, 1.0]. Returns 1.0 if no collision occurs.
// - hitNormal: The normal vector to be used for sliding (matches the triangle normal).
// - hit: Boolean indicating if a collision occurred in this frame.
func (a *AABB) SweepAABB(vx, vy, vz float64, p0, p1, p2, normal Point) (float64, Point, bool) {
	cx := (a.minX + a.maxX) * 0.5
	cy := (a.minY + a.maxY) * 0.5
	cz := (a.minZ + a.maxZ) * 0.5
	ex := (a.maxX - a.minX) * 0.5
	ey := (a.maxY - a.minY) * 0.5
	ez := (a.maxZ - a.minZ) * 0.5
	// 2. Project the AABB onto the plane normal (Minkowski Radius)
	// This is the "thickness" that the plane acquires from the perspective of the AABB center
	r := ex*math.Abs(normal.x) + ey*math.Abs(normal.y) + ez*math.Abs(normal.z)
	// 3. Distance from the AABB center to the plane at time t=0
	distStart := (cx-p0.x)*normal.x + (cy-p0.y)*normal.y + (cz-p0.z)*normal.z
	// Projection of velocity onto the normal
	vDotN := vx*normal.x + vy*normal.y + vz*normal.z
	// 4. Directional broad-phase on the plane
	if math.Abs(vDotN) < 1e-8 {
		// Movement perfectly parallel to the plane.
		// If distStart <= r we would already be in penetration (handled by the static un-stuck routine),
		// but there is no frontal impact along V.
		return 1.0, normal, false
	}
	// 5. Calculate the intersection times (Time of Impact) with the upper and lower "crust" of the expanded plane
	t0 := (r - distStart) / vDotN
	t1 := (-r - distStart) / vDotN
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	// If the intersection occurs entirely in the past or future, no hit in this frame
	if t0 > 1.0 || t1 < 0.0 {
		return 1.0, normal, false
	}
	// Clamp tHit to 0.0 in case of slight pre-existing penetration
	tHit := t0
	if tHit < 0.0 {
		tHit = 0.0
	}
	// 6. SAT (Separating Axis Theorem) Testing: Does the impact point fall INSIDE the triangle?
	// Move the AABB center to the theoretical contact point on the plane
	hitCx := cx + vx*tHit
	hitCy := cy + vy*tHit
	hitCz := cz + vz*tHit
	// Inline Edge 1 (p0 -> p1)
	edge1X, edge1Y, edge1Z := p1.x-p0.x, p1.y-p0.y, p1.z-p0.z
	nx1 := edge1Y*normal.z - edge1Z*normal.y
	ny1 := edge1Z*normal.x - edge1X*normal.z
	nz1 := edge1X*normal.y - edge1Y*normal.x
	l1 := math.Sqrt(nx1*nx1 + ny1*ny1 + nz1*nz1)
	if l1 > 0 {
		nx1 /= l1
		ny1 /= l1
		nz1 /= l1
	}
	er1 := ex*math.Abs(nx1) + ey*math.Abs(ny1) + ez*math.Abs(nz1)
	if (hitCx-p0.x)*nx1+(hitCy-p0.y)*ny1+(hitCz-p0.z)*nz1 > er1 {
		return 1.0, normal, false
	}
	// Inline Edge 2 (p1 -> p2)
	edge2X, edge2Y, edge2Z := p2.x-p1.x, p2.y-p1.y, p2.z-p1.z
	nx2 := edge2Y*normal.z - edge2Z*normal.y
	ny2 := edge2Z*normal.x - edge2X*normal.z
	nz2 := edge2X*normal.y - edge2Y*normal.x
	l2 := math.Sqrt(nx2*nx2 + ny2*ny2 + nz2*nz2)
	if l2 > 0 {
		nx2 /= l2
		ny2 /= l2
		nz2 /= l2
	}
	er2 := ex*math.Abs(nx2) + ey*math.Abs(ny2) + ez*math.Abs(nz2)
	if (hitCx-p1.x)*nx2+(hitCy-p1.y)*ny2+(hitCz-p1.z)*nz2 > er2 {
		return 1.0, normal, false
	}
	// Inline Edge 3 (p2 -> p0)
	edge3X, edge3Y, edge3Z := p0.x-p2.x, p0.y-p2.y, p0.z-p2.z
	nx3 := edge3Y*normal.z - edge3Z*normal.y
	ny3 := edge3Z*normal.x - edge3X*normal.z
	nz3 := edge3X*normal.y - edge3Y*normal.x
	l3 := math.Sqrt(nx3*nx3 + ny3*ny3 + nz3*nz3)
	if l3 > 0 {
		nx3 /= l3
		ny3 /= l3
		nz3 /= l3
	}
	er3 := ex*math.Abs(nx3) + ey*math.Abs(ny3) + ez*math.Abs(nz3)
	if (hitCx-p2.x)*nx3+(hitCy-p2.y)*ny3+(hitCz-p2.z)*nz3 > er3 {
		return 1.0, normal, false
	}
	// return frontal hit.
	return tHit, normal, true
}
