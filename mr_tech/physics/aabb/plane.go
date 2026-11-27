package aabb

import "math"

// Plane rappresenta un piano matematico nello spazio 3D (Ax + By + Cz + D = 0).
type Plane struct {
	NormalX float64
	NormalY float64
	NormalZ float64
	D       float64
}

// NewPlane returns a new Plane with zero-valued coefficients.
func NewPlane() *Plane {
	return &Plane{}
}

// Update sets the plane coefficients and normalizes the resulting plane.
func (p *Plane) Update(nX, nY, nZ, d float64) {
	p.NormalX = nX
	p.NormalY = nY
	p.NormalZ = nZ
	p.D = d
	p.Normalize()
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
