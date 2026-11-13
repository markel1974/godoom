package rigid_body

// Matrix3x3 represents a 3 by 3 matrix of float64 values.
type Matrix3x3 struct {
	data [3][3]float64
}

// NewMatrix3x3 returns a new zero-initialized 3x3 matrix.
func NewMatrix3x3() *Matrix3x3 {
	return &Matrix3x3{}
}

// Set assigns the matrix entries in row-major order.
func (r *Matrix3x3) Set(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) {
	r.data[0][0], r.data[0][1], r.data[0][2] = a00, a01, a02
	r.data[1][0], r.data[1][1], r.data[1][2] = a10, a11, a12
	r.data[2][0], r.data[2][1], r.data[2][2] = a20, a21, a22
}

// Multiply accumulates body and bodyInv transformed by r into world and worldInv.
func (r *Matrix3x3) Multiply(world *Matrix3x3, worldInv *Matrix3x3, body *Matrix3x3, bodyInv *Matrix3x3) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				for l := 0; l < 3; l++ {
					world.data[i][j] += r.data[k][i] * body.data[k][l] * r.data[l][j]
					worldInv.data[i][j] += r.data[k][i] * bodyInv.data[k][l] * r.data[l][j]
				}
			}
		}
	}
}
