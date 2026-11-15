package aabb

import "math"

// MaxManifoldPoints is the maximum number of contact points retained in a collision manifold.
const MaxManifoldPoints = 4
const MaxCandidatePoints = 8

// OBBManifold stores the result of an oriented bounding box contact test.
// It includes collision state, contact normal, penetration or gap, and generated contact points.
type OBBManifold struct {
	Hit                       bool
	Gap                       float64
	NormalX, NormalY, NormalZ float64
	Points                    *ContactPoints
	candidates                *ContactPoints
	sat                       *Sat
	clipA                     *ClipPolygon
	clipB                     *ClipPolygon
	qInterpA                  *Quaternion
	qInterpB                  *Quaternion
	axesA                     *Axes
	axesB                     *Axes
}

// NewOBBManifold creates a reusable OBB contact manifold with initialized SAT scratch state.
func NewOBBManifold() *OBBManifold {
	return &OBBManifold{
		sat:        NewSat(),
		Points:     NewContactPoints(),
		candidates: NewContactPoints(),
		clipA:      NewClipPolygon(),
		clipB:      NewClipPolygon(),
		qInterpA:   NewQuaternionEmpty(),
		qInterpB:   NewQuaternionEmpty(),
		axesA:      NewAxis(0, 0, 0, 0, 0, 0, 0, 0, 0),
		axesB:      NewAxis(0, 0, 0, 0, 0, 0, 0, 0, 0),
	}
}

// ComputeAtOffset computes the contact manifold between r and other with other translated by the given offset.
func (om *OBBManifold) ComputeAtOffset(self *BoundingBox, other *BoundingBox, offX, offY, offZ float64) {
	tx, ty, tz := other.center.Get()
	px, py, pz := self.center.Get()
	scx, scy, scz := self.size.GetCenter()
	ocx, ocy, ocz := other.size.GetCenter()

	om.sat.Compute(tx-px+offX, ty-py+offY, tz-pz+offZ, self.axis, scx, scy, scz, other.axis, ocx, ocy, ocz)
	if !om.sat.hit {
		om.Hit = false
		om.Gap = om.sat.maxGap
		om.NormalX, om.NormalY, om.NormalZ = om.sat.GetNormal()
		om.Points.SetPenetration(0)
		om.Points.SetCount(0)
		//om.Penetration = 0.0
		//om.Count = 0
		return
	}

	om.buildFromSat(om.sat, om.clipA, om.clipB, px, py, pz, self.axis, scx, scy, scz, tx+offX, ty+offY, tz+offZ, other.axis, ocx, ocy, ocz)
}

// ComputeSweptAtOffset evaluates discrete t=1 SAT first and falls back to angular+linear conservative advancement CCD
// over t in [0, 1] when boxes move rapidly or rotate through each other. It returns the impact time toi in [0, 1].
func (om *OBBManifold) ComputeSweptAtOffset(self *BoundingBox, other *BoundingBox, dAX, dAY, dAZ, dBX, dBY, dBZ float64) float64 {
	offX, offY, offZ := dBX-dAX, dBY-dAY, dBZ-dAZ
	om.ComputeAtOffset(self, other, offX, offY, offZ)

	sweepA := self.GetAngularSweepArc()
	sweepB := other.GetAngularSweepArc()
	rotBound := sweepA + sweepB
	totalSweep := math.Hypot(math.Hypot(offX, offY), offZ) + rotBound
	if totalSweep <= 1e-4 {
		return 1.0
	}

	if om.Hit {
		minDim := math.Min(
			math.Min(self.GetWidth(), math.Min(self.GetHeight(), self.GetDepth())),
			math.Min(other.GetWidth(), math.Min(other.GetHeight(), other.GetDepth())),
		)
		if totalSweep <= minDim*0.5 {
			return 1.0
		}
		if (om.NormalX*offX + om.NormalY*offY + om.NormalZ*offZ) >= -1e-4 {
			// Skip CCD only if bodies were already overlapping at t=0; if separated at t=0 they tunneled past halfway.
			ax0, ay0, az0 := self.center.Get()
			bx0, by0, bz0 := other.center.Get()
			scx, scy, scz := self.size.GetCenter()
			ocx, ocy, ocz := other.size.GetCenter()
			om.qInterpA.CopyFrom(self.qPrev)
			om.axesA.Set(om.qInterpA.ToAxes())
			om.qInterpB.CopyFrom(other.qPrev)
			om.axesB.Set(om.qInterpB.ToAxes())
			om.sat.Compute(bx0-ax0, by0-ay0, bz0-az0, om.axesA, scx, scy, scz, om.axesB, ocx, ocy, ocz)
			if om.sat.hit {
				return 1.0
			}
		}
	}

	missHit := om.Hit
	missGap := om.Gap
	missNX, missNY, missNZ := om.NormalX, om.NormalY, om.NormalZ
	missPen := om.Points.GetPenetration()
	missCount := om.Points.GetCount()

	if toi, ok := om.findSweptImpact(om.sat, om.clipA, om.clipB, self, other, dAX, dAY, dAZ, dBX, dBY, dBZ, offX, offY, offZ, rotBound); ok {
		return toi
	}

	om.Hit = missHit
	om.Gap = missGap
	om.NormalX, om.NormalY, om.NormalZ = missNX, missNY, missNZ
	om.Points.SetPenetration(missPen)
	om.Points.SetCount(missCount)
	return 1.0
}

// findSweptImpact advances two moving OBBs to their first swept contact and builds the contact manifold.
// It returns the impact time in [0,1] and whether an impact was found.
func (om *OBBManifold) findSweptImpact(sat *Sat, clipA, clipB *ClipPolygon, r, other *BoundingBox, dAX, dAY, dAZ, dBX, dBY, dBZ float64, offX, offY, offZ, rotBound float64) (float64, bool) {
	const maxCAIter = 16
	const ccdGapTol = 1e-3
	const minStep = 1e-3

	ax0, ay0, az0 := r.center.Get()
	bx0, by0, bz0 := other.center.Get()
	scx, scy, scz := r.size.GetCenter()
	ocx, ocy, ocz := other.size.GetCenter()
	baseDX, baseDY, baseDZ := bx0-ax0, by0-ay0, bz0-az0

	t := 0.0
	closeRate := 0.0
	hitFound := false

	for iter := 0; iter < maxCAIter; iter++ {
		r.qPrev.Nlerp(r.qRotation, t, om.qInterpA)
		om.axesA.Set(om.qInterpA.ToAxes())
		other.qPrev.Nlerp(other.qRotation, t, om.qInterpB)
		om.axesB.Set(om.qInterpB.ToAxes())

		sat.Compute(baseDX+offX*t, baseDY+offY*t, baseDZ+offZ*t, om.axesA, scx, scy, scz, om.axesB, ocx, ocy, ocz)
		if sat.hit {
			hitFound = true
			break
		}
		gap := sat.maxGap
		gx, gy, gz := sat.gx, sat.gy, sat.gz
		closeRate = math.Max(0.0, offX*gx+offY*gy+offZ*gz) + rotBound
		if gap <= ccdGapTol {
			if closeRate > 1e-9 && t < 1.0 {
				t = math.Min(1.0, t+(gap+1e-4)/closeRate)
				r.qPrev.Nlerp(r.qRotation, t, om.qInterpA)
				om.axesA.Set(om.qInterpA.ToAxes())
				other.qPrev.Nlerp(other.qRotation, t, om.qInterpB)
				om.axesB.Set(om.qInterpB.ToAxes())
				sat.Compute(baseDX+offX*t, baseDY+offY*t, baseDZ+offZ*t, om.axesA, scx, scy, scz, om.axesB, ocx, ocy, ocz)
			}
			hitFound = true
			break
		}
		if closeRate <= 1e-9 {
			return 1.0, false
		}
		dtStep := gap / closeRate
		if dtStep < minStep {
			dtStep = minStep
		}
		t += dtStep
		if t > 1.0 {
			return 1.0, false
		}
	}

	if !hitFound {
		return 1.0, false
	}

	axT, ayT, azT := ax0+dAX*t, ay0+dAY*t, az0+dAZ*t
	bxT, byT, bzT := bx0+dBX*t, by0+dBY*t, bz0+dBZ*t
	if sat.hit && sat.contactAxis >= 0 {
		om.buildFromSat(sat, clipA, clipB, axT, ayT, azT, om.axesA, scx, scy, scz, bxT, byT, bzT, om.axesB, ocx, ocy, ocz)
	} else {
		om.Hit = true
		om.NormalX, om.NormalY, om.NormalZ = sat.gx, sat.gy, sat.gz
		om.Points.SetPenetration(math.Max(1e-4, -sat.maxGap))
		om.Gap = 0.0
		sx, sy, sz := om.axesA.ComputeStabilizedContactPoint(axT, ayT, azT, -om.NormalX, -om.NormalY, -om.NormalZ, scx, scy, scz)
		om.Points.SetSinglePoint(sx, sy, sz, om.Points.GetPenetration())
		//om.setSinglePoint(sx, sy, sz, om.Penetration)
	}

	remT := 1.0 - t
	linAdv := math.Max(0.0, offX*om.NormalX+offY*om.NormalY+offZ*om.NormalZ) * remT
	angAdv := rotBound * remT * 0.5
	if advPen := linAdv + angAdv; advPen > om.Points.GetPenetration() {
		om.Points.SetPenetration(advPen)
	}
	shiftX := 0.5 * remT * (dAX + dBX)
	shiftY := 0.5 * remT * (dAY + dBY)
	shiftZ := 0.5 * remT * (dAZ + dBZ)
	om.Points.Shift(shiftX, shiftY, shiftZ)
	/*
		for i := 0; i < om.Count; i++ {
			p := &om.Points[i]
			p.X += shiftX
			p.Y += shiftY
			p.Z += shiftZ
			if p.Penetration < om.Penetration {
				p.Penetration = om.Penetration
			}
		}

	*/
	return t, true
}

// buildFromSat populates the manifold from a successful SAT result and builds edge or face contacts.
func (om *OBBManifold) buildFromSat(sat *Sat, clipA, clipB *ClipPolygon, ax, ay, az float64, aAxes *Axes, ahx, ahy, ahz float64, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) {
	om.Hit = true
	om.Points.SetPenetration(sat.contactPen)
	om.Points.SetCount(0)
	om.Gap = 0.0
	om.NormalX, om.NormalY, om.NormalZ = sat.GetNormal()

	if sat.contactAxis >= 6 {
		om.buildEdgeManifoldAxes(sat, ax, ay, az, aAxes, ahx, ahy, ahz, bx, by, bz, bAxes, bhx, bhy, bhz)
		return
	}
	om.buildFaceManifoldAxes(sat, clipA, clipB, ax, ay, az, aAxes, ahx, ahy, ahz, bx, by, bz, bAxes, bhx, bhy, bhz)
}

// buildEdgeManifoldAxes computes the closest-points contact for an edge-edge OBB collision from SAT state and axes.
func (om *OBBManifold) buildEdgeManifoldAxes(sat *Sat, ax, ay, az float64, aAxes *Axes, ahx, ahy, ahz float64, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) {
	axisIdx := sat.contactAxis - 6
	i := axisIdx / 3
	j := axisIdx % 3

	nx, ny, nz := sat.nx, sat.ny, sat.nz
	tieX, tieY, tieZ := bx-ax, by-ay, bz-az
	ptAx, ptAy, ptAz := edgeMidpointAnchor(ax, ay, az, aAxes, ahx, ahy, ahz, i, -nx, -ny, -nz, tieX, tieY, tieZ)
	ptBx, ptBy, ptBz := edgeMidpointAnchor(bx, by, bz, bAxes, bhx, bhy, bhz, j, nx, ny, nz, -tieX, -tieY, -tieZ)

	uAx, uAy, uAz := aAxes.GetRow(i)
	uBx, uBy, uBz := bAxes.GetRow(j)
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
	om.Points.SetSinglePoint(cx, cy, cz, sat.contactPen)
}

// buildFaceManifoldAxes constructs clipped contact points for a face-axis OBB collision from SAT state and axes.
func (om *OBBManifold) buildFaceManifoldAxes(sat *Sat, clipA, clipB *ClipPolygon, ax, ay, az float64, aAxes *Axes, ahx, ahy, ahz float64, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) {
	var refX, refY, refZ, refHX, refHY, refHZ float64
	var incX, incY, incZ, incHX, incHY, incHZ float64
	var refAxes, incAxes *Axes
	var refIdx int
	var rnx, rny, rnz float64

	if sat.contactAxis < 3 {
		refIdx = sat.contactAxis
		refX, refY, refZ, refHX, refHY, refHZ, refAxes = ax, ay, az, ahx, ahy, ahz, aAxes
		incX, incY, incZ, incHX, incHY, incHZ, incAxes = bx, by, bz, bhx, bhy, bhz, bAxes
		// SAT normal points from B to A, so outward normal of reference face A toward B is -n.
		rnx, rny, rnz = -sat.nx, -sat.ny, -sat.nz
	} else {
		refIdx = sat.contactAxis - 3
		refX, refY, refZ, refHX, refHY, refHZ, refAxes = bx, by, bz, bhx, bhy, bhz, bAxes
		incX, incY, incZ, incHX, incHY, incHZ, incAxes = ax, ay, az, ahx, ahy, ahz, aAxes
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

	clipA.FillIncidentFaceQuad(incX, incY, incZ, incAxes, incHX, incHY, incHZ, rnx, rny, rnz)

	// Clip against the 4 side planes bounding the reference face rectangle.
	rcU := rcx*rux + rcy*ruy + rcz*ruz
	rcV := rcx*rvx + rcy*rvy + rcz*rvz
	if clipA.ClipPlane(clipB, rux, ruy, ruz, rcU+eU) == 0 || clipB.ClipPlane(clipA, -rux, -ruy, -ruz, -rcU+eU) == 0 ||
		clipA.ClipPlane(clipB, rvx, rvy, rvz, rcV+eV) == 0 || clipB.ClipPlane(clipA, -rvx, -rvy, -rvz, -rcV+eV) == 0 {
		om.Points.SetSinglePoint(rcx, rcy, rcz, sat.contactPen)
		//om.setSinglePoint(rcx, rcy, rcz, sat.contactPen)
		return
	}

	candCount := om.collectPlaneCandidates(clipA, rcx, rcy, rcz, rnx, rny, rnz, 0.5, 0.0)
	om.finalizePlaneCandidates(candCount, rcx, rcy, rcz, sat.contactPen)
}

// ComputeStaticTriangle clips the incident face of OBB self (shifted by offX, offY, offZ) against the static triangle
// (v0, v1, v2) with plane normal (nx, ny, nz) pointing toward self, retaining up to MaxManifoldPoints contact points.
// It reports whether clipping produced at least one penetrating contact point.
func (om *OBBManifold) ComputeStaticTriangle(self *BoundingBox, offX, offY, offZ float64, nx, ny, nz float64, v0x, v0y, v0z, v1x, v1y, v1z, v2x, v2y, v2z float64, maxPen float64) bool {
	om.Hit = true
	om.NormalX, om.NormalY, om.NormalZ = nx, ny, nz
	om.Points.SetPenetration(maxPen)
	om.Gap = 0.0

	px, py, pz := self.center.Get()
	hx, hy, hz := self.size.GetCenter()
	om.clipA.FillIncidentFaceQuad(px+offX, py+offY, pz+offZ, self.axis, hx, hy, hz, nx, ny, nz)

	// Align triangle winding normal with (nx, ny, nz) so edge×effN points strictly outward from each edge.
	e1x, e1y, e1z := v1x-v0x, v1y-v0y, v1z-v0z
	e2x, e2y, e2z := v2x-v0x, v2y-v0y, v2z-v0z
	tnx := e1y*e2z - e1z*e2y
	tny := e1z*e2x - e1x*e2z
	tnz := e1x*e2y - e1y*e2x
	effNX, effNY, effNZ := nx, ny, nz
	if tnx*nx+tny*ny+tnz*nz < 0.0 {
		effNX, effNY, effNZ = -nx, -ny, -nz
	}

	if !om.clipA.ClipEdgeHalfSpace(om.clipB, v0x, v0y, v0z, v1x, v1y, v1z, effNX, effNY, effNZ) ||
		!om.clipB.ClipEdgeHalfSpace(om.clipA, v1x, v1y, v1z, v2x, v2y, v2z, effNX, effNY, effNZ) ||
		!om.clipA.ClipEdgeHalfSpace(om.clipB, v2x, v2y, v2z, v0x, v0y, v0z, effNX, effNY, effNZ) {
		om.Points.SetCount(0)
		return false
	}

	candidatesCount := om.collectPlaneCandidates(om.clipB, v0x, v0y, v0z, nx, ny, nz, 1.0, maxPen)
	if candidatesCount == 0 {
		om.Points.SetCount(0)
		return false
	}
	om.finalizePlaneCandidates(candidatesCount, v0x, v0y, v0z, maxPen)
	return om.Points.GetCount() > 0
}

// collectPlaneCandidates projects clipped polygon vertices onto plane (planeX/Y/Z, planeNX/Y/Z) with depth > 0 into om.candidates.
func (om *OBBManifold) collectPlaneCandidates(poly *ClipPolygon, planeX, planeY, planeZ, planeNX, planeNY, planeNZ, shiftScale, maxDepth float64) int {
	candidatesCount := om.candidates.CollectPlaneCandidates(poly, planeX, planeY, planeZ, planeNX, planeNY, planeNZ, shiftScale, maxDepth)
	return candidatesCount
}

// finalizePlaneCandidates reduces candCount candidate contact points to at most MaxManifoldPoints or falls back to a single point.
func (om *OBBManifold) finalizePlaneCandidates(candidatesCount int, fallbackX, fallbackY, fallbackZ, fallbackPen float64) {
	if candidatesCount == 0 {
		om.Points.SetSinglePoint(fallbackX, fallbackY, fallbackZ, fallbackPen)
		return
	}
	if candidatesCount <= MaxManifoldPoints {
		om.candidates.Finalize(candidatesCount)
		om.Points.CopyFrom(om.candidates, candidatesCount)
		return
	}
	i0, i1, i2, i3 := om.candidates.Reduce(candidatesCount)
	om.Points.SetCount(MaxManifoldPoints)
	om.Points.AssignIndex(0, i0)
	om.Points.AssignIndex(1, i1)
	om.Points.AssignIndex(2, i2)
	om.Points.AssignIndex(3, i3)
}

// edgeMidpointAnchor returns an anchor point at the midpoint of a box edge selected by skipIdx and direction.
func edgeMidpointAnchor(cx, cy, cz float64, axes *Axes, hx, hy, hz float64, skipIdx int, dirX, dirY, dirZ, tieX, tieY, tieZ float64) (float64, float64, float64) {
	px, py, pz := cx, cy, cz
	for k := 0; k < 3; k++ {
		if k == skipIdx {
			continue
		}
		ax, ay, az := axes.GetRow(k)
		step := -axisHalfExtent(hx, hy, hz, k)
		dot := dirX*ax + dirY*ay + dirZ*az
		if math.Abs(dot) <= 1e-12 {
			dot = tieX*ax + tieY*ay + tieZ*az
		}
		if dot > 0.0 {
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
