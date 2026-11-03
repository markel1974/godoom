package doom3

import "github.com/markel1974/godoom/mr_tech/geometry"

// HeadJointTransformType indicates whether a head joint maps directly to a body joint or to a child bone.
type HeadJointTransformType int

const (
	HeadTransformBody HeadJointTransformType = iota
	HeadTransformChild
)

type HeadJointBinding struct {
	kind      HeadJointTransformType
	bodyIdx   int
	parentIdx int
	pLocal    geometry.XYZ
	qLocal    [4]float64
}

func NewHeadJointBinding(kind HeadJointTransformType, bodyIdx int) *HeadJointBinding {
	return &HeadJointBinding{
		kind:      kind,
		bodyIdx:   bodyIdx,
		parentIdx: 0,
		pLocal:    geometry.XYZ{},
		qLocal:    [4]float64{},
	}
}

// HeadAttachment handles linking a separate head MD5 mesh (e.g. helmets, zombie heads) to an animated body skeleton.
type HeadAttachment struct {
	mesh     *MD5Mesh
	bindings []*HeadJointBinding
}

// NewHeadAttachment builds joint bindings between the head mesh and the body skeleton.
func NewHeadAttachment(bodyMesh *MD5Mesh, headMesh *MD5Mesh, attachJoint string) *HeadAttachment {
	if bodyMesh == nil || headMesh == nil || len(headMesh.joints) == 0 {
		return nil
	}

	bodyJoints := make(map[string]int, len(bodyMesh.joints))
	for i, j := range bodyMesh.joints {
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

	bindings := make([]*HeadJointBinding, len(headMesh.joints))
	for i, hj := range headMesh.joints {
		if hj.name == "origin" {
			bindings[i] = NewHeadJointBinding(HeadTransformBody, attachIdx)
		} else if bIdx, exists := bodyJoints[hj.name]; exists {
			bindings[i] = NewHeadJointBinding(HeadTransformBody, bIdx)
		} else {
			pIdx := hj.parent
			if pIdx < 0 || pIdx >= len(headMesh.joints) {
				bindings[i] = NewHeadJointBinding(HeadTransformBody, attachIdx)
				continue
			}
			pj := headMesh.joints[pIdx]
			invQ := QuatConjugate(pj.orient)
			dPos := geometry.XYZ{X: hj.pos.X - pj.pos.X, Y: hj.pos.Y - pj.pos.Y, Z: hj.pos.Z - pj.pos.Z}
			hb := NewHeadJointBinding(HeadTransformChild, 0)
			hb.parentIdx = pIdx
			hb.pLocal = QuatRotate(invQ, dPos)
			hb.qLocal = QuatMultiplies(invQ, hj.orient)
			bindings[i] = hb
		}
	}

	return &HeadAttachment{mesh: headMesh, bindings: bindings}
}

// BuildHeadSkeleton calculates the world pose of all head joints for the given body skeleton pose.
func (h *HeadAttachment) BuildHeadSkeleton(bodySkeleton []md5Joint) []md5Joint {
	if h == nil || h.mesh == nil || len(bodySkeleton) == 0 {
		return nil
	}
	headSkeleton := make([]md5Joint, len(h.mesh.joints))
	for i, hj := range h.mesh.joints {
		b := h.bindings[i]
		if b.kind == HeadTransformBody {
			if b.bodyIdx >= 0 && b.bodyIdx < len(bodySkeleton) {
				bj := bodySkeleton[b.bodyIdx]
				headSkeleton[i] = md5Joint{
					name:   hj.name,
					parent: hj.parent,
					pos:    bj.pos,
					orient: bj.orient,
				}
			}
		} else {
			if b.parentIdx >= 0 && b.parentIdx < len(headSkeleton) {
				pEval := headSkeleton[b.parentIdx]
				rPos := QuatRotate(pEval.orient, b.pLocal)
				cPos := geometry.XYZ{
					X: pEval.pos.X + rPos.X,
					Y: pEval.pos.Y + rPos.Y,
					Z: pEval.pos.Z + rPos.Z,
				}
				cOrient := QuatNormalize(QuatMultiplies(pEval.orient, b.qLocal))
				headSkeleton[i] = md5Joint{
					name:   hj.name,
					parent: hj.parent,
					pos:    cPos,
					orient: cOrient,
				}
			}
		}
	}
	return headSkeleton
}
