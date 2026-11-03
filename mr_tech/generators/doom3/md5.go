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
	shader   string
	vertices []md5Vert
	tris     [][3]int
	weights  []md5Weight
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

// MD5Mesh represents a parsed MD5 mesh structure containing joints and sub-mesh data.
type MD5Mesh struct {
	joints    []md5Joint
	subMeshes []md5SubMesh
}

// NewMD5Mesh creates and returns a new instance of MD5Mesh with uninitialized joints and sub-mesh data.
func NewMD5Mesh() *MD5Mesh {
	return &MD5Mesh{}
}

// Parse reads an MD5Mesh file from the provided reader and populates the mesh with parsed joints and sub-mesh data.
func (mesh *MD5Mesh) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
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
				if len(currentSubMesh.vertices) > 0 && len(currentSubMesh.tris) > 0 {
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

						currentSubMesh.vertices = append(currentSubMesh.vertices, md5Vert{
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

	if inMesh && len(currentSubMesh.vertices) > 0 && len(currentSubMesh.tris) > 0 {
		mesh.subMeshes = append(mesh.subMeshes, currentSubMesh)
	}

	if len(mesh.subMeshes) == 0 {
		return fmt.Errorf("no valid sub-meshes parsed from md5mesh")
	}

	return nil
}

// MD5Anim holds the parsed animation data of an .md5anim file.
type MD5Anim struct {
	numFrames int
	frameRate int
	hierarchy []md5HierarchyJoint
	baseFrame []md5BaseFrameJoint
	frames    [][]float64
}

// NewMD5Anim initializes and returns a new MD5Anim object for parsing and managing .md5anim animation data.
func NewMD5Anim() *MD5Anim {
	return &MD5Anim{}
}

// Parse parses an .md5anim file.
func (anim *MD5Anim) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
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

					anim.baseFrame = append(anim.baseFrame, md5BaseFrameJoint{
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

	return nil
}

// BuildFrameSkeleton constructs the animated skeleton joints in model space for a specific frame index.
func (anim *MD5Anim) BuildFrameSkeleton(frameIdx int) []md5Joint {
	skel := make([]md5Joint, len(anim.hierarchy))
	if frameIdx < 0 || frameIdx >= len(anim.frames) {
		return skel
	}
	frameData := anim.frames[frameIdx]

	for i, h := range anim.hierarchy {
		base := anim.baseFrame[i]
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
			rPos := QuatRotate(parentJoint.orient, pos)
			skel[i] = md5Joint{
				name:   h.name,
				parent: h.parent,
				pos: geometry.XYZ{
					X: parentJoint.pos.X + rPos.X,
					Y: parentJoint.pos.Y + rPos.Y,
					Z: parentJoint.pos.Z + rPos.Z,
				},
				orient: QuatNormalize(QuatMultiplies(parentJoint.orient, orient)),
			}
		}
	}

	return skel
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
			computedVerts := make([]geometry.XYZ, len(sub.vertices))
			for i, vert := range sub.vertices {
				var p geometry.XYZ
				for w := 0; w < vert.weightSpan; w++ {
					weight := sub.weights[vert.weightIdx+w]
					joint := skel[weight.joint]

					rotated := QuatRotate(joint.orient, weight.pos)
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
					v := sub.vertices[vi]
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
				computedVerts := make([]geometry.XYZ, len(sub.vertices))
				for i, vert := range sub.vertices {
					var p geometry.XYZ
					for w := 0; w < vert.weightSpan; w++ {
						weight := sub.weights[vert.weightIdx+w]
						joint := headSkel[weight.joint]

						rotated := QuatRotate(joint.orient, weight.pos)
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
						v := sub.vertices[vi]
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
			skeleton := anim.BuildFrameSkeleton(f)
			frame := bakeFrameWithSkel(skeleton)
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
