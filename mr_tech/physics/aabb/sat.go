package aabb

import "math"

// Sat stores reusable scratch state and results for a separating axis test between oriented bounding boxes.
type Sat struct {
	hit         bool
	minPen      float64
	maxGap      float64
	contactPen  float64
	contactAxis int
	nx, ny, nz  float64
	gx, gy, gz  float64
	r           [3][3]float64
	absR        [3][3]float64
	t           [3]float64
	hA          [3]float64
	hB          [3]float64
}

func NewSat() *Sat {
	return &Sat{}
}

// GetNormal returns the normal vector from the last SAT computation.
func (res *Sat) GetNormal() (float64, float64, float64) {
	return res.nx, res.ny, res.nz
}

// GetContact returns the latest SAT contact result: hit, axis index, penetration, maximum gap, and normal.
// If hit is false, penetration is zero and the normal points along the largest separating gap.
func (res *Sat) GetContact() (bool, int, float64, float64, float64, float64, float64) {
	return res.hit, res.contactAxis, res.contactPen, res.maxGap, res.nx, res.ny, res.nz
}

// Compute runs a separating axis test between two oriented boxes and stores the contact or separation result.
// dx, dy, and dz are the center offset from the first box to the second.
// a and b provide box axes; scx, scy, scz and ocx, ocy, ocz provide their half-extents.
func (res *Sat) Compute(dx, dy, dz float64, a *Axes, scx, scy, scz float64, b *Axes, ocx, ocy, ocz float64) {
	const epsilon = 1e-12
	const crossAxisLenEps = 1e-6
	const edgeAxisMinLen = 1e-4

	res.hA[0], res.hA[1], res.hA[2] = scx, scy, scz
	res.hB[0], res.hB[1], res.hB[2] = ocx, ocy, ocz

	for i := 0; i < 3; i++ {
		ax, ay, az := a.data[i][0], a.data[i][1], a.data[i][2]
		for j := 0; j < 3; j++ {
			value := ax*b.data[j][0] + ay*b.data[j][1] + az*b.data[j][2]
			res.r[i][j] = value
			res.absR[i][j] = math.Abs(value) + epsilon
		}
		res.t[i] = dx*ax + dy*ay + dz*az
	}

	res.hit = false
	res.minPen = math.MaxFloat64
	res.maxGap = 0.0
	res.contactPen = math.MaxFloat64
	res.contactAxis = -1
	res.nx, res.ny, res.nz = 0, 0, 0
	res.gx, res.gy, res.gz = 0, 0, 0
	separated := false
	faceBestPen := math.MaxFloat64

	// Test the three axes of A (indices 0..2).
	for i := 0; i < 3; i++ {
		rb := res.hB[0]*res.absR[i][0] + res.hB[1]*res.absR[i][1] + res.hB[2]*res.absR[i][2]
		proj := res.t[i]
		distance := math.Abs(proj)
		nx, ny, nz := a.data[i][0], a.data[i][1], a.data[i][2]
		if proj > 0.0 {
			nx, ny, nz = -nx, -ny, -nz
		}
		if distance > res.hA[i]+rb {
			separated = true
			res.recordGap(distance-(res.hA[i]+rb), nx, ny, nz)
		} else if !separated {
			p := res.hA[i] + rb - distance
			if p < res.minPen {
				res.minPen = p
			}
			if p < res.contactPen {
				res.contactPen = p
				faceBestPen = p
				res.contactAxis = i
				res.nx, res.ny, res.nz = nx, ny, nz
			}
		}
	}

	// Test the three axes of B (indices 3..5).
	for j := 0; j < 3; j++ {
		ra := res.hA[0]*res.absR[0][j] + res.hA[1]*res.absR[1][j] + res.hA[2]*res.absR[2][j]
		proj := res.t[0]*res.r[0][j] + res.t[1]*res.r[1][j] + res.t[2]*res.r[2][j]
		distance := math.Abs(proj)
		nx, ny, nz := b.data[j][0], b.data[j][1], b.data[j][2]
		if proj > 0.0 {
			nx, ny, nz = -nx, -ny, -nz
		}
		if distance > ra+res.hB[j] {
			separated = true
			res.recordGap(distance-(ra+res.hB[j]), nx, ny, nz)
		} else if !separated {
			p := ra + res.hB[j] - distance
			if p < res.minPen {
				res.minPen = p
			}
			if p < res.contactPen {
				res.contactPen = p
				faceBestPen = p
				res.contactAxis = 3 + j
				res.nx, res.ny, res.nz = nx, ny, nz
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
			ra := res.hA[i1]*res.absR[i2][j] + res.hA[i2]*res.absR[i1][j]
			rb := res.hB[j1]*res.absR[i][j2] + res.hB[j2]*res.absR[i][j1]
			proj := res.t[i2]*res.r[i1][j] - res.t[i1]*res.r[i2][j]
			distance := math.Abs(proj)
			axisLen := math.Hypot(res.r[i1][j], res.r[i2][j])
			if distance > ra+rb {
				separated = true
				if axisLen > crossAxisLenEps {
					if g := (distance - (ra + rb)) / axisLen; g > res.maxGap {
						nx, ny, nz := a.WorldCrossUnitNormal(i, b, j, axisLen, proj)
						res.recordGap(g, nx, ny, nz)
					}
				}
			} else if !separated && axisLen > crossAxisLenEps {
				p := (ra + rb - distance) / axisLen
				if p < res.minPen {
					res.minPen = p
				}
				// Slightly bias against edge-edge axes to prevent numerical angle chatter on flat face contacts.
				if axisLen > edgeAxisMinLen && p < faceBestPen*0.98-1e-6 && p < res.contactPen {
					res.contactPen = p
					res.contactAxis = 6 + i*3 + j
					res.nx, res.ny, res.nz = a.WorldCrossUnitNormal(i, b, j, axisLen, proj)
				}
			}
		}
	}

	res.hit = !separated
	if separated {
		res.minPen = 0.0
		res.contactPen = 0.0
		res.nx, res.ny, res.nz = res.gx, res.gy, res.gz
	}
}

// BuildEdgeManifoldAxes computes the closest-points contact for an edge-edge OBB collision from SAT state and axes.
func (res *Sat) BuildEdgeManifoldAxes(ax, ay, az float64, aAxes *Axes, ahx, ahy, ahz float64, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) (float64, float64, float64, float64) {
	axisIdx := res.contactAxis - 6
	i := axisIdx / 3
	j := axisIdx % 3

	nx, ny, nz := res.nx, res.ny, res.nz
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

	return cx, cy, cz, res.contactPen
	//om.points.SetSinglePoint(cx, cy, cz, sat.contactPen)
}

// BuildFaceManifoldAxes constructs clipped contact points for a face-axis OBB collision from SAT state and axes.
func (res *Sat) BuildFaceManifoldAxes(clipA, clipB *ClipPolygon, ax, ay, az float64, aAxes *Axes, ahx, ahy, ahz float64, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) (float64, float64, float64, float64, float64, float64, bool) {
	var refX, refY, refZ, refHX, refHY, refHZ float64
	var incX, incY, incZ, incHX, incHY, incHZ float64
	var refAxes, incAxes *Axes
	var refIdx int
	var rnx, rny, rnz float64

	if res.contactAxis < 3 {
		refIdx = res.contactAxis
		refX, refY, refZ, refHX, refHY, refHZ, refAxes = ax, ay, az, ahx, ahy, ahz, aAxes
		incX, incY, incZ, incHX, incHY, incHZ, incAxes = bx, by, bz, bhx, bhy, bhz, bAxes
		// SAT normal points from B to A, so outward normal of reference face A toward B is -n.
		rnx, rny, rnz = -res.nx, -res.ny, -res.nz
	} else {
		refIdx = res.contactAxis - 3
		refX, refY, refZ, refHX, refHY, refHZ, refAxes = bx, by, bz, bhx, bhy, bhz, bAxes
		incX, incY, incZ, incHX, incHY, incHZ, incAxes = ax, ay, az, ahx, ahy, ahz, aAxes
		// Outward normal of reference face B toward A is +n.
		rnx, rny, rnz = res.nx, res.ny, res.nz
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
	if clipA.ClipPlane(clipB, rux, ruy, ruz, rcU+eU) == 0 || clipB.ClipPlane(clipA, -rux, -ruy, -ruz, -rcU+eU) == 0 || clipA.ClipPlane(clipB, rvx, rvy, rvz, rcV+eV) == 0 || clipB.ClipPlane(clipA, -rvx, -rvy, -rvz, -rcV+eV) == 0 {
		return rcx, rcy, rcz, rnx, rny, rnz, true
		//om.points.SetSinglePoint(rcx, rcy, rcz, sat.contactPen)
		//return
	}
	return rcx, rcy, rcz, rnx, rny, rnz, false
	//candidatesCount := om.candidates.CollectPlaneCandidates(clipA, rcx, rcy, rcz, rnx, rny, rnz, 0.5, 0.0)
	//om.candidates.FinalizeInto(om.points, candidatesCount, rcx, rcy, rcz, sat.contactPen)
}

// recordGap records gap and its normal when it exceeds the current maximum separating gap.
func (res *Sat) recordGap(gap, nx, ny, nz float64) {
	if gap > res.maxGap {
		res.maxGap = gap
		res.gx, res.gy, res.gz = nx, ny, nz
	}
}
