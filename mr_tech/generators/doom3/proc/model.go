package proc

// Model represents a geometric model in Doom 3 containing named surfaces.
type Model struct {
	Name     string
	Surfaces []*Surface
}

// NewModel creates a new Model instance with the provided name and preallocated surfaces based on numSurfaces.
func NewModel(name string, numSurfaces int) *Model {
	return &Model{
		Name:     name,
		Surfaces: make([]*Surface, 0, numSurfaces),
	}
}
