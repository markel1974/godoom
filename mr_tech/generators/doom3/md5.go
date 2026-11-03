package doom3

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// md5Joint holds the position and orientation of a single bone in model space.
type md5Joint struct {
	name   string
	parent int
	pos    geometry.XYZ
	orient [4]float64 // (qx, qy, qz, qw)
}

// md5Weight represents a vertex skinning weight attached to a specific joint.
type md5Weight struct {
	joint int
	bias  float64
	pos   geometry.XYZ
}

// md5Vert represents a vertex definition in the MD5 mesh.
type md5Vert struct {
	u          float32
	v          float32
	weightIdx  int
	weightSpan int
}

// md5SubMesh holds the parsed geometry and weight list of an individual sub-mesh.
type md5SubMesh struct {
	shader  string
	verts   []md5Vert
	tris    [][3]int
	weights []md5Weight
}

// MD5Mesh holds the parsed joints and submeshes of a .md5mesh file.
type MD5Mesh struct {
	joints    []md5Joint
	subMeshes []md5SubMesh
}

// md5HierarchyJoint represents a bone in the hierarchy section of an .md5anim file.
type md5HierarchyJoint struct {
	name       string
	parent     int
	flags      int
	startIndex int
}

// md5BaseFrameJoint represents a joint's default local position and orientation in an .md5anim file.
type md5BaseFrameJoint struct {
	pos    geometry.XYZ
	orient [4]float64
}

// MD5Anim holds the parsed animation data of an .md5anim file.
type MD5Anim struct {
	numFrames int
	frameRate int
	hierarchy []md5HierarchyJoint
	baseframe []md5BaseFrameJoint
	frames    [][]float64
}

// quatRotate rotates a 3D vector by a unit quaternion (qx, qy, qz, qw).
func quatRotate(q [4]float64, v geometry.XYZ) geometry.XYZ {
	qx, qy, qz, qw := q[0], q[1], q[2], q[3]

	tx := 2.0 * (qy*v.Z - qz*v.Y)
	ty := 2.0 * (qz*v.X - qx*v.Z)
	tz := 2.0 * (qx*v.Y - qy*v.X)

	rx := v.X + qw*tx + (qy*tz - qz*ty)
	ry := v.Y + qw*ty + (qz*tx - qx*tz)
	rz := v.Z + qw*tz + (qx*ty - qy*tx)

	return geometry.XYZ{X: rx, Y: ry, Z: rz}
}

// quatMult multiplies two unit quaternions q1 and q2.
func quatMult(q1, q2 [4]float64) [4]float64 {
	x1, y1, z1, w1 := q1[0], q1[1], q1[2], q1[3]
	x2, y2, z2, w2 := q2[0], q2[1], q2[2], q2[3]

	return [4]float64{
		w1*x2 + x1*w2 + y1*z2 - z1*y2,
		w1*y2 - x1*z2 + y1*w2 + z1*x2,
		w1*z2 + x1*y2 - y1*x2 + z1*w2,
		w1*w2 - x1*x2 - y1*y2 - z1*z2,
	}
}

// quatNormalize normalizes a quaternion to unit length.
func quatNormalize(q [4]float64) [4]float64 {
	lenSq := q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3]
	if lenSq <= 0.0 {
		return [4]float64{0, 0, 0, 1}
	}
	l := math.Sqrt(lenSq)
	return [4]float64{q[0] / l, q[1] / l, q[2] / l, q[3] / l}
}

// quatConj returns the conjugate of a unit quaternion, which is its inverse.
func quatConj(q [4]float64) [4]float64 {
	return [4]float64{-q[0], -q[1], -q[2], q[3]}
}

// ParseMD5Mesh parses a Doom 3 .md5mesh file.
func ParseMD5Mesh(r io.Reader) (*MD5Mesh, error) {
	scanner := bufio.NewScanner(r)

	mesh := &MD5Mesh{}
	inJoints := false
	inMesh := false

	var currentSubMesh md5SubMesh

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "//") {
			continue
		}

		if commentIdx := strings.Index(line, "//"); commentIdx >= 0 {
			line = strings.TrimSpace(line[:commentIdx])
		}

		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		if tokens[0] == "joints" && len(tokens) > 1 && tokens[1] == "{" {
			inJoints = true
			continue
		}
		if tokens[0] == "mesh" && len(tokens) > 1 && tokens[1] == "{" {
			inMesh = true
			currentSubMesh = md5SubMesh{}
			continue
		}
		if tokens[0] == "}" {
			if inJoints {
				inJoints = false
			} else if inMesh {
				if len(currentSubMesh.verts) > 0 && len(currentSubMesh.tris) > 0 {
					mesh.subMeshes = append(mesh.subMeshes, currentSubMesh)
				}
				inMesh = false
			}
			continue
		}

		if inJoints {
			quoteStart := strings.Index(line, "\"")
			quoteEnd := strings.LastIndex(line, "\"")
			if quoteStart < 0 || quoteEnd <= quoteStart {
				continue
			}
			name := line[quoteStart+1 : quoteEnd]
			rest := strings.TrimSpace(line[quoteEnd+1:])
			subTokens := strings.Fields(rest)
			if len(subTokens) < 9 {
				continue
			}
			parent, _ := strconv.Atoi(subTokens[0])

			pStart := strings.Index(rest, "(")
			pEnd := strings.Index(rest, ")")
			if pStart < 0 || pEnd <= pStart {
				continue
			}
			pFields := strings.Fields(rest[pStart+1 : pEnd])
			if len(pFields) < 3 {
				continue
			}
			px, _ := strconv.ParseFloat(pFields[0], 64)
			py, _ := strconv.ParseFloat(pFields[1], 64)
			pz, _ := strconv.ParseFloat(pFields[2], 64)

			qRest := rest[pEnd+1:]
			qStart := strings.Index(qRest, "(")
			qEnd := strings.Index(qRest, ")")
			if qStart < 0 || qEnd <= qStart {
				continue
			}
			qFields := strings.Fields(qRest[qStart+1 : qEnd])
			if len(qFields) < 3 {
				continue
			}
			qx, _ := strconv.ParseFloat(qFields[0], 64)
			qy, _ := strconv.ParseFloat(qFields[1], 64)
			qz, _ := strconv.ParseFloat(qFields[2], 64)

			// In Doom 3 MD5 format, qw is strictly negative
			t := 1.0 - (qx*qx + qy*qy + qz*qz)
			var qw float64
			if t > 0.0 {
				qw = -math.Sqrt(t)
			}

			mesh.joints = append(mesh.joints, md5Joint{
				name:   name,
				parent: parent,
				pos:    geometry.XYZ{X: px, Y: py, Z: pz},
				orient: [4]float64{qx, qy, qz, qw},
			})
			continue
		}

		if inMesh {
			switch tokens[0] {
			case "shader":
				if len(tokens) > 1 {
					currentSubMesh.shader = strings.Trim(tokens[1], "\"")
				}
			case "vert":
				pStart := strings.Index(line, "(")
				pEnd := strings.Index(line, ")")
				if pStart >= 0 && pEnd > pStart {
					uvFields := strings.Fields(line[pStart+1 : pEnd])
					tailFields := strings.Fields(line[pEnd+1:])
					if len(uvFields) >= 2 && len(tailFields) >= 2 {
						u, _ := strconv.ParseFloat(uvFields[0], 32)
						v, _ := strconv.ParseFloat(uvFields[1], 32)
						startW, _ := strconv.Atoi(tailFields[0])
						countW, _ := strconv.Atoi(tailFields[1])

						currentSubMesh.verts = append(currentSubMesh.verts, md5Vert{
							u:          float32(u),
							v:          float32(v),
							weightIdx:  startW,
							weightSpan: countW,
						})
					}
				}
			case "tri":
				if len(tokens) >= 5 {
					i0, _ := strconv.Atoi(tokens[2])
					i1, _ := strconv.Atoi(tokens[3])
					i2, _ := strconv.Atoi(tokens[4])
					currentSubMesh.tris = append(currentSubMesh.tris, [3]int{i0, i1, i2})
				}
			case "weight":
				pStart := strings.Index(line, "(")
				pEnd := strings.Index(line, ")")
				if pStart >= 0 && pEnd > pStart {
					headFields := strings.Fields(line[:pStart])
					posFields := strings.Fields(line[pStart+1 : pEnd])
					if len(headFields) >= 4 && len(posFields) >= 3 {
						joint, _ := strconv.Atoi(headFields[2])
						bias, _ := strconv.ParseFloat(headFields[3], 64)
						wx, _ := strconv.ParseFloat(posFields[0], 64)
						wy, _ := strconv.ParseFloat(posFields[1], 64)
						wz, _ := strconv.ParseFloat(posFields[2], 64)

						currentSubMesh.weights = append(currentSubMesh.weights, md5Weight{
							joint: joint,
							bias:  bias,
							pos:   geometry.XYZ{X: wx, Y: wy, Z: wz},
						})
					}
				}
			}
		}
	}

	if inMesh && len(currentSubMesh.verts) > 0 && len(currentSubMesh.tris) > 0 {
		mesh.subMeshes = append(mesh.subMeshes, currentSubMesh)
	}

	if len(mesh.subMeshes) == 0 {
		return nil, fmt.Errorf("no valid sub-meshes parsed from md5mesh")
	}

	return mesh, nil
}

// ParseMD5Anim parses an .md5anim file.
func ParseMD5Anim(r io.Reader) (*MD5Anim, error) {
	scanner := bufio.NewScanner(r)

	anim := &MD5Anim{}
	inHierarchy := false
	inBaseFrame := false
	inFrame := false
	currentFrameIdx := -1
	var currentFrameVals []float64

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "//") {
			continue
		}

		if commentIdx := strings.Index(line, "//"); commentIdx >= 0 {
			line = strings.TrimSpace(line[:commentIdx])
		}

		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		if tokens[0] == "numFrames" && len(tokens) > 1 {
			anim.numFrames, _ = strconv.Atoi(tokens[1])
			continue
		}
		if tokens[0] == "frameRate" && len(tokens) > 1 {
			anim.frameRate, _ = strconv.Atoi(tokens[1])
			continue
		}
		if tokens[0] == "hierarchy" && len(tokens) > 1 && tokens[1] == "{" {
			inHierarchy = true
			continue
		}
		if tokens[0] == "baseframe" && len(tokens) > 1 && tokens[1] == "{" {
			inBaseFrame = true
			continue
		}
		if tokens[0] == "frame" && len(tokens) > 2 && tokens[2] == "{" {
			inFrame = true
			currentFrameIdx, _ = strconv.Atoi(tokens[1])
			currentFrameVals = make([]float64, 0, 64)
			_ = currentFrameIdx
			continue
		}

		if tokens[0] == "}" {
			if inHierarchy {
				inHierarchy = false
			} else if inBaseFrame {
				inBaseFrame = false
			} else if inFrame {
				anim.frames = append(anim.frames, currentFrameVals)
				inFrame = false
			}
			continue
		}

		if inHierarchy {
			quoteStart := strings.Index(line, "\"")
			quoteEnd := strings.LastIndex(line, "\"")
			if quoteStart < 0 || quoteEnd <= quoteStart {
				continue
			}
			name := line[quoteStart+1 : quoteEnd]
			rest := strings.TrimSpace(line[quoteEnd+1:])
			subTokens := strings.Fields(rest)
			if len(subTokens) >= 3 {
				parent, _ := strconv.Atoi(subTokens[0])
				flags, _ := strconv.Atoi(subTokens[1])
				sIdx, _ := strconv.Atoi(subTokens[2])
				anim.hierarchy = append(anim.hierarchy, md5HierarchyJoint{
					name:       name,
					parent:     parent,
					flags:      flags,
					startIndex: sIdx,
				})
			}
			continue
		}

		if inBaseFrame {
			pStart := strings.Index(line, "(")
			pEnd := strings.Index(line, ")")
			qStart := strings.LastIndex(line, "(")
			qEnd := strings.LastIndex(line, ")")
			if pStart >= 0 && pEnd > pStart && qStart >= 0 && qEnd > qStart && qStart > pEnd {
				pFields := strings.Fields(line[pStart+1 : pEnd])
				qFields := strings.Fields(line[qStart+1 : qEnd])
				if len(pFields) >= 3 && len(qFields) >= 3 {
					px, _ := strconv.ParseFloat(pFields[0], 64)
					py, _ := strconv.ParseFloat(pFields[1], 64)
					pz, _ := strconv.ParseFloat(pFields[2], 64)

					qx, _ := strconv.ParseFloat(qFields[0], 64)
					qy, _ := strconv.ParseFloat(qFields[1], 64)
					qz, _ := strconv.ParseFloat(qFields[2], 64)

					anim.baseframe = append(anim.baseframe, md5BaseFrameJoint{
						pos:    geometry.XYZ{X: px, Y: py, Z: pz},
						orient: [4]float64{qx, qy, qz, 0},
					})
				}
			}
			continue
		}

		if inFrame {
			for _, tStr := range tokens {
				val, err := strconv.ParseFloat(tStr, 64)
				if err == nil {
					currentFrameVals = append(currentFrameVals, val)
				}
			}
			continue
		}
	}

	return anim, nil
}

// BuildFrameSkeleton constructs the animated skeleton joints in model space for a specific frame index.
func (anim *MD5Anim) BuildFrameSkeleton(frameIdx int) []md5Joint {
	skel := make([]md5Joint, len(anim.hierarchy))
	if frameIdx < 0 || frameIdx >= len(anim.frames) {
		return skel
	}
	frameData := anim.frames[frameIdx]

	for i, h := range anim.hierarchy {
		base := anim.baseframe[i]
		pos := base.pos
		orient := base.orient
		k := 0

		if h.flags&1 != 0 {
			if h.startIndex+k < len(frameData) {
				pos.X = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&2 != 0 {
			if h.startIndex+k < len(frameData) {
				pos.Y = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&4 != 0 {
			if h.startIndex+k < len(frameData) {
				pos.Z = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&8 != 0 {
			if h.startIndex+k < len(frameData) {
				orient[0] = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&16 != 0 {
			if h.startIndex+k < len(frameData) {
				orient[1] = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&32 != 0 {
			if h.startIndex+k < len(frameData) {
				orient[2] = frameData[h.startIndex+k]
				k++
			}
		}

		// Recompute qw with strict negative convention for Doom 3
		t := 1.0 - (orient[0]*orient[0] + orient[1]*orient[1] + orient[2]*orient[2])
		if t > 0.0 {
			orient[3] = -math.Sqrt(t)
		} else {
			orient[3] = 0.0
		}

		if h.parent < 0 {
			skel[i] = md5Joint{
				name:   h.name,
				parent: h.parent,
				pos:    pos,
				orient: orient,
			}
		} else {
			parentJoint := skel[h.parent]
			rpos := quatRotate(parentJoint.orient, pos)
			skel[i] = md5Joint{
				name:   h.name,
				parent: h.parent,
				pos: geometry.XYZ{
					X: parentJoint.pos.X + rpos.X,
					Y: parentJoint.pos.Y + rpos.Y,
					Z: parentJoint.pos.Z + rpos.Z,
				},
				orient: quatNormalize(quatMult(parentJoint.orient, orient)),
			}
		}
	}

	return skel
}

// headJointTransformType indicates whether a head joint maps directly to a body joint or to a child bone.
type headJointTransformType int

const (
	headTransformBody headJointTransformType = iota
	headTransformChild
)

type headJointBinding struct {
	kind      headJointTransformType
	bodyIdx   int
	parentIdx int
	pLocal    geometry.XYZ
	qLocal    [4]float64
}

// HeadAttachment handles linking a separate head MD5 mesh (e.g. helmets, zombie heads) to an animated body skeleton.
type HeadAttachment struct {
	mesh     *MD5Mesh
	bindings []headJointBinding
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

	bindings := make([]headJointBinding, len(headMesh.joints))
	for i, hj := range headMesh.joints {
		if hj.name == "origin" {
			bindings[i] = headJointBinding{kind: headTransformBody, bodyIdx: attachIdx}
		} else if bIdx, exists := bodyJoints[hj.name]; exists {
			bindings[i] = headJointBinding{kind: headTransformBody, bodyIdx: bIdx}
		} else {
			pIdx := hj.parent
			if pIdx < 0 || pIdx >= len(headMesh.joints) {
				bindings[i] = headJointBinding{kind: headTransformBody, bodyIdx: attachIdx}
				continue
			}
			pj := headMesh.joints[pIdx]
			invQ := quatConj(pj.orient)
			dPos := geometry.XYZ{
				X: hj.pos.X - pj.pos.X,
				Y: hj.pos.Y - pj.pos.Y,
				Z: hj.pos.Z - pj.pos.Z,
			}
			pLoc := quatRotate(invQ, dPos)
			qLoc := quatMult(invQ, hj.orient)
			bindings[i] = headJointBinding{
				kind:      headTransformChild,
				parentIdx: pIdx,
				pLocal:    pLoc,
				qLocal:    qLoc,
			}
		}
	}

	return &HeadAttachment{
		mesh:     headMesh,
		bindings: bindings,
	}
}

// BuildHeadSkeleton calculates the world pose of all head joints for the given body skeleton pose.
func (h *HeadAttachment) BuildHeadSkeleton(bodySkel []md5Joint) []md5Joint {
	if h == nil || h.mesh == nil || len(bodySkel) == 0 {
		return nil
	}
	headSkel := make([]md5Joint, len(h.mesh.joints))
	for i, hj := range h.mesh.joints {
		b := h.bindings[i]
		if b.kind == headTransformBody {
			if b.bodyIdx >= 0 && b.bodyIdx < len(bodySkel) {
				bj := bodySkel[b.bodyIdx]
				headSkel[i] = md5Joint{
					name:   hj.name,
					parent: hj.parent,
					pos:    bj.pos,
					orient: bj.orient,
				}
			}
		} else {
			if b.parentIdx >= 0 && b.parentIdx < len(headSkel) {
				pEval := headSkel[b.parentIdx]
				rPos := quatRotate(pEval.orient, b.pLocal)
				cPos := geometry.XYZ{
					X: pEval.pos.X + rPos.X,
					Y: pEval.pos.Y + rPos.Y,
					Z: pEval.pos.Z + rPos.Z,
				}
				cOrient := quatNormalize(quatMult(pEval.orient, b.qLocal))
				headSkel[i] = md5Joint{
					name:   hj.name,
					parent: hj.parent,
					pos:    cPos,
					orient: cOrient,
				}
			}
		}
	}
	return headSkel
}

// BakeMD5Model evaluates an MD5Mesh (and optional attached headMesh) with optional animations and compiles it into a Config Model3DEntry.
func BakeMD5Model(mesh *MD5Mesh, headMesh *MD5Mesh, headJointName string, anims map[string]*MD5Anim, materials *Materials) (*config.Model3DEntry, error) {
	if mesh == nil || len(mesh.subMeshes) == 0 {
		return nil, fmt.Errorf("empty md5 mesh")
	}

	// Prepare body submesh materials
	subMaterials := make([]*config.Material, len(mesh.subMeshes))
	for sIdx, sub := range mesh.subMeshes {
		var mat *config.Material
		if materials != nil && len(sub.shader) > 0 {
			mat = materials.CreateConfigMaterial(sub.shader)
		}
		if mat == nil {
			mat = config.NewConfigMaterial("md5.mesh", []string{sub.shader}, config.MaterialKindSolid)
		}
		subMaterials[sIdx] = mat
	}

	// Prepare head submesh materials and attachment if present
	var headAttachment *HeadAttachment
	var headMaterials []*config.Material
	if headMesh != nil && len(headMesh.subMeshes) > 0 {
		headAttachment = NewHeadAttachment(mesh, headMesh, headJointName)
		headMaterials = make([]*config.Material, len(headMesh.subMeshes))
		for sIdx, sub := range headMesh.subMeshes {
			var mat *config.Material
			if materials != nil && len(sub.shader) > 0 {
				mat = materials.CreateConfigMaterial(sub.shader)
			}
			if mat == nil {
				mat = config.NewConfigMaterial("md5.mesh", []string{sub.shader}, config.MaterialKindSolid)
			}
			headMaterials[sIdx] = mat
		}
	}

	var allFrames []config.Model3DEntryFrame
	var actionNames []string
	var actionIntervals [][2]int

	bakeFrameWithSkel := func(skel []md5Joint) config.Model3DEntryFrame {
		var frameTriangles []config.Model3DEntryTriangle

		// Bake body submeshes
		for sIdx, sub := range mesh.subMeshes {
			mat := subMaterials[sIdx]
			computedVerts := make([]geometry.XYZ, len(sub.verts))
			for i, vert := range sub.verts {
				var p geometry.XYZ
				for w := 0; w < vert.weightSpan; w++ {
					weight := sub.weights[vert.weightIdx+w]
					joint := skel[weight.joint]

					rotated := quatRotate(joint.orient, weight.pos)
					p.X += (joint.pos.X + rotated.X) * weight.bias
					p.Y += (joint.pos.Y + rotated.Y) * weight.bias
					p.Z += (joint.pos.Z + rotated.Z) * weight.bias
				}
				computedVerts[i] = p
			}

			for _, triIdx := range sub.tris {
				if triIdx[0] >= len(computedVerts) || triIdx[1] >= len(computedVerts) || triIdx[2] >= len(computedVerts) {
					continue
				}

				tri := config.NewModel3DEntryTriangle(mat)
				for k := 0; k < 3; k++ {
					vi := triIdx[k]
					v := sub.verts[vi]
					tri.Vertices[k] = config.Model3DEntryVertex{
						Pos: computedVerts[vi],
						U:   v.u,
						V:   v.v,
					}
				}
				frameTriangles = append(frameTriangles, tri)
			}
		}

		// Bake attached head submeshes
		if headAttachment != nil && headMesh != nil {
			headSkel := headAttachment.BuildHeadSkeleton(skel)
			for sIdx, sub := range headMesh.subMeshes {
				mat := headMaterials[sIdx]
				computedVerts := make([]geometry.XYZ, len(sub.verts))
				for i, vert := range sub.verts {
					var p geometry.XYZ
					for w := 0; w < vert.weightSpan; w++ {
						weight := sub.weights[vert.weightIdx+w]
						joint := headSkel[weight.joint]

						rotated := quatRotate(joint.orient, weight.pos)
						p.X += (joint.pos.X + rotated.X) * weight.bias
						p.Y += (joint.pos.Y + rotated.Y) * weight.bias
						p.Z += (joint.pos.Z + rotated.Z) * weight.bias
					}
					computedVerts[i] = p
				}

				for _, triIdx := range sub.tris {
					if triIdx[0] >= len(computedVerts) || triIdx[1] >= len(computedVerts) || triIdx[2] >= len(computedVerts) {
						continue
					}

					tri := config.NewModel3DEntryTriangle(mat)
					for k := 0; k < 3; k++ {
						vi := triIdx[k]
						v := sub.verts[vi]
						tri.Vertices[k] = config.Model3DEntryVertex{
							Pos: computedVerts[vi],
							U:   v.u,
							V:   v.v,
						}
					}
					frameTriangles = append(frameTriangles, tri)
				}
			}
		}

		return config.NewModel3DEntryFrame(frameTriangles)
	}

	// Process actions in standard order: idle, walk, run
	actionKeys := []string{"idle", "walk", "run"}
	for _, aKey := range actionKeys {
		anim, hasAnim := anims[aKey]
		if !hasAnim || anim == nil || len(anim.frames) == 0 {
			continue
		}

		startFrame := len(allFrames)
		for f := 0; f < len(anim.frames); f++ {
			skel := anim.BuildFrameSkeleton(f)
			frame := bakeFrameWithSkel(skel)
			allFrames = append(allFrames, frame)
		}
		endFrame := len(allFrames) - 1

		actionNames = append(actionNames, aKey)
		actionIntervals = append(actionIntervals, [2]int{startFrame, endFrame})
	}

	// Fallback to bind pose if no animations provided
	if len(allFrames) == 0 {
		frame := bakeFrameWithSkel(mesh.joints)
		allFrames = append(allFrames, frame)
		actionNames = []string{"idle"}
		actionIntervals = [][2]int{{0, 0}}
	}

	cModel := config.NewModel3DEntry(len(allFrames), actionNames)
	for i, f := range allFrames {
		cModel.Frames[i] = f
	}
	cModel.ActionDefinitions = actionNames
	cModel.ActionIntervals = actionIntervals

	// Center the model in X, Y, and Z so that its center is at (0, 0, 0).
	// In Godoom, rendering places the model at (center.x, center.z, -center.y).
	// Centering ensures that the feet touch the floor at pos.Z and the head reaches pos.Z + height.
	CenterModel(cModel)

	return cModel, nil
}

// CenterModel centers the 3D model's bounding box around (0, 0, 0) across all frames.
func CenterModel(m *config.Model3DEntry) {
	if m == nil || len(m.Frames) == 0 {
		return
	}

	minX, minY, minZ := math.Inf(1), math.Inf(1), math.Inf(1)
	maxX, maxY, maxZ := math.Inf(-1), math.Inf(-1), math.Inf(-1)

	for _, frame := range m.Frames {
		for _, tri := range frame.Triangles {
			for _, vertex := range tri.Vertices {
				p := vertex.Pos
				if p.X < minX {
					minX = p.X
				}
				if p.X > maxX {
					maxX = p.X
				}
				if p.Y < minY {
					minY = p.Y
				}
				if p.Y > maxY {
					maxY = p.Y
				}
				if p.Z < minZ {
					minZ = p.Z
				}
				if p.Z > maxZ {
					maxZ = p.Z
				}
			}
		}
	}

	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2
	centerZ := (minZ + maxZ) / 2

	for frameIdx := range m.Frames {
		for triIdx := range m.Frames[frameIdx].Triangles {
			for vertexIdx := range m.Frames[frameIdx].Triangles[triIdx].Vertices {
				p := &m.Frames[frameIdx].Triangles[triIdx].Vertices[vertexIdx].Pos
				p.X -= centerX
				p.Y -= centerY
				p.Z -= centerZ
			}
		}
	}
}
