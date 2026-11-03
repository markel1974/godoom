package md5

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// HeadJointTransform represents different types of transformations applied to head joints in a 3D space.
type HeadJointTransform int

// HeadTransformBody indicates a transformation tied to the body's reference point.
// HeadTransformChild indicates a transformation tied to a child joint or reference point.
const (
	HeadTransformBody HeadJointTransform = iota
	HeadTransformChild
)

// HeadJointBinding represents a binding between head and body joints, defining local transforms for hierarchical attachment.
type HeadJointBinding struct {
	kind      HeadJointTransform
	bodyIdx   int
	parentIdx int
	pLocal    geometry.XYZ
	qLocal    [4]float64
}

// NewHeadJointBinding initializes and returns a new HeadJointBinding with the given joint transformation type and body index.
func NewHeadJointBinding(kind HeadJointTransform, bodyIdx int) *HeadJointBinding {
	return &HeadJointBinding{
		kind:      kind,
		bodyIdx:   bodyIdx,
		parentIdx: 0,
		pLocal:    geometry.XYZ{},
		qLocal:    [4]float64{},
	}
}

// HeadAttachment represents a structure for attaching a head mesh to a body mesh using joint bindings.
type HeadAttachment struct {
	mesh     *Mesh
	bindings []*HeadJointBinding
}

// NewHeadAttachment creates a new HeadAttachment by binding a head mesh to a body mesh using joint mappings and transformations.
func NewHeadAttachment(bodyMesh *Mesh, headMesh *Mesh, attachJoint string) *HeadAttachment {
	if bodyMesh == nil || headMesh == nil || len(headMesh.Joints) == 0 {
		return nil
	}

	bodyJoints := make(map[string]int, len(bodyMesh.Joints))
	for i, j := range bodyMesh.Joints {
		bodyJoints[j.name] = i
	}

	attachIdx := -1
	if len(attachJoint) > 0 {
		if idx, ok := bodyJoints[attachJoint]; ok {
			attachIdx = idx
		}
	}
	if attachIdx < 0 {
		if idx, ok := bodyJoints["Shoulders"]; ok {
			attachIdx = idx
		} else {
			attachIdx = 0
		}
	}

	bindings := make([]*HeadJointBinding, len(headMesh.Joints))
	for i, hj := range headMesh.Joints {
		if hj.name == "origin" {
			bindings[i] = NewHeadJointBinding(HeadTransformBody, attachIdx)
		} else if bIdx, exists := bodyJoints[hj.name]; exists {
			bindings[i] = NewHeadJointBinding(HeadTransformBody, bIdx)
		} else {
			pIdx := hj.parent
			if pIdx < 0 || pIdx >= len(headMesh.Joints) {
				bindings[i] = NewHeadJointBinding(HeadTransformBody, attachIdx)
				continue
			}
			pj := headMesh.Joints[pIdx]
			invQ := QuatConjugate(pj.Orient)
			dPos := geometry.XYZ{X: hj.Pos.X - pj.Pos.X, Y: hj.Pos.Y - pj.Pos.Y, Z: hj.Pos.Z - pj.Pos.Z}
			hb := NewHeadJointBinding(HeadTransformChild, 0)
			hb.parentIdx = pIdx
			hb.pLocal = QuatRotate(invQ, dPos)
			hb.qLocal = QuatMultiplies(invQ, hj.Orient)
			bindings[i] = hb
		}
	}

	return &HeadAttachment{mesh: headMesh, bindings: bindings}
}

// BuildHeadSkeleton constructs a head skeleton by transforming and aligning it with the provided body skeleton joints.
func (h *HeadAttachment) BuildHeadSkeleton(bodySkeleton []Joint) []Joint {
	if h == nil || h.mesh == nil || len(bodySkeleton) == 0 {
		return nil
	}
	headSkeleton := make([]Joint, len(h.mesh.Joints))
	for i, hj := range h.mesh.Joints {
		b := h.bindings[i]
		if b.kind == HeadTransformBody {
			if b.bodyIdx >= 0 && b.bodyIdx < len(bodySkeleton) {
				bj := bodySkeleton[b.bodyIdx]
				headSkeleton[i] = Joint{
					name:   hj.name,
					parent: hj.parent,
					Pos:    bj.Pos,
					Orient: bj.Orient,
				}
			}
		} else {
			if b.parentIdx >= 0 && b.parentIdx < len(headSkeleton) {
				pEval := headSkeleton[b.parentIdx]
				rPos := QuatRotate(pEval.Orient, b.pLocal)
				cPos := geometry.XYZ{X: pEval.Pos.X + rPos.X, Y: pEval.Pos.Y + rPos.Y, Z: pEval.Pos.Z + rPos.Z}
				cOrient := QuatNormalize(QuatMultiplies(pEval.Orient, b.qLocal))
				headSkeleton[i] = Joint{name: hj.name, parent: hj.parent, Pos: cPos, Orient: cOrient}
			}
		}
	}
	return headSkeleton
}
