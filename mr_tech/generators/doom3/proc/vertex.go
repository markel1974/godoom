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

func NewVertex(x, y, z, nx, ny, nz, u, v float64) *Vertex {
	//vert.Pos = geometry.XYZ{X: x, Y: y, Z: z}
	//vert.U = u
	//vert.V = v
	//vert.Normal = geometry.XYZ{X: nx, Y: ny, Z: nz}

	return &Vertex{
		Pos:    geometry.XYZ{X: x, Y: y, Z: z},
		U:      u,
		V:      v,
		Normal: geometry.XYZ{X: nx, Y: ny, Z: nz},
	}
}
