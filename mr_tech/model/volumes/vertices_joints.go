package volumes

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// PartJointState stores the physical state and configuration of a joint connecting articulated parts.
type PartJointState struct {
	localCenter     geometry.XYZ
	partRadius      float64
	dofScaleX       float64
	dofScaleY       float64
	dofScaleZ       float64
	maxAngleAlive   float64
	maxAngleRagdoll float64
	stiffness       float64
	damping         float64
	canDetach       bool
	detachThreshold float64

	angX           float64
	angY           float64
	angZ           float64
	velX           float64
	velY           float64
	velZ           float64
	ragdollTargetX float64
	ragdollTargetY float64
	ragdollTargetZ float64
	damage         float64
	hasDeflection  bool

	detached   bool
	detSettled bool
	detWorldX  float64
	detWorldY  float64
	detWorldZ  float64
	detVx      float64
	detVy      float64
	detVz      float64
	detWx      float64
	detWy      float64
	detWz      float64
	detFloorZ  float64

	quat      *physics.Quaternion
	detQuat   *physics.Quaternion
	axis      *TagAxis
	combinedA *TagAxis
	combinedB *TagAxis
}

// NewPartJointState creates a PartJointState initialized with identity quaternions and axes.
func NewPartJointState() *PartJointState {
	return &PartJointState{
		quat:      physics.NewQuaternionIdentity(),
		detQuat:   physics.NewQuaternionIdentity(),
		axis:      NewTagAxis(geometry.XYZ{}, _im),
		combinedA: NewTagAxis(geometry.XYZ{}, _im),
		combinedB: NewTagAxis(geometry.XYZ{}, _im),
	}
}

// PartJoints stores joint state data for model parts during animated and ragdoll physics.
type PartJoints struct {
	data []*PartJointState
}

// NewPartJoints creates a PartJoints collection with the specified number of initialized joint states.
func NewPartJoints(size int) *PartJoints {
	pj := &PartJoints{
		data: make([]*PartJointState, size),
	}
	for i := 0; i < size; i++ {
		pj.data[i] = NewPartJointState()
	}
	return pj
}
