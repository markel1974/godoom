package aabb

import "math"

// MaxManifoldPoints defines the maximum number of contact points in an OBB contact manifold.
const MaxManifoldPoints = 4

// ContactPoint represents a single contact vertex and its penetration depth in world space.
type ContactPoint struct {
	X, Y, Z     float64
	Penetration float64
}

func (cp *ContactPoint) GetXYZ() (float64, float64, float64) {
	return cp.X, cp.Y, cp.Z
}

func (cp *ContactPoint) SetXYZ(x, y, z float64) {
	cp.X, cp.Y, cp.Z = x, y, z
}

// OBBManifold describes the result of a 3D SAT OBB-vs-OBB query including the unsnapped
// contact normal (pointing from remote box B toward local box A), up to 4 contact points,
// and reusable internal clipping scratch buffers so repeated calls perform zero allocations.
type OBBManifold struct {
	Hit                       bool
	Penetration               float64
	Gap                       float64
	NormalX, NormalY, NormalZ float64
	Count                     int
	Points                    [MaxManifoldPoints]ContactPoint
	polyA                     [8][3]float64
	polyB                     [8][3]float64
	candidates                [8]ContactPoint
	sat                       *Sat
}

// NewOBBManifold creates a reusable OBB contact manifold initialized for SAT-based OBB queries.
func NewOBBManifold() *OBBManifold {
	return &OBBManifold{
		sat: &Sat{},
	}
}

// ComputeAtOffset computes an unsnapped SAT normal and a clipped multi-point
// contact manifold (up to 4 points) into out between this OBB and another OBB shifted by (offX, offY, offZ).
func (om *OBBManifold) ComputeAtOffset(r *BoundingBox, other *BoundingBox, offX, offY, offZ float64) {
	tx, ty, tz := other.center.Get()
	px, py, pz := r.center.Get()
	dx := tx - px + offX
	dy := ty - py + offY
	dz := tz - pz + offZ
	scx, scy, scz := r.size.GetCenter()
	ocx, ocy, ocz := other.size.GetCenter()

	om.sat.Compute(dx, dy, dz, &r.axis, scx, scy, scz, &other.axis, ocx, ocy, ocz)

	sat := om.sat

	if !sat.hit {
		om.Hit = false
		om.Penetration = 0.0
		om.Gap = sat.maxGap
		om.NormalX, om.NormalY, om.NormalZ = sat.GetNormal()
		om.Count = 0
		return
	}

	om.Hit = true
	om.Penetration = sat.contactPen
	om.Gap = 0.0
	om.NormalX, om.NormalY, om.NormalZ = sat.GetNormal()
	om.Count = 0

	pA := [3]float64{px, py, pz}
	pB := [3]float64{tx + offX, ty + offY, tz + offZ}
	hA := [3]float64{scx, scy, scz}
	hB := [3]float64{ocx, ocy, ocz}

	if sat.contactAxis >= 6 {
		// Edge-edge SAT contact: compute closest points between the two supporting edges.
		axisIdx := sat.contactAxis - 6
		i := axisIdx / 3
		j := axisIdx % 3
		ptX, ptY, ptZ := closestEdgeEdgePoint(pA, pB, &r.axis, hA, &other.axis, hB, i, j, sat.normal)
		om.setSinglePoint(ptX, ptY, ptZ, sat.contactPen)
		return
	}

	// Face-face / face-vertex / face-edge contact: clip the incident face against the reference face.
	om.buildFaceManifold(pA, pB, &r.axis, hA, &other.axis, hB, sat.contactAxis, sat.normal, sat.contactPen)
}

func (om *OBBManifold) setSinglePoint(x, y, z, penetration float64) {
	om.Count = 1
	om.Points[0].X = x
	om.Points[0].Y = y
	om.Points[0].Z = z
	om.Points[0].Penetration = penetration
}

func (om *OBBManifold) buildFaceManifold(pA, pB [3]float64, a *Axis, hA [3]float64, b *Axis, hB [3]float64, bestAxis int, n [3]float64, bestPen float64) {
	var refPos, incPos, refH, incH [3]float64
	var refAxes, incAxes *Axis
	var refIdx int
	var refN [3]float64

	if bestAxis < 3 {
		refIdx = bestAxis
		refPos, refAxes, refH = pA, a, hA
		incPos, incAxes, incH = pB, b, hB
		// Normal n points from B to A, so outward normal of reference face A toward B is -n.
		refN = [3]float64{-n[0], -n[1], -n[2]}
	} else {
		refIdx = bestAxis - 3
		refPos, refAxes, refH = pB, b, hB
		incPos, incAxes, incH = pA, a, hA
		// Outward normal of reference face B toward A is +n.
		refN = n
	}

	refCenter := [3]float64{
		refPos[0] + refN[0]*refH[refIdx],
		refPos[1] + refN[1]*refH[refIdx],
		refPos[2] + refN[2]*refH[refIdx],
	}
	uIdx := (refIdx + 1) % 3
	vIdx := (refIdx + 2) % 3
	refU := refAxes.data[uIdx]
	refV := refAxes.data[vIdx]
	eU := refH[uIdx]
	eV := refH[vIdx]

	// Pick incident face whose outward normal is most anti-parallel to refN.
	incFaceIdx := 0
	maxAbsDot := -1.0
	incDotSign := 1.0
	for k := 0; k < 3; k++ {
		d := refN[0]*incAxes.data[k][0] + refN[1]*incAxes.data[k][1] + refN[2]*incAxes.data[k][2]
		ad := math.Abs(d)
		if ad > maxAbsDot {
			maxAbsDot = ad
			incFaceIdx = k
			if d > 0.0 {
				incDotSign = -1.0
			} else {
				incDotSign = 1.0
			}
		}
	}

	incN := incAxes.data[incFaceIdx]
	incCenter := [3]float64{
		incPos[0] + incN[0]*incDotSign*incH[incFaceIdx],
		incPos[1] + incN[1]*incDotSign*incH[incFaceIdx],
		incPos[2] + incN[2]*incDotSign*incH[incFaceIdx],
	}
	iu := (incFaceIdx + 1) % 3
	iv := (incFaceIdx + 2) % 3
	uAxis := incAxes.data[iu]
	vAxis := incAxes.data[iv]
	hu := incH[iu]
	hv := incH[iv]

	polyA := &om.polyA
	polyB := &om.polyB
	polyA[0] = [3]float64{incCenter[0] + uAxis[0]*hu + vAxis[0]*hv, incCenter[1] + uAxis[1]*hu + vAxis[1]*hv, incCenter[2] + uAxis[2]*hu + vAxis[2]*hv}
	polyA[1] = [3]float64{incCenter[0] - uAxis[0]*hu + vAxis[0]*hv, incCenter[1] - uAxis[1]*hu + vAxis[1]*hv, incCenter[2] - uAxis[2]*hu + vAxis[2]*hv}
	polyA[2] = [3]float64{incCenter[0] - uAxis[0]*hu - vAxis[0]*hv, incCenter[1] - uAxis[1]*hu - vAxis[1]*hv, incCenter[2] - uAxis[2]*hu - vAxis[2]*hv}
	polyA[3] = [3]float64{incCenter[0] + uAxis[0]*hu - vAxis[0]*hv, incCenter[1] + uAxis[1]*hu - vAxis[1]*hv, incCenter[2] + uAxis[2]*hu - vAxis[2]*hv}
	count := 4

	// Clip against the 4 side planes bounding the reference face rectangle.
	rcU := refCenter[0]*refU[0] + refCenter[1]*refU[1] + refCenter[2]*refU[2]
	rcV := refCenter[0]*refV[0] + refCenter[1]*refV[1] + refCenter[2]*refV[2]

	count = clipPolygonPlane(polyA, count, refU, rcU+eU, polyB)
	if count == 0 {
		om.setSinglePoint(refCenter[0], refCenter[1], refCenter[2], bestPen)
		return
	}
	count = clipPolygonPlane(polyB, count, [3]float64{-refU[0], -refU[1], -refU[2]}, -rcU+eU, polyA)
	if count == 0 {
		om.setSinglePoint(refCenter[0], refCenter[1], refCenter[2], bestPen)
		return
	}
	count = clipPolygonPlane(polyA, count, refV, rcV+eV, polyB)
	if count == 0 {
		om.setSinglePoint(refCenter[0], refCenter[1], refCenter[2], bestPen)
		return
	}
	count = clipPolygonPlane(polyB, count, [3]float64{-refV[0], -refV[1], -refV[2]}, -rcV+eV, polyA)
	if count == 0 {
		om.setSinglePoint(refCenter[0], refCenter[1], refCenter[2], bestPen)
		return
	}

	candidates := &om.candidates
	candCount := 0
	const clipDedupSq = 1e-8
	for i := 0; i < count; i++ {
		v := polyA[i]
		depth := (refCenter[0]-v[0])*refN[0] + (refCenter[1]-v[1])*refN[1] + (refCenter[2]-v[2])*refN[2]
		if depth <= 0.0 {
			continue
		}
		// Place contact point symmetrically halfway along penetration depth.
		halfD := 0.5 * depth
		cx := v[0] + refN[0]*halfD
		cy := v[1] + refN[1]*halfD
		cz := v[2] + refN[2]*halfD
		duplicate := false
		for j := 0; j < candCount; j++ {
			ddx := cx - candidates[j].X
			ddy := cy - candidates[j].Y
			ddz := cz - candidates[j].Z
			if ddx*ddx+ddy*ddy+ddz*ddz < clipDedupSq {
				if depth > candidates[j].Penetration {
					candidates[j].Penetration = depth
				}
				duplicate = true
				break
			}
		}
		if !duplicate && candCount < len(candidates) {
			candidates[candCount].SetXYZ(cx, cy, cz)
			candidates[candCount].Penetration = depth
			candCount++
		}
	}

	if candCount == 0 {
		om.setSinglePoint(refCenter[0], refCenter[1], refCenter[2], bestPen)
		return
	}

	if candCount <= MaxManifoldPoints {
		deepestIdx := 0
		for i := 1; i < candCount; i++ {
			if candidates[i].Penetration > candidates[deepestIdx].Penetration {
				deepestIdx = i
			}
		}
		if deepestIdx != 0 {
			candidates[0].SetXYZ(candidates[deepestIdx].GetXYZ())
			candidates[deepestIdx].SetXYZ(candidates[0].GetXYZ())
		}
		om.Count = candCount
		for i := 0; i < candCount; i++ {
			om.Points[i].SetXYZ(candidates[i].GetXYZ())
		}
		return
	}

	om.reduceManifoldPoints(candCount)
}

func (om *OBBManifold) reduceManifoldPoints(candCount int) {
	// Deepest vertex.
	cands := &om.candidates
	i0 := 0
	for i := 1; i < candCount; i++ {
		if cands[i].Penetration > cands[i0].Penetration {
			i0 = i
		}
	}
	// Farthest from p0.
	i1 := 0
	maxD1 := -1.0
	for i := 0; i < candCount; i++ {
		if i == i0 {
			continue
		}
		ax, ay, az := cands[i].GetXYZ()
		bx, by, bz := cands[i0].GetXYZ()
		if d := pointDistSq(ax, ay, az, bx, by, bz); d > maxD1 {
			maxD1 = d
			i1 = i
		}
	}
	//Largest triangle area with (p0, p1).
	i2 := 0
	maxArea := -1.0
	e01x, e01y, e01z := cands[i1].X-cands[i0].X, cands[i1].Y-cands[i0].Y, cands[i1].Z-cands[i0].Z
	for i := 0; i < candCount; i++ {
		if i == i0 || i == i1 {
			continue
		}
		vx, vy, vz := cands[i].X-cands[i0].X, cands[i].Y-cands[i0].Y, cands[i].Z-cands[i0].Z
		cx := e01y*vz - e01z*vy
		cy := e01z*vx - e01x*vz
		cz := e01x*vy - e01y*vx
		if areaSq := cx*cx + cy*cy + cz*cz; areaSq > maxArea {
			maxArea = areaSq
			i2 = i
		}
	}
	// Widest spread from existing triangle {p0, p1, p2}.
	i3 := 0
	maxMinD := -1.0
	for i := 0; i < candCount; i++ {
		if i == i0 || i == i1 || i == i2 {
			continue
		}
		ax, ay, az := cands[i].GetXYZ()
		ai0x, ai0y, ai0z := cands[i0].GetXYZ()
		ai1x, ai1y, ai1z := cands[i1].GetXYZ()
		ai2x, ai2y, ai2z := cands[i2].GetXYZ()
		d0 := pointDistSq(ax, ay, az, ai0x, ai0y, ai0z)
		d1 := pointDistSq(ax, ay, az, ai1x, ai1y, ai1z)
		d2 := pointDistSq(ax, ay, az, ai2x, ai2y, ai2z)
		minD := math.Min(d0, math.Min(d1, d2))
		if minD > maxMinD {
			maxMinD = minD
			i3 = i
		}
	}
	om.Count = MaxManifoldPoints
	om.Points[0] = cands[i0]
	om.Points[1] = cands[i1]
	om.Points[2] = cands[i2]
	om.Points[3] = cands[i3]
}

func worldCrossUnitNormal(a *Axis, i int, b *Axis, j int, axisLen, proj float64) [3]float64 {
	ax, ay, az := a.data[i][0], a.data[i][1], a.data[i][2]
	bx, by, bz := b.data[j][0], b.data[j][1], b.data[j][2]
	cx := (ay*bz - az*by) / axisLen
	cy := (az*bx - ax*bz) / axisLen
	cz := (ax*by - ay*bx) / axisLen
	if proj > 0.0 {
		cx, cy, cz = -cx, -cy, -cz
	}
	return [3]float64{cx, cy, cz}
}

func closestEdgeEdgePoint(pA, pB [3]float64, a *Axis, hA [3]float64, b *Axis, hB [3]float64, i, j int, n [3]float64) (float64, float64, float64) {
	ptA := pA
	for k := 0; k < 3; k++ {
		if k == i {
			continue
		}
		dot := n[0]*a.data[k][0] + n[1]*a.data[k][1] + n[2]*a.data[k][2]
		step := -hA[k]
		if dot < 0.0 {
			step = hA[k]
		}
		ptA[0] += a.data[k][0] * step
		ptA[1] += a.data[k][1] * step
		ptA[2] += a.data[k][2] * step
	}

	ptB := pB
	for l := 0; l < 3; l++ {
		if l == j {
			continue
		}
		dot := n[0]*b.data[l][0] + n[1]*b.data[l][1] + n[2]*b.data[l][2]
		step := -hB[l]
		if dot > 0.0 {
			step = hB[l]
		}
		ptB[0] += b.data[l][0] * step
		ptB[1] += b.data[l][1] * step
		ptB[2] += b.data[l][2] * step
	}

	ua := a.data[i]
	ub := b.data[j]
	k := ua[0]*ub[0] + ua[1]*ub[1] + ua[2]*ub[2]
	rx, ry, rz := ptA[0]-ptB[0], ptA[1]-ptB[1], ptA[2]-ptB[2]
	ca := ua[0]*rx + ua[1]*ry + ua[2]*rz
	cb := ub[0]*rx + ub[1]*ry + ub[2]*rz
	denom := 1.0 - k*k
	var sA, sB float64
	if denom > 1e-12 {
		sA = (k*cb - ca) / denom
		sB = (cb - k*ca) / denom
	}
	if sA < -hA[i] {
		sA = -hA[i]
		sB = clampFloat(cb+k*sA, -hB[j], hB[j])
	} else if sA > hA[i] {
		sA = hA[i]
		sB = clampFloat(cb+k*sA, -hB[j], hB[j])
	}
	if sB < -hB[j] {
		sB = -hB[j]
		sA = clampFloat(k*sB-ca, -hA[i], hA[i])
	} else if sB > hB[j] {
		sB = hB[j]
		sA = clampFloat(k*sB-ca, -hA[i], hA[i])
	}

	cAx, cAy, cAz := ptA[0]+ua[0]*sA, ptA[1]+ua[1]*sA, ptA[2]+ua[2]*sA
	cBx, cBy, cBz := ptB[0]+ub[0]*sB, ptB[1]+ub[1]*sB, ptB[2]+ub[2]*sB
	return (cAx + cBx) * 0.5, (cAy + cBy) * 0.5, (cAz + cBz) * 0.5
}

func clipPolygonPlane(in *[8][3]float64, inCount int, planeN [3]float64, limit float64, out *[8][3]float64) int {
	outCount := 0
	for i := 0; i < inCount; i++ {
		s := in[i]
		e := in[(i+1)%inCount]
		dS := s[0]*planeN[0] + s[1]*planeN[1] + s[2]*planeN[2] - limit
		dE := e[0]*planeN[0] + e[1]*planeN[1] + e[2]*planeN[2] - limit
		if dS <= 0.0 && dE <= 0.0 {
			if outCount < len(out) {
				out[outCount] = e
				outCount++
			}
		} else if dS <= 0.0 && dE > 0.0 {
			if outCount < len(out) {
				t := dS / (dS - dE)
				out[outCount] = [3]float64{s[0] + t*(e[0]-s[0]), s[1] + t*(e[1]-s[1]), s[2] + t*(e[2]-s[2])}
				outCount++
			}
		} else if dS > 0.0 && dE <= 0.0 {
			t := dS / (dS - dE)
			if outCount < len(out) {
				out[outCount] = [3]float64{s[0] + t*(e[0]-s[0]), s[1] + t*(e[1]-s[1]), s[2] + t*(e[2]-s[2])}
				outCount++
			}
			if outCount < len(out) {
				out[outCount] = e
				outCount++
			}
		}
	}
	return outCount
}

func pointDistSq(aX, aY, aZ float64, bX, bY, bZ float64) float64 {
	dx, dy, dz := aX-bX, aY-bY, aZ-bZ
	return dx*dx + dy*dy + dz*dz
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
