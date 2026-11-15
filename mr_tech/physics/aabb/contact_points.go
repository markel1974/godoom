package aabb

import (
	"math"
)

type ContactPoints struct {
	data        [MaxManifoldPoints]ContactPoint
	ptr         [MaxManifoldPoints]*ContactPoint
	penetration float64
}

func NewContactPoints() *ContactPoints {
	z := &ContactPoints{}
	for i := range z.ptr {
		z.ptr[i] = &z.data[i]
	}
	return z
}

func (cp *ContactPoints) Shift(count int, shiftX, shiftY, shiftZ float64) {
	for i := 0; i < count; i++ {
		p := cp.ptr[i]
		p.X += shiftX
		p.Y += shiftY
		p.Z += shiftZ
		if p.Penetration < cp.penetration {
			p.Penetration = cp.penetration
		}
	}
}

func (cp *ContactPoints) Set(idx int, x, y, z, penetration float64) {
	p := cp.ptr[idx]
	p.Set(x, y, z, penetration)
}

func (cp *ContactPoints) Copy(dstIdx, srcIdx int) {
	src := cp.ptr[srcIdx]
	dst := cp.ptr[dstIdx]
	dst.Set(src.X, src.Y, src.Z, src.Penetration)
}

func (cp *ContactPoints) CollectPlaneCandidates(poly *ClipPolygon, planeX, planeY, planeZ, planeNX, planeNY, planeNZ, shiftScale, maxDepth float64) int {
	count := 0
	const clipDedupSq = 1e-8
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
		duplicate := false
		for j := 0; j < count; j++ {
			target := cp.ptr[j]
			if pointDistSq(cx, cy, cz, target.X, target.Y, target.Z) < clipDedupSq {
				if depth > target.Penetration {
					target.Penetration = depth
				}
				duplicate = true
				break
			}
		}
		if !duplicate && count < len(cp.data) {
			cp.ptr[count].Set(cx, cy, cz, depth)
			count++
		}
	}
	return count
}

// ReduceManifoldPoints keeps the deepest candidate and spatially spreads the remaining manifold points.
func (cp *ContactPoints) ReduceManifoldPoints(count int) (int, int, int, int) {
	// Deepest vertex.
	i0 := 0
	for i := 1; i < count; i++ {
		if cp.ptr[i].Penetration > cp.ptr[i0].Penetration {
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
		if d := pointDistSq(cp.ptr[i].X, cp.ptr[i].Y, cp.ptr[i].Z, p0.X, p0.Y, p0.Z); d > maxD1 {
			maxD1 = d
			i1 = i
		}
	}
	p1 := cp.ptr[i1]

	// Largest triangle area with (p0, p1).
	i2 := 0
	maxArea := -1.0
	e01x, e01y, e01z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	for i := 0; i < count; i++ {
		if i == i0 || i == i1 {
			continue
		}
		vx, vy, vz := cp.ptr[i].X-p0.X, cp.ptr[i].Y-p0.Y, cp.ptr[i].Z-p0.Z
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
		d0 := pointDistSq(c.X, c.Y, c.Z, p0.X, p0.Y, p0.Z)
		d1 := pointDistSq(c.X, c.Y, c.Z, p1.X, p1.Y, p1.Z)
		d2 := pointDistSq(c.X, c.Y, c.Z, p2.X, p2.Y, p2.Z)
		minD := math.Min(d0, math.Min(d1, d2))
		if minD > maxMinD {
			maxMinD = minD
			i3 = i
		}
	}

	return i0, i1, i2, i3
}

func (cp *ContactPoints) Len() int {
	return len(cp.data)
}
