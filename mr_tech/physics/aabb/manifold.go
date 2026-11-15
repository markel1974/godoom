package aabb

import "math"

// MaxManifoldPoints is the maximum number of contact points retained in a collision manifold.
const MaxManifoldPoints = 4
const MaxCandidatePoints = 8

// OBBManifold stores the result of an oriented bounding box contact test.
// It includes collision state, contact normal, penetration or gap, and generated contact points.
type OBBManifold struct {
	hit                       bool
	gap                       float64
	normalX, normalY, normalZ float64
	points                    *ContactPoints
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
		points:     NewContactPoints(),
		candidates: NewContactPoints(),
		clipA:      NewClipPolygon(),
		clipB:      NewClipPolygon(),
		qInterpA:   NewQuaternionEmpty(),
		qInterpB:   NewQuaternionEmpty(),
		axesA:      NewAxis(0, 0, 0, 0, 0, 0, 0, 0, 0),
		axesB:      NewAxis(0, 0, 0, 0, 0, 0, 0, 0, 0),
	}
}

// GetNormal returns the contact normal components for the manifold.
func (om *OBBManifold) GetNormal() (float64, float64, float64) {
	return om.normalX, om.normalY, om.normalZ
}

// GetHit reports whether the last contact test found an intersection.
func (om *OBBManifold) GetHit() bool {
	return om.hit
}

// GetGap returns the separation gap from the last contact test, or zero when the boxes overlap.
func (om *OBBManifold) GetGap() float64 {
	return om.gap
}

// GetPenetration returns the contact penetration depth from the current manifold.
func (om *OBBManifold) GetPenetration() float64 {
	return om.points.GetPenetration()
}

// GetCount returns the number of contact points in the manifold.
func (om *OBBManifold) GetCount() int {
	return om.points.GetCount()
}

// GetPoint returns the world-space x, y, z coordinates and penetration depth of contact point i.
func (om *OBBManifold) GetPoint(i int) (float64, float64, float64, float64) {
	return om.points.GetIndex(i)
}

// ComputeAtOffset computes the contact manifold between r and other with other translated by the given offset.
func (om *OBBManifold) ComputeAtOffset(self *BoundingBox, other *BoundingBox, offX, offY, offZ float64) {
	tx, ty, tz := other.center.Get()
	px, py, pz := self.center.Get()
	scx, scy, scz := self.size.GetCenter()
	ocx, ocy, ocz := other.size.GetCenter()

	om.sat.Compute(tx-px+offX, ty-py+offY, tz-pz+offZ, self.axis, scx, scy, scz, other.axis, ocx, ocy, ocz)
	if !om.sat.hit {
		om.hit = false
		om.gap = om.sat.maxGap
		om.normalX, om.normalY, om.normalZ = om.sat.GetNormal()
		om.points.SetPenetration(0)
		om.points.SetCount(0)
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

	if om.hit {
		minDim := math.Min(
			math.Min(self.GetWidth(), math.Min(self.GetHeight(), self.GetDepth())),
			math.Min(other.GetWidth(), math.Min(other.GetHeight(), other.GetDepth())),
		)
		if totalSweep <= minDim*0.5 {
			return 1.0
		}
		if (om.normalX*offX + om.normalY*offY + om.normalZ*offZ) >= -1e-4 {
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

	missHit := om.hit
	missGap := om.gap
	missNX, missNY, missNZ := om.normalX, om.normalY, om.normalZ
	missPen := om.points.GetPenetration()
	missCount := om.points.GetCount()

	if toi, ok := om.findSweptImpact(om.sat, om.clipA, om.clipB, self, other, dAX, dAY, dAZ, dBX, dBY, dBZ, offX, offY, offZ, rotBound); ok {
		return toi
	}

	om.hit = missHit
	om.gap = missGap
	om.normalX, om.normalY, om.normalZ = missNX, missNY, missNZ
	om.points.SetPenetration(missPen)
	om.points.SetCount(missCount)
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
		om.hit = true
		om.normalX, om.normalY, om.normalZ = sat.gx, sat.gy, sat.gz
		om.points.SetPenetration(math.Max(1e-4, -sat.maxGap))
		om.gap = 0.0
		sx, sy, sz := om.axesA.ComputeStabilizedContactPoint(axT, ayT, azT, -om.normalX, -om.normalY, -om.normalZ, scx, scy, scz)
		om.points.SetSinglePoint(sx, sy, sz, om.points.GetPenetration())
	}

	remT := 1.0 - t
	linAdv := math.Max(0.0, offX*om.normalX+offY*om.normalY+offZ*om.normalZ) * remT
	angAdv := rotBound * remT * 0.5
	if advPen := linAdv + angAdv; advPen > om.points.GetPenetration() {
		om.points.SetPenetration(advPen)
	}
	shiftX := 0.5 * remT * (dAX + dBX)
	shiftY := 0.5 * remT * (dAY + dBY)
	shiftZ := 0.5 * remT * (dAZ + dBZ)
	om.points.Shift(shiftX, shiftY, shiftZ)
	return t, true
}

// buildFromSat populates the manifold from a successful SAT result and builds edge or face contacts.
func (om *OBBManifold) buildFromSat(sat *Sat, clipA, clipB *ClipPolygon, ax, ay, az float64, aAxes *Axes, ahx, ahy, ahz float64, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) {
	om.hit = true
	om.points.SetPenetration(sat.contactPen)
	om.points.SetCount(0)
	om.gap = 0.0
	om.normalX, om.normalY, om.normalZ = sat.GetNormal()

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
	ptAx, ptAy, ptAz := EdgeMidpointAnchor(ax, ay, az, aAxes, ahx, ahy, ahz, i, -nx, -ny, -nz, tieX, tieY, tieZ)
	ptBx, ptBy, ptBz := EdgeMidpointAnchor(bx, by, bz, bAxes, bhx, bhy, bhz, j, nx, ny, nz, -tieX, -tieY, -tieZ)

	uAx, uAy, uAz := aAxes.GetRow(i)
	uBx, uBy, uBz := bAxes.GetRow(j)
	limA := AxisHalfExtent(ahx, ahy, ahz, i)
	limB := AxisHalfExtent(bhx, bhy, bhz, j)

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
		sB = ClampFloat(cb+k*sA, -limB, limB)
	} else if sA > limA {
		sA = limA
		sB = ClampFloat(cb+k*sA, -limB, limB)
	}
	if sB < -limB {
		sB = -limB
		sA = ClampFloat(k*sB-ca, -limA, limA)
	} else if sB > limB {
		sB = limB
		sA = ClampFloat(k*sB-ca, -limA, limA)
	}

	cx := 0.5 * (ptAx + uAx*sA + ptBx + uBx*sB)
	cy := 0.5 * (ptAy + uAy*sA + ptBy + uBy*sB)
	cz := 0.5 * (ptAz + uAz*sA + ptBz + uBz*sB)
	om.points.SetSinglePoint(cx, cy, cz, sat.contactPen)
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

	refDepthHalf := AxisHalfExtent(refHX, refHY, refHZ, refIdx)
	rcx := refX + rnx*refDepthHalf
	rcy := refY + rny*refDepthHalf
	rcz := refZ + rnz*refDepthHalf

	uIdx := (refIdx + 1) % 3
	vIdx := (refIdx + 2) % 3
	rux, ruy, ruz := refAxes.GetRow(uIdx)
	rvx, rvy, rvz := refAxes.GetRow(vIdx)
	eU := AxisHalfExtent(refHX, refHY, refHZ, uIdx)
	eV := AxisHalfExtent(refHX, refHY, refHZ, vIdx)

	clipA.FillIncidentFaceQuad(incX, incY, incZ, incAxes, incHX, incHY, incHZ, rnx, rny, rnz)

	// Clip against the 4 side planes bounding the reference face rectangle.
	rcU := rcx*rux + rcy*ruy + rcz*ruz
	rcV := rcx*rvx + rcy*rvy + rcz*rvz
	if clipA.ClipPlane(clipB, rux, ruy, ruz, rcU+eU) == 0 || clipB.ClipPlane(clipA, -rux, -ruy, -ruz, -rcU+eU) == 0 ||
		clipA.ClipPlane(clipB, rvx, rvy, rvz, rcV+eV) == 0 || clipB.ClipPlane(clipA, -rvx, -rvy, -rvz, -rcV+eV) == 0 {
		om.points.SetSinglePoint(rcx, rcy, rcz, sat.contactPen)
		return
	}

	candidatesCount := om.candidates.CollectPlaneCandidates(clipA, rcx, rcy, rcz, rnx, rny, rnz, 0.5, 0.0)
	om.candidates.FinalizeInto(om.points, candidatesCount, rcx, rcy, rcz, sat.contactPen)
}

// ComputeStaticTriangle clips the incident face of OBB self (shifted by offX, offY, offZ) against the static triangle
// (v0, v1, v2) with plane normal (nx, ny, nz) pointing toward self, retaining up to MaxManifoldPoints contact points.
// It reports whether clipping produced at least one penetrating contact point.
func (om *OBBManifold) ComputeStaticTriangle(self *BoundingBox, offX, offY, offZ float64, nx, ny, nz float64, v0x, v0y, v0z, v1x, v1y, v1z, v2x, v2y, v2z float64, maxPen float64) bool {
	om.hit = true
	om.normalX, om.normalY, om.normalZ = nx, ny, nz
	om.points.SetPenetration(maxPen)
	om.gap = 0.0

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
		om.points.SetCount(0)
		return false
	}

	candidatesCount := om.candidates.CollectPlaneCandidates(om.clipB, v0x, v0y, v0z, nx, ny, nz, 1.0, maxPen)
	if candidatesCount == 0 {
		om.points.SetCount(0)
		return false
	}
	om.candidates.FinalizeInto(om.points, candidatesCount, v0x, v0y, v0z, maxPen)
	return om.points.GetCount() > 0
}
