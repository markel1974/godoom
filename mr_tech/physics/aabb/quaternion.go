package aabb

import "math"

// Quaternion represents a 3D rotation or orientation supporting rotation, normalization, and composition.
type Quaternion struct {
	data [4]float64
}

// NewQuaternion creates a quaternion initialized with the supplied component array.
func NewQuaternion(data [4]float64) *Quaternion {
	return &Quaternion{data: data}
}

// NewQuaternionIdentity creates a unit identity quaternion (0, 0, 0, 1).
func NewQuaternionIdentity() *Quaternion {
	return &Quaternion{data: [4]float64{0, 0, 0, 1}}
}

// NewQuaternionEmpty creates a quaternion with all components initialized to zero.
func NewQuaternionEmpty() *Quaternion {
	return &Quaternion{data: [4]float64{}}
}

// Set updates the quaternion components in place.
func (q *Quaternion) Set(x, y, z, w float64) {
	q.data[0], q.data[1], q.data[2], q.data[3] = x, y, z, w
}

// SetIdentity resets the quaternion to the multiplicative identity (0, 0, 0, 1).
func (q *Quaternion) SetIdentity() {
	q.data[0], q.data[1], q.data[2], q.data[3] = 0, 0, 0, 1
}

// CopyFrom copies the components from in into q without allocating.
func (q *Quaternion) CopyFrom(in *Quaternion) {
	q.data = in.data
}

// Get returns the quaternion components in (x, y, z, w) order.
func (q *Quaternion) Get() (float64, float64, float64, float64) {
	return q.data[0], q.data[1], q.data[2], q.data[3]
}

// Clone returns a new Quaternion with the same components as q.
func (q *Quaternion) Clone() *Quaternion {
	return &Quaternion{data: q.data}
}

// SetAxisAngle sets q from a rotation axis (ax, ay, az) and an angle in radians.
func (q *Quaternion) SetAxisAngle(ax, ay, az, angleRad float64) {
	lenSq := ax*ax + ay*ay + az*az
	if lenSq < 1e-24 {
		q.SetIdentity()
		return
	}
	half := angleRad * 0.5
	s := math.Sin(half) / math.Sqrt(lenSq)
	q.data[0] = ax * s
	q.data[1] = ay * s
	q.data[2] = az * s
	q.data[3] = math.Cos(half)
}

// SetEulerZYX sets q from intrinsic ZYX (yaw around Z, pitch around Y, roll around X) angles in radians.
func (q *Quaternion) SetEulerZYX(rollX, pitchY, yawZ float64) {
	hr := rollX * 0.5
	hp := pitchY * 0.5
	hy := yawZ * 0.5
	sr, cr := math.Sin(hr), math.Cos(hr)
	sp, cp := math.Sin(hp), math.Cos(hp)
	sy, cy := math.Sin(hy), math.Cos(hy)

	q.data[0] = sr*cp*cy - cr*sp*sy
	q.data[1] = cr*sp*cy + sr*cp*sy
	q.data[2] = cr*cp*sy - sr*sp*cy
	q.data[3] = cr*cp*cy + sr*sp*sy
}

// Rotate applies the quaternion rotation (q * v * q^*) to the vector components and returns the rotated components.
func (q *Quaternion) Rotate(vX, vY, vZ float64) (float64, float64, float64) {
	qx, qy, qz, qw := q.data[0], q.data[1], q.data[2], q.data[3]
	tx := 2.0 * (qy*vZ - qz*vY)
	ty := 2.0 * (qz*vX - qx*vZ)
	tz := 2.0 * (qx*vY - qy*vX)
	rx := vX + qw*tx + (qy*tz - qz*ty)
	ry := vY + qw*ty + (qz*tx - qx*tz)
	rz := vZ + qw*tz + (qx*ty - qy*tx)
	return rx, ry, rz
}

// RotateZ returns the Z component of the vector rotated by the quaternion.
func (q *Quaternion) RotateZ(vX, vY, vZ float64) float64 {
	qx, qy, qz, qw := q.data[0], q.data[1], q.data[2], q.data[3]
	tx := 2.0 * (qy*vZ - qz*vY)
	ty := 2.0 * (qz*vX - qx*vZ)
	tz := 2.0 * (qx*vY - qy*vX)
	rz := vZ + qw*tz + (qx*ty - qy*tx)
	return rz
}

// RotateInverse applies the inverse rotation (q^* * v * q) to the vector components without mutating q.
func (q *Quaternion) RotateInverse(vX, vY, vZ float64) (float64, float64, float64) {
	qx, qy, qz, qw := q.data[0], q.data[1], q.data[2], q.data[3]
	tx := 2.0 * (qy*vZ - qz*vY)
	ty := 2.0 * (qz*vX - qx*vZ)
	tz := 2.0 * (qx*vY - qy*vX)
	rx := vX - qw*tx + (qy*tz - qz*ty)
	ry := vY - qw*ty + (qz*tx - qx*tz)
	rz := vZ - qw*tz + (qx*ty - qy*tx)
	return rx, ry, rz
}

// ToAxes extracts the three orthonormal local axes (X, Y, Z) rotated into world space.
func (q *Quaternion) ToAxes() (a0x, a0y, a0z, a1x, a1y, a1z, a2x, a2y, a2z float64) {
	x, y, z, w := q.data[0], q.data[1], q.data[2], q.data[3]
	x2 := x + x
	y2 := y + y
	z2 := z + z

	xx := x * x2
	yy := y * y2
	zz := z * z2
	xy := x * y2
	xz := x * z2
	yz := y * z2
	wx := w * x2
	wy := w * y2
	wz := w * z2

	a0x = 1.0 - (yy + zz)
	a0y = xy + wz
	a0z = xz - wy

	a1x = xy - wz
	a1y = 1.0 - (xx + zz)
	a1z = yz + wx

	a2x = xz + wy
	a2y = yz - wx
	a2z = 1.0 - (xx + yy)
	return
}

// SetFromAxes populates q from three orthonormal column basis vectors a0=(a0x,a0y,a0z), a1=(a1x,a1y,a1z), a2=(a2x,a2y,a2z).
func (q *Quaternion) SetFromAxes(a0x, a0y, a0z, a1x, a1y, a1z, a2x, a2y, a2z float64) {
	trace := a0x + a1y + a2z
	if trace > 0.0 {
		s := math.Sqrt(trace+1.0) * 2.0
		invS := 1.0 / s
		q.data[0] = (a1z - a2y) * invS
		q.data[1] = (a2x - a0z) * invS
		q.data[2] = (a0y - a1x) * invS
		q.data[3] = 0.25 * s
	} else if a0x > a1y && a0x > a2z {
		s := math.Sqrt(1.0+a0x-a1y-a2z) * 2.0
		invS := 1.0 / s
		q.data[0] = 0.25 * s
		q.data[1] = (a0y + a1x) * invS
		q.data[2] = (a2x + a0z) * invS
		q.data[3] = (a1z - a2y) * invS
	} else if a1y > a2z {
		s := math.Sqrt(1.0+a1y-a0x-a2z) * 2.0
		invS := 1.0 / s
		q.data[0] = (a0y + a1x) * invS
		q.data[1] = 0.25 * s
		q.data[2] = (a1z + a2y) * invS
		q.data[3] = (a2x - a0z) * invS
	} else {
		s := math.Sqrt(1.0+a2z-a0x-a1y) * 2.0
		invS := 1.0 / s
		q.data[0] = (a2x + a0z) * invS
		q.data[1] = (a1z + a2y) * invS
		q.data[2] = 0.25 * s
		q.data[3] = (a0y - a1x) * invS
	}
	q.Normalize()
}

// MultipliesSelf multiplies this quaternion by q2 and stores the result in this quaternion.
func (q *Quaternion) MultipliesSelf(q2 *Quaternion) {
	q.Multiplies(q2, q)
}

// Multiplies multiplies q by q2 and writes the resulting quaternion to out (safe if out aliases q or q2).
func (q *Quaternion) Multiplies(q2 *Quaternion, out *Quaternion) {
	x1, y1, z1, w1 := q.data[0], q.data[1], q.data[2], q.data[3]
	x2, y2, z2, w2 := q2.data[0], q2.data[1], q2.data[2], q2.data[3]

	out.data[0] = w1*x2 + x1*w2 + y1*z2 - z1*y2
	out.data[1] = w1*y2 - x1*z2 + y1*w2 + z1*x2
	out.data[2] = w1*z2 + x1*y2 - y1*x2 + z1*w2
	out.data[3] = w1*w2 - x1*x2 - y1*y2 - z1*z2
}

// Normalize scales q in place so its components have unit length.
func (q *Quaternion) Normalize() {
	lenSq := q.data[0]*q.data[0] + q.data[1]*q.data[1] + q.data[2]*q.data[2] + q.data[3]*q.data[3]
	if lenSq < 1e-24 {
		q.data[0], q.data[1], q.data[2], q.data[3] = 0, 0, 0, 1
		return
	}
	invL := 1.0 / math.Sqrt(lenSq)
	q.data[0] *= invL
	q.data[1] *= invL
	q.data[2] *= invL
	q.data[3] *= invL
}

// Conjugate sets q to the conjugate of in, negating the vector part and preserving the scalar part.
func (q *Quaternion) Conjugate(in *Quaternion) {
	q.data[0], q.data[1], q.data[2], q.data[3] = -in.data[0], -in.data[1], -in.data[2], in.data[3]
}

// IntegrateAngular integrates a world-space angular velocity (wx, wy, wz) over timestep dt into q in-place
// using the exponential map and normalizes the resulting orientation.
func (q *Quaternion) IntegrateAngular(wx, wy, wz, dt float64) {
	omegaSq := wx*wx + wy*wy + wz*wz
	halfThetaSq := 0.25 * omegaSq * dt * dt
	if halfThetaSq < 1e-24 {
		return
	}

	var s, c float64
	if halfThetaSq < 1e-8 {
		// Taylor series expansion for small angles
		s = (0.5 * dt) * (1.0 - halfThetaSq/6.0)
		c = 1.0 - 0.5*halfThetaSq
	} else {
		halfTheta := math.Sqrt(halfThetaSq)
		s = math.Sin(halfTheta) / math.Sqrt(omegaSq)
		c = math.Cos(halfTheta)
	}

	dqx := wx * s
	dqy := wy * s
	dqz := wz * s
	dqw := c

	qx, qy, qz, qw := q.data[0], q.data[1], q.data[2], q.data[3]
	q.data[0] = dqw*qx + dqx*qw + dqy*qz - dqz*qy
	q.data[1] = dqw*qy - dqx*qz + dqy*qw + dqz*qx
	q.data[2] = dqw*qz + dqx*qy - dqy*qx + dqz*qw
	q.data[3] = dqw*qw - dqx*qx - dqy*qy - dqz*qz
	q.Normalize()
}

// Nlerp performs shortest-path normalized linear interpolation between q and q2 at factor t in [0, 1], writing to out.
func (q *Quaternion) Nlerp(q2 *Quaternion, t float64, out *Quaternion) {
	x1, y1, z1, w1 := q.data[0], q.data[1], q.data[2], q.data[3]
	x2, y2, z2, w2 := q2.data[0], q2.data[1], q2.data[2], q2.data[3]

	dot := x1*x2 + y1*y2 + z1*z2 + w1*w2
	if dot < 0.0 {
		x2, y2, z2, w2 = -x2, -y2, -z2, -w2
	}

	invT := 1.0 - t
	out.data[0] = invT*x1 + t*x2
	out.data[1] = invT*y1 + t*y2
	out.data[2] = invT*z1 + t*z2
	out.data[3] = invT*w1 + t*w2
	out.Normalize()
}

// Slerp performs shortest-path spherical linear interpolation between q and q2 at factor t in [0, 1], writing to out.
func (q *Quaternion) Slerp(q2 *Quaternion, t float64, out *Quaternion) {
	x1, y1, z1, w1 := q.data[0], q.data[1], q.data[2], q.data[3]
	x2, y2, z2, w2 := q2.data[0], q2.data[1], q2.data[2], q2.data[3]

	dot := x1*x2 + y1*y2 + z1*z2 + w1*w2
	if dot < 0.0 {
		dot = -dot
		x2, y2, z2, w2 = -x2, -y2, -z2, -w2
	}

	if dot > 0.9995 {
		invT := 1.0 - t
		out.data[0] = invT*x1 + t*x2
		out.data[1] = invT*y1 + t*y2
		out.data[2] = invT*z1 + t*z2
		out.data[3] = invT*w1 + t*w2
		out.Normalize()
		return
	}

	theta0 := math.Acos(dot)
	theta := theta0 * t
	sinTheta := math.Sin(theta)
	sinTheta0 := math.Sin(theta0)

	s0 := math.Cos(theta) - dot*sinTheta/sinTheta0
	s1 := sinTheta / sinTheta0

	out.data[0] = s0*x1 + s1*x2
	out.data[1] = s0*y1 + s1*y2
	out.data[2] = s0*z1 + s1*z2
	out.data[3] = s0*w1 + s1*w2
	out.Normalize()
}
