package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// Slope represents a geometric slope with X and Y gradients and a precomputed gradient value for calculations.
type Slope struct {
	G        geometry.XY `json:"g"`
	Gradient float64     `json:"gradient"`
}

// NewConfigSlope initializes and returns a pointer to a new Sloped instance with default values.
func NewConfigSlope() *Slope {
	return &Slope{
		G:        geometry.XY{X: 0, Y: 0},
		Gradient: 0,
	}
}
