package volumes

import (
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Vertices3DJoint stores the physical state and configuration of a joint connecting articulated parts.
type Vertices3DJoint struct {
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

// NewVertices3DJoint creates a Vertices3DJoint initialized with identity quaternions and axes.
func NewVertices3DJoint() *Vertices3DJoint {
	return &Vertices3DJoint{
		quat:      physics.NewQuaternionIdentity(),
		detQuat:   physics.NewQuaternionIdentity(),
		axis:      NewTagAxis(geometry.XYZ{}, _im),
		combinedA: NewTagAxis(geometry.XYZ{}, _im),
		combinedB: NewTagAxis(geometry.XYZ{}, _im),
	}
}

// Vertices3DJoints stores joint state data for model parts during animated and ragdoll physics.
type Vertices3DJoints struct {
	data []*Vertices3DJoint
}

// NewVertices3DJoints creates a Vertices3DJoints collection with the specified number of initialized joint states.
func NewVertices3DJoints() *Vertices3DJoints {
	pj := &Vertices3DJoints{}
	return pj
}

// Setup configures anatomical joint parameters, local bounds, and pre-allocated matrices for each part.
func (v *Vertices3DJoints) Setup(_ *config.Model3D, elements []*Vertices3DEntry, links *Vertices3DLinks) error {
	//TODO REWRITE
	numPart := len(elements)
	v.data = make([]*Vertices3DJoint, numPart)
	for i := 0; i < numPart; i++ {
		v.data[i] = NewVertices3DJoint()
	}

	for i := range v.data {
		j := v.data[i]
		j.quat.SetIdentity()
		j.detQuat.SetIdentity()
		j.axis.Reset()
		j.combinedA.Reset()
		j.combinedB.Reset()
		j.partRadius = 8.0

		if i < len(elements) && elements[i] != nil && len(elements[i].volumes) > 0 && elements[i].volumes[0] != nil {
			vol := elements[i].volumes[0]
			fc := vol.GetFaceCount()
			if fc > 0 {
				minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
				maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
				for fIdx := 0; fIdx < fc; fIdx++ {
					f := vol.GetFace(fIdx)
					minX, minY, minZ, maxX, maxY, maxZ = f.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
				}
				j.localCenter = geometry.XYZ{X: (minX + maxX) * 0.5, Y: (minY + maxY) * 0.5, Z: (minZ + maxZ) * 0.5}
				dx, dy, dz := maxX-minX, maxY-minY, maxZ-minZ
				r := math.Max(dx, math.Max(dy, dz)) * 0.5
				j.partRadius = math.Max(4.0, math.Min(48.0, r))
			}
		}

		tagLow := ""
		if i < len(links.data) {
			tagLow = strings.ToLower(links.data[i].tag)
		}

		switch {
		case i == 0 || links.data[i].parent < 0:
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 0, 0, 0
			j.maxAngleAlive = 0
			j.maxAngleRagdoll = 0
			j.stiffness = 80.0
			j.damping = 10.0
			j.canDetach = false

		case strings.Contains(tagLow, "torso") || strings.Contains(tagLow, "chest") ||
			strings.Contains(tagLow, "waist") || strings.Contains(tagLow, "pelvis") ||
			strings.Contains(tagLow, "hips") || strings.Contains(tagLow, "belly") ||
			strings.Contains(tagLow, "spine") || strings.Contains(tagLow, "shoulders") ||
			strings.Contains(tagLow, "upper"):
			// Torso / Spine: 3 DoF (Pitch, Roll, Yaw), core trunk stays connected
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.85
			j.maxAngleAlive = 0.48
			j.maxAngleRagdoll = 0.85
			j.stiffness = 58.0
			j.damping = 7.5
			j.canDetach = false

		case strings.Contains(tagLow, "head") || strings.Contains(tagLow, "neck") || strings.Contains(tagLow, "jaw"):
			// Head / Neck: 2-3 DoF with decapitation threshold
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.75
			j.maxAngleAlive = 0.60
			j.maxAngleRagdoll = 1.15
			j.stiffness = 48.0
			j.damping = 6.5
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 1.25

		case strings.Contains(tagLow, "lloarm") || strings.Contains(tagLow, "rloarm") ||
			strings.Contains(tagLow, "lhand") || strings.Contains(tagLow, "rhand") ||
			strings.Contains(tagLow, "lloleg") || strings.Contains(tagLow, "rloleg") ||
			strings.Contains(tagLow, "lfoot") || strings.Contains(tagLow, "rfoot"):
			// Distal limbs (forearms, hands, lower legs): 1-2 DoF hinge, detachable
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 0.90, 0.40
			j.maxAngleAlive = 0.72
			j.maxAngleRagdoll = 1.35
			j.stiffness = 40.0
			j.damping = 5.5
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 1.05

		case strings.Contains(tagLow, "weapon"):
			// Held weapon: disarms/flies off on hit
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.65
			j.maxAngleAlive = 0.68
			j.maxAngleRagdoll = 1.35
			j.stiffness = 42.0
			j.damping = 5.5
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 0.95

		default:
			// Upper arms and upper legs (luparm, ruparm, lupleg, rupleg, tail, etc.): 2-3 DoF ball joint, detachable on heavy/ragdoll hits
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.65
			j.maxAngleAlive = 0.62
			j.maxAngleRagdoll = 1.20
			j.stiffness = 46.0
			j.damping = 6.2
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 1.30
		}
	}

	return nil
}
