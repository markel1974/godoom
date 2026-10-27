package sectors

import "github.com/markel1974/godoom/mr_tech/geometry"

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
