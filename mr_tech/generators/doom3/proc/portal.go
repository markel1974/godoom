package proc

import "github.com/markel1974/godoom/mr_tech/geometry"

// Portal defines an inter-area portal between two areas in Doom 3.
type Portal struct {
	Points       []geometry.XYZ
	PositiveArea int
	NegativeArea int
}
