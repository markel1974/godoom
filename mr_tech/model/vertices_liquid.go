package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// VerticesLiquid represents a 2D grid structure of vertices, primarily used for creating and managing liquid-like surfaces.
type VerticesLiquid struct {
	volume       *Volume
	width, depth int
}

// NewVerticesLiquid initializes a liquid surface with a grid-based structure and associated physical volume.
func NewVerticesLiquid(cfg *config.Thing, materials *Materials) *VerticesLiquid {
	const res = 32

	v := &VerticesLiquid{
		volume: NewVolume(0, "liquid_surface", "liquid", cfg.Mass, 0.0, 0.0, 0.0),
		width:  res,
		depth:  res,
	}

	v.buildGrid(cfg.Radius)

	return v
}

// buildGrid constructs a grid of vertices and triangular faces within the specified radius for a liquid simulation.
func (v *VerticesLiquid) buildGrid(radius float64) {
	size := radius * 2.0
	step := size / float64(v.width)

	// Generiamo i vertici della griglia
	vertices := make([][]geometry.XYZ, v.width+1)
	for i := range vertices {
		vertices[i] = make([]geometry.XYZ, v.depth+1)
		for j := range vertices[i] {
			vertices[i][j] = geometry.XYZ{
				X: float64(i)*step - radius,
				Y: 0.0, // Piano di base
				Z: float64(j)*step - radius,
			}
		}
	}

	// Costruzione delle facce (triangoli)
	for i := 0; i < v.width; i++ {
		for j := 0; j < v.depth; j++ {
			// Triangolo 1
			v00 := vertices[i][j]
			v10 := vertices[i+1][j]
			v01 := vertices[i][j+1]
			v11 := vertices[i+1][j+1]

			// Ogni faccia viene aggiunta al volume.
			// Il vertex shader userà la posizione locale per l'offset temporale.
			f1 := NewFace([3]geometry.XYZ{v00, v10, v01}, "liquid_mat", nil)
			f2 := NewFace([3]geometry.XYZ{v10, v11, v01}, "liquid_mat", nil)

			v.volume.AddFace(f1)
			v.volume.AddFace(f2)
		}
	}
	v.volume.Rebuild()
}

// GetVolume returns the Volume instance associated with the VerticesLiquid.
func (v *VerticesLiquid) GetVolume() *Volume { return v.volume }

// GetEntity retrieves the physics.Entity associated with the VerticesLiquid instance.
func (v *VerticesLiquid) GetEntity() *physics.Entity { return v.volume.GetEntity() }

// GetAABB retrieves the axis-aligned bounding box (AABB) of the entity associated with the VerticesLiquid instance.
func (v *VerticesLiquid) GetAABB() *physics.AABB {
	return v.volume.GetEntity().GetAABB()
}

// GetVertices retrieves vertex and face information for liquid rendering at a specific simulation tick.
// Returns two sets of face pointers, their counts, and a float64 placeholder.
func (v *VerticesLiquid) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	f, c := v.volume.GetFaces()
	return f, c, f, c, 0.0, v.GetBillboard()
}

// SetAction updates the state or behavior of the liquid vertices at the specified index.
func (v *VerticesLiquid) SetAction(idx int) {

}

// GetDisplacement retrieves the bottom-left displacement coordinates (X, Y, Z) from the associated entity.
func (v *VerticesLiquid) GetDisplacement() (float64, float64, float64) {
	return v.volume.entity.GetBottomLeft()
}

// GetBillboard returns a float64 value representing the default billboard value associated with the VerticesLiquid instance.
func (v *VerticesLiquid) GetBillboard() float64 {
	return 0.0
}
