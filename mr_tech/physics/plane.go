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
