package collisions

import "math"

// obbPrismEpsilon is the tolerance used for OBB-prism bounds and half-space checks.
const obbPrismEpsilon = 1e-9

// TrianglePrism stores reusable geometry for testing an oriented box against a triangular face prism.
type TrianglePrism struct {
	hx, hy, hz    float64
	lnx, lny, lnz float64
	cz            float64
	az0, az1, az2 float64
	p0x, p0y, p0z float64
	px, py, pz    [4]float64
	mx, my, mz    [4]float64
	ex, ey, ez    [3]float64
}

// MergeContact tests a local point against the prism and merges its contact depth and lowest world Z.
// It returns the updated best depth, minimum world Z, and whether any contact has been found.
func (pr *TrianglePrism) MergeContact(bestDepth, minWorldZ float64, found bool, qx, qy, qz float64) (float64, float64, bool) {
	if math.Abs(qx) > pr.hx+obbPrismEpsilon || math.Abs(qy) > pr.hy+obbPrismEpsilon || math.Abs(qz) > pr.hz+obbPrismEpsilon {
		return bestDepth, minWorldZ, found
	}
	for i := range 4 {
		if (qx-pr.px[i])*pr.mx[i]+(qy-pr.py[i])*pr.my[i]+(qz-pr.pz[i])*pr.mz[i] > obbPrismEpsilon {
			return bestDepth, minWorldZ, found
		}
	}
	depth := (pr.p0x-qx)*pr.lnx + (pr.p0y-qy)*pr.lny + (pr.p0z-qz)*pr.lnz
	wz := pr.cz + qx*pr.az0 + qy*pr.az1 + qz*pr.az2
	if !found {
		return depth, wz, true
	}
	if depth > bestDepth {
		bestDepth = depth
	}
	if wz < minWorldZ {
		minWorldZ = wz
	}
	return bestDepth, minWorldZ, true
}

// unitPrismHalfSpaceNormal returns the unit normal for a prism side half-space from an edge and face normal.
// It returns false when the cross product is too small to normalize.
func (pr *TrianglePrism) unitPrismHalfSpaceNormal(ex, ey, ez, nx, ny, nz float64) (float64, float64, float64, bool) {
	mx := ey*nz - ez*ny
	my := ez*nx - ex*nz
	mz := ex*ny - ey*nx
	lenSq := mx*mx + my*my + mz*mz
	if lenSq <= 1e-24 {
		return 0.0, 0.0, 0.0, false
	}
	inv := 1.0 / math.Sqrt(lenSq)
	return mx * inv, my * inv, mz * inv, true
}

// updateRaySlabRange clips a ray parameter interval against one centered slab and reports whether it remains valid.
func (pr *TrianglePrism) updateRaySlabRange(origin, dir, halfExtent, tMin, tMax float64) (float64, float64, bool) {
	if math.Abs(dir) <= 1e-12 {
		if math.Abs(origin) > halfExtent+obbPrismEpsilon {
			return tMin, tMax, false
		}
		return tMin, tMax, true
	}
	inv := 1.0 / dir
	t1 := (-halfExtent - origin) * inv
	t2 := (halfExtent - origin) * inv
	if t1 > t2 {
		t1, t2 = t2, t1
	}
	if t1 > tMin {
		tMin = t1
	}
	if t2 < tMax {
		tMax = t2
	}
	return tMin, tMax, tMin <= tMax
}

// Penetration computes the deepest overlap of a cage with a triangular face prism.
// It returns penetration depth, lowest contact world Z, and whether a contact was found.
func (pr *TrianglePrism) Penetration(s *Cage, rFace IFace, cx, cy, cz, nx, ny, nz float64) (float64, float64, bool) {
	hx, hy, hz := s.entity.GetSizeCenter()
	if hx <= 0.0 || hy <= 0.0 || hz <= 0.0 {
		return 0.0, 0.0, false
	}
	ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := s.entity.GetAxes()
	v0x, v0y, v0z := rFace.Get0()
	v1x, v1y, v1z := rFace.Get1()
	v2x, v2y, v2z := rFace.Get2()

	d0x, d0y, d0z := v0x-cx, v0y-cy, v0z-cz
	d1x, d1y, d1z := v1x-cx, v1y-cy, v1z-cz
	d2x, d2y, d2z := v2x-cx, v2y-cy, v2z-cz

	p0x, p0y, p0z := d0x*ax0+d0y*ay0+d0z*az0, d0x*ax1+d0y*ay1+d0z*az1, d0x*ax2+d0y*ay2+d0z*az2
	p1x, p1y, p1z := d1x*ax0+d1y*ay0+d1z*az0, d1x*ax1+d1y*ay1+d1z*az1, d1x*ax2+d1y*ay2+d1z*az2
	p2x, p2y, p2z := d2x*ax0+d2y*ay0+d2z*az0, d2x*ax1+d2y*ay1+d2z*az1, d2x*ax2+d2y*ay2+d2z*az2

	lnx := nx*ax0 + ny*ay0 + nz*az0
	lny := nx*ax1 + ny*ay1 + nz*az1
	lnz := nx*ax2 + ny*ay2 + nz*az2

	e0x, e0y, e0z := p1x-p0x, p1y-p0y, p1z-p0z
	e1x, e1y, e1z := p2x-p1x, p2y-p1y, p2z-p1z
	e2x, e2y, e2z := p0x-p2x, p0y-p2y, p0z-p2z

	effNX, effNY, effNZ := lnx, lny, lnz
	tnx, tny, tnz := e0y*e1z-e0z*e1y, e0z*e1x-e0x*e1z, e0x*e1y-e0y*e1x
	if tnx*lnx+tny*lny+tnz*lnz < 0.0 {
		effNX, effNY, effNZ = -lnx, -lny, -lnz
	}

	m0x, m0y, m0z, ok0 := pr.unitPrismHalfSpaceNormal(e0x, e0y, e0z, effNX, effNY, effNZ)
	m1x, m1y, m1z, ok1 := pr.unitPrismHalfSpaceNormal(e1x, e1y, e1z, effNX, effNY, effNZ)
	m2x, m2y, m2z, ok2 := pr.unitPrismHalfSpaceNormal(e2x, e2y, e2z, effNX, effNY, effNZ)
	if !ok0 || !ok1 || !ok2 {
		return 0.0, 0.0, false
	}

	pr.hx, pr.hy, pr.hz = hx, hy, hz
	pr.lnx, pr.lny, pr.lnz = lnx, lny, lnz
	pr.cz, pr.az0, pr.az1, pr.az2 = cz, az0, az1, az2
	pr.p0x, pr.p0y, pr.p0z = p0x, p0y, p0z
	pr.px[0], pr.px[1], pr.px[2], pr.px[3] = p0x, p1x, p2x, p0x
	pr.py[0], pr.py[1], pr.py[2], pr.py[3] = p0y, p1y, p2y, p0y
	pr.pz[0], pr.pz[1], pr.pz[2], pr.pz[3] = p0z, p1z, p2z, p0z
	pr.mx[0], pr.mx[1], pr.mx[2], pr.mx[3] = m0x, m1x, m2x, lnx
	pr.my[0], pr.my[1], pr.my[2], pr.my[3] = m0y, m1y, m2y, lny
	pr.mz[0], pr.mz[1], pr.mz[2], pr.mz[3] = m0z, m1z, m2z, lnz
	pr.ex[0], pr.ex[1], pr.ex[2] = e0x, e1x, e2x
	pr.ey[0], pr.ey[1], pr.ey[2] = e0y, e1y, e2y
	pr.ez[0], pr.ez[1], pr.ez[2] = e0z, e1z, e2z

	best := 0.0
	minWZ := 0.0
	found := false
	for sx := -1.0; sx <= 1.0; sx += 2.0 {
		for sy := -1.0; sy <= 1.0; sy += 2.0 {
			for sz := -1.0; sz <= 1.0; sz += 2.0 {
				best, minWZ, found = pr.MergeContact(best, minWZ, found, sx*hx, sy*hy, sz*hz)
			}
		}
	}
	for k := range 4 {
		for s1 := -1.0; s1 <= 1.0; s1 += 2.0 {
			for s2 := -1.0; s2 <= 1.0; s2 += 2.0 {
				if math.Abs(pr.mx[k]) > 1e-12 {
					qy, qz := s1*hy, s2*hz
					qx := pr.px[k] - ((qy-pr.py[k])*pr.my[k]+(qz-pr.pz[k])*pr.mz[k])/pr.mx[k]
					best, minWZ, found = pr.MergeContact(best, minWZ, found, qx, qy, qz)
				}
				if math.Abs(pr.my[k]) > 1e-12 {
					qx, qz := s1*hx, s2*hz
					qy := pr.py[k] - ((qx-pr.px[k])*pr.mx[k]+(qz-pr.pz[k])*pr.mz[k])/pr.my[k]
					best, minWZ, found = pr.MergeContact(best, minWZ, found, qx, qy, qz)
				}
				if math.Abs(pr.mz[k]) > 1e-12 {
					qx, qy := s1*hx, s2*hy
					qz := pr.pz[k] - ((qx-pr.px[k])*pr.mx[k]+(qy-pr.py[k])*pr.my[k])/pr.mz[k]
					best, minWZ, found = pr.MergeContact(best, minWZ, found, qx, qy, qz)
				}
			}
		}
	}
	for k := range 3 {
		tMin, tMax, ok := pr.updateRaySlabRange(pr.px[k], -lnx, hx, 0.0, math.MaxFloat64)
		if ok {
			tMin, tMax, ok = pr.updateRaySlabRange(pr.py[k], -lny, hy, tMin, tMax)
		}
		if ok {
			tMin, tMax, ok = pr.updateRaySlabRange(pr.pz[k], -lnz, hz, tMin, tMax)
		}
		if ok {
			best, minWZ, found = pr.MergeContact(best, minWZ, found, pr.px[k]-tMin*lnx, pr.py[k]-tMin*lny, pr.pz[k]-tMin*lnz)
			best, minWZ, found = pr.MergeContact(best, minWZ, found, pr.px[k]-tMax*lnx, pr.py[k]-tMax*lny, pr.pz[k]-tMax*lnz)
		}
		eMin, eMax, eOk := pr.updateRaySlabRange(pr.px[k], pr.ex[k], hx, 0.0, 1.0)
		if eOk {
			eMin, eMax, eOk = pr.updateRaySlabRange(pr.py[k], pr.ey[k], hy, eMin, eMax)
		}
		if eOk {
			eMin, eMax, eOk = pr.updateRaySlabRange(pr.pz[k], pr.ez[k], hz, eMin, eMax)
		}
		if eOk {
			best, minWZ, found = pr.MergeContact(best, minWZ, found, pr.px[k]+eMin*pr.ex[k], pr.py[k]+eMin*pr.ey[k], pr.pz[k]+eMin*pr.ez[k])
			best, minWZ, found = pr.MergeContact(best, minWZ, found, pr.px[k]+eMax*pr.ex[k], pr.py[k]+eMax*pr.ey[k], pr.pz[k]+eMax*pr.ez[k])
		}
	}
	return best, minWZ, found
}
