package physics

import "math"

// Quaternion represents a 3D rotation or orientation supporting rotation, normalization, and composition.
type Quaternion struct {
	data [4]float64
}

// NewQuaternion creates a quaternion initialized with the supplied component array.
func NewQuaternion(data [4]float64) *Quaternion {
	return &Quaternion{data: data}
}

// NewQuaternionEmpty creates a quaternion with all components initialized to zero.
func NewQuaternionEmpty() *Quaternion {
	return &Quaternion{data: [4]float64{}}
}

// Set updates the quaternion components in place.
func (q *Quaternion) Set(x, y, z, w float64) {
	q.data[0], q.data[1], q.data[2], q.data[3] = x, y, z, w
}

// Clone returns a new Quaternion with the same components as q.
func (q *Quaternion) Clone() *Quaternion {
	return &Quaternion{data: q.data}
}

// Rotate applies the quaternion rotation to the vector components and returns the rotated components.
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

// MultipliesSelf multiplies this quaternion by q2 and stores the result in this quaternion.
func (q *Quaternion) MultipliesSelf(q2 *Quaternion) {
	q.Multiplies(q2, q)
}

// Multiplies multiplies q1 by q2 and writes the resulting quaternion to out.
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
	if lenSq <= 0.0 {
		q.data[0], q.data[1], q.data[2], q.data[3] = 0, 0, 0, 1
		return
	}
	l := math.Sqrt(lenSq)
	q.data[0], q.data[1], q.data[2], q.data[3] = q.data[0]/l, q.data[1]/l, q.data[2]/l, q.data[3]/l
}

// Conjugate sets q1 to the conjugate of in, negating the vector part and preserving the scalar part.
func (q *Quaternion) Conjugate(in *Quaternion) {
	q.data[0], q.data[1], q.data[2], q.data[3] = -in.data[0], -in.data[1], -in.data[2], in.data[3]
}
