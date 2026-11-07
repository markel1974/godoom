package collisions

import "github.com/markel1974/godoom/mr_tech/model/volumes"

// Faces stores a pair of related volume faces, typically representing left and right sides.
type Faces struct {
	lFace *volumes.Face
	rFace *volumes.Face
}

// NewFaces creates and returns an empty Faces instance.
func NewFaces() *Faces {
	return &Faces{}
}
