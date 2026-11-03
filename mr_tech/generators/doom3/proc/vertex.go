package proc

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Vertex represents a vertex in a Doom 3 .proc surface with coordinates, UVs, and normal.
type Vertex struct {
	Pos    geometry.XYZ
	U, V   float64
	Normal geometry.XYZ
}
