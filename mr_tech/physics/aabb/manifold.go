package aabb

import "math"

// MaxManifoldPoints is the maximum number of contact points retained in a collision manifold.
const MaxManifoldPoints = 4

// OBBManifold stores the result of an oriented bounding box contact test.
// It includes collision state, contact normal, penetration or gap, and generated contact points.
type OBBManifold struct {
	Hit                       bool
	Penetration               float64
	Gap                       float64
	NormalX, NormalY, NormalZ float64
	Count                     int
	Points                    [MaxManifoldPoints]ContactPoint
	clipA                     ClipPolygon
	clipB                     ClipPolygon
	candidates                [8]ContactPoint
	sat                       *Sat
}

// NewOBBManifold creates a reusable OBB contact manifold with initialized SAT scratch state.
func NewOBBManifold() *OBBManifold {
	return &OBBManifold{
		sat: &Sat{},
	}
}

// ComputeAtOffset computes the contact manifold between r and other with other translated by the given offset.
func (om *OBBManifold) ComputeAtOffset(r *BoundingBox, other *BoundingBox, offX, offY, offZ float64) {
	tx, ty, tz := other.center.Get()
	px, py, pz := r.center.Get()
	scx, scy, scz := r.size.GetCenter()
	ocx, ocy, ocz := other.size.GetCenter()

	om.sat.Compute(tx-px+offX, ty-py+offY, tz-pz+offZ, &r.axis, scx, scy, scz, &other.axis, ocx, ocy, ocz)

	sat := om.sat
	om.NormalX, om.NormalY, om.NormalZ = sat.GetNormal()
	if !sat.hit {
		om.Hit = false
		om.Penetration = 0.0
		om.Gap = sat.maxGap
		om.Count = 0
		return
	}

	om.Hit = true
	om.Penetration = sat.contactPen
	om.Gap = 0.0
	om.Count = 0

	if sat.contactAxis >= 6 {
		om.buildEdgeManifold(r, other, offX, offY, offZ)
		return
	}
	om.buildFaceManifold(r, other, offX, offY, offZ)
}

// setSinglePoint resets the manifold to one contact point with the given position and penetration depth.
func (om *OBBManifold) setSinglePoint(x, y, z, penetration float64) {
	om.Count = 1
	om.Points[0].Set(x, y, z, penetration)
}

// copyCandidateToPoint copies a candidate contact point into the manifold point at the given indices.
func (om *OBBManifold) copyCandidateToPoint(dstIdx, srcIdx int) {
	c := &om.candidates[srcIdx]
	om.Points[dstIdx].Set(c.X, c.Y, c.Z, c.Penetration)
}

// buildEdgeManifold computes the contact point for an edge-edge OBB collision using the cached SAT result.
func (om *OBBManifold) buildEdgeManifold(r *BoundingBox, other *BoundingBox, offX, offY, offZ float64) {
	sat := om.sat
	axisIdx := sat.contactAxis - 6
	i := axisIdx / 3
	j := axisIdx % 3

	ax, ay, az := r.center.Get()
	ahx, ahy, ahz := r.size.GetCenter()
	bx, by, bz := other.center.Get()
	bhx, bhy, bhz := other.size.GetCenter()

	nx, ny, nz := sat.nx, sat.ny, sat.nz
	ptAx, ptAy, ptAz := edgeMidpointAnchor(ax, ay, az, &r.axis, ahx, ahy, ahz, i, -nx, -ny, -nz)
	ptBx, ptBy, ptBz := edgeMidpointAnchor(bx+offX, by+offY, bz+offZ, &other.axis, bhx, bhy, bhz, j, nx, ny, nz)

	uAx, uAy, uAz := r.axis.GetRow(i)
	uBx, uBy, uBz := other.axis.GetRow(j)
	limA := axisHalfExtent(ahx, ahy, ahz, i)
	limB := axisHalfExtent(bhx, bhy, bhz, j)

	k := uAx*uBx + uAy*uBy + uAz*uBz
	rx, ry, rz := ptAx-ptBx, ptAy-ptBy, ptAz-ptBz
	ca := uAx*rx + uAy*ry + uAz*rz
	cb := uBx*rx + uBy*ry + uBz*rz
	denom := 1.0 - k*k
	var sA, sB float64
	if denom > 1e-12 {
		sA = (k*cb - ca) / denom
		sB = (cb - k*ca) / denom
	}
	if sA < -limA {
		sA = -limA
		sB = clampFloat(cb+k*sA, -limB, limB)
	} else if sA > limA {
		sA = limA
		sB = clampFloat(cb+k*sA, -limB, limB)
	}
	if sB < -limB {
		sB = -limB
		sA = clampFloat(k*sB-ca, -limA, limA)
	} else if sB > limB {
		sB = limB
		sA = clampFloat(k*sB-ca, -limA, limA)
	}

	cx := 0.5 * (ptAx + uAx*sA + ptBx + uBx*sB)
	cy := 0.5 * (ptAy + uAy*sA + ptBy + uBy*sB)
	cz := 0.5 * (ptAz + uAz*sA + ptBz + uBz*sB)
	om.setSinglePoint(cx, cy, cz, sat.contactPen)
}

// buildFaceManifold constructs contact points for a face-axis OBB collision using the current SAT result.
// It clips the incident face against the reference face and reduces candidates to the manifold limit.
func (om *OBBManifold) buildFaceManifold(r *BoundingBox, other *BoundingBox, offX, offY, offZ float64) {
	sat := om.sat
	ax, ay, az := r.center.Get()
	ahx, ahy, ahz := r.size.GetCenter()
	bx, by, bz := other.center.Get()
	bx += offX
	by += offY
	bz += offZ
	bhx, bhy, bhz := other.size.GetCenter()

	var refX, refY, refZ, refHX, refHY, refHZ float64
	var incX, incY, incZ, incHX, incHY, incHZ float64
	var refAxes, incAxes *Axis
	var refIdx int
	var rnx, rny, rnz float64

	if sat.contactAxis < 3 {
		refIdx = sat.contactAxis
		refX, refY, refZ, refHX, refHY, refHZ, refAxes = ax, ay, az, ahx, ahy, ahz, &r.axis
		incX, incY, incZ, incHX, incHY, incHZ, incAxes = bx, by, bz, bhx, bhy, bhz, &other.axis
		// SAT normal points from B to A, so outward normal of reference face A toward B is -n.
		rnx, rny, rnz = -sat.nx, -sat.ny, -sat.nz
	} else {
		refIdx = sat.contactAxis - 3
		refX, refY, refZ, refHX, refHY, refHZ, refAxes = bx, by, bz, bhx, bhy, bhz, &other.axis
		incX, incY, incZ, incHX, incHY, incHZ, incAxes = ax, ay, az, ahx, ahy, ahz, &r.axis
		// Outward normal of reference face B toward A is +n.
		rnx, rny, rnz = sat.nx, sat.ny, sat.nz
	}

	refDepthHalf := axisHalfExtent(refHX, refHY, refHZ, refIdx)
	rcx := refX + rnx*refDepthHalf
	rcy := refY + rny*refDepthHalf
	rcz := refZ + rnz*refDepthHalf

	uIdx := (refIdx + 1) % 3
	vIdx := (refIdx + 2) % 3
	rux, ruy, ruz := refAxes.GetRow(uIdx)
	rvx, rvy, rvz := refAxes.GetRow(vIdx)
	eU := axisHalfExtent(refHX, refHY, refHZ, uIdx)
	eV := axisHalfExtent(refHX, refHY, refHZ, vIdx)

	// Pick incident face whose outward normal is most anti-parallel to refN.
	incFaceIdx := 0
	maxAbsDot := -1.0
	incDotSign := 1.0
	for k := 0; k < 3; k++ {
		kx, ky, kz := incAxes.GetRow(k)
		d := rnx*kx + rny*ky + rnz*kz
		if ad := math.Abs(d); ad > maxAbsDot {
			maxAbsDot = ad
			incFaceIdx = k
			if d > 0.0 {
				incDotSign = -1.0
			} else {
				incDotSign = 1.0
			}
		}
	}

	inx, iny, inz := incAxes.GetRow(incFaceIdx)
	incHalfN := incDotSign * axisHalfExtent(incHX, incHY, incHZ, incFaceIdx)
	icx := incX + inx*incHalfN
	icy := incY + iny*incHalfN
	icz := incZ + inz*incHalfN

	iu := (incFaceIdx + 1) % 3
	iv := (incFaceIdx + 2) % 3
	ux, uy, uz := incAxes.GetRow(iu)
	vx, vy, vz := incAxes.GetRow(iv)
	hu := axisHalfExtent(incHX, incHY, incHZ, iu)
	hv := axisHalfExtent(incHX, incHY, incHZ, iv)
	ux *= hu
	uy *= hu
	uz *= hu
	vx *= hv
	vy *= hv
	vz *= hv

	clipA := &om.clipA
	clipB := &om.clipB
	clipA.count = 0
	clipA.Add(icx+ux+vx, icy+uy+vy, icz+uz+vz)
	clipA.Add(icx-ux+vx, icy-uy+vy, icz-uz+vz)
	clipA.Add(icx-ux-vx, icy-uy-vy, icz-uz-vz)
	clipA.Add(icx+ux-vx, icy+uy-vy, icz+uz-vz)

	// Clip against the 4 side planes bounding the reference face rectangle.
	rcU := rcx*rux + rcy*ruy + rcz*ruz
	rcV := rcx*rvx + rcy*rvy + rcz*rvz
	if clipA.ClipPlane(rux, ruy, ruz, rcU+eU, clipB) == 0 ||
		clipB.ClipPlane(-rux, -ruy, -ruz, -rcU+eU, clipA) == 0 ||
		clipA.ClipPlane(rvx, rvy, rvz, rcV+eV, clipB) == 0 ||
		clipB.ClipPlane(-rvx, -rvy, -rvz, -rcV+eV, clipA) == 0 {
		om.setSinglePoint(rcx, rcy, rcz, sat.contactPen)
		return
	}

	candidates := &om.candidates
	candCount := 0
	const clipDedupSq = 1e-8
	for i := 0; i < clipA.count; i++ {
		px, py, pz := clipA.x[i], clipA.y[i], clipA.z[i]
		depth := (rcx-px)*rnx + (rcy-py)*rny + (rcz-pz)*rnz
		if depth <= 0.0 {
			continue
		}
		// Place contact point symmetrically halfway along penetration depth.
		halfD := 0.5 * depth
		cx := px + rnx*halfD
		cy := py + rny*halfD
		cz := pz + rnz*halfD
		duplicate := false
		for j := 0; j < candCount; j++ {
			if pointDistSq(cx, cy, cz, candidates[j].X, candidates[j].Y, candidates[j].Z) < clipDedupSq {
				if depth > candidates[j].Penetration {
					candidates[j].Penetration = depth
				}
				duplicate = true
				break
			}
		}
		if !duplicate && candCount < len(candidates) {
			candidates[candCount].Set(cx, cy, cz, depth)
			candCount++
		}
	}

	if candCount == 0 {
		om.setSinglePoint(rcx, rcy, rcz, sat.contactPen)
		return
	}

	if candCount <= MaxManifoldPoints {
		deepestIdx := 0
		for i := 1; i < candCount; i++ {
			if candidates[i].Penetration > candidates[deepestIdx].Penetration {
				deepestIdx = i
			}
		}
		om.Count = candCount
		om.copyCandidateToPoint(0, deepestIdx)
		dst := 1
		for i := 0; i < candCount; i++ {
			if i == deepestIdx {
				continue
			}
			om.copyCandidateToPoint(dst, i)
			dst++
		}
		return
	}

	om.reduceManifoldPoints(candCount)
}

// reduceManifoldPoints keeps the deepest candidate and spatially spreads the remaining manifold points.
func (om *OBBManifold) reduceManifoldPoints(candCount int) {
	cands := &om.candidates

	// Deepest vertex.
	i0 := 0
	for i := 1; i < candCount; i++ {
		if cands[i].Penetration > cands[i0].Penetration {
			i0 = i
		}
	}
	p0 := &cands[i0]

	// Farthest from p0.
	i1 := 0
	maxD1 := -1.0
	for i := 0; i < candCount; i++ {
		if i == i0 {
			continue
		}
		if d := pointDistSq(cands[i].X, cands[i].Y, cands[i].Z, p0.X, p0.Y, p0.Z); d > maxD1 {
			maxD1 = d
			i1 = i
		}
	}
	p1 := &cands[i1]

	// Largest triangle area with (p0, p1).
	i2 := 0
	maxArea := -1.0
	e01x, e01y, e01z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	for i := 0; i < candCount; i++ {
		if i == i0 || i == i1 {
			continue
		}
		vx, vy, vz := cands[i].X-p0.X, cands[i].Y-p0.Y, cands[i].Z-p0.Z
		cx := e01y*vz - e01z*vy
		cy := e01z*vx - e01x*vz
		cz := e01x*vy - e01y*vx
		if areaSq := cx*cx + cy*cy + cz*cz; areaSq > maxArea {
			maxArea = areaSq
			i2 = i
		}
	}
	p2 := &cands[i2]

	// Widest spread from existing triangle {p0, p1, p2}.
	i3 := 0
	maxMinD := -1.0
	for i := 0; i < candCount; i++ {
		if i == i0 || i == i1 || i == i2 {
			continue
		}
		c := &cands[i]
		d0 := pointDistSq(c.X, c.Y, c.Z, p0.X, p0.Y, p0.Z)
		d1 := pointDistSq(c.X, c.Y, c.Z, p1.X, p1.Y, p1.Z)
		d2 := pointDistSq(c.X, c.Y, c.Z, p2.X, p2.Y, p2.Z)
		minD := math.Min(d0, math.Min(d1, d2))
		if minD > maxMinD {
			maxMinD = minD
			i3 = i
		}
	}

	om.Count = MaxManifoldPoints
	om.copyCandidateToPoint(0, i0)
	om.copyCandidateToPoint(1, i1)
	om.copyCandidateToPoint(2, i2)
	om.copyCandidateToPoint(3, i3)
}

// edgeMidpointAnchor returns an anchor point at the midpoint of a box edge selected by skipIdx and direction.
func edgeMidpointAnchor(cx, cy, cz float64, axes *Axis, hx, hy, hz float64, skipIdx int, dirX, dirY, dirZ float64) (float64, float64, float64) {
	px, py, pz := cx, cy, cz
	for k := 0; k < 3; k++ {
		if k == skipIdx {
			continue
		}
		ax, ay, az := axes.GetRow(k)
		step := -axisHalfExtent(hx, hy, hz, k)
		if dirX*ax+dirY*ay+dirZ*az > 0.0 {
			step = -step
		}
		px += ax * step
		py += ay * step
		pz += az * step
	}
	return px, py, pz
}

// axisHalfExtent returns the half extent for axis index idx: x for 0, y for 1, and z otherwise.
func axisHalfExtent(x, y, z float64, idx int) float64 {
	switch idx {
	case 0:
		return x
	case 1:
		return y
	default:
		return z
	}
}

// pointDistSq returns the squared Euclidean distance between two 3D points.
func pointDistSq(aX, aY, aZ, bX, bY, bZ float64) float64 {
	dx, dy, dz := aX-bX, aY-bY, aZ-bZ
	return dx*dx + dy*dy + dz*dz
}

// clampFloat returns v constrained to the inclusive range from minV to maxV.
func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
