package rigid_body

import (
	"math"
)

// ComputeOrthonormalBasis constructs two unit vectors (u, v) that are mutually perpendicular
// to unit normal n = (nx, ny, nz) so that (n, u, v) forms a right-handed orthonormal basis.
func ComputeOrthonormalBasis(nx, ny, nz float64) (float64, float64, float64, float64, float64, float64) {
	var ux, uy, uz, vx, vy, vz float64
	if math.Abs(nx) < 0.9 {
		ux = 0.0
		uy = nz
		uz = -ny
	} else {
		ux = -nz
		uy = 0.0
		uz = nx
	}
	uLen := math.Sqrt(ux*ux + uy*uy + uz*uz)
	if uLen > 1e-12 {
		invLen := 1.0 / uLen
		ux *= invLen
		uy *= invLen
		uz *= invLen
	}
	vx = ny*uz - nz*uy
	vy = nz*ux - nx*uz
	vz = nx*uy - ny*ux
	return ux, uy, uz, vx, vy, vz
}

// AnchorWorld computes world-space anchor coordinates and lever arm from center of mass for e.
func (e *Entity) AnchorWorld(localX, localY, localZ float64) (px, py, pz, rx, ry, rz float64) {
	return e.anchorWorld(localX, localY, localZ)
}

// ApplyLinearImpulse applies a 1D bilateral/unilateral impulse j along world axis n between e (A) and other (B).
func (e *Entity) ApplyLinearImpulse(other *Entity, rAx, rAy, rAz, rBx, rBy, rBz float64, nx, ny, nz, j float64) {
	if j == 0.0 {
		return
	}
	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
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

// SolveLinearAxis executes one 1D bilateral or unilateral PGS impulse step along world axis n.
func (e *Entity) SolveLinearAxis(other *Entity, rAx, rAy, rAz, rBx, rBy, rBz float64, nx, ny, nz float64, accJ *float64, targetRelV, minJ, maxJ float64) float64 {
	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
	if invMassA == 0.0 && invMassB == 0.0 {
		return 0.0
	}

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
		return 0.0
	}

	errV := relV - targetRelV
	j := -errV / k

	if accJ != nil {
		oldJ := *accJ
		newJ := math.Max(minJ, math.Min(maxJ, oldJ+j))
		j = newJ - oldJ
		*accJ = newJ
	} else if j < minJ {
		j = minJ
	} else if j > maxJ {
		j = maxJ
	}

	if j == 0.0 {
		return 0.0
	}

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
	return j
}

// ApplyAngularImpulse applies a 1D bilateral/unilateral angular impulse j along world axis n.
func (e *Entity) ApplyAngularImpulse(other *Entity, nx, ny, nz, j float64) {
	if j == 0.0 {
		return
	}
	if e.Cinematic.invMass > 0.0 && !e.Cinematic.keepUpright {
		uAx, uAy, uAz := e.Cinematic.applyWorldInvInertia(nx, ny, nz)
		e.applyClampedAngularDelta(-j*uAx, -j*uAy, -j*uAz)
	}
	if other.Cinematic.invMass > 0.0 && !other.Cinematic.keepUpright {
		uBx, uBy, uBz := other.Cinematic.applyWorldInvInertia(nx, ny, nz)
		other.applyClampedAngularDelta(j*uBx, j*uBy, j*uBz)
	}
}

// SolveAngularAxis executes one 1D bilateral or unilateral angular PGS impulse step along world axis n.
func (e *Entity) SolveAngularAxis(other *Entity, nx, ny, nz float64, accJ *float64, targetRelW, minJ, maxJ float64) float64 {
	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
	if invMassA == 0.0 && invMassB == 0.0 {
		return 0.0
	}

	relW := (other.Cinematic.wx-e.Cinematic.wx)*nx +
		(other.Cinematic.wy-e.Cinematic.wy)*ny +
		(other.Cinematic.wz-e.Cinematic.wz)*nz

	var uAx, uAy, uAz, kA float64
	if invMassA > 0.0 && !e.Cinematic.keepUpright {
		uAx, uAy, uAz = e.Cinematic.applyWorldInvInertia(nx, ny, nz)
		kA = nx*uAx + ny*uAy + nz*uAz
	}
	var uBx, uBy, uBz, kB float64
	if invMassB > 0.0 && !other.Cinematic.keepUpright {
		uBx, uBy, uBz = other.Cinematic.applyWorldInvInertia(nx, ny, nz)
		kB = nx*uBx + ny*uBy + nz*uBz
	}

	k := kA + kB
	if k <= 1e-12 {
		return 0.0
	}

	errW := relW - targetRelW
	j := -errW / k

	if accJ != nil {
		oldJ := *accJ
		newJ := math.Max(minJ, math.Min(maxJ, oldJ+j))
		j = newJ - oldJ
		*accJ = newJ
	} else if j < minJ {
		j = minJ
	} else if j > maxJ {
		j = maxJ
	}

	if j == 0.0 {
		return 0.0
	}

	if invMassA > 0.0 && !e.Cinematic.keepUpright {
		e.applyClampedAngularDelta(-j*uAx, -j*uAy, -j*uAz)
	}
	if invMassB > 0.0 && !other.Cinematic.keepUpright {
		other.applyClampedAngularDelta(j*uBx, j*uBy, j*uBz)
	}
	return j
}

// ProjectLinearDistance corrects anchor separation dist along world axis n in position space.
func (e *Entity) ProjectLinearDistance(other *Entity, rAx, rAy, rAz, rBx, rBy, rBz float64, nx, ny, nz, dist, posSlop, relax float64) {
	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
	totalInvMass := invMassA + invMassB
	if totalInvMass <= 0.0 {
		return
	}

	absDist := math.Abs(dist)
	if absDist <= posSlop {
		return
	}

	sign := 1.0
	if dist < 0.0 {
		sign = -1.0
	}
	excess := (absDist - posSlop) * sign

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

	lambda := excess * relax / k
	if lambda == 0.0 {
		return
	}

	if invMassA > 0.0 {
		e.AddTo(nx*lambda*invMassA, ny*lambda*invMassA, nz*lambda*invMassA)
		e.applyClampedPosRotation(lambda*uAx, lambda*uAy, lambda*uAz)
	}
	if invMassB > 0.0 {
		other.AddTo(-nx*lambda*invMassB, -ny*lambda*invMassB, -nz*lambda*invMassB)
		other.applyClampedPosRotation(-lambda*uBx, -lambda*uBy, -lambda*uBz)
	}
}

// ProjectAngularAngle corrects relative angular error along world axis n in position space.
func (e *Entity) ProjectAngularAngle(other *Entity, nx, ny, nz, angle, angSlop, relax float64) {
	absAng := math.Abs(angle)
	if absAng <= angSlop {
		return
	}

	sign := 1.0
	if angle < 0.0 {
		sign = -1.0
	}
	excess := (absAng - angSlop) * sign

	invMassA := e.Cinematic.invMass
	invMassB := other.Cinematic.invMass
	if invMassA == 0.0 && invMassB == 0.0 {
		return
	}

	var uAx, uAy, uAz, kA float64
	if invMassA > 0.0 && !e.Cinematic.keepUpright {
		uAx, uAy, uAz = e.Cinematic.applyWorldInvInertia(nx, ny, nz)
		kA = nx*uAx + ny*uAy + nz*uAz
	}
	var uBx, uBy, uBz, kB float64
	if invMassB > 0.0 && !other.Cinematic.keepUpright {
		uBx, uBy, uBz = other.Cinematic.applyWorldInvInertia(nx, ny, nz)
		kB = nx*uBx + ny*uBy + nz*uBz
	}

	kTot := kA + kB
	if kTot <= 1e-12 {
		return
	}

	corr := excess * relax
	if corr > 0.15 {
		corr = 0.15
	} else if corr < -0.15 {
		corr = -0.15
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

	if wA != 0.0 && !e.Cinematic.keepUpright {
		e.rotateAroundCOM(nx*wA, ny*wA, nz*wA)
	}
	if wB != 0.0 && !other.Cinematic.keepUpright {
		other.rotateAroundCOM(-nx*wB, -ny*wB, -nz*wB)
	}
}

// RotateAroundCOM rotates e around its center of mass by rotation vector (ax, ay, az).
func (e *Entity) RotateAroundCOM(ax, ay, az float64) {
	e.rotateAroundCOM(ax, ay, az)
}
