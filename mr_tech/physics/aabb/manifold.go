package aabb

import "math"

// MaxManifoldPoints defines the maximum number of contact points in an OBB contact manifold.
const MaxManifoldPoints = 4

// ContactPoint represents a single contact vertex and its penetration depth in world space.
type ContactPoint struct {
	X, Y, Z     float64
	Penetration float64
}

type Sat struct {
	hit         bool
	minPen      float64
	maxGap      float64
	contactPen  float64
	contactAxis int
	normal      [3]float64
	gapNormal   [3]float64
}

func (res *Sat) GetNormal() (float64, float64, float64) {
	return res.normal[0], res.normal[1], res.normal[2]
}

// Compute runs a separating axis test between two oriented boxes using center delta, axes, and half-extents.
// It updates res with hit state, minimum penetration, contact normal, contact axis, and largest separation gap.
func (res *Sat) Compute(dx, dy, dz float64, a *Axis, scx, scy, scz float64, b *Axis, ocx, ocy, ocz float64) {
	hA := [3]float64{scx, scy, scz}
	hB := [3]float64{ocx, ocy, ocz}
	var R [3][3]float64
	var AbsR [3][3]float64
	const epsilon = 1e-12
	const crossAxisLenEps = 1e-6
	const edgeAxisMinLen = 1e-4

	for i := 0; i < 3; i++ {
		ax, ay, az := a.data[i][0], a.data[i][1], a.data[i][2]
		for j := 0; j < 3; j++ {
			value := ax*b.data[j][0] + ay*b.data[j][1] + az*b.data[j][2]
			R[i][j] = value
			AbsR[i][j] = math.Abs(value) + epsilon
		}
	}

	t := [3]float64{
		dx*a.data[0][0] + dy*a.data[0][1] + dz*a.data[0][2],
		dx*a.data[1][0] + dy*a.data[1][1] + dz*a.data[1][2],
		dx*a.data[2][0] + dy*a.data[2][1] + dz*a.data[2][2],
	}

	res.minPen = math.MaxFloat64
	res.contactPen = math.MaxFloat64
	res.contactAxis = -1
	separated := false
	faceBestPen := math.MaxFloat64

	// Test the three axes of A (indices 0..2).
	for i := 0; i < 3; i++ {
		rb := hB[0]*AbsR[i][0] + hB[1]*AbsR[i][1] + hB[2]*AbsR[i][2]
		proj := t[i]
		distance := math.Abs(proj)
		nx, ny, nz := a.data[i][0], a.data[i][1], a.data[i][2]
		if proj > 0.0 {
			nx, ny, nz = -nx, -ny, -nz
		}
		if distance > hA[i]+rb {
			separated = true
			if g := distance - (hA[i] + rb); g > res.maxGap {
				res.maxGap = g
				res.gapNormal = [3]float64{nx, ny, nz}
			}
		} else if !separated {
			p := hA[i] + rb - distance
			if p < res.minPen {
				res.minPen = p
			}
			if p < res.contactPen {
				res.contactPen = p
				faceBestPen = p
				res.contactAxis = i
				res.normal = [3]float64{nx, ny, nz}
			}
		}
	}

	// Test the three axes of B (indices 3..5).
	for j := 0; j < 3; j++ {
		ra := hA[0]*AbsR[0][j] + hA[1]*AbsR[1][j] + hA[2]*AbsR[2][j]
		proj := t[0]*R[0][j] + t[1]*R[1][j] + t[2]*R[2][j]
		distance := math.Abs(proj)
		nx, ny, nz := b.data[j][0], b.data[j][1], b.data[j][2]
		if proj > 0.0 {
			nx, ny, nz = -nx, -ny, -nz
		}
		if distance > ra+hB[j] {
			separated = true
			if g := distance - (ra + hB[j]); g > res.maxGap {
				res.maxGap = g
				res.gapNormal = [3]float64{nx, ny, nz}
			}
		} else if !separated {
			p := ra + hB[j] - distance
			if p < res.minPen {
				res.minPen = p
			}
			if p < res.contactPen {
				res.contactPen = p
				faceBestPen = p
				res.contactAxis = 3 + j
				res.normal = [3]float64{nx, ny, nz}
			}
		}
	}

	// Test the nine cross-product axes A[i] x B[j] (indices 6..14).
	for i := 0; i < 3; i++ {
		i1 := (i + 1) % 3
		i2 := (i + 2) % 3
		for j := 0; j < 3; j++ {
			j1 := (j + 1) % 3
			j2 := (j + 2) % 3
			ra := hA[i1]*AbsR[i2][j] + hA[i2]*AbsR[i1][j]
			rb := hB[j1]*AbsR[i][j2] + hB[j2]*AbsR[i][j1]
			proj := t[i2]*R[i1][j] - t[i1]*R[i2][j]
			distance := math.Abs(proj)
			axisLen := math.Hypot(R[i1][j], R[i2][j])
			if distance > ra+rb {
				separated = true
				if axisLen > crossAxisLenEps {
					if g := (distance - (ra + rb)) / axisLen; g > res.maxGap {
						res.maxGap = g
						res.gapNormal = worldCrossUnitNormal(a, i, b, j, axisLen, proj)
					}
				}
			} else if !separated && axisLen > crossAxisLenEps {
				p := (ra + rb - distance) / axisLen
				if p < res.minPen {
					res.minPen = p
				}
				// Slightly bias against edge-edge axes to prevent numerical angle chatter on flat face contacts.
				if axisLen > edgeAxisMinLen && p < faceBestPen*0.98-1e-4 && p < res.contactPen {
					res.contactPen = p
					res.contactAxis = 6 + i*3 + j
					res.normal = worldCrossUnitNormal(a, i, b, j, axisLen, proj)
				}
			}
		}
	}
	res.hit = !separated
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

// ComputeOBBManifoldAtOffset computes an unsnapped SAT normal and a clipped multi-point
// contact manifold (up to 4 points) into out between this OBB and another OBB shifted by (offX, offY, offZ).
func (om *OBBManifold) ComputeOBBManifoldAtOffset(r *BoundingBox, other *BoundingBox, offX, offY, offZ float64) {
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
		om.NormalX = sat.gapNormal[0]
		om.NormalY = sat.gapNormal[1]
		om.NormalZ = sat.gapNormal[2]
		om.Count = 0
		return
	}

	om.Hit = true
	om.Penetration = sat.contactPen
	om.Gap = 0.0
	om.NormalX = sat.normal[0]
	om.NormalY = sat.normal[1]
	om.NormalZ = sat.normal[2]
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
		om.Count = 1
		om.Points[0].X = ptX
		om.Points[0].Y = ptY
		om.Points[0].Z = ptZ
		om.Points[0].Penetration = sat.contactPen
		return
	}

	// Face-face / face-vertex / face-edge contact: clip the incident face against the reference face.
	om.buildFaceManifold(pA, pB, &r.axis, hA, &other.axis, hB, sat.contactAxis, sat.normal, sat.contactPen, om)
}

func (om *OBBManifold) buildFaceManifold(pA, pB [3]float64, a *Axis, hA [3]float64, b *Axis, hB [3]float64, bestAxis int, n [3]float64, bestPen float64, out *OBBManifold) {
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
		om.Count = 1
		om.Points[0] = ContactPoint{X: refCenter[0], Y: refCenter[1], Z: refCenter[2], Penetration: bestPen}
		return
	}
	count = clipPolygonPlane(polyB, count, [3]float64{-refU[0], -refU[1], -refU[2]}, -rcU+eU, polyA)
	if count == 0 {
		om.Count = 1
		om.Points[0] = ContactPoint{X: refCenter[0], Y: refCenter[1], Z: refCenter[2], Penetration: bestPen}
		return
	}
	count = clipPolygonPlane(polyA, count, refV, rcV+eV, polyB)
	if count == 0 {
		om.Count = 1
		om.Points[0] = ContactPoint{X: refCenter[0], Y: refCenter[1], Z: refCenter[2], Penetration: bestPen}
		return
	}
	count = clipPolygonPlane(polyB, count, [3]float64{-refV[0], -refV[1], -refV[2]}, -rcV+eV, polyA)
	if count == 0 {
		om.Count = 1
		om.Points[0] = ContactPoint{X: refCenter[0], Y: refCenter[1], Z: refCenter[2], Penetration: bestPen}
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
			candidates[candCount] = ContactPoint{X: cx, Y: cy, Z: cz, Penetration: depth}
			candCount++
		}
	}

	if candCount == 0 {
		om.Count = 1
		om.Points[0] = ContactPoint{X: refCenter[0], Y: refCenter[1], Z: refCenter[2], Penetration: bestPen}
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
			candidates[0], candidates[deepestIdx] = candidates[deepestIdx], candidates[0]
		}
		om.Count = candCount
		for i := 0; i < candCount; i++ {
			om.Points[i] = candidates[i]
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
		if d := pointDistSq(cands[i], cands[i0]); d > maxD1 {
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
		d0 := pointDistSq(cands[i], cands[i0])
		d1 := pointDistSq(cands[i], cands[i1])
		d2 := pointDistSq(cands[i], cands[i2])
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

func pointDistSq(a, b ContactPoint) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
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
