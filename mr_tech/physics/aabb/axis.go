package aabb

import "math"

var _axisId = NewAxis(1, 0, 0, 0, 1, 0, 0, 0, 1)

// Axis represents a 3x3 matrix used to store local axes for oriented bounding box calculations.
type Axis struct {
	data [3][3]float64
}

// NewAxis returns a new Axis initialized with the provided 3x3 matrix values in row-major order.
func NewAxis(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) *Axis {
	return &Axis{data: [3][3]float64{{a00, a01, a02}, {a10, a11, a12}, {a20, a21, a22}}}
}

func (r *Axis) Set(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) {
	r.data[0][0], r.data[0][1], r.data[0][2], r.data[1][0], r.data[1][1], r.data[1][2], r.data[2][0], r.data[2][1], r.data[2][2] = a00, a01, a02, a10, a11, a12, a20, a21, a22
}

// Get returns all nine matrix values in row-major order.
func (r *Axis) Get() (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	return r.data[0][0], r.data[0][1], r.data[0][2], r.data[1][0], r.data[1][1], r.data[1][2], r.data[2][0], r.data[2][1], r.data[2][2]
}

// Get0 returns the three values in the first row of the matrix.
func (r *Axis) Get0() (float64, float64, float64) {
	return r.data[0][0], r.data[0][1], r.data[0][2]
}

// Get1 returns the three values in the second row of the matrix.
func (r *Axis) Get1() (float64, float64, float64) {
	return r.data[1][0], r.data[1][1], r.data[1][2]
}

// Get2 returns the three values in the third row of the matrix.
func (r *Axis) Get2() (float64, float64, float64) {
	return r.data[2][0], r.data[2][1], r.data[2][2]
}

// GetIdentity returns the shared identity axis matrix.
func (r *Axis) GetIdentity() *Axis {
	return _axisId
}

// Project returns the axis-aligned extents of the oriented half-extents hx, hy, and hz.
func (r *Axis) Project(hx, hy, hz float64) (float64, float64, float64) {
	ex := math.Abs(r.data[0][0])*hx + math.Abs(r.data[1][0])*hy + math.Abs(r.data[2][0])*hz
	ey := math.Abs(r.data[0][1])*hx + math.Abs(r.data[1][1])*hy + math.Abs(r.data[2][1])*hz
	ez := math.Abs(r.data[0][2])*hx + math.Abs(r.data[1][2])*hy + math.Abs(r.data[2][2])*hz
	return ex, ey, ez
}

// ProjectNormal returns the oriented half-extent projected onto unit normal (nx, ny, nz).
func (r *Axis) ProjectNormal(nx, ny, nz, hx, hy, hz float64) float64 {
	return math.Abs(nx*r.data[0][0]+ny*r.data[0][1]+nz*r.data[0][2])*hx +
		math.Abs(nx*r.data[1][0]+ny*r.data[1][1]+nz*r.data[1][2])*hy +
		math.Abs(nx*r.data[2][0]+ny*r.data[2][1]+nz*r.data[2][2])*hz
}

// ComputeSupportPoint computes the support point for an oriented box from its center, direction, and half-extents.
func (r *Axis) ComputeSupportPoint(cx, cy, cz float64, dirX, dirY, dirZ float64, hw, hh, hd float64) (float64, float64, float64) {
	px, py, pz := cx, cy, cz
	f0 := supportAxisFactor(r.data[0][0]*dirX+r.data[0][1]*dirY+r.data[0][2]*dirZ) * hw
	if f0 != 0.0 {
		px += r.data[0][0] * f0
		py += r.data[0][1] * f0
		pz += r.data[0][2] * f0
	}
	f1 := supportAxisFactor(r.data[1][0]*dirX+r.data[1][1]*dirY+r.data[1][2]*dirZ) * hh
	if f1 != 0.0 {
		px += r.data[1][0] * f1
		py += r.data[1][1] * f1
		pz += r.data[1][2] * f1
	}
	f2 := supportAxisFactor(r.data[2][0]*dirX+r.data[2][1]*dirY+r.data[2][2]*dirZ) * hd
	if f2 != 0.0 {
		px += r.data[2][0] * f2
		py += r.data[2][1] * f2
		pz += r.data[2][2] * f2
	}
	return px, py, pz
}

// supportAxisFactor smoothly maps an axis-normal alignment dot product to [-1, 1],
// suppressing discontinuous corner snaps within a small near-flat deadband.
func supportAxisFactor(d float64) float64 {
	const flatTol = 0.012
	const cornerTol = 0.06
	ad := math.Abs(d)
	if ad <= flatTol {
		return 0.0
	}
	if ad >= cornerTol {
		return math.Copysign(1.0, d)
	}
	t := (ad - flatTol) / (cornerTol - flatTol)
	s := t * t * (3.0 - 2.0*t)
	return math.Copysign(s, d)
}
