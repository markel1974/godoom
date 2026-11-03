package proc

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Portal defines an inter-area portal between two areas in Doom 3.
type Portal struct {
	Points       []geometry.XYZ
	PositiveArea int
	NegativeArea int
}

// NewPorta creates and initializes a Portal with specified points, positive area, and negative area.
func NewPorta(numPoints, positiveArea, negativeArea int) *Portal {
	return &Portal{
		Points:       make([]geometry.XYZ, numPoints),
		PositiveArea: positiveArea,
		NegativeArea: negativeArea,
	}
}
