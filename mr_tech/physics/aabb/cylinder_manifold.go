package aabb

import "math"

// cylOBBSatState stores intermediate SAT data for cylinder versus oriented bounding box overlap tests.
type cylOBBSatState struct {
	dx, dy, dz    float64
	cR, cHZ       float64
	bAxes         *Axes
	bhx, bhy, bhz float64
	separated     bool
	maxGap        float64
	gx, gy, gz    float64
	minPen        float64
	nx, ny, nz    float64
}

// TestAxis evaluates a candidate SAT axis for the cylinder-OBB pair.
// It normalizes valid axes and records the largest separating gap or smallest penetration.
func (st *cylOBBSatState) TestAxis(ax, ay, az float64) {
	lenSq := ax*ax + ay*ay + az*az
	if lenSq <= 1e-24 {
		return
	}
	if math.Abs(lenSq-1.0) > 1e-12 {
		inv := 1.0 / math.Sqrt(lenSq)
		ax *= inv
		ay *= inv
		az *= inv
	}
	proj := st.dx*ax + st.dy*ay + st.dz*az
	if proj < 0.0 {
		ax, ay, az, proj = -ax, -ay, -az, -proj
	}
	cylExt := st.cR*math.Hypot(ax, ay) + st.cHZ*math.Abs(az)
	obbExt := st.bAxes.ProjectNormal(ax, ay, az, st.bhx, st.bhy, st.bhz)
	ov := cylExt + obbExt - proj
	if ov <= 0.0 {
		g := -ov
		st.separated = true
		if g > st.maxGap {
			st.maxGap = g
			st.gx, st.gy, st.gz = ax, ay, az
		}
	} else if !st.separated && ov < st.minPen {
		st.minPen = ov
		st.nx, st.ny, st.nz = ax, ay, az
	}
}

// ComputeCylinderSweptAtOffset computes swept cylinder-cylinder contact at offset motion and returns time of impact.
func (om *OBBManifold) ComputeCylinderSweptAtOffset(cAX, cAY, cAZ, dAX, dAY, dAZ, rA, hzA float64, cBX, cBY, cBZ, dBX, dBY, dBZ, rB, hzB float64) float64 {
	rel0X, rel0Y, rel0Z := cAX-cBX, cAY-cBY, cAZ-cBZ
	relVX, relVY, relVZ := dAX-dBX, dAY-dBY, dAZ-dBZ
	rel1X, rel1Y, rel1Z := rel0X+relVX, rel0Y+relVY, rel0Z+relVZ
	sumR, sumZ := rA+rB, hzA+hzB

	hit1, pen1, gap1, nx1, ny1, nz1 := evalCylinderCylinder(rel1X, rel1Y, rel1Z, sumR, sumZ)
	totalSweep := math.Hypot(math.Hypot(relVX, relVY), relVZ)
	if totalSweep > 1e-4 {
		minDim := 2.0 * math.Min(math.Min(rA, hzA), math.Min(rB, hzB))
		needCCD := !hit1
		if hit1 && totalSweep > minDim*0.5 {
			needCCD = true
			if -(relVX*nx1 + relVY*ny1 + relVZ*nz1) >= -1e-4 {
				hit0, _, _, _, _, _ := evalCylinderCylinder(rel0X, rel0Y, rel0Z, sumR, sumZ)
				if hit0 {
					needCCD = false
				}
			}
		}
		if needCCD {
			if t, ok, fromZCap := sweepCylinderInterval(rel0X, rel0Y, rel0Z, relVX, relVY, relVZ, sumR, sumZ); ok {
				relTX, relTY, relTZ := rel0X+relVX*t, rel0Y+relVY*t, rel0Z+relVZ*t
				nxT, nyT, nzT := sweptCylinderNormal(relTX, relTY, relTZ, relVX, relVY, relVZ, fromZCap)
				remT := 1.0 - t
				linAdv := math.Max(0.0, -(relVX*nxT+relVY*nyT+relVZ*nzT)) * remT
				if linAdv > 1e-6 || (hit1 && pen1 > 0.0) {
					pen := math.Max(1e-4, linAdv)
					om.hit = true
					om.gap = 0.0
					om.normalX, om.normalY, om.normalZ = nxT, nyT, nzT
					om.populateCylinderCylinderContact(cAX+dAX*t, cAY+dAY*t, cAZ+dAZ*t, rA, hzA, cBX+dBX*t, cBY+dBY*t, cBZ+dBZ*t, rB, hzB, nxT, nyT, nzT, pen)
					om.points.Shift(0.5*remT*(dAX+dBX), 0.5*remT*(dAY+dBY), 0.5*remT*(dAZ+dBZ))
					return t
				}
			}
		}
	}

	if !hit1 {
		om.hit = false
		om.gap = gap1
		om.normalX, om.normalY, om.normalZ = nx1, ny1, nz1
		om.points.SetPenetration(0)
		om.points.SetCount(0)
		return 1.0
	}

	om.hit = true
	om.gap = 0.0
	om.normalX, om.normalY, om.normalZ = nx1, ny1, nz1
	om.populateCylinderCylinderContact(cAX+dAX, cAY+dAY, cAZ+dAZ, rA, hzA, cBX+dBX, cBY+dBY, cBZ+dBZ, rB, hzB, nx1, ny1, nz1, pen1)
	return 1.0
}

// ComputeCylinderOBBSweptAtOffset computes swept cylinder-vs-OBB contact at the given motion offsets.
// It updates the manifold with end-state contact data or impact time and returns the time of impact.
func (om *OBBManifold) ComputeCylinderOBBSweptAtOffset(cX, cY, cZ, dCX, dCY, dCZ, cR, cHZ float64, box *BoundingBox, dBX, dBY, dBZ float64) float64 {
	bx0, by0, bz0 := box.center.Get()
	bhx, bhy, bhz := box.size.GetCenter()
	c1X, c1Y, c1Z := cX+dCX, cY+dCY, cZ+dCZ
	b1X, b1Y, b1Z := bx0+dBX, by0+dBY, bz0+dBZ

	hit1, pen1, gap1, nx1, ny1, nz1 := evalCylinderOBB(c1X, c1Y, c1Z, cR, cHZ, b1X, b1Y, b1Z, box.axis, bhx, bhy, bhz)

	offX, offY, offZ := dBX-dCX, dBY-dCY, dBZ-dCZ
	rotBound := box.GetAngularSweepArc()
	totalSweep := math.Hypot(math.Hypot(offX, offY), offZ) + rotBound
	if totalSweep > 1e-4 {
		minDim := math.Min(math.Min(2.0*cR, 2.0*cHZ), math.Min(box.GetWidth(), math.Min(box.GetHeight(), box.GetDepth())))
		needCCD := !hit1
		if hit1 && totalSweep > minDim*0.5 {
			needCCD = true
			if (nx1*offX + ny1*offY + nz1*offZ) >= -1e-4 {
				om.qInterpB.CopyFrom(box.qPrev)
				om.axesB.Set(om.qInterpB.ToAxes())
				if hit0, _, _, _, _, _ := evalCylinderOBB(cX, cY, cZ, cR, cHZ, bx0, by0, bz0, om.axesB, bhx, bhy, bhz); hit0 {
					needCCD = false
				}
			}
		}
		if needCCD {
			if toi, ok := om.findCylinderOBBSweptImpact(cX, cY, cZ, dCX, dCY, dCZ, cR, cHZ, box, dBX, dBY, dBZ, offX, offY, offZ, rotBound); ok {
				return toi
			}
		}
	}

	if !hit1 {
		om.hit = false
		om.gap = gap1
		om.normalX, om.normalY, om.normalZ = nx1, ny1, nz1
		om.points.SetPenetration(0)
		om.points.SetCount(0)
		return 1.0
	}

	om.hit = true
	om.gap = 0.0
	om.normalX, om.normalY, om.normalZ = nx1, ny1, nz1
	om.populateCylinderOBBContact(c1X, c1Y, c1Z, cR, cHZ, b1X, b1Y, b1Z, box.axis, bhx, bhy, bhz, nx1, ny1, nz1, pen1)
	return 1.0
}

// ComputeOBBCylinderSweptAtOffset computes swept OBB-vs-cylinder contact and returns the impact time in [0, 1].
// The resulting manifold normal is oriented for the OBB as the first shape.
func (om *OBBManifold) ComputeOBBCylinderSweptAtOffset(box *BoundingBox, dAX, dAY, dAZ float64, cX, cY, cZ, dBX, dBY, dBZ, cR, cHZ float64) float64 {
	toi := om.ComputeCylinderOBBSweptAtOffset(cX, cY, cZ, dBX, dBY, dBZ, cR, cHZ, box, dAX, dAY, dAZ)
	om.normalX = -om.normalX
	om.normalY = -om.normalY
	om.normalZ = -om.normalZ
	return toi
}

// evalCylinderCylinder evaluates overlap or separation between two aligned cylinders using relative offset.
// It returns whether they intersect, the penetration or gap distance, and the separating/contact normal.
func evalCylinderCylinder(rx, ry, rz, sumR, sumZ float64) (hit bool, pen, gap, nx, ny, nz float64) {
	dXY := math.Hypot(rx, ry)
	penR := sumR - dXY
	penZ := sumZ - math.Abs(rz)
	signZ := 1.0
	if rz < 0.0 {
		signZ = -1.0
	}
	if penR < 0.0 || penZ < 0.0 {
		if penR < 0.0 && penZ < 0.0 {
			gR, gZ := -penR, -penZ
			g := math.Hypot(gR, gZ)
			if g > 1e-12 && dXY > 1e-12 {
				return false, 0.0, g, (rx / dXY) * (gR / g), (ry / dXY) * (gR / g), signZ * (gZ / g)
			}
			return false, 0.0, g, 0.0, 0.0, signZ
		}
		if penZ < 0.0 {
			return false, 0.0, -penZ, 0.0, 0.0, signZ
		}
		if dXY > 1e-12 {
			return false, 0.0, -penR, rx / dXY, ry / dXY, 0.0
		}
		return false, 0.0, -penR, 1.0, 0.0, 0.0
	}
	if penZ <= penR {
		return true, penZ, 0.0, 0.0, 0.0, signZ
	}
	if dXY > 1e-12 {
		return true, penR, 0.0, rx / dXY, ry / dXY, 0.0
	}
	return true, penR, 0.0, 1.0, 0.0, 0.0
}

// populateCylinderCylinderContact builds a single contact point and penetration for an overlapping cylinder pair.
// It selects cap or side contact placement from the contact normal and cylinder overlap.
func (om *OBBManifold) populateCylinderCylinderContact(lx, ly, lz, rA, hzA, rx, ry, rz, rB, hzB, nx, ny, nz, pen float64) {
	om.points.SetPenetration(pen)
	if math.Abs(nz) >= 0.7 {
		relX, relY := lx-rx, ly-ry
		dXY := math.Hypot(relX, relY)
		px, py := lx, ly
		if dXY > 1e-12 {
			uMin := math.Max(-rB, dXY-rA)
			uMax := math.Min(rB, dXY+rA)
			uMid := 0.5 * (uMin + uMax)
			ux, uy := relX/dXY, relY/dXY
			px = rx + ux*uMid
			py = ry + uy*uMid
		}
		signZ := 1.0
		if nz < 0.0 {
			signZ = -1.0
		}
		pz := 0.5 * ((lz - signZ*hzA) + (rz + signZ*hzB))
		om.points.SetSinglePoint(px, py, pz, pen)
		return
	}

	dx, dy := lx-rx, ly-ry
	dXY := math.Hypot(dx, dy)
	uMid := 0.5 * ((dXY - rA) + rB)
	px := rx + nx*uMid
	py := ry + ny*uMid
	zMin := math.Max(lz-hzA, rz-hzB)
	zMax := math.Min(lz+hzA, rz+hzB)
	var pz float64
	if zMin <= zMax {
		pz = 0.5 * (zMin + zMax)
	} else {
		pz = 0.5 * (lz + rz)
	}
	om.points.SetSinglePoint(px, py, pz, pen)
}

// findCylinderOBBSweptImpact advances a swept cylinder and OBB to the first impact time.
// It returns the impact fraction and whether contact was found, updating the manifold on hit.
func (om *OBBManifold) findCylinderOBBSweptImpact(cX, cY, cZ, dCX, dCY, dCZ, cR, cHZ float64, box *BoundingBox, dBX, dBY, dBZ, offX, offY, offZ, rotBound float64) (float64, bool) {
	const maxCAIter = 32
	const ccdGapTol = 1e-3
	const minProgressStep = 1e-12

	bx0, by0, bz0 := box.center.Get()
	bhx, bhy, bhz := box.size.GetCenter()
	baseDX, baseDY, baseDZ := bx0-cX, by0-cY, bz0-cZ
	offLen := math.Hypot(math.Hypot(offX, offY), offZ)
	thetaB, radB := box.GetAngularSweepBounds()
	cylBoundRad := math.Hypot(cR, cHZ)

	t := 0.0
	hitFound := false
	var hitPen, hitGap, hitNX, hitNY, hitNZ float64

	for iter := 0; iter < maxCAIter; iter++ {
		box.qPrev.Slerp(box.qRotation, t, om.qInterpB)
		om.axesB.Set(om.qInterpB.ToAxes())

		curCX, curCY, curCZ := cX+dCX*t, cY+dCY*t, cZ+dCZ*t
		curBX, curBY, curBZ := bx0+dBX*t, by0+dBY*t, bz0+dBZ*t
		hit, pen, gap, nx, ny, nz := evalCylinderOBB(curCX, curCY, curCZ, cR, cHZ, curBX, curBY, curBZ, om.axesB, bhx, bhy, bhz)
		if hit {
			hitFound = true
			hitPen, hitGap, hitNX, hitNY, hitNZ = pen, 0.0, nx, ny, nz
			break
		}

		curDX := baseDX + offX*t
		curDY := baseDY + offY*t
		curDZ := baseDZ + offZ*t
		maxFutureDist := math.Sqrt(curDX*curDX+curDY*curDY+curDZ*curDZ) + offLen*(1.0-t)
		effRadB := math.Min(radB, maxFutureDist+cylBoundRad)
		closeRate := math.Max(0.0, offX*nx+offY*ny+offZ*nz) + effRadB*thetaB
		if gap <= ccdGapTol {
			if closeRate > 1e-9 && t < 1.0 {
				t = math.Min(1.0, t+(gap+1e-4)/closeRate)
				box.qPrev.Slerp(box.qRotation, t, om.qInterpB)
				om.axesB.Set(om.qInterpB.ToAxes())
				_, hitPen, _, hitNX, hitNY, hitNZ = evalCylinderOBB(cX+dCX*t, cY+dCY*t, cZ+dCZ*t, cR, cHZ, bx0+dBX*t, by0+dBY*t, bz0+dBZ*t, om.axesB, bhx, bhy, bhz)
			} else {
				hitPen, hitGap, hitNX, hitNY, hitNZ = math.Max(1e-4, gap), gap, nx, ny, nz
			}
			hitFound = true
			break
		}
		if closeRate <= 1e-9 {
			return 1.0, false
		}
		dtStep := gap / closeRate
		if dtStep < minProgressStep {
			dtStep = minProgressStep
		}
		t += dtStep
		if t > 1.0 {
			return 1.0, false
		}
	}

	if !hitFound {
		return 1.0, false
	}

	remT := 1.0 - t
	linAdv := math.Max(0.0, offX*hitNX+offY*hitNY+offZ*hitNZ) * remT
	angAdv := rotBound * remT * 0.5
	pen := math.Max(math.Max(1e-4, hitPen), linAdv+angAdv)
	_ = hitGap

	om.hit = true
	om.gap = 0.0
	om.normalX, om.normalY, om.normalZ = hitNX, hitNY, hitNZ
	curCX, curCY, curCZ := cX+dCX*t, cY+dCY*t, cZ+dCZ*t
	curBX, curBY, curBZ := bx0+dBX*t, by0+dBY*t, bz0+dBZ*t
	om.populateCylinderOBBContact(curCX, curCY, curCZ, cR, cHZ, curBX, curBY, curBZ, om.axesB, bhx, bhy, bhz, hitNX, hitNY, hitNZ, pen)
	om.points.Shift(0.5*remT*(dCX+dBX), 0.5*remT*(dCY+dBY), 0.5*remT*(dCZ+dBZ))
	return t, true
}

// populateCylinderOBBContact builds contact points for a cylinder-OBB overlap using the supplied normal and penetration.
// It emits a midpoint fallback for side contacts and clips cap contacts against the OBB face when applicable.
func (om *OBBManifold) populateCylinderOBBContact(cx, cy, cz, cR, cHZ float64, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64, nx, ny, nz, pen float64) {
	// Single-point midpoint fallback shared by side contacts and tilted Z-cap fallback.
	var cxS, cyS float64
	if hXY := math.Hypot(nx, ny); hXY > 1e-9 {
		cxS = cx - (nx/hXY)*cR
		cyS = cy - (ny/hXY)*cR
	} else {
		cxS, cyS = cx, cy
	}
	_, _, bExtZ := bAxes.Project(bhx, bhy, bhz)
	zMin := math.Max(cz-cHZ, bz-bExtZ)
	zMax := math.Min(cz+cHZ, bz+bExtZ)
	var pzMid float64
	if zMin <= zMax {
		pzMid = 0.5 * (zMin + zMax)
	} else {
		pzMid = math.Max(cz-cHZ, math.Min(cz+cHZ, bz))
	}
	qBx, qBy, qBz := closestPointOnOBBWorld(cx, cy, pzMid, bx, by, bz, bAxes, bhx, bhy, bhz)
	if math.Abs(qBx-cx) <= 1e-9 && math.Abs(qBy-cy) <= 1e-9 && math.Abs(qBz-pzMid) <= 1e-9 {
		dB := (cx-bx)*nx + (cy-by)*ny + (pzMid-bz)*nz
		obbExt := bAxes.ProjectNormal(nx, ny, nz, bhx, bhy, bhz)
		shift := obbExt - dB
		qBx = cx + shift*nx
		qBy = cy + shift*ny
		qBz = pzMid + shift*nz
	}
	fx := 0.5 * (cxS + qBx)
	fy := 0.5 * (cyS + qBy)
	var fz float64
	if math.Abs(nz) >= 0.7 {
		signZ := 1.0
		if nz < 0.0 {
			signZ = -1.0
		}
		zCap := cz - signZ*cHZ
		fz = 0.5 * (zCap + qBz)
	} else {
		fz = pzMid
	}

	om.points.SetPenetration(pen)
	if math.Abs(nz) < 0.7 {
		om.points.SetSinglePoint(fx, fy, fz, pen)
		return
	}

	h := [3]float64{bhx, bhy, bhz}
	dom := 0
	bestDot := -1.0
	bestSignedDot := 0.0
	for i := range 3 {
		ax, ay, az := bAxes.GetRow(i)
		d := ax*nx + ay*ny + az*nz
		if ad := math.Abs(d); ad > bestDot {
			bestDot = ad
			bestSignedDot = d
			dom = i
		}
	}
	sDom := 1.0
	if bestSignedDot < 0.0 {
		sDom = -1.0
	}
	fnX, fnY, fnZ := bAxes.GetRow(dom)
	fnX, fnY, fnZ = fnX*sDom, fnY*sDom, fnZ*sDom
	if math.Abs(fnZ) < 0.5 {
		om.points.SetSinglePoint(fx, fy, fz, pen)
		return
	}
	fcX := bx + h[dom]*fnX
	fcY := by + h[dom]*fnY
	fcZ := bz + h[dom]*fnZ

	uIdx, vIdx := (dom+1)%3, (dom+2)%3
	buX, buY, buZ := bAxes.GetRow(uIdx)
	bvX, bvY, bvZ := bAxes.GetRow(vIdx)
	hu, hv := h[uIdx], h[vIdx]

	signZ := 1.0
	if nz < 0.0 {
		signZ = -1.0
	}
	zCap := cz - signZ*cHZ

	signs := [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}
	candCount := 0
	for ci := range 4 {
		e0, e1 := signs[ci][0]*hu, signs[ci][1]*hv
		px := fcX + e0*buX + e1*bvX
		py := fcY + e0*buY + e1*bvY
		pz := fcZ + e0*buZ + e1*bvZ
		// Alternate projection between horizontal cylinder disk and OBB face rectangle so overhanging faces clip to disk edge.
		for range 3 {
			vx, vy := px-cx, py-cy
			if d := math.Hypot(vx, vy); d > cR {
				inv := cR / d
				px = cx + vx*inv
				py = cy + vy*inv
			}
			pz = fcZ - (fnX*(px-fcX)+fnY*(py-fcY))/fnZ
			rx, ry, rz := px-fcX, py-fcY, pz-fcZ
			du := math.Max(-hu, math.Min(hu, rx*buX+ry*buY+rz*buZ))
			dv := math.Max(-hv, math.Min(hv, rx*bvX+ry*bvY+rz*bvZ))
			px = fcX + du*buX + dv*bvX
			py = fcY + du*buY + dv*bvY
			pz = fcZ + du*buZ + dv*bvZ
		}
		if (px-cx)*(px-cx)+(py-cy)*(py-cy) > (cR+1e-2)*(cR+1e-2) {
			continue
		}
		cornerPen := signZ * (pz - zCap)
		if cornerPen <= 0.0 {
			continue
		}
		if cornerPen > pen {
			cornerPen = pen
		}
		midZ := 0.5 * (zCap + pz)
		candCount = om.candidates.RemoveDuplicate(candCount, cornerPen, px, py, midZ)
	}

	om.candidates.FinalizeInto(om.points, candCount, fx, fy, fz, pen)
}

// evalCylinderOBB tests a vertical cylinder against an oriented box and returns SAT contact or separation data.
// It reports penetration depth and normal on overlap, or positive gap and separating normal otherwise.
func evalCylinderOBB(cx, cy, cz, cR, cHZ, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) (hit bool, pen, gap, nx, ny, nz float64) {
	st := cylOBBSatState{
		dx: cx - bx, dy: cy - by, dz: cz - bz,
		cR: cR, cHZ: cHZ,
		bAxes: bAxes, bhx: bhx, bhy: bhy, bhz: bhz,
		maxGap: -1.0,
		minPen: math.MaxFloat64,
		gz:     1.0,
		nz:     1.0,
	}
	st.TestAxis(0.0, 0.0, 1.0)
	for i := range 3 {
		ax, ay, az := bAxes.GetRow(i)
		st.TestAxis(ax, ay, az)
		st.TestAxis(-ay, ax, 0.0)
		st.TestAxis(ax, ay, 0.0)
	}

	sBest, qx, qy, qz := closestOBBVerticalSegment(cx, cy, cz, cHZ, bx, by, bz, bAxes, bhx, bhy, bhz)
	st.TestAxis(cx-qx, cy-qy, 0.0)
	st.TestAxis(cx-qx, cy-qy, (cz+sBest)-qz)

	// Alternating projection steps refine skew 3D OBB edge vs circular cylinder rim separating directions.
	curQX, curQY, curQZ := qx, qy, qz
	for range 3 {
		vx, vy := curQX-cx, curQY-cy
		px, py := curQX, curQY
		if vxy := math.Hypot(vx, vy); vxy > cR {
			inv := cR / vxy
			px, py = cx+vx*inv, cy+vy*inv
		}
		pz := math.Max(cz-cHZ, math.Min(cz+cHZ, curQZ))
		st.TestAxis(px-curQX, py-curQY, pz-curQZ)
		curQX, curQY, curQZ = closestPointOnOBBWorld(px, py, pz, bx, by, bz, bAxes, bhx, bhy, bhz)
		st.TestAxis(px-curQX, py-curQY, pz-curQZ)
	}

	if st.separated {
		return false, 0.0, math.Max(0.0, st.maxGap), st.gx, st.gy, st.gz
	}
	return true, st.minPen, 0.0, st.nx, st.ny, st.nz
}

// closestPointOnOBBWorld returns the closest world-space point on an oriented bounding box to the given point.
// It clamps the point's projection onto each box axis to the corresponding half-extent.
func closestPointOnOBBWorld(px, py, pz, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) (float64, float64, float64) {
	rx, ry, rz := px-bx, py-by, pz-bz
	h := [3]float64{bhx, bhy, bhz}
	qx, qy, qz := bx, by, bz
	for k := range 3 {
		ax, ay, az := bAxes.GetRow(k)
		d := rx*ax + ry*ay + rz*az
		if d < -h[k] {
			d = -h[k]
		} else if d > h[k] {
			d = h[k]
		}
		qx += d * ax
		qy += d * ay
		qz += d * az
	}
	return qx, qy, qz
}

// evalOBBVertSegDistSq returns the squared distance from a segment point to an OBB in local coordinates.
// The segment point is evaluated as u+s*ez and clamped against half-extents h on each axis.
func evalOBBVertSegDistSq(u, ez, h [3]float64, s float64) float64 {
	sum := 0.0
	for k := range 3 {
		p := u[k] + s*ez[k]
		if p < -h[k] {
			d := p + h[k]
			sum += d * d
		} else if p > h[k] {
			d := p - h[k]
			sum += d * d
		}
	}
	return sum
}

// closestOBBVerticalSegment finds the closest point pair between a vertical segment and an oriented box.
// It returns the segment offset and the closest point on the box in world coordinates.
func closestOBBVerticalSegment(cx, cy, cz, cHZ, bx, by, bz float64, bAxes *Axes, bhx, bhy, bhz float64) (float64, float64, float64, float64) {
	rx, ry, rz := cx-bx, cy-by, cz-bz
	var u, ez [3]float64
	h := [3]float64{bhx, bhy, bhz}
	for k := range 3 {
		ax, ay, az := bAxes.GetRow(k)
		u[k] = rx*ax + ry*ay + rz*az
		ez[k] = az
	}
	var pts [8]float64
	pts[0] = -cHZ
	pts[1] = cHZ
	n := 2
	for k := range 3 {
		if math.Abs(ez[k]) <= 1e-12 {
			continue
		}
		inv := 1.0 / ez[k]
		t0 := (-h[k] - u[k]) * inv
		if t0 > -cHZ && t0 < cHZ {
			pts[n] = t0
			n++
		}
		t1 := (h[k] - u[k]) * inv
		if t1 > -cHZ && t1 < cHZ {
			pts[n] = t1
			n++
		}
	}
	for i := 1; i < n; i++ {
		v := pts[i]
		j := i - 1
		for j >= 0 && pts[j] > v {
			pts[j+1] = pts[j]
			j--
		}
		pts[j+1] = v
	}

	sBest := -cHZ
	bestSq := evalOBBVertSegDistSq(u, ez, h, sBest)
	if dEnd := evalOBBVertSegDistSq(u, ez, h, cHZ); dEnd < bestSq {
		bestSq = dEnd
		sBest = cHZ
	}
	for i := 0; i < n-1; i++ {
		a, b := pts[i], pts[i+1]
		if b-a <= 1e-12 {
			continue
		}
		mid := 0.5 * (a + b)
		num, denom := 0.0, 0.0
		for k := range 3 {
			p := u[k] + mid*ez[k]
			if p < -h[k] {
				num -= ez[k] * (u[k] + h[k])
				denom += ez[k] * ez[k]
			} else if p > h[k] {
				num -= ez[k] * (u[k] - h[k])
				denom += ez[k] * ez[k]
			}
		}
		sCand := mid
		if denom > 1e-12 {
			sCand = num / denom
			if sCand < a {
				sCand = a
			} else if sCand > b {
				sCand = b
			}
		}
		if d := evalOBBVertSegDistSq(u, ez, h, sCand); d < bestSq {
			bestSq = d
			sBest = sCand
		}
	}

	qx, qy, qz := bx, by, bz
	for k := range 3 {
		ax, ay, az := bAxes.GetRow(k)
		qk := u[k] + sBest*ez[k]
		if qk < -h[k] {
			qk = -h[k]
		} else if qk > h[k] {
			qk = h[k]
		}
		qx += qk * ax
		qy += qk * ay
		qz += qk * az
	}
	return sBest, qx, qy, qz
}

// sweepCylinderInterval finds the first normalized time two swept vertical cylinders overlap.
// It returns whether an interval exists and whether entry is governed by the Z cap constraint.
func sweepCylinderInterval(rel0X, rel0Y, rel0Z, relVX, relVY, relVZ, sumR, sumZ float64) (tEnter float64, ok bool, fromZCap bool) {
	var tz0, tz1 float64
	if math.Abs(relVZ) <= 1e-12 {
		if math.Abs(rel0Z) > sumZ+1e-9 {
			return 0, false, false
		}
		tz0, tz1 = -math.MaxFloat64, math.MaxFloat64
	} else {
		t1 := (-sumZ - rel0Z) / relVZ
		t2 := (sumZ - rel0Z) / relVZ
		if t1 < t2 {
			tz0, tz1 = t1, t2
		} else {
			tz0, tz1 = t2, t1
		}
	}

	var tr0, tr1 float64
	a := relVX*relVX + relVY*relVY
	b := rel0X*relVX + rel0Y*relVY
	c := rel0X*rel0X + rel0Y*rel0Y - sumR*sumR
	if a <= 1e-18 {
		if c > 1e-9 {
			return 0, false, false
		}
		tr0, tr1 = -math.MaxFloat64, math.MaxFloat64
	} else {
		disc := b*b - a*c
		if disc < 0.0 {
			return 0, false, false
		}
		sqrtD := math.Sqrt(disc)
		tr0 = (-b - sqrtD) / a
		tr1 = (-b + sqrtD) / a
	}

	tStart := math.Max(0.0, math.Max(tz0, tr0))
	tEnd := math.Min(1.0, math.Min(tz1, tr1))
	if tStart > tEnd+1e-9 || tEnd < -1e-9 || tStart > 1.0+1e-9 {
		return 0, false, false
	}
	if tEnd <= tStart+1e-12 {
		return 0, false, false
	}
	if tStart < 0.0 {
		tStart = 0.0
	}
	return tStart, true, tz0 > tr0
}

// sweptCylinderNormal returns the contact normal for a swept cylinder hit at the given relative position and velocity.
// It selects an axial cap normal when fromZCap is true, otherwise a radial normal with velocity fallback.
func sweptCylinderNormal(rx, ry, rz, vx, vy, vz float64, fromZCap bool) (float64, float64, float64) {
	if fromZCap {
		if rz > 0.0 || (rz == 0.0 && vz < 0.0) {
			return 0.0, 0.0, 1.0
		}
		return 0.0, 0.0, -1.0
	}
	if d := math.Hypot(rx, ry); d > 1e-12 {
		return rx / d, ry / d, 0.0
	}
	if dv := math.Hypot(vx, vy); dv > 1e-12 {
		return -vx / dv, -vy / dv, 0.0
	}
	return 1.0, 0.0, 0.0
}
