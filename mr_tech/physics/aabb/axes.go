package aabb

import "math"

var _axisId = NewAxis(1, 0, 0, 0, 1, 0, 0, 0, 1)

// Axes represents a 3x3 matrix used to store local axes for oriented bounding box calculations.
type Axes struct {
	data [3][3]float64
}

// NewAxis returns a new Axes initialized with the provided 3x3 matrix values in row-major order.
func NewAxis(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) *Axes {
	return &Axes{data: [3][3]float64{{a00, a01, a02}, {a10, a11, a12}, {a20, a21, a22}}}
}

func (r *Axes) Set(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) {
	r.data[0][0], r.data[0][1], r.data[0][2], r.data[1][0], r.data[1][1], r.data[1][2], r.data[2][0], r.data[2][1], r.data[2][2] = a00, a01, a02, a10, a11, a12, a20, a21, a22
}

// Get returns all nine matrix values in row-major order.
func (r *Axes) Get() (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	return r.data[0][0], r.data[0][1], r.data[0][2], r.data[1][0], r.data[1][1], r.data[1][2], r.data[2][0], r.data[2][1], r.data[2][2]
}

// Get0 returns the three values in the first row of the matrix.
func (r *Axes) Get0() (float64, float64, float64) {
	return r.data[0][0], r.data[0][1], r.data[0][2]
}

// Get1 returns the three values in the second row of the matrix.
func (r *Axes) Get1() (float64, float64, float64) {
	return r.data[1][0], r.data[1][1], r.data[1][2]
}

// Get2 returns the three values in the third row of the matrix.
func (r *Axes) Get2() (float64, float64, float64) {
	return r.data[2][0], r.data[2][1], r.data[2][2]
}

// GetRow returns the three values in row i (0..2) of the matrix.
func (r *Axes) GetRow(i int) (float64, float64, float64) {
	return r.data[i][0], r.data[i][1], r.data[i][2]
}

// GetIdentity returns the shared identity axis matrix.
func (r *Axes) GetIdentity() *Axes {
	return _axisId
}

// Project returns the axis-aligned extents of the oriented half-extents hx, hy, and hz.
func (r *Axes) Project(hx, hy, hz float64) (float64, float64, float64) {
	ex := math.Abs(r.data[0][0])*hx + math.Abs(r.data[1][0])*hy + math.Abs(r.data[2][0])*hz
	ey := math.Abs(r.data[0][1])*hx + math.Abs(r.data[1][1])*hy + math.Abs(r.data[2][1])*hz
	ez := math.Abs(r.data[0][2])*hx + math.Abs(r.data[1][2])*hy + math.Abs(r.data[2][2])*hz
	return ex, ey, ez
}

// ProjectNormal returns the oriented half-extent projected onto unit normal (nx, ny, nz).
func (r *Axes) ProjectNormal(nx, ny, nz, hx, hy, hz float64) float64 {
	return math.Abs(nx*r.data[0][0]+ny*r.data[0][1]+nz*r.data[0][2])*hx +
		math.Abs(nx*r.data[1][0]+ny*r.data[1][1]+nz*r.data[1][2])*hy +
		math.Abs(nx*r.data[2][0]+ny*r.data[2][1]+nz*r.data[2][2])*hz
}

// ComputeSupportPoint computes the exact extreme support point for an oriented box
// from its center, direction, and half-extents without deadband or smoothing.
func (r *Axes) ComputeSupportPoint(cx, cy, cz float64, dirX, dirY, dirZ float64, hw, hh, hd float64) (float64, float64, float64) {
	return r.computeExtremalPoint(cx, cy, cz, dirX, dirY, dirZ, hw, hh, hd, false)
}

// ComputeStabilizedContactPoint computes a contact-stabilized extreme point for an oriented box
// along direction (dirX, dirY, dirZ), suppressing near-flat corner jitter via supportAxisFactor.
func (r *Axes) ComputeStabilizedContactPoint(cx, cy, cz float64, dirX, dirY, dirZ float64, hw, hh, hd float64) (float64, float64, float64) {
	lenSq := dirX*dirX + dirY*dirY + dirZ*dirZ
	if lenSq <= 1e-24 {
		return cx, cy, cz
	}
	if math.Abs(lenSq-1.0) > 1e-12 {
		invLen := 1.0 / math.Sqrt(lenSq)
		dirX *= invLen
		dirY *= invLen
		dirZ *= invLen
	}
	return r.computeExtremalPoint(cx, cy, cz, dirX, dirY, dirZ, hw, hh, hd, true)
}

// computeExtremalPoint returns the oriented box point farthest along the given direction.
// It offsets the center by signed half-extents, optionally smoothing near-flat axis alignment.
func (r *Axes) computeExtremalPoint(cx, cy, cz float64, dirX, dirY, dirZ float64, hw, hh, hd float64, stabilized bool) (float64, float64, float64) {
	px, py, pz := cx, cy, cz
	extents := [3]float64{hw, hh, hd}
	for i := 0; i < 3; i++ {
		dot := r.data[i][0]*dirX + r.data[i][1]*dirY + r.data[i][2]*dirZ
		var sign float64
		if stabilized {
			sign = supportAxisFactor(dot)
		} else if dot > 0.0 {
			sign = 1.0
		} else if dot < 0.0 {
			sign = -1.0
		}
		if f := sign * extents[i]; f != 0.0 {
			px += r.data[i][0] * f
			py += r.data[i][1] * f
			pz += r.data[i][2] * f
		}
	}
	return px, py, pz
}

// WorldCrossUnitNormal returns the unit normal of the world-space cross axis from rows a[i] and b[j].
// It divides by axisLen and flips the result when proj is positive to keep the normal consistently oriented.
func (r *Axes) WorldCrossUnitNormal(i int, b *Axes, j int, axisLen, proj float64) (float64, float64, float64) {
	ax, ay, az := r.data[i][0], r.data[i][1], r.data[i][2]
	bx, by, bz := b.data[j][0], b.data[j][1], b.data[j][2]
	cx := (ay*bz - az*by) / axisLen
	cy := (az*bx - ax*bz) / axisLen
	cz := (ax*by - ay*bx) / axisLen
	if proj > 0.0 {
		cx, cy, cz = -cx, -cy, -cz
	}
	return cx, cy, cz
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
