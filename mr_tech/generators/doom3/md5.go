package doom3

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/md5"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

type MD5 struct {
	arc       proc.IPkArchive
	materials *proc.Materials
}

func NewMD5(arc proc.IPkArchive, materials *proc.Materials) *MD5 {
	return &MD5{
		arc:       arc,
		materials: materials,
	}
}

// Bake evaluates an MD5Mesh (and optional attached headMesh) with optional animations and compiles it into a Config Model3DEntry.
func (t *MD5) Bake(mesh *md5.Mesh, headMesh *md5.Mesh, headJointName string, anims map[string]*md5.Anim) (*config.Model3DEntry, error) {
	if mesh == nil || len(mesh.SubMeshes) == 0 {
		return nil, fmt.Errorf("empty md5 mesh")
	}
	// Prepare body submesh materials
	subMaterials := make([]*config.Material, len(mesh.SubMeshes))
	for sIdx, sub := range mesh.SubMeshes {
		var mat *config.Material
		var matDef *proc.MaterialDef
		if len(sub.Shader) > 0 {
			if matDef = t.materials.Resolve(sub.Shader); matDef != nil {
				mat = ResolveConfigMaterial(matDef)
			}
		}
		if mat == nil {
			mat = config.NewConfigMaterial("md5.mesh", []string{sub.Shader}, config.MaterialKindSolid)
		}
		ConfigureThingMaterial(mat, sub.Shader, matDef)
		subMaterials[sIdx] = mat
	}

	// Prepare head submesh materials and attachment if present
	var headAttachment *md5.HeadAttachment
	var headMaterials []*config.Material
	if headMesh != nil && len(headMesh.SubMeshes) > 0 {
		headAttachment = md5.NewHeadAttachment(mesh, headMesh, headJointName)
		headMaterials = make([]*config.Material, len(headMesh.SubMeshes))
		for sIdx, sub := range headMesh.SubMeshes {
			var mat *config.Material
			var matDef *proc.MaterialDef
			if len(sub.Shader) > 0 {
				if matDef = t.materials.Resolve(sub.Shader); matDef != nil {
					mat = ResolveConfigMaterial(matDef)
				}
			}
			if mat == nil {
				mat = config.NewConfigMaterial("md5.mesh", []string{sub.Shader}, config.MaterialKindSolid)
			}
			ConfigureThingMaterial(mat, sub.Shader, matDef)
			headMaterials[sIdx] = mat
		}
	}

	var allFrames []config.Model3DEntryFrame
	var actionNames []string
	var actionIntervals [][2]int

	bakeFrameWithSkel := func(skel []*md5.Joint) config.Model3DEntryFrame {
		var frameTriangles []config.Model3DEntryTriangle
		// Bake body submeshes
		for sIdx, sub := range mesh.SubMeshes {
			mat := subMaterials[sIdx]
			computedVerts := make([]geometry.XYZ, len(sub.Vertices))
			for i, vert := range sub.Vertices {
				var p geometry.XYZ
				for w := 0; w < vert.WeightSpan; w++ {
					weight := sub.Weights[vert.WeightIdx+w]
					joint := skel[weight.JointId]
					rotatedX, rotatedY, rotatedZ := joint.Orient.Rotate(weight.Pos.X, weight.Pos.Y, weight.Pos.Z)
					p.X += (joint.Pos.X + rotatedX) * weight.Bias
					p.Y += (joint.Pos.Y + rotatedY) * weight.Bias
					p.Z += (joint.Pos.Z + rotatedZ) * weight.Bias
				}
				computedVerts[i] = p
			}

			for _, triIdx := range sub.Tris {
				if triIdx[0] >= len(computedVerts) || triIdx[1] >= len(computedVerts) || triIdx[2] >= len(computedVerts) {
					continue
				}
				tri := config.NewModel3DEntryTriangle(mat)
				for k := 0; k < 3; k++ {
					vi := triIdx[k]
					v := sub.Vertices[vi]
					tri.Vertices[k] = config.Model3DEntryVertex{Pos: computedVerts[vi], U: v.U, V: v.V}
				}
				frameTriangles = append(frameTriangles, tri)
			}
		}

		// Bake attached head submeshes
		if headAttachment != nil && headMesh != nil {
			headSkel := headAttachment.BuildHeadSkeleton(skel)
			for sIdx, sub := range headMesh.SubMeshes {
				mat := headMaterials[sIdx]
				computedVerts := make([]geometry.XYZ, len(sub.Vertices))
				for i, vert := range sub.Vertices {
					var p geometry.XYZ
					for w := 0; w < vert.WeightSpan; w++ {
						weight := sub.Weights[vert.WeightIdx+w]
						joint := headSkel[weight.JointId]
						rotatedX, rotatedY, rotatedZ := joint.Orient.Rotate(weight.Pos.X, weight.Pos.Y, weight.Pos.Z)
						p.X += (joint.Pos.X + rotatedX) * weight.Bias
						p.Y += (joint.Pos.Y + rotatedY) * weight.Bias
						p.Z += (joint.Pos.Z + rotatedZ) * weight.Bias
					}
					computedVerts[i] = p
				}

				for _, triIdx := range sub.Tris {
					if triIdx[0] >= len(computedVerts) || triIdx[1] >= len(computedVerts) || triIdx[2] >= len(computedVerts) {
						continue
					}

					tri := config.NewModel3DEntryTriangle(mat)
					for k := 0; k < 3; k++ {
						vi := triIdx[k]
						v := sub.Vertices[vi]
						tri.Vertices[k] = config.Model3DEntryVertex{Pos: computedVerts[vi], U: v.U, V: v.V}
					}
					frameTriangles = append(frameTriangles, tri)
				}
			}
		}

		return config.NewModel3DEntryFrame(frameTriangles)
	}

	// Process actions in standard order: idle, walk, run, attack, pain, death
	actionKeys := []string{"idle", "walk", "run", "attack", "pain", "death"}
	for _, aKey := range actionKeys {
		anim, hasAnim := anims[aKey]
		if !hasAnim || anim == nil || len(anim.Frames) == 0 {
			continue
		}
		if aKey == "death" && len(anim.Frames) <= 1 {
			continue
		}
		startFrame := len(allFrames)
		for f := 0; f < len(anim.Frames); f++ {
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
		frame := bakeFrameWithSkel(mesh.Joints)
		allFrames = append(allFrames, frame)
		actionNames = []string{"idle"}
		actionIntervals = [][2]int{{0, 0}}
	}

	// In Doom 3 (id Tech 4), regular monsters do not have keyframed death.md5anim files
	// because id Tech 4 switches them to Ragdoll physics (.af) upon death.
	// When no multi-frame death animation exists in the PK4, synthesize a collapse death
	// sequence from the pain/idle skeleton pitching onto the floor.
	hasDeath := false
	for _, aName := range actionNames {
		if aName == "death" {
			hasDeath = true
			break
		}
	}
	if !hasDeath && len(anims) > 0 && len(allFrames) > 0 {
		floorZ := math.MaxFloat64
		for _, tri := range allFrames[0].Triangles {
			for k := 0; k < 3; k++ {
				if tri.Vertices[k].Pos.Z < floorZ {
					floorZ = tri.Vertices[k].Pos.Z
				}
			}
		}
		if floorZ == math.MaxFloat64 {
			floorZ = 0
		}

		baseAnim := anims["pain"]
		if baseAnim == nil || len(baseAnim.Frames) == 0 {
			baseAnim = anims["idle"]
		}

		const deathFrames = 16
		startFrame := len(allFrames)
		qFall := aabb.NewQuaternionEmpty()
		for f := 0; f < deathFrames; f++ {
			var skel []*md5.Joint
			if baseAnim != nil && len(baseAnim.Frames) > 0 {
				srcIdx := (f * len(baseAnim.Frames)) / deathFrames
				if srcIdx >= len(baseAnim.Frames) {
					srcIdx = len(baseAnim.Frames) - 1
				}
				skel = baseAnim.BuildFrameSkeleton(srcIdx)
			} else {
				skel = make([]*md5.Joint, len(mesh.Joints))
				for jIdx, j := range mesh.Joints {
					skel[jIdx] = j.Clone()
				}
			}

			tNorm := float64(f) / float64(deathFrames-1)
			theta := -(math.Pi * 0.5) * (tNorm * tNorm)
			half := theta * 0.5
			qFall.Set(0, math.Sin(half), 0, math.Cos(half))

			for _, j := range skel {
				j.Pos.X, j.Pos.Y, j.Pos.Z = qFall.Rotate(j.Pos.X, j.Pos.Y, j.Pos.Z)
				qFall.Multiplies(j.Orient, j.Orient)
				j.Orient.Normalize()
			}

			frame := bakeFrameWithSkel(skel)
			minZ := math.MaxFloat64
			for _, tri := range frame.Triangles {
				for k := 0; k < 3; k++ {
					if tri.Vertices[k].Pos.Z < minZ {
						minZ = tri.Vertices[k].Pos.Z
					}
				}
			}
			if minZ < math.MaxFloat64 {
				dz := floorZ - minZ
				for tIdx := range frame.Triangles {
					for k := 0; k < 3; k++ {
						frame.Triangles[tIdx].Vertices[k].Pos.Z += dz
					}
				}
			}
			allFrames = append(allFrames, frame)
		}
		endFrame := len(allFrames) - 1
		actionNames = append(actionNames, "death")
		actionIntervals = append(actionIntervals, [2]int{startFrame, endFrame})
	}

	cModel := config.NewModel3DEntry(len(allFrames), actionNames, []string{"death", "die"})
	for i, f := range allFrames {
		cModel.Frames[i] = f
	}
	cModel.ActionDefinitions = actionNames
	cModel.ActionIntervals = actionIntervals

	// Center the model in X, Y, and Z so that its center is at (0, 0, 0).
	// In Godoom, presentation aligns all objects to their center.
	cModel.Center([]string{"stand", "idle", "legs_idle", "hover"})

	return cModel, nil
}
