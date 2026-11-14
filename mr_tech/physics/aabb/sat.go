package aabb

import "math"

// Sat holds reusable result state and scratch buffers for a 15-axis OBB Separating Axis Test.
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

// GetNormal returns the winning contact normal when hit is true, or the largest separation gap normal otherwise.
func (res *Sat) GetNormal() (float64, float64, float64) {
	return res.nx, res.ny, res.nz
}

// GetContact returns the winning SAT state: overlap flag, winning axis index (0..5 face, 6..14 edge-edge),
// contact penetration depth, separation gap, and unit normal pointing from B toward A.
func (res *Sat) GetContact() (bool, int, float64, float64, float64, float64, float64) {
	return res.hit, res.contactAxis, res.contactPen, res.maxGap, res.nx, res.ny, res.nz
}

// Compute runs a separating axis test between two oriented boxes using center delta, axes, and half-extents.
// It reuses internal scratch buffers and writes hit state, penetration depth, contact normal, axis index, and gap.
func (res *Sat) Compute(dx, dy, dz float64, a *Axis, scx, scy, scz float64, b *Axis, ocx, ocy, ocz float64) {
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
						nx, ny, nz := worldCrossUnitNormal(a, i, b, j, axisLen, proj)
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
					res.nx, res.ny, res.nz = worldCrossUnitNormal(a, i, b, j, axisLen, proj)
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

func (res *Sat) recordGap(gap, nx, ny, nz float64) {
	if gap > res.maxGap {
		res.maxGap = gap
		res.gx, res.gy, res.gz = nx, ny, nz
	}
}

func worldCrossUnitNormal(a *Axis, i int, b *Axis, j int, axisLen, proj float64) (float64, float64, float64) {
	ax, ay, az := a.data[i][0], a.data[i][1], a.data[i][2]
	bx, by, bz := b.data[j][0], b.data[j][1], b.data[j][2]
	cx := (ay*bz - az*by) / axisLen
	cy := (az*bx - ax*bz) / axisLen
	cz := (ax*by - ay*bx) / axisLen
	if proj > 0.0 {
		cx, cy, cz = -cx, -cy, -cz
	}
	return cx, cy, cz
}
