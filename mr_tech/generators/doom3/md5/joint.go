package md5

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Joint holds the position and orientation of a single bone in model space.
type Joint struct {
	name   string
	parent int
	Pos    geometry.XYZ
	Orient *aabb.Quaternion // (qx, qy, qz, qw)
}

// NewJoint creates a joint with the given name, parent index, position, and orientation.
func NewJoint(name string, parent int, pos geometry.XYZ, orient *aabb.Quaternion) *Joint {
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

// Clone returns a deep copy of the joint, including a cloned orientation quaternion.
func (j *Joint) Clone() *Joint {
	var orient *aabb.Quaternion
	if j.Orient != nil {
		orient = j.Orient.Clone()
	}
	return &Joint{
		name:   j.name,
		parent: j.parent,
		Pos:    j.Pos,
		Orient: orient,
	}
}
