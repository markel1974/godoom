package model

import "github.com/markel1974/godoom/mr_tech/geometry"

// Slope represents a 2D inclined plane characterized by its normal vector and gradient for geometric calculations.
type Slope struct {
	G        geometry.XY
	Gradient float64
	Start    geometry.XY
	End      geometry.XY
	N        geometry.XY
}

func NewSlope(G geometry.XY, Gradient float64, Start geometry.XY, End geometry.XY) *Slope {
	return &Slope{
		G:        G,
		Gradient: Gradient,
		Start:    Start,
		End:      End,
	}
}

func (s *Slope) ComputeZ(base float64, x float64, y float64) float64 {
	z := base + s.G.X*(x-s.Start.X) + s.G.Y*(y-s.Start.Y)
	return z
}
