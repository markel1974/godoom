package proc

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Vertex represents a point in 3D space with position, texture coordinates, and a normal vector.
type Vertex struct {
	Pos    geometry.XYZ
	U, V   float64
	Normal geometry.XYZ
}

// NewVertex creates a new Vertex with the given position (x, y, z), normal (nx, ny, nz), and texture coordinates (u, v).
func NewVertex(x, y, z, nx, ny, nz, u, v float64) *Vertex {
	return &Vertex{
		Pos:    geometry.XYZ{X: x, Y: y, Z: z},
		U:      u,
		V:      v,
		Normal: geometry.XYZ{X: nx, Y: ny, Z: nz},
	}
}
