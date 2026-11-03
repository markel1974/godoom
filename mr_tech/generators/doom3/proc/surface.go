package proc

// Surface represents a single material surface in a Doom 3 .proc model.
type Surface struct {
	Material string
	Vertices []Vertex
	Indices  []int
}
