package collisions

// Faces stores a pair of related volume faces, typically representing left and right sides.
type Faces struct {
	lFace IFace
	rFace IFace
}

// NewFaces creates and returns an empty Faces instance.
func NewFaces() *Faces {
	return &Faces{}
}
