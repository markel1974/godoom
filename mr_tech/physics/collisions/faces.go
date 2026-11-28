package collisions

// Faces stores a reusable pair of Face wrappers, typically representing left and right sides.
type Faces struct {
	lFace *Face
	rFace *Face
}

// NewFaces creates and returns an initialized Faces pair.
func NewFaces() *Faces {
	return &Faces{
		lFace: NewFace(),
		rFace: NewFace(),
	}
}
