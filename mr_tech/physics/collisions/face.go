package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Face wraps an IFace implementation for triangle face geometry and collision queries.
type Face struct {
	IFace
}

// NewFace creates and returns a new Face instance.
func NewFace() *Face {
	return &Face{}
}

func (s *Face) SetIFace(face IFace) {
	s.IFace = face
}

// OverlapsTriangleLocalOBB reports whether the face triangle overlaps the entity's local oriented box at the given center.
func (s *Face) OverlapsTriangleLocalOBB(ent *rigid_body.Entity, cx, cy, cz float64) bool {
	rFace := s.IFace
	hx, hy, hz := ent.GetSizeCenter()
	if hx <= 0.0 || hy <= 0.0 || hz <= 0.0 {
		return true
	}
	ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := ent.GetAxes()
	v0x, v0y, v0z := rFace.Get0()
	v1x, v1y, v1z := rFace.Get1()
	v2x, v2y, v2z := rFace.Get2()

	d0x, d0y, d0z := v0x-cx, v0y-cy, v0z-cz
	d1x, d1y, d1z := v1x-cx, v1y-cy, v1z-cz
	d2x, d2y, d2z := v2x-cx, v2y-cy, v2z-cz

	p0x := d0x*ax0 + d0y*ay0 + d0z*az0
	p0y := d0x*ax1 + d0y*ay1 + d0z*az1
	p0z := d0x*ax2 + d0y*ay2 + d0z*az2

	p1x := d1x*ax0 + d1y*ay0 + d1z*az0
	p1y := d1x*ax1 + d1y*ay1 + d1z*az1
	p1z := d1x*ax2 + d1y*ay2 + d1z*az2

	p2x := d2x*ax0 + d2y*ay0 + d2z*az0
	p2y := d2x*ax1 + d2y*ay1 + d2z*az1
	p2z := d2x*ax2 + d2y*ay2 + d2z*az2

	if math.Min(p0x, math.Min(p1x, p2x)) > hx || math.Max(p0x, math.Max(p1x, p2x)) < -hx {
		return false
	}
	if math.Min(p0y, math.Min(p1y, p2y)) > hy || math.Max(p0y, math.Max(p1y, p2y)) < -hy {
		return false
	}
	if math.Min(p0z, math.Min(p1z, p2z)) > hz || math.Max(p0z, math.Max(p1z, p2z)) < -hz {
		return false
	}

	e0x, e0y, e0z := p1x-p0x, p1y-p0y, p1z-p0z
	e1x, e1y, e1z := p2x-p1x, p2y-p1y, p2z-p1z
	e2x, e2y, e2z := p0x-p2x, p0y-p2y, p0z-p2z

	tnx := e0y*e1z - e0z*e1y
	tny := e0z*e1x - e0x*e1z
	tnz := e0x*e1y - e0y*e1x
	if math.Abs(p0x*tnx+p0y*tny+p0z*tnz) > hx*math.Abs(tnx)+hy*math.Abs(tny)+hz*math.Abs(tnz) {
		return false
	}

	return overlapsTriangleEdgeAxes(hx, hy, hz, p0x, p0y, p0z, p1x, p1y, p1z, p2x, p2y, p2z, e0x, e0y, e0z) &&
		overlapsTriangleEdgeAxes(hx, hy, hz, p0x, p0y, p0z, p1x, p1y, p1z, p2x, p2y, p2z, e1x, e1y, e1z) &&
		overlapsTriangleEdgeAxes(hx, hy, hz, p0x, p0y, p0z, p1x, p1y, p1z, p2x, p2y, p2z, e2x, e2y, e2z)
}

/*
func overlapsTriangleEdgeAxes(hx, hy, hz, p0x, p0y, p0z, p1x, p1y, p1z, p2x, p2y, p2z, ex, ey, ez float64) bool {
	absX, absY, absZ := math.Abs(ex), math.Abs(ey), math.Abs(ez)

	if rad := hy*absZ + hz*absY; rad > 1e-12 {
		a0 := -p0y*ez + p0z*ey
		a1 := -p1y*ez + p1z*ey
		a2 := -p2y*ez + p2z*ey
		if math.Min(a0, math.Min(a1, a2)) > rad || math.Max(a0, math.Max(a1, a2)) < -rad {
			return false
		}
	}
	if rad := hx*absZ + hz*absX; rad > 1e-12 {
		a0 := p0x*ez - p0z*ex
		a1 := p1x*ez - p1z*ex
		a2 := p2x*ez - p2z*ex
		if math.Min(a0, math.Min(a1, a2)) > rad || math.Max(a0, math.Max(a1, a2)) < -rad {
			return false
		}
	}
	if rad := hx*absY + hy*absX; rad > 1e-12 {
		a0 := -p0x*ey + p0y*ex
		a1 := -p1x*ey + p1y*ex
		a2 := -p2x*ey + p2y*ex
		if math.Min(a0, math.Min(a1, a2)) > rad || math.Max(a0, math.Max(a1, a2)) < -rad {
			return false
		}
	}
	return true
}

*/

// IntersectsFiniteTriangleOBB reports whether the swept finite triangle intersects the entity's oriented bounding box.
// It tests the target pose and the interpolated plane-crossing pose when the sweep crosses the face plane.
func (s *Face) IntersectsFiniteTriangleOBB(ent *rigid_body.Entity, cX, cY, cZ, dX, dY, dZ, tX, tY, tZ, distStart, distTarget float64) bool {
	if s.OverlapsTriangleLocalOBB(ent, tX, tY, tZ) {
		return true
	}
	if distStart >= -satFilterEpsilon && distTarget < 0.0 {
		tauPlane := 0.5
		if denom := distStart - distTarget; denom > 1e-12 {
			tauPlane = math.Max(0.0, math.Min(1.0, distStart/denom))
		}
		if s.OverlapsTriangleLocalOBB(ent, cX+tauPlane*dX, cY+tauPlane*dY, cZ+tauPlane*dZ) {
			return true
		}
	}
	return false
}

// IntersectsFiniteTriangle3D reports whether the finite swept volume intersects this face's triangle in 3D.
// It tests against ellipsoid-scaled triangle geometry and optional plane-crossing contact.
// Inactive triangle edge features are ignored unless allowInactiveEdge is true.
func (s *Face) IntersectsFiniteTriangle3D(eRadX, eRadY, eRadZ, cX, cY, cZ, dX, dY, dZ, tX, tY, tZ, distStart, distTarget float64, allowInactiveEdge bool) bool {
	if eRadX <= 0.0 || eRadY <= 0.0 || eRadZ <= 0.0 {
		return true
	}
	rFace := s.IFace
	activeMask := rFace.ActiveEdgeMask()
	v0x, v0y, v0z := rFace.Get0()
	v1x, v1y, v1z := rFace.Get1()
	v2x, v2y, v2z := rFace.Get2()
	invRX := 1.0 / eRadX
	invRY := 1.0 / eRadY
	invRZ := 1.0 / eRadZ

	qx, qy, qz, featT := closestPointOnTriangleFeature((v0x-tX)*invRX, (v0y-tY)*invRY, (v0z-tZ)*invRZ, (v1x-tX)*invRX, (v1y-tY)*invRY, (v1z-tZ)*invRZ, (v2x-tX)*invRX, (v2y-tY)*invRY, (v2z-tZ)*invRZ)
	if qx*qx+qy*qy+qz*qz < 1.0-1e-9 {
		if allowInactiveEdge || !isInactiveTriangleFeature(featT, activeMask) {
			return true
		}
	}

	if distStart >= -satFilterEpsilon && distTarget < 0.0 {
		tauPlane := 0.5
		if denom := distStart - distTarget; denom > 1e-12 {
			tauPlane = math.Max(0.0, math.Min(1.0, distStart/denom))
		}
		px := cX + tauPlane*dX
		py := cY + tauPlane*dY
		pz := cZ + tauPlane*dZ
		qpx, qpy, qpz, featPlane := closestPointOnTriangleFeature((v0x-px)*invRX, (v0y-py)*invRY, (v0z-pz)*invRZ, (v1x-px)*invRX, (v1y-py)*invRY, (v1z-pz)*invRZ, (v2x-px)*invRX, (v2y-py)*invRY, (v2z-pz)*invRZ)
		if qpx*qpx+qpy*qpy+qpz*qpz < 1.0-1e-9 {
			if allowInactiveEdge || !isInactiveTriangleFeature(featPlane, activeMask) {
				return true
			}
		}
	}

	_, _, _, _, _, _, segSq, _, pierced, segFeat := closestSegmentToTriangleFeature(dX*invRX, dY*invRY, dZ*invRZ, (v0x-cX)*invRX, (v0y-cY)*invRY, (v0z-cZ)*invRZ, (v1x-cX)*invRX, (v1y-cY)*invRY, (v1z-cZ)*invRZ, (v2x-cX)*invRX, (v2y-cY)*invRY, (v2z-cZ)*invRZ)
	if segSq >= 1.0-1e-9 {
		return false
	}
	return allowInactiveEdge || pierced || !isInactiveTriangleFeature(segFeat, activeMask)
}

// EvalKeepUprightFace evaluates an upright ellipsoid face contact and optional edge or corner adjustment.
// It reports whether the face is valid, whether the response was adjusted, and any contact point.
func (s *Face) EvalKeepUprightFace(ent *rigid_body.Entity, bucket BucketType, solidWE, solidNS bool, eRadX, eRadY, eRadZ, cX, cY, cZ, dX, dY, dZ, tX, tY, tZ, nX, nY, nZ, distStart, distTarget, penetration float64) (valid, adjusted, cornerHit bool, cNX, cNY, cNZ, cPen, qxW, qyW, qzW float64, hasPt bool) {
	if eRadX <= 0.0 || eRadY <= 0.0 || eRadZ <= 0.0 {
		return false, false, false, 0, 0, 0, 0, 0, 0, 0, false
	}
	rFace := s.IFace
	invRX := 1.0 / eRadX
	invRY := 1.0 / eRadY
	invRZ := 1.0 / eRadZ
	activeMask := rFace.ActiveEdgeMask()

	v0x, v0y, v0z := rFace.Get0()
	v1x, v1y, v1z := rFace.Get1()
	v2x, v2y, v2z := rFace.Get2()

	qx, qy, qz, feat := closestPointOnTriangleFeature((v0x-tX)*invRX, (v0y-tY)*invRY, (v0z-tZ)*invRZ, (v1x-tX)*invRX, (v1y-tY)*invRY, (v1z-tZ)*invRZ, (v2x-tX)*invRX, (v2y-tY)*invRY, (v2z-tZ)*invRZ)
	bestSq := qx*qx + qy*qy + qz*qz
	inactiveFeat := isInactiveTriangleFeature(feat, activeMask)
	crossedPlane := distStart >= -satFilterEpsilon && distTarget < 0.0
	tunneledWall := (solidWE || solidNS) && distStart >= effectiveRadius(ent, eRadX, eRadY, eRadZ, nX, nY, nZ)-satFilterEpsilon && distTarget < 0.0

	if bestSq >= 1.0-1e-9 {
		var rx, ry, rz float64
		var checkGraze, resolvedPlane bool
		if crossedPlane {
			tauPlane := 0.5
			if denom := distStart - distTarget; denom > 1e-12 {
				tauPlane = math.Max(0.0, math.Min(1.0, distStart/denom))
			}
			px := cX + tauPlane*dX
			py := cY + tauPlane*dY
			pz := cZ + tauPlane*dZ
			qpx, qpy, qpz, featPlane := closestPointOnTriangleFeature((v0x-px)*invRX, (v0y-py)*invRY, (v0z-pz)*invRZ, (v1x-px)*invRX, (v1y-py)*invRY, (v1z-pz)*invRZ, (v2x-px)*invRX, (v2y-py)*invRY, (v2z-pz)*invRZ)
			if planeSq := qpx*qpx + qpy*qpy + qpz*qpz; planeSq < 1.0-1e-9 {
				resolvedPlane = true
				qxW, qyW, qzW, hasPt = px+qpx*eRadX, py+qpy*eRadY, pz+qpz*eRadZ, true
				if planeSq > 1e-12 && !isInactiveTriangleFeature(featPlane, activeMask) {
					rx, ry, rz = qpx, qpy, qpz
					checkGraze = true
				}
			} else if tunneledWall {
				return false, false, false, 0, 0, 0, 0, 0, 0, 0, false
			}
		}
		if !resolvedPlane {
			if !crossedPlane && dX*dX+dY*dY+dZ*dZ <= 1e-12 {
				return false, false, false, 0, 0, 0, 0, 0, 0, 0, false
			}
			sx, sy, sz, sqx, sqy, sqz, segSq, startSq, pierced, segFeat := closestSegmentToTriangleFeature(dX*invRX, dY*invRY, dZ*invRZ, (v0x-cX)*invRX, (v0y-cY)*invRY, (v0z-cZ)*invRZ, (v1x-cX)*invRX, (v1y-cY)*invRY, (v1z-cZ)*invRZ, (v2x-cX)*invRX, (v2y-cY)*invRY, (v2z-cZ)*invRZ)
			if segSq >= 1.0-1e-9 || (startSq < 1.0-1e-9 && !(crossedPlane && pierced)) {
				return false, false, false, 0, 0, 0, 0, 0, 0, 0, false
			}
			qxW, qyW, qzW, hasPt = cX+sqx*eRadX, cY+sqy*eRadY, cZ+sqz*eRadZ, true
			if !pierced {
				if isInactiveTriangleFeature(segFeat, activeMask) {
					if !crossedPlane {
						return false, false, false, 0, 0, 0, 0, 0, 0, 0, false
					}
				} else if segSq > 1e-12 {
					rx, ry, rz = sqx-sx, sqy-sy, sqz-sz
					checkGraze = true
				}
			}
		}
		if checkGraze {
			if rXY := math.Hypot(rx, ry); rXY > 1e-6 {
				if rZ := math.Sqrt(math.Max(0.0, 1.0-rz*rz)); rZ > rXY {
					scaleH := rZ/rXY - 1.0
					dxH := -rx * scaleH * eRadX
					dyH := -ry * scaleH * eRadY
					if horizPen := math.Hypot(dxH, dyH); horizPen > 1e-12 && horizPen <= satFilterEpsilon {
						hx, hy := dxH/horizPen, dyH/horizPen
						nXY := math.Hypot(nX, nY)
						if nXY <= 1e-9 || (hx*(nX/nXY)+hy*(nY/nXY)) < 0.1 {
							return true, true, true, hx, hy, 0.0, horizPen, qxW, qyW, qzW, true
						}
					}
				}
			}
		}
		return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, hasPt
	}

	qxW, qyW, qzW, hasPt = tX+qx*eRadX, tY+qy*eRadY, tZ+qz*eRadZ, true
	if bestSq <= 1e-12 || inactiveFeat {
		return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, true
	}

	qLen := math.Sqrt(bestSq)
	scale := 1.0/qLen - 1.0
	dx := -qx * scale * eRadX
	dy := -qy * scale * eRadY
	dz := -qz * scale * eRadZ
	pen3D := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if pen3D <= 1e-12 {
		return false, false, false, 0, 0, 0, 0, 0, 0, 0, false
	}

	// If the sweep crossed the triangle interior at tauPlane and merely drifted to an active edge at t, keep face normal.
	if crossedPlane && feat != 0 {
		tauPlane := 0.5
		if denom := distStart - distTarget; denom > 1e-12 {
			tauPlane = math.Max(0.0, math.Min(1.0, distStart/denom))
		}
		px := cX + tauPlane*dX
		py := cY + tauPlane*dY
		pz := cZ + tauPlane*dZ
		qpx, qpy, qpz, featPlane := closestPointOnTriangleFeature((v0x-px)*invRX, (v0y-py)*invRY, (v0z-pz)*invRZ, (v1x-px)*invRX, (v1y-py)*invRY, (v1z-pz)*invRZ, (v2x-px)*invRX, (v2y-py)*invRY, (v2z-pz)*invRZ)
		if qpx*qpx+qpy*qpy+qpz*qpz < 1.0-1e-9 && featPlane == 0 {
			return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, true
		}
	}

	onBoundary := feat != 0
	if !solidWE && !solidNS {
		// Floor/ceiling: separate horizontally when grazing mid-body ledge or wrong-side overhang edge,
		// and use exact 3D separation on active sloped edges to avoid infinite-plane vertical catapults.
		if qXY := math.Hypot(qx, qy); !crossedPlane && qXY > 1e-6 && math.Abs(qz) < 1.0-1e-9 {
			if rZ := math.Sqrt(math.Max(0.0, 1.0-qz*qz)); rZ > qXY {
				scaleH := rZ/qXY - 1.0
				dxH := -qx * scaleH * eRadX
				dyH := -qy * scaleH * eRadY
				if penHoriz := math.Hypot(dxH, dyH); penHoriz > 1e-12 {
					wrongSideOfPlane := (bucket == BucketCeiling && qz < 0.0) || (bucket == BucketFloor && qz > 0.0)
					midBodyEdge := onBoundary && math.Abs(qz) < 0.707 && penHoriz < penetration*0.85
					if (wrongSideOfPlane && penHoriz <= pen3D*2.0) || midBodyEdge {
						return true, true, onBoundary, dxH / penHoriz, dyH / penHoriz, 0.0, penHoriz, qxW, qyW, qzW, true
					}
				}
			}
		}
		if math.Abs(nZ) < 0.999 && (onBoundary || pen3D < penetration*0.75) {
			nx3, ny3, nz3 := dx/pen3D, dy/pen3D, dz/pen3D
			if pen3D < penetration-satFilterEpsilon || (nx3*nX+ny3*nY+nz3*nZ) > 0.0 {
				return true, true, onBoundary, nx3, ny3, nz3, pen3D, qxW, qyW, qzW, true
			}
		} else if onBoundary && !crossedPlane {
			// On flat/near-flat floor/ceiling boundary edges, use curved ellipsoid rim separation
			// instead of infinite-plane depth so balanced upright bodies settle smoothly without catapulting.
			nx3, ny3, nz3 := dx/pen3D, dy/pen3D, dz/pen3D
			sameVerticalBucket := (bucket == BucketFloor && nz3 >= floorNormalThreshold) || (bucket == BucketCeiling && nz3 <= -floorNormalThreshold)
			if sameVerticalBucket && pen3D < penetration {
				return true, true, onBoundary, nx3, ny3, nz3, pen3D, qxW, qyW, qzW, true
			}
			if qXY := math.Hypot(qx, qy); qXY > 1e-6 && qXY < 1.0-1e-9 {
				var rimPen float64
				if bucket == BucketFloor && qz < 0.0 {
					rimPen = (qz + math.Sqrt(math.Max(0.0, 1.0-qXY*qXY))) * eRadZ
				} else if bucket == BucketCeiling && qz > 0.0 {
					rimPen = (math.Sqrt(math.Max(0.0, 1.0-qXY*qXY)) - qz) * eRadZ
				}
				if rimPen > 0.0 && rimPen < penetration {
					return true, true, onBoundary, nX, nY, nZ, rimPen, qxW, qyW, qzW, true
				}
			}
		}
		return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, true
	}

	// Wall: separate horizontally around active vertical/bevel corners and cap ends unless already flat-aligned.
	qXY := math.Hypot(qx, qy)
	rZ := math.Sqrt(math.Max(0.0, 1.0-qz*qz))
	if qXY <= 1e-6 || rZ <= qXY {
		return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, true
	}
	scaleH := rZ/qXY - 1.0
	dxH := -qx * scaleH * eRadX
	dyH := -qy * scaleH * eRadY
	horizPen := math.Hypot(dxH, dyH)
	if horizPen <= 1e-12 {
		return false, false, false, 0, 0, 0, 0, 0, 0, 0, false
	}
	hx, hy := dxH/horizPen, dyH/horizPen
	dot := 1.0
	if nXY := math.Hypot(nX, nY); nXY > 1e-9 {
		dot = hx*(nX/nXY) + hy*(nY/nXY)
	}
	if dot < -0.5 {
		return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, true
	}
	if horizPen >= penetration-1e-4 && dot > 0.999 {
		return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, true
	}
	rMinZ := math.Min(v0z, math.Min(v1z, v2z))
	rMaxZ := math.Max(v0z, math.Max(v1z, v2z))
	if rMaxZ < tZ && qz < -floorNormalThreshold {
		nx3, ny3, nz3 := dx/pen3D, dy/pen3D, dz/pen3D
		if nz3 >= floorNormalThreshold && horizPen > pen3D {
			return true, true, onBoundary, nx3, ny3, nz3, pen3D, qxW, qyW, qzW, true
		}
	}
	if (rMinZ > tZ || rMaxZ < tZ || math.Abs(nZ) > 1e-3) && horizPen >= penetration*1.5 && dot > 0.7 {
		return true, false, false, 0, 0, 0, 0, qxW, qyW, qzW, true
	}
	return true, true, onBoundary, hx, hy, 0.0, horizPen, qxW, qyW, qzW, true
}
