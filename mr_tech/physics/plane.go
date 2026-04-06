package physics

import "math"

// Plane rappresenta un piano matematico nello spazio 3D (Ax + By + Cz + D = 0).
type Plane struct {
	NormalX float64
	NormalY float64
	NormalZ float64
	D       float64
}

// Normalize normalizza il vettore normale del piano e aggiorna la distanza D.
func (p *Plane) Normalize() {
	mag := math.Sqrt(p.NormalX*p.NormalX + p.NormalY*p.NormalY + p.NormalZ*p.NormalZ)
	if mag > 0 {
		p.NormalX /= mag
		p.NormalY /= mag
		p.NormalZ /= mag
		p.D /= mag
	}
}

// Frustum rappresenta il volume visivo della telecamera delimitato da 6 piani.
type Frustum struct {
	Planes [6]*Plane
}

// NewFrustum estrae i 6 piani del Frustum da una matrice combinata View-Projection (Column-Major).
func NewFrustum(vp [16]float32) *Frustum {
	f := &Frustum{}

	// La matrice in Go/OpenGL è un array 1D column-major: vp[col*4 + row]
	// Estrazione dei piani (A, B, C, D)

	// Left Plane (w + x)
	f.Planes[0] = &Plane{
		NormalX: float64(vp[3] + vp[0]),
		NormalY: float64(vp[7] + vp[4]),
		NormalZ: float64(vp[11] + vp[8]),
		D:       float64(vp[15] + vp[12]),
	}

	// Right Plane (w - x)
	f.Planes[1] = &Plane{
		NormalX: float64(vp[3] - vp[0]),
		NormalY: float64(vp[7] - vp[4]),
		NormalZ: float64(vp[11] - vp[8]),
		D:       float64(vp[15] - vp[12]),
	}

	// Bottom Plane (w + y)
	f.Planes[2] = &Plane{
		NormalX: float64(vp[3] + vp[1]),
		NormalY: float64(vp[7] + vp[5]),
		NormalZ: float64(vp[11] + vp[9]),
		D:       float64(vp[15] + vp[13]),
	}

	// Top Plane (w - y)
	f.Planes[3] = &Plane{
		NormalX: float64(vp[3] - vp[1]),
		NormalY: float64(vp[7] - vp[5]),
		NormalZ: float64(vp[11] - vp[9]),
		D:       float64(vp[15] - vp[13]),
	}

	// Near Plane (w + z) -> Assumendo OpenGL depth clipping da -1 a 1
	f.Planes[4] = &Plane{
		NormalX: float64(vp[3] + vp[2]),
		NormalY: float64(vp[7] + vp[6]),
		NormalZ: float64(vp[11] + vp[10]),
		D:       float64(vp[15] + vp[14]),
	}

	// Far Plane (w - z)
	f.Planes[5] = &Plane{
		NormalX: float64(vp[3] - vp[2]),
		NormalY: float64(vp[7] - vp[6]),
		NormalZ: float64(vp[11] - vp[10]),
		D:       float64(vp[15] - vp[14]),
	}

	// Normalizza tutti i piani
	for i := 0; i < 6; i++ {
		f.Planes[i].Normalize()
	}

	return f
}
