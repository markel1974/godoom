package md5

import "github.com/markel1974/godoom/mr_tech/geometry"

// Joint holds the position and orientation of a single bone in model space.
type Joint struct {
	name   string
	parent int
	Pos    geometry.XYZ
	Orient [4]float64 // (qx, qy, qz, qw)
}

// NewJoint creates a joint with the given name, parent index, position, and orientation.
func NewJoint(name string, parent int, pos geometry.XYZ, orient [4]float64) *Joint {
	return &Joint{name: name, parent: parent, Pos: pos, Orient: orient}
}

// Name returns the joint's bone name.
func (j *Joint) Name() string {
	return j.name
}

// Parent returns the index of the parent joint (-1 for root).
func (j *Joint) Parent() int {
	return j.parent
}
