package physics

// Frustum represents a view frustum in 3D space, defined by six planes.
type Frustum struct {
	Planes [6]*Plane
}

// NewFrustum creates and returns a pointer to a new Frustum instance with initialized Plane objects.
func NewFrustum() *Frustum {
	f := &Frustum{}
	f.Planes[0] = &Plane{}
	f.Planes[1] = &Plane{}
	f.Planes[2] = &Plane{}
	f.Planes[3] = &Plane{}
	f.Planes[4] = &Plane{}
	f.Planes[5] = &Plane{}
	return f
}

// Rebuild updates the 6 frustum planes using the given 4x4 column-major view-projection matrix.
func (f *Frustum) Rebuild(vp [16]float32) *Frustum {
	// La matrice in Go/OpenGL è un array 1D column-major: vp[col*4 + row]
	// Estrazione dei piani (A, B, C, D)

	// Left Plane (w + x)
	f.Planes[0].NormalX = float64(vp[3] + vp[0])
	f.Planes[0].NormalY = float64(vp[7] + vp[4])
	f.Planes[0].NormalZ = float64(vp[11] + vp[8])
	f.Planes[0].D = float64(vp[15] + vp[12])

	// Right Plane (w - x)
	f.Planes[1].NormalX = float64(vp[3] - vp[0])
	f.Planes[1].NormalY = float64(vp[7] - vp[4])
	f.Planes[1].NormalZ = float64(vp[11] - vp[8])
	f.Planes[1].D = float64(vp[15] - vp[12])

	// Bottom Plane (w + y)
	f.Planes[2].NormalX = float64(vp[3] + vp[1])
	f.Planes[2].NormalY = float64(vp[7] + vp[5])
	f.Planes[2].NormalZ = float64(vp[11] + vp[9])
	f.Planes[2].D = float64(vp[15] + vp[13])

	// Top Plane (w - y)
	f.Planes[3].NormalX = float64(vp[3] - vp[1])
	f.Planes[3].NormalY = float64(vp[7] - vp[5])
	f.Planes[3].NormalZ = float64(vp[11] - vp[9])
	f.Planes[3].D = float64(vp[15] - vp[13])

	// Near Plane (w + z) -> Assumendo OpenGL depth clipping da -1 a 1
	f.Planes[4].NormalX = float64(vp[3] + vp[2])
	f.Planes[4].NormalY = float64(vp[7] + vp[6])
	f.Planes[4].NormalZ = float64(vp[11] + vp[10])
	f.Planes[4].D = float64(vp[15] + vp[14])

	// Far Plane (w - z)
	f.Planes[5].NormalX = float64(vp[3] - vp[2])
	f.Planes[5].NormalY = float64(vp[7] - vp[6])
	f.Planes[5].NormalZ = float64(vp[11] - vp[10])
	f.Planes[5].D = float64(vp[15] - vp[14])
	// Normalizza tutti i piani
	for i := 0; i < 6; i++ {
		f.Planes[i].Normalize()
	}
	return f
}
