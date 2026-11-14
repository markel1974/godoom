package aabb

import "math"

type Sat struct {
	hit         bool
	minPen      float64
	maxGap      float64
	contactPen  float64
	contactAxis int
	normal      [3]float64
	gapNormal   [3]float64
}

func (res *Sat) GetNormal() (float64, float64, float64) {
	return res.normal[0], res.normal[1], res.normal[2]
}

// Compute runs a separating axis test between two oriented boxes using center delta, axes, and half-extents.
// It updates res with hit state, minimum penetration, contact normal, contact axis, and largest separation gap.
func (res *Sat) Compute(dx, dy, dz float64, a *Axis, scx, scy, scz float64, b *Axis, ocx, ocy, ocz float64) {
	hA := [3]float64{scx, scy, scz}
	hB := [3]float64{ocx, ocy, ocz}
	var R [3][3]float64
	var AbsR [3][3]float64
	const epsilon = 1e-12
	const crossAxisLenEps = 1e-6
	const edgeAxisMinLen = 1e-4

	for i := 0; i < 3; i++ {
		ax, ay, az := a.data[i][0], a.data[i][1], a.data[i][2]
		for j := 0; j < 3; j++ {
			value := ax*b.data[j][0] + ay*b.data[j][1] + az*b.data[j][2]
			R[i][j] = value
			AbsR[i][j] = math.Abs(value) + epsilon
		}
	}

	t := [3]float64{
		dx*a.data[0][0] + dy*a.data[0][1] + dz*a.data[0][2],
		dx*a.data[1][0] + dy*a.data[1][1] + dz*a.data[1][2],
		dx*a.data[2][0] + dy*a.data[2][1] + dz*a.data[2][2],
	}

	res.hit = false
	res.minPen = math.MaxFloat64
	res.maxGap = 0.0
	res.contactPen = math.MaxFloat64
	res.contactAxis = -1
	res.normal[0], res.normal[1], res.normal[2] = 0, 0, 0
	res.gapNormal[0], res.gapNormal[1], res.gapNormal[2] = 0, 0, 0
	separated := false
	faceBestPen := math.MaxFloat64

	// Test the three axes of A (indices 0..2).
	for i := 0; i < 3; i++ {
		rb := hB[0]*AbsR[i][0] + hB[1]*AbsR[i][1] + hB[2]*AbsR[i][2]
		proj := t[i]
		distance := math.Abs(proj)
		nx, ny, nz := a.data[i][0], a.data[i][1], a.data[i][2]
		if proj > 0.0 {
			nx, ny, nz = -nx, -ny, -nz
		}
		if distance > hA[i]+rb {
			separated = true
			if g := distance - (hA[i] + rb); g > res.maxGap {
				res.maxGap = g
				res.gapNormal = [3]float64{nx, ny, nz}
			}
		} else if !separated {
			p := hA[i] + rb - distance
			if p < res.minPen {
				res.minPen = p
			}
			if p < res.contactPen {
				res.contactPen = p
				faceBestPen = p
				res.contactAxis = i
				res.normal[0] = nx
				res.normal[1] = ny
				res.normal[2] = nz
			}
		}
	}

	// Test the three axes of B (indices 3..5).
	for j := 0; j < 3; j++ {
		ra := hA[0]*AbsR[0][j] + hA[1]*AbsR[1][j] + hA[2]*AbsR[2][j]
		proj := t[0]*R[0][j] + t[1]*R[1][j] + t[2]*R[2][j]
		distance := math.Abs(proj)
		nx, ny, nz := b.data[j][0], b.data[j][1], b.data[j][2]
		if proj > 0.0 {
			nx, ny, nz = -nx, -ny, -nz
		}
		if distance > ra+hB[j] {
			separated = true
			if g := distance - (ra + hB[j]); g > res.maxGap {
				res.maxGap = g
				res.gapNormal[0] = nx
				res.gapNormal[1] = ny
				res.gapNormal[2] = nz
			}
		} else if !separated {
			p := ra + hB[j] - distance
			if p < res.minPen {
				res.minPen = p
			}
			if p < res.contactPen {
				res.contactPen = p
				faceBestPen = p
				res.contactAxis = 3 + j
				res.normal[0] = nx
				res.normal[1] = ny
				res.normal[2] = nz
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
			ra := hA[i1]*AbsR[i2][j] + hA[i2]*AbsR[i1][j]
			rb := hB[j1]*AbsR[i][j2] + hB[j2]*AbsR[i][j1]
			proj := t[i2]*R[i1][j] - t[i1]*R[i2][j]
			distance := math.Abs(proj)
			axisLen := math.Hypot(R[i1][j], R[i2][j])
			if distance > ra+rb {
				separated = true
				if axisLen > crossAxisLenEps {
					if g := (distance - (ra + rb)) / axisLen; g > res.maxGap {
						res.maxGap = g
						res.gapNormal = worldCrossUnitNormal(a, i, b, j, axisLen, proj)
					}
				}
			} else if !separated && axisLen > crossAxisLenEps {
				p := (ra + rb - distance) / axisLen
				if p < res.minPen {
					res.minPen = p
				}
				// Slightly bias against edge-edge axes to prevent numerical angle chatter on flat face contacts.
				if axisLen > edgeAxisMinLen && p < faceBestPen*0.98-1e-4 && p < res.contactPen {
					res.contactPen = p
					res.contactAxis = 6 + i*3 + j
					res.normal = worldCrossUnitNormal(a, i, b, j, axisLen, proj)
				}
			}
		}
	}
	res.hit = !separated
	if separated {
		res.minPen = 0.0
		res.contactPen = 0.0
	}
}
