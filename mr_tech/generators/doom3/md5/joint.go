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
