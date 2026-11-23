package draws

import "github.com/markel1974/godoom/mr_tech/textures"

// FrameBatch represents a single drawing batch associated with a blend mode and material.
type FrameBatch struct {
	blendMode  int
	indexCount int32
	material   *textures.Material
	centroidX  float32
	centroidY  float32
	centroidZ  float32
	centCount  int32
}

// NewFrameBatches creates n initialized FrameBatch pointers.
func NewFrameBatches(n int) []*FrameBatch {
	b := make([]*FrameBatch, n)
	for i := range b {
		b[i] = &FrameBatch{}
	}
	return b
}
