package proc

// Surface represents a geometric surface in a Doom 3 model with material, vertices, and triangle indices.
type Surface struct {
	Material string
	Vertices []Vertex
	Indices  []int
}

// NewSurface creates a new Surface with the specified material, vertices, and indices.
// It returns a pointer to the constructed Surface.
func NewSurface(material string, vertices []Vertex, indices []int) *Surface {
	return &Surface{
		Material: material,
		Vertices: vertices,
		Indices:  indices,
	}
}
