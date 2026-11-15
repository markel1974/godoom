package aabb

import (
	"math"
)

// ContactPoints stores a bounded set of collision contact candidates and their aggregate penetration state.
type ContactPoints struct {
	data        [MaxCandidatePoints]ContactPoint
	ptr         [MaxCandidatePoints]*ContactPoint
	penetration float64
	count       int
}

// NewContactPoints creates a ContactPoints value with its internal point pointers initialized.
// It returns an empty contact point collection ready for reuse.
func NewContactPoints() *ContactPoints {
	z := &ContactPoints{}
	for i := range z.ptr {
		z.ptr[i] = &z.data[i]
	}
	return z
}

// SetPenetration sets the aggregate penetration depth for the contact points.
func (cp *ContactPoints) SetPenetration(p float64) {
	cp.penetration = p
}

// SetCount sets the number of active contact points.
func (cp *ContactPoints) SetCount(c int) {
	cp.count = c
}

// GetPenetration returns the stored penetration depth for the contact points.
func (cp *ContactPoints) GetPenetration() float64 {
	return cp.penetration
}

// GetCount returns the number of active contact points.
func (cp *ContactPoints) GetCount() int {
	return cp.count
}

// Shift translates active contact points and raises any shallower penetration to the current penetration.
func (cp *ContactPoints) Shift(shiftX, shiftY, shiftZ float64) {
	for i := 0; i < cp.count; i++ {
		p := cp.ptr[i]
		p.x += shiftX
		p.y += shiftY
		p.z += shiftZ
		if p.penetration < cp.penetration {
			p.penetration = cp.penetration
		}
	}
}

// Finalize clamps the candidate count, records it, and moves the deepest penetration point to index 0.
func (cp *ContactPoints) Finalize(candCount int) {
	if candCount <= 0 {
		cp.count = 0
		return
	}
	if candCount > len(cp.ptr) {
		candCount = len(cp.ptr)
	}
	deepestIdx := 0
	for i := 1; i < candCount; i++ {
		if cp.ptr[i].penetration > cp.ptr[deepestIdx].penetration {
			deepestIdx = i
		}
	}
	cp.count = candCount
	if deepestIdx > 0 {
		deepPtr := cp.ptr[deepestIdx]
		for i := deepestIdx; i > 0; i-- {
			cp.ptr[i] = cp.ptr[i-1]
		}
		cp.ptr[0] = deepPtr
	}
}

// CopyFrom copies up to count contact points from src and updates the stored count.
// It clamps count to the valid range for both source and destination.
func (cp *ContactPoints) CopyFrom(src *ContactPoints, count int) {
	if count < 0 {
		count = 0
	}
	if count > len(cp.ptr) {
		count = len(cp.ptr)
	}
	if count > len(src.ptr) {
		count = len(src.ptr)
	}
	cp.count = count
	for i := 0; i < count; i++ {
		s := src.ptr[i]
		cp.ptr[i].Set(s.x, s.y, s.z, s.penetration)
	}
}

// SetSinglePoint sets this collection to a single contact point with the given position and penetration depth.
func (cp *ContactPoints) SetSinglePoint(x, y, z, penetration float64) {
	cp.count = 1
	cp.ptr[0].Set(x, y, z, penetration)
}

// SetIndex updates the contact point at idx with the given coordinates and penetration depth.
func (cp *ContactPoints) SetIndex(idx int, x, y, z, penetration float64) {
	p := cp.ptr[idx]
	p.Set(x, y, z, penetration)
}

// SetPenetrationIndex updates the penetration depth of the contact point at idx.
func (cp *ContactPoints) SetPenetrationIndex(idx int, penetration float64) {
	p := cp.ptr[idx]
	p.SetPenetration(penetration)
}

// GetIndex returns the x, y, z coordinates and penetration depth for the contact point at idx.
func (cp *ContactPoints) GetIndex(idx int) (float64, float64, float64, float64) {
	p := cp.ptr[idx]
	return p.Get()
}

// AssignIndex copies src coordinates and penetration into the contact point at dstIdx.
func (cp *ContactPoints) AssignIndex(dstIdx int, src *ContactPoint) {
	dst := cp.ptr[dstIdx]
	dst.Set(src.x, src.y, src.z, src.penetration)
}

// CopyIndex copies the contact point at srcIdx into dstIdx, including coordinates and penetration depth.
func (cp *ContactPoints) CopyIndex(dstIdx int, srcIdx int) {
	dst := cp.ptr[dstIdx]
	src := cp.ptr[srcIdx]
	dst.Set(src.x, src.y, src.z, src.penetration)
}

// CollectPlaneCandidates gathers penetrating polygon vertices projected along the plane normal as contact candidates.
// It clamps stored depth to maxDepth when positive, removes duplicates, and returns the candidate count.
func (cp *ContactPoints) CollectPlaneCandidates(poly *ClipPolygon, planeX, planeY, planeZ, planeNX, planeNY, planeNZ, shiftScale, maxDepth float64) int {
	count := 0
	for i := 0; i < poly.count; i++ {
		px, py, pz := poly.Get(i)
		rawDepth := (planeX-px)*planeNX + (planeY-py)*planeNY + (planeZ-pz)*planeNZ
		if rawDepth <= 0.0 {
			continue
		}
		depth := rawDepth
		if maxDepth > 0.0 && depth > maxDepth {
			depth = maxDepth
		}
		shift := shiftScale * rawDepth
		cx := px + planeNX*shift
		cy := py + planeNY*shift
		cz := pz + planeNZ*shift
		count = cp.RemoveDuplicate(count, depth, cx, cy, cz)
	}
	return count
}

// Reduce selects four representative contact points from the first count candidates.
// It keeps the deepest point, then chooses remaining points to maximize spatial spread.
func (cp *ContactPoints) Reduce(count int) (*ContactPoint, *ContactPoint, *ContactPoint, *ContactPoint) {
	// Deepest vertex.
	i0 := 0
	for i := 1; i < count; i++ {
		if cp.ptr[i].penetration > cp.ptr[i0].penetration {
			i0 = i
		}
	}
	p0 := cp.ptr[i0]

	// Farthest from p0.
	i1 := 0
	maxD1 := -1.0
	for i := 0; i < count; i++ {
		if i == i0 {
			continue
		}
		if d := pointDistSq(cp.ptr[i].x, cp.ptr[i].y, cp.ptr[i].z, p0.x, p0.y, p0.z); d > maxD1 {
			maxD1 = d
			i1 = i
		}
	}
	p1 := cp.ptr[i1]

	// Largest triangle area with (p0, p1).
	i2 := 0
	maxArea := -1.0
	e01x, e01y, e01z := p1.x-p0.x, p1.y-p0.y, p1.z-p0.z
	for i := 0; i < count; i++ {
		if i == i0 || i == i1 {
			continue
		}
		vx, vy, vz := cp.ptr[i].x-p0.x, cp.ptr[i].y-p0.y, cp.ptr[i].z-p0.z
		cx := e01y*vz - e01z*vy
		cy := e01z*vx - e01x*vz
		cz := e01x*vy - e01y*vx
		if areaSq := cx*cx + cy*cy + cz*cz; areaSq > maxArea {
			maxArea = areaSq
			i2 = i
		}
	}
	p2 := cp.ptr[i2]

	// Widest spread from existing triangle {p0, p1, p2}.
	i3 := 0
	maxMinD := -1.0
	for i := 0; i < count; i++ {
		if i == i0 || i == i1 || i == i2 {
			continue
		}
		c := cp.ptr[i]
		d0 := pointDistSq(c.x, c.y, c.z, p0.x, p0.y, p0.z)
		d1 := pointDistSq(c.x, c.y, c.z, p1.x, p1.y, p1.z)
		d2 := pointDistSq(c.x, c.y, c.z, p2.x, p2.y, p2.z)
		minD := math.Min(d0, math.Min(d1, d2))
		if minD > maxMinD {
			maxMinD = minD
			i3 = i
		}
	}

	return cp.ptr[i0], cp.ptr[i1], cp.ptr[i2], cp.ptr[i3]
}

// ReduceInto selects MaxManifoldPoints well-spread representative candidates from cp and writes them into dst.
func (cp *ContactPoints) ReduceInto(dst *ContactPoints, count int) {
	i0, i1, i2, i3 := cp.Reduce(count)
	dst.SetCount(MaxManifoldPoints)
	dst.AssignIndex(0, i0)
	dst.AssignIndex(1, i1)
	dst.AssignIndex(2, i2)
	dst.AssignIndex(3, i3)
}

// FinalizeInto writes up to MaxManifoldPoints finalized candidates from cp into dst, falling back to a single point when empty.
func (cp *ContactPoints) FinalizeInto(dst *ContactPoints, count int, fallbackX, fallbackY, fallbackZ, fallbackPen float64) {
	if count <= 0 {
		dst.SetSinglePoint(fallbackX, fallbackY, fallbackZ, fallbackPen)
		return
	}
	if count <= MaxManifoldPoints {
		cp.Finalize(count)
		dst.CopyFrom(cp, count)
		return
	}
	cp.ReduceInto(dst, count)
}

// RemoveDuplicate adds a contact point unless it matches an existing one, updating duplicate penetration if deeper.
// It returns the resulting number of stored candidate contact points.
func (cp *ContactPoints) RemoveDuplicate(count int, depth float64, cx, cy, cz float64) int {
	const clipDedupSq = 1e-8
	duplicate := false
	for j := 0; j < count; j++ {
		ptr := cp.ptr[j]
		if pointDistSq(cx, cy, cz, ptr.x, ptr.y, ptr.z) < clipDedupSq {
			if depth > ptr.penetration {
				ptr.penetration = depth
			}
			duplicate = true
			break
		}
	}
	if !duplicate && count < len(cp.ptr) {
		cp.ptr[count].Set(cx, cy, cz, depth)
		count++
	}
	return count
}

// Len returns the fixed storage capacity for contact points.
func (cp *ContactPoints) Len() int {
	return len(cp.data)
}
