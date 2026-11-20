package rigid_body

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// anchorWorld returns the world-space anchor point and its offset from the entity's center of mass.
func (e *Entity) anchorWorld(localX, localY, localZ float64) (px, py, pz, rx, ry, rz float64) {
	gx, gy, gz := e.BoundingBox.GetCenter()
	ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := e.BoundingBox.GetAxes()
	px = gx + localX*ax0 + localY*ax1 + localZ*ax2
	py = gy + localX*ay0 + localY*ay1 + localZ*ay2
	pz = gz + localX*az0 + localY*az1 + localZ*az2
	cx, cy, cz := e.getCenterOfMass()
	rx = px - cx
	ry = py - cy
	rz = pz - cz
	return px, py, pz, rx, ry, rz
}

// WarmStartJoint reapplies accumulated ball-socket joint impulses between two entities at their local anchors.
func (e *Entity) WarmStartJoint(other *Entity, ax, ay, az, bx, by, bz float64, accJx, accJy, accJz *float64) {
	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
	if (invMassA == 0.0 && invMassB == 0.0) || e.Cinematic.GetDt() <= 0.0 {
		return
	}
	_, _, _, rAx, rAy, rAz := e.anchorWorld(ax, ay, az)
	_, _, _, rBx, rBy, rBz := other.anchorWorld(bx, by, bz)

	const warmFactor = 1.0
	*accJx *= warmFactor
	*accJy *= warmFactor
	*accJz *= warmFactor

	e.applyBallSocketImpulse(other, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, *accJx, invMassA, invMassB)
	e.applyBallSocketImpulse(other, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, *accJy, invMassA, invMassB)
	e.applyBallSocketImpulse(other, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, *accJz, invMassA, invMassB)
}

// ResolveJointVelocity applies velocity-level joint constraints between two entities at their local anchor points.
// It enforces ball-socket motion, optional angular damping, and an optional cone limit relative to restQ.
func (e *Entity) ResolveJointVelocity(
	other *Entity,
	ax, ay, az, bx, by, bz float64,
	restQ *aabb.Quaternion,
	maxAngle, damping float64,
	accJx, accJy, accJz *float64,
	qErrScratch, qTargetScratch *aabb.Quaternion,
) {
	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
	if invMassA == 0.0 && invMassB == 0.0 {
		return
	}

	_, _, _, rAx, rAy, rAz := e.anchorWorld(ax, ay, az)
	_, _, _, rBx, rBy, rBz := other.anchorWorld(bx, by, bz)

	dt := e.Cinematic.GetDt()
	if dt <= 0.0 {
		return
	}
	// Positional error is eliminated by ResolveJointPosition; zero velocity bias avoids injecting phantom kinetic energy at rest.
	e.solveBallSocketAxis(other, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, invMassA, invMassB, accJx)
	e.solveBallSocketAxis(other, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, invMassA, invMassB, accJy)
	e.solveBallSocketAxis(other, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, invMassA, invMassB, accJz)

	// Viscous relative angular damping inside the joint socket so ragdolls settle naturally.
	if damping > 0.0 {
		rwx := other.Cinematic.wx - e.Cinematic.wx
		rwy := other.Cinematic.wy - e.Cinematic.wy
		rwz := other.Cinematic.wz - e.Cinematic.wz
		rwSq := rwx*rwx + rwy*rwy + rwz*rwz
		if rwSq > 1e-10 {
			rwLen := math.Sqrt(rwSq)
			dx, dy, dz := rwx/rwLen, rwy/rwLen, rwz/rwLen
			uAx, uAy, uAz := e.Cinematic.applyWorldInvInertia(dx, dy, dz)
			uBx, uBy, uBz := other.Cinematic.applyWorldInvInertia(dx, dy, dz)
			kAng := (dx*uAx + dy*uAy + dz*uAz) + (dx*uBx + dy*uBy + dz*uBz)
			if kAng > 1e-12 {
				frac := math.Min(damping*dt*0.50, 0.75)
				jDamp := (rwLen * frac) / kAng
				e.applyClampedAngularDelta(jDamp*uAx, jDamp*uAy, jDamp*uAz)
				other.applyClampedAngularDelta(-jDamp*uBx, -jDamp*uBy, -jDamp*uBz)
			}
		}
	}

	// Unilateral angular cone limit around restQ.
	if maxAngle > 0.0 && restQ != nil {
		qA := e.BoundingBox.GetQRotation()
		qB := other.BoundingBox.GetQRotation()
		qA.Multiplies(restQ, qTargetScratch)
		qErrScratch.Conjugate(qTargetScratch)
		qErrScratch.MultipliesSelf(qB)

		ex, ey, ez, ew := qErrScratch.Get()
		if ew < 0.0 {
			ex, ey, ez, ew = -ex, -ey, -ez, -ew
		}
		vSq := ex*ex + ey*ey + ez*ez
		if vSq > 1e-12 {
			vLen := math.Sqrt(vSq)
			angle := 2.0 * math.Atan2(vLen, ew)
			if angle > maxAngle {
				axW, ayW, azW := qTargetScratch.Rotate(ex/vLen, ey/vLen, ez/vLen)
				uAx, uAy, uAz := e.Cinematic.applyWorldInvInertia(axW, ayW, azW)
				uBx, uBy, uBz := other.Cinematic.applyWorldInvInertia(axW, ayW, azW)
				kAng := (axW*uAx + ayW*uAy + azW*uAz) + (axW*uBx + ayW*uBy + azW*uBz)
				if kAng > 1e-12 {
					wRel := (other.Cinematic.wx-e.Cinematic.wx)*axW +
						(other.Cinematic.wy-e.Cinematic.wy)*ayW +
						(other.Cinematic.wz-e.Cinematic.wz)*azW
					jAng := -wRel / kAng
					if jAng < 0.0 {
						e.applyClampedAngularDelta(-jAng*uAx, -jAng*uAy, -jAng*uAz)
						other.applyClampedAngularDelta(jAng*uBx, jAng*uBy, jAng*uBz)
					}
				}
			}
		}
	}
}

// applyBallSocketImpulse applies a scalar ball-socket constraint impulse to both entities along the given axis.
// It updates linear velocity and clamped angular velocity deltas using each body's inverse mass.
func (e *Entity) applyBallSocketImpulse(other *Entity, rAx, rAy, rAz, rBx, rBy, rBz float64, nx, ny, nz, j, invMassA, invMassB float64) {
	if j == 0.0 {
		return
	}
	if invMassA > 0.0 {
		uAx, uAy, uAz, _ := e.Cinematic.rotationalMass(rAx, rAy, rAz, nx, ny, nz)
		e.Cinematic.vx -= j * invMassA * nx
		e.Cinematic.vy -= j * invMassA * ny
		e.Cinematic.vz -= j * invMassA * nz
		e.applyClampedAngularDelta(-j*uAx, -j*uAy, -j*uAz)
	}
	if invMassB > 0.0 {
		uBx, uBy, uBz, _ := other.Cinematic.rotationalMass(rBx, rBy, rBz, nx, ny, nz)
		other.Cinematic.vx += j * invMassB * nx
		other.Cinematic.vy += j * invMassB * ny
		other.Cinematic.vz += j * invMassB * nz
		other.applyClampedAngularDelta(j*uBx, j*uBy, j*uBz)
	}
}

// solveBallSocketAxis applies an impulse along one axis to cancel relative velocity at a ball-socket joint.
// It updates linear and angular velocities for both entities and accumulates the applied impulse.
func (e *Entity) solveBallSocketAxis(other *Entity, rAx, rAy, rAz, rBx, rBy, rBz float64, nx, ny, nz, invMassA, invMassB float64, accJ *float64) {
	vAx := e.Cinematic.vx + (e.Cinematic.wy*rAz - e.Cinematic.wz*rAy)
	vAy := e.Cinematic.vy + (e.Cinematic.wz*rAx - e.Cinematic.wx*rAz)
	vAz := e.Cinematic.vz + (e.Cinematic.wx*rAy - e.Cinematic.wy*rAx)

	vBx := other.Cinematic.vx + (other.Cinematic.wy*rBz - other.Cinematic.wz*rBy)
	vBy := other.Cinematic.vy + (other.Cinematic.wz*rBx - other.Cinematic.wx*rBz)
	vBz := other.Cinematic.vz + (other.Cinematic.wx*rBy - other.Cinematic.wy*rBx)

	relV := (vBx-vAx)*nx + (vBy-vAy)*ny + (vBz-vAz)*nz

	var uAx, uAy, uAz, rotMassA float64
	if invMassA > 0.0 {
		uAx, uAy, uAz, rotMassA = e.Cinematic.rotationalMass(rAx, rAy, rAz, nx, ny, nz)
	}
	var uBx, uBy, uBz, rotMassB float64
	if invMassB > 0.0 {
		uBx, uBy, uBz, rotMassB = other.Cinematic.rotationalMass(rBx, rBy, rBz, nx, ny, nz)
	}

	k := invMassA + rotMassA + invMassB + rotMassB
	if k <= 1e-12 {
		return
	}

	j := -relV / k
	*accJ += j

	if invMassA > 0.0 {
		e.Cinematic.vx -= j * invMassA * nx
		e.Cinematic.vy -= j * invMassA * ny
		e.Cinematic.vz -= j * invMassA * nz
		e.applyClampedAngularDelta(-j*uAx, -j*uAy, -j*uAz)
	}
	if invMassB > 0.0 {
		other.Cinematic.vx += j * invMassB * nx
		other.Cinematic.vy += j * invMassB * ny
		other.Cinematic.vz += j * invMassB * nz
		other.applyClampedAngularDelta(j*uBx, j*uBy, j*uBz)
	}
}

// applyClampedAngularDelta adds angular velocity deltas while respecting upright locks and axis velocity limits.
func (e *Entity) applyClampedAngularDelta(dwx, dwy, dwz float64) {
	if e.Cinematic.keepUpright {
		return
	}
	e.Cinematic.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.Cinematic.wx+dwx))
	e.Cinematic.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.Cinematic.wy+dwy))
	e.Cinematic.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.Cinematic.wz+dwz))
}

// ResolveJointPosition projects a joint constraint between two entities toward its rest anchors and angular limit.
func (e *Entity) ResolveJointPosition(other *Entity, ax, ay, az, bx, by, bz float64, restQ *aabb.Quaternion, maxAngle float64, qErrScratch, qTargetScratch *aabb.Quaternion) {
	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
	totalInvMass := invMassA + invMassB
	if totalInvMass <= 0.0 {
		return
	}

	// Project angular cone limits first so subsequent anchor projection closes any COM-rotation lever offset.
	if maxAngle > 0.0 && restQ != nil {
		qA := e.BoundingBox.GetQRotation()
		qB := other.BoundingBox.GetQRotation()
		qA.Multiplies(restQ, qTargetScratch)
		qErrScratch.Conjugate(qTargetScratch)
		qErrScratch.MultipliesSelf(qB)
		ex, ey, ez, ew := qErrScratch.Get()
		if ew < 0.0 {
			ex, ey, ez, ew = -ex, -ey, -ez, -ew
		}
		vSq := ex*ex + ey*ey + ez*ez
		if vSq > 1e-12 {
			vLen := math.Sqrt(vSq)
			angle := 2.0 * math.Atan2(vLen, ew)
			const angSlop = 0.002
			if excess := angle - maxAngle; excess > angSlop {
				axW, ayW, azW := qTargetScratch.Rotate(ex/vLen, ey/vLen, ez/vLen)
				uAx, uAy, uAz := e.Cinematic.applyWorldInvInertia(axW, ayW, azW)
				uBx, uBy, uBz := other.Cinematic.applyWorldInvInertia(axW, ayW, azW)
				kA := axW*uAx + ayW*uAy + azW*uAz
				kB := axW*uBx + ayW*uBy + azW*uBz
				kTot := kA + kB
				if kTot > 1e-12 {
					corr := (excess - angSlop) * 0.30
					if corr > 0.15 {
						corr = 0.15
					}
					groundedA := e.Cinematic.onGround && !other.Cinematic.onGround
					groundedB := other.Cinematic.onGround && !e.Cinematic.onGround
					var wA, wB float64
					switch {
					case groundedA:
						wB = corr
					case groundedB:
						wA = corr
					default:
						wA = corr * (kA / kTot)
						wB = corr * (kB / kTot)
					}
					if wA > 0.0 {
						e.rotateAroundCOM(axW*wA, ayW*wA, azW*wA)
					}
					if wB > 0.0 {
						other.rotateAroundCOM(-axW*wB, -ayW*wB, -azW*wB)
					}
				}
			}
		}
	}

	pAx, pAy, pAz, rAx, rAy, rAz := e.anchorWorld(ax, ay, az)
	pBx, pBy, pBz, rBx, rBy, rBz := other.anchorWorld(bx, by, bz)

	cx := pBx - pAx
	cy := pBy - pAy
	cz := pBz - pAz
	distSq := cx*cx + cy*cy + cz*cz
	const posSlop = 0.02
	const posSlopSq = posSlop * posSlop
	if distSq > posSlopSq {
		dist := math.Sqrt(distSq)
		nx, ny, nz := cx/dist, cy/dist, cz/dist

		var uAx, uAy, uAz, rotMassA float64
		if invMassA > 0.0 {
			uAx, uAy, uAz, rotMassA = e.Cinematic.rotationalMass(rAx, rAy, rAz, nx, ny, nz)
		}
		var uBx, uBy, uBz, rotMassB float64
		if invMassB > 0.0 {
			uBx, uBy, uBz, rotMassB = other.Cinematic.rotationalMass(rBx, rBy, rBz, nx, ny, nz)
		}

		k := invMassA + rotMassA + invMassB + rotMassB
		if k > 1e-12 {
			const relax = 0.70
			lambda := (dist - posSlop) * relax / k
			if lambda > 0.0 {
				if invMassA > 0.0 {
					e.AddTo(nx*lambda*invMassA, ny*lambda*invMassA, nz*lambda*invMassA)
					e.applyClampedPosRotation(lambda*uAx, lambda*uAy, lambda*uAz)
				}
				if invMassB > 0.0 {
					other.AddTo(-nx*lambda*invMassB, -ny*lambda*invMassB, -nz*lambda*invMassB)
					other.applyClampedPosRotation(-lambda*uBx, -lambda*uBy, -lambda*uBz)
				}
			}
		}
	}
}

// applyClampedPosRotation applies a bounded position-solver rotation around the entity center of mass.
// It ignores near-zero corrections and clamps larger angular steps to a safe maximum.
func (e *Entity) applyClampedPosRotation(dpx, dpy, dpz float64) {
	const maxStepAng = 0.15
	magSq := dpx*dpx + dpy*dpy + dpz*dpz
	if magSq < 1e-12 {
		return
	}
	if magSq > maxStepAng*maxStepAng {
		s := maxStepAng / math.Sqrt(magSq)
		dpx *= s
		dpy *= s
		dpz *= s
	}
	e.rotateAroundCOM(dpx, dpy, dpz)
}

// rotateAroundCOM applies an angular rotation while keeping the entity's center of mass fixed.
func (e *Entity) rotateAroundCOM(ax, ay, az float64) {
	if e.Cinematic.keepUpright {
		return
	}
	cx, cy, cz := e.getCenterOfMass()
	e.BoundingBox.GetQRotation().IntegrateAngular(ax, ay, az, 1.0)
	e.SyncRotation()
	ncx, ncy, ncz := e.getCenterOfMass()
	e.BoundingBox.AddTo(cx-ncx, cy-ncy, cz-ncz)
}
