package rigid_body

// Matrix3x3 represents a 3 by 3 matrix of float64 values.
type Matrix3x3 struct {
	data [3][3]float64
}

// NewMatrix3x3 returns a new zero-initialized 3x3 matrix.
func NewMatrix3x3() *Matrix3x3 {
	return &Matrix3x3{}
}

func (r *Matrix3x3) Reset() {
	r.data[0][0], r.data[0][1], r.data[0][2] = 0, 0, 0
	r.data[1][0], r.data[1][1], r.data[1][2] = 0, 0, 0
	r.data[2][0], r.data[2][1], r.data[2][2] = 0, 0, 0
}

// Set assigns the matrix entries in row-major order.
func (r *Matrix3x3) Set(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) {
	r.data[0][0], r.data[0][1], r.data[0][2] = a00, a01, a02
	r.data[1][0], r.data[1][1], r.data[1][2] = a10, a11, a12
	r.data[2][0], r.data[2][1], r.data[2][2] = a20, a21, a22
}

func (r *Matrix3x3) Get0() (float64, float64, float64) {
	return r.data[0][0], r.data[0][1], r.data[0][2]
}

func (r *Matrix3x3) Get1() (float64, float64, float64) {
	return r.data[1][0], r.data[1][1], r.data[1][2]
}

// Get2 returns the third row of the matrix.
func (r *Matrix3x3) Get2() (float64, float64, float64) {
	return r.data[2][0], r.data[2][1], r.data[2][2]
}

// GetSymmetric returns the six unique entries (00, 01, 02, 11, 12, 22) of a symmetric 3x3 matrix.
func (r *Matrix3x3) GetSymmetric() (float64, float64, float64, float64, float64, float64) {
	return r.data[0][0], r.data[0][1], r.data[0][2], r.data[1][1], r.data[1][2], r.data[2][2]
}

// MultiplySymmetric returns the product of the symmetric matrix and the vector (tx, ty, tz).
func (r *Matrix3x3) MultiplySymmetric(tx, ty, tz float64) (float64, float64, float64) {
	invIxx, invIxy, invIxz, invIyy, invIyz, invIzz := r.data[0][0], r.data[0][1], r.data[0][2], r.data[1][1], r.data[1][2], r.data[2][2]
	return invIxx*tx + invIxy*ty + invIxz*tz, invIxy*tx + invIyy*ty + invIyz*tz, invIxz*tx + invIyz*ty + invIzz*tz
}

// Multiply writes body and bodyInv transformed by r (R^T * M * R) into world and worldInv.
func (r *Matrix3x3) Multiply(world *Matrix3x3, worldInv *Matrix3x3, body *Matrix3x3, bodyInv *Matrix3x3) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			sumW := 0.0
			sumWI := 0.0
			for k := 0; k < 3; k++ {
				rki := r.data[k][i]
				for l := 0; l < 3; l++ {
					rlj := r.data[l][j]
					sumW += rki * body.data[k][l] * rlj
					sumWI += rki * bodyInv.data[k][l] * rlj
				}
			}
			world.data[i][j] = sumW
			worldInv.data[i][j] = sumWI
		}
	}
}
