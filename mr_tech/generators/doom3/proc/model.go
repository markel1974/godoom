package proc

// Model represents a geometric model in Doom 3 containing named surfaces.
type Model struct {
	Name     string
	Surfaces []*Surface
}
