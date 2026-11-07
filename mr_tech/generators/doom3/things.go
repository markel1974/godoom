package doom3

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/md5"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/textures"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

type cachedRagdoll struct {
	md3    *config.Model3D
	radius float64
}

// Things manages the loading, caching, and instantiation of 3D entities (monsters, items, weapons, ammo) in Doom 3.
type Things struct {
	arc          proc.IPkArchive
	materials    *proc.Materials
	defs         *proc.Defs
	texManager   *textures.Textures
	imgLoader    *proc.ImageLoader
	modelCache   map[string]*config.Model3DEntry
	ragdollCache map[string]*cachedRagdoll
}

// NewThings initializes and returns a new Things manager.
func NewThings(arc proc.IPkArchive, materials *proc.Materials, defs *proc.Defs, texManager *textures.Textures, imgLoader *proc.ImageLoader) *Things {
	return &Things{
		arc:          arc,
		materials:    materials,
		defs:         defs,
		texManager:   texManager,
		imgLoader:    imgLoader,
		modelCache:   make(map[string]*config.Model3DEntry),
		ragdollCache: make(map[string]*cachedRagdoll),
	}
}

/*
func (t *Things) resolveMeshSpec(res *proc.ResolvedEntity) {
	// If it's an MD5 mesh, enrich animations via ReadDirFilter on the mesh directory in the PK4
	if strings.HasSuffix(res.ModelPath, ".md5mesh") {
		res.Anims = t.defs.DiscoverModelAnims(res.ModelPath, res.Anims)
		res.IdleAnim = t.defs.PickAnim(
			res.Anims,
			[]string{"idle", "idle1", "idle2", "stand", "alert_idle", "crawl_idle", "on4_idle", "idle_stand1", "zidle1", "zctidle", "chaingun_idle", "machinegun_stand", "shield_stand", "playdead"},
			[]string{"idle", "stand"},
		)
		res.WalkAnim = t.defs.PickAnim(
			res.Anims,
			[]string{"walk", "walk1", "walk2", "walk3", "walk4", "walk7", "walk1_r", "zwalk1", "fastwalk", "crawl1", "travel", "travel3", "chaingun_walk", "machinegun_walk", "shield_walk", "run"},
			[]string{"walk", "crawl", "travel"},
		)
		res.RunAnim = t.defs.PickAnim(
			res.Anims,
			[]string{"run", "run1", "run2", "run3", "run_r", "faster_walk", "fastwalk", "zctrun5", "crawl2", "chaingun_run", "machinegun_run", "shield_run", "walk", "walk1", "walk2"},
			[]string{"run", "fastwalk", "faster_walk"},
		)
		if len(res.RunAnim) == 0 {
			res.RunAnim = res.WalkAnim
		}
		if len(res.WalkAnim) == 0 {
			res.WalkAnim = res.RunAnim
		}
		res.AttackAnim = t.defs.PickAnim(
			res.Anims,
			[]string{
				"range_attack1", "range_attack2", "range1", "attack", "attack1", "attack2", "attack3",
				"melee_attack1", "melee_attack2", "melee1", "melee_left", "melee1_walk", "on4_melee1",
				"stand_fire", "fire1", "chaingun_stand_fire", "machinegun_stand_fire", "shield_stand_fire",
				"oneshot", "attack_bfg", "range_tentacle", "leftclawattack", "slash1", "swing1", "mocapswing1", "pipeattack1",
			},
			[]string{"attack", "fire", "melee", "range", "swing", "slash"},
		)
		res.PainAnim = t.defs.PickAnim(
			res.Anims,
			[]string{
				"pain", "pain_chest", "pain_chest1", "pain_chest2", "chest_pain", "chest_pain1", "chestpain", "chestpain1",
				"chaingun_pain_chest", "machinegun_chest_pain", "shield_chest_pain", "crawl_pain_chest", "pain_chest_on4",
				"pain1", "pain3", "pain_big1", "pain_head", "pain_head1", "pain_head3", "head_pain", "headpain", "headpain1",
				"bellypain", "leftarmpain", "walk1_pain",
			},
			[]string{"pain"},
		)
		res.DeathAnim = t.defs.PickAnim(
			res.Anims,
			[]string{"death", "die", "playdead", "death1_pose", "death_pose"},
			[]string{"death", "die", "playdead"},
		)
	}
}
*/

// loadAnimatedMonsterMD3 loads and caches a multi-part articulated MD5 model decomposed into ragdoll limbs
// connected via hierarchical Model3DLink and joint attachment tags.
func (t *Things) loadAnimatedMonsterMD3(spec *ThingSpec) (*config.Model3D, float64, error) {
	cleanPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(spec.ModelPath, "\\", "/")))
	cacheKey := cleanPath
	if len(spec.HeadMesh) > 0 {
		cacheKey += "#" + strings.TrimSpace(strings.ToLower(strings.ReplaceAll(spec.HeadMesh, "\\", "/")))
	}
	if len(spec.Ragdoll) > 0 {
		cacheKey += "@" + strings.TrimSpace(strings.ToLower(spec.Ragdoll))
	}
	if cached, ok := t.ragdollCache[cacheKey]; ok {
		return cached.md3, cached.radius, nil
	}

	mf, err := t.arc.Open(cleanPath)
	if err != nil {
		return nil, 0, fmt.Errorf("opening md5mesh %s: %w", cleanPath, err)
	}

	mesh := md5.NewMesh()
	if err := mesh.Parse(mf); err != nil {
		return nil, 0, fmt.Errorf("parsing md5mesh %s: %w", cleanPath, err)
	}

	var headMesh *md5.Mesh
	if len(spec.HeadMesh) > 0 {
		cHeadPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(spec.HeadMesh, "\\", "/")))
		if hf, err := t.arc.Open(cHeadPath); err == nil {
			parsedHead := md5.NewMesh()
			if err := parsedHead.Parse(hf); err == nil {
				headMesh = parsedHead
			}
		}
	}

	anims := make(map[string]*md5.Anim)
	loadAnim := func(actionKey, animPath string) {
		if len(animPath) == 0 {
			return
		}
		cAnimPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(animPath, "\\", "/")))
		if af, err := t.arc.Open(cAnimPath); err == nil {
			anim := md5.NewAnim()
			if err := anim.Parse(af); err == nil && anim.NumJoints() == len(mesh.Joints) {
				anims[actionKey] = anim
			}
		}
	}

	loadAnim("idle", spec.IdleAnim)
	loadAnim("walk", spec.WalkAnim)
	loadAnim("run", spec.RunAnim)
	loadAnim("attack", spec.AttackAnim)
	loadAnim("pain", spec.PainAnim)
	loadAnim("death", spec.DeathAnim)

	parts, links, radius, err := t.bakeMD5Ragdoll(mesh, headMesh, spec.HeadJoint, spec.AFBodies, anims)
	if err != nil || len(parts) == 0 {
		return nil, 0, fmt.Errorf("baking md5 ragdoll %s: %w", cleanPath, err)
	}

	// Preload all texture maps (diffuse, normal, specular) across all ragdoll parts
	if t.imgLoader != nil {
		for _, part := range parts {
			if len(part.Frames) > 0 {
				for _, tri := range part.Frames[0].Triangles {
					if tri.Material != nil {
						for _, texName := range tri.Material.Textures {
							t.imgLoader.LoadMaterialMaps(texName)
						}
					}
				}
			}
		}
	}

	md3 := config.NewMD3(parts, links)
	md3.ActionDefinitions = parts[0].ActionDefinitions
	md3.ActionIntervals = parts[0].ActionIntervals
	numActions := len(md3.ActionDefinitions)
	md3.ActionMaps = make([][]int, len(parts))
	for i := range md3.ActionMaps {
		md3.ActionMaps[i] = make([]int, numActions)
		for a := 0; a < numActions; a++ {
			md3.ActionMaps[i][a] = a
		}
	}

	t.ragdollCache[cacheKey] = &cachedRagdoll{
		md3:    md3,
		radius: radius,
	}
	return md3, radius, nil
}

// loadModel loads and caches a static 3D model (.md5mesh or .lwo) from the PK4 archive.
func (t *Things) loadModel(modelPath string) (*config.Model3DEntry, error) {
	cleanPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(modelPath, "\\", "/")))
	if cached, ok := t.modelCache[cleanPath]; ok {
		return cached, nil
	}

	f, err := t.arc.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("opening model %s: %w", cleanPath, err)
	}
	//defer f.Close()

	var cModel *config.Model3DEntry
	if strings.HasSuffix(cleanPath, ".md5mesh") {
		mesh := md5.NewMesh()
		if mErr := mesh.Parse(f); mErr != nil {
			return nil, mErr
		}
		cModel, err = t.bakeMD5Model(mesh, nil, "", nil)
	} else if strings.HasSuffix(cleanPath, ".lwo") {
		cModel, err = t.bakeLWO(f)
	} else {
		return nil, fmt.Errorf("unsupported model format for %s", cleanPath)
	}

	if err != nil {
		return nil, fmt.Errorf("parsing model %s: %w", cleanPath, err)
	}

	// Preload all texture maps (diffuse, normal, specular) for all materials referenced by the model
	if t.imgLoader != nil && len(cModel.Frames) > 0 {
		for _, tri := range cModel.Frames[0].Triangles {
			if tri.Material != nil {
				for _, texName := range tri.Material.Textures {
					t.imgLoader.LoadMaterialMaps(texName)
				}
			}
		}
	}

	t.modelCache[cleanPath] = cModel
	return cModel, nil
}

// Create creates a new Thing entity based on the parsed MapEntity properties.
func (t *Things) Create(ent *proc.MapEntity) (*config.Thing, error) {
	classname, ok := ent.GetProperty("classname")
	if !ok {
		return nil, fmt.Errorf("entity has no classname")
	}

	// In Doom 3, scripted monsters with teleport, hide, or trigger keys are dormant spawn events
	// and are not physically active in the world at map start.
	//if ent.IsDormant() {
	//	return nil, fmt.Errorf("dormant entity: %s", classname)
	//}

	origin, err := ent.GetOrigin()
	if err != nil {
		return nil, fmt.Errorf("entity %s origin error: %w", classname, err)
	}

	angle := ent.GetAngle()

	spec := NewThingSpec()
	hasSpec := spec.ResolveEntity(classname, t.defs)
	if !hasSpec {
		return nil, fmt.Errorf("unhandled entity classname: %s", classname)
	}

	// Entity might override model explicitly
	modelPath := spec.ModelPath
	if explicitModel, hasModel := ent.GetProperty("model"); hasModel {
		expLower := strings.ToLower(explicitModel)
		if strings.HasSuffix(expLower, ".md5mesh") || strings.HasSuffix(expLower, ".lwo") {
			modelPath = explicitModel
			spec.ModelPath = explicitModel
		} else if t.defs != nil {
			if mMesh, _, mOk := t.defs.ResolveModel(explicitModel); mOk && len(mMesh) > 0 {
				modelPath = mMesh
				spec.ModelPath = mMesh
			}
		}
	}
	if len(modelPath) == 0 {
		return nil, fmt.Errorf("no model specified for entity %s", classname)
	}

	var cModel *config.Model3DEntry
	var md3 *config.Model3D
	var radius float64

	if spec.Kind == config.ThingEnemyDef && strings.HasSuffix(strings.ToLower(modelPath), ".md5mesh") {
		md3, radius, err = t.loadAnimatedMonsterMD3(spec)
	} else {
		cModel, err = t.loadModel(modelPath)
		if cModel != nil {
			radius, _ = cModel.GetSize()
		}
	}
	if err != nil {
		return nil, fmt.Errorf("loading model for %s (%s): %w", classname, modelPath, err)
	}

	// Presentation centers all objects around GetCenter() = bottomLeft + (radius, radius, height/2).
	// In Doom 3, origin is at (centerX, centerY, floorZ).
	// To align the bottom-left coordinate so center lands at (origin.X, origin.Y, origin.Z + height/2):
	pos := geometry.XYZ{X: origin.X - radius, Y: origin.Y - radius, Z: origin.Z}

	thingCfg := config.NewConfigThing(classname, pos, angle, spec.Kind, spec.Mass, 0, 0, spec.Speed)
	thingCfg.GForce = GForce

	if spec.Kind == config.ThingEnemyDef {
		if md3 == nil && cModel != nil {
			parts := []*config.Model3DEntry{cModel}
			links := []config.Model3DLink{{Parent: -1, Tag: ""}}
			md3 = config.NewMD3(parts, links)
			md3.ActionDefinitions = cModel.ActionDefinitions
			md3.ActionIntervals = cModel.ActionIntervals
			numActions := len(cModel.ActionDefinitions)
			md3.ActionMaps = make([][]int, len(parts))
			for i := range md3.ActionMaps {
				md3.ActionMaps[i] = make([]int, numActions)
				for a := 0; a < numActions; a++ {
					md3.ActionMaps[i][a] = a
				}
			}
		}

		actions := config.NewEnemyAnimations()
		if md3 != nil && len(md3.ActionDefinitions) > 0 {
			ConvertAnimation(actions, md3.ActionDefinitions, md3.ActionIntervals)
		}
		if len(actions.Content[config.EnemyStateIdle].Indices) == 0 {
			actions.Add(config.EnemyStateIdle, 0, 1)
			actions.Add(config.EnemyStateWalk, 0, 1)
			actions.Add(config.EnemyStateRun, 0, 1)
		}
		thingCfg.Model3D = md3
		thingCfg.JumpForce = JumpForce
		thingCfg.Impulse = 3.0
		thingCfg.WakeupDistance = 300.0
		_ = common.NewEnemy(thingCfg, actions)
	} else if spec.Kind == config.ThingItemDef {
		thingCfg.Model3DEntry = cModel
		_ = common.NewItem(thingCfg)
	} else {
		thingCfg.Model3DEntry = cModel
		_ = common.NewItem(thingCfg)
	}

	return thingCfg, nil
}

type ragdollBodyPart struct {
	name        string
	anchorJoint int
}

// partitionSkeletonIntoRagdollBodies assigns every joint in mesh.Joints to an articulated ragdoll body
// using Doom 3's .af (Articulated Figure) body declarations or standard anatomical limb roots.
func partitionSkeletonIntoRagdollBodies(joints []*md5.Joint, afBodies []*proc.AFBody) ([]ragdollBodyPart, []int) {
	numJoints := len(joints)
	if numJoints == 0 {
		return nil, nil
	}

	jointByName := make(map[string]int, numJoints)
	children := make([][]int, numJoints)
	for idx, j := range joints {
		jointByName[strings.ToLower(j.Name())] = idx
		p := j.Parent()
		if p >= 0 && p < numJoints && p != idx {
			children[p] = append(children[p], idx)
		}
	}

	var collectSubtree func(root int, out map[int]bool)
	collectSubtree = func(root int, out map[int]bool) {
		out[root] = true
		for _, ch := range children[root] {
			if !out[ch] {
				collectSubtree(ch, out)
			}
		}
	}

	jointToBody := make([]int, numJoints)
	for i := range jointToBody {
		jointToBody[i] = -1
	}

	var parts []ragdollBodyPart

	if len(afBodies) > 0 {
		for _, afb := range afBodies {
			anchorIdx := -1
			if len(afb.Joint) > 0 {
				if idx, ok := jointByName[strings.ToLower(afb.Joint)]; ok {
					anchorIdx = idx
				}
			}
			included := make(map[int]bool)
			for _, token := range strings.Fields(afb.ContainedJoints) {
				tok := strings.ToLower(strings.TrimSpace(token))
				if len(tok) == 0 {
					continue
				}
				switch {
				case strings.HasPrefix(tok, "-*"):
					jName := strings.TrimPrefix(tok, "-*")
					if jIdx, ok := jointByName[jName]; ok {
						excl := make(map[int]bool)
						collectSubtree(jIdx, excl)
						for k := range excl {
							delete(included, k)
						}
					}
				case strings.HasPrefix(tok, "*"):
					jName := strings.TrimPrefix(tok, "*")
					if jIdx, ok := jointByName[jName]; ok {
						if anchorIdx < 0 {
							anchorIdx = jIdx
						}
						collectSubtree(jIdx, included)
					}
				case strings.HasPrefix(tok, "-"):
					jName := strings.TrimPrefix(tok, "-")
					if jIdx, ok := jointByName[jName]; ok {
						delete(included, jIdx)
					}
				default:
					if jIdx, ok := jointByName[tok]; ok {
						if anchorIdx < 0 {
							anchorIdx = jIdx
						}
						included[jIdx] = true
					}
				}
			}
			if anchorIdx < 0 && len(included) > 0 {
				minIdx := numJoints
				for jIdx := range included {
					if jIdx < minIdx {
						minIdx = jIdx
					}
				}
				if minIdx < numJoints {
					anchorIdx = minIdx
				}
			}
			if anchorIdx < 0 {
				continue
			}
			bodyIdx := len(parts)
			parts = append(parts, ragdollBodyPart{
				name:        afb.Name,
				anchorJoint: anchorIdx,
			})
			for jIdx := range included {
				jointToBody[jIdx] = bodyIdx
			}
		}
	}

	// Fallback if no .af matched the skeleton joints: partition by standard anatomical limb roots
	if len(parts) == 0 {
		parts = append(parts, ragdollBodyPart{name: "origin", anchorJoint: 0})
		jointToBody[0] = 0
		limbKeywords := []string{
			"waist", "pelvis", "hips", "belly", "chest", "torso", "spine", "shoulders",
			"neck", "head", "jaw", "luparm", "lloarm", "lhand", "ruparm", "rloarm", "rhand",
			"lupleg", "lloleg", "lfoot", "rupleg", "rloleg", "rfoot", "tail",
		}
		for jIdx := 1; jIdx < numJoints; jIdx++ {
			jLow := strings.ToLower(joints[jIdx].Name())
			matched := false
			for _, kw := range limbKeywords {
				if jLow == kw {
					matched = true
					break
				}
			}
			if matched {
				bIdx := len(parts)
				parts = append(parts, ragdollBodyPart{name: jLow, anchorJoint: jIdx})
				jointToBody[jIdx] = bIdx
			}
		}
	}

	// Propagate body assignment from parent joints for any unassigned joints
	for jIdx := 0; jIdx < numJoints; jIdx++ {
		if jointToBody[jIdx] >= 0 {
			continue
		}
		curr := joints[jIdx].Parent()
		assigned := -1
		for curr >= 0 && curr < numJoints {
			if jointToBody[curr] >= 0 {
				assigned = jointToBody[curr]
				break
			}
			next := joints[curr].Parent()
			if next == curr {
				break
			}
			curr = next
		}
		if assigned < 0 {
			assigned = 0
		}
		jointToBody[jIdx] = assigned
	}

	return parts, jointToBody
}

type ragdollTriSpec struct {
	partIdx    int
	mat        *config.Material
	isHead     bool
	subIdx     int
	triIndices [3]int
}

type rawSkelFrame struct {
	skel    []*md5.Joint
	bodyPos [][]geometry.XYZ
	headPos [][]geometry.XYZ
}

// bakeMD5Ragdoll decomposes an MD5Mesh (plus optional attached headMesh) into multiple articulated
// Model3DEntry parts linked via Model3DLink and per-frame Model3DTag joint transforms.
func (t *Things) bakeMD5Ragdoll(mesh *md5.Mesh, headMesh *md5.Mesh, headJointName string, afBodies []*proc.AFBody, anims map[string]*md5.Anim) ([]*config.Model3DEntry, []config.Model3DLink, float64, error) {
	if mesh == nil || len(mesh.SubMeshes) == 0 || len(mesh.Joints) == 0 {
		return nil, nil, 0, fmt.Errorf("empty md5 mesh")
	}

	// 1. Prepare materials for body submeshes
	subMaterials := make([]*config.Material, len(mesh.SubMeshes))
	for sIdx, sub := range mesh.SubMeshes {
		var mat *config.Material
		if len(sub.Shader) > 0 {
			if matDef := t.materials.Resolve(sub.Shader); matDef != nil {
				mat = ResolveConfigMaterial(matDef)
			}
		}
		if mat == nil {
			mat = config.NewConfigMaterial("md5.mesh", []string{sub.Shader}, config.MaterialKindSolid)
		}
		subMaterials[sIdx] = mat
	}

	// 2. Prepare head attachment and materials if present
	var headAttachment *md5.HeadAttachment
	var headMaterials []*config.Material
	if headMesh != nil && len(headMesh.SubMeshes) > 0 {
		headAttachment = md5.NewHeadAttachment(mesh, headMesh, headJointName)
		headMaterials = make([]*config.Material, len(headMesh.SubMeshes))
		for sIdx, sub := range headMesh.SubMeshes {
			var mat *config.Material
			if len(sub.Shader) > 0 {
				if matDef := t.materials.Resolve(sub.Shader); matDef != nil {
					mat = ResolveConfigMaterial(matDef)
				}
			}
			if mat == nil {
				mat = config.NewConfigMaterial("md5.mesh", []string{sub.Shader}, config.MaterialKindSolid)
			}
			headMaterials[sIdx] = mat
		}
	}

	// 3. Partition skeleton into ragdoll bodies
	rawParts, jointToBody := partitionSkeletonIntoRagdollBodies(mesh.Joints, afBodies)
	if len(rawParts) == 0 {
		return nil, nil, 0, fmt.Errorf("failed to partition md5 skeleton")
	}

	// 4. Assign each triangle in mesh.SubMeshes to a ragdoll body based on dominant joint weight
	var allTriSpecs []ragdollTriSpec
	partTriCount := make([]int, len(rawParts))

	for sIdx, sub := range mesh.SubMeshes {
		mat := subMaterials[sIdx]
		for _, triIdx := range sub.Tris {
			if triIdx[0] >= len(sub.Vertices) || triIdx[1] >= len(sub.Vertices) || triIdx[2] >= len(sub.Vertices) {
				continue
			}
			bodyWeights := make(map[int]float64)
			for k := 0; k < 3; k++ {
				vert := sub.Vertices[triIdx[k]]
				for w := 0; w < vert.WeightSpan; w++ {
					weight := sub.Weights[vert.WeightIdx+w]
					if weight.JointId >= 0 && weight.JointId < len(jointToBody) {
						bIdx := jointToBody[weight.JointId]
						if bIdx >= 0 {
							bodyWeights[bIdx] += weight.Bias
						}
					}
				}
			}
			bestBody := 0
			bestWeight := -1.0
			for bIdx, wSum := range bodyWeights {
				if wSum > bestWeight || (wSum == bestWeight && bIdx < bestBody) {
					bestWeight = wSum
					bestBody = bIdx
				}
			}
			partTriCount[bestBody]++
			allTriSpecs = append(allTriSpecs, ragdollTriSpec{
				partIdx:    bestBody,
				mat:        mat,
				isHead:     false,
				subIdx:     sIdx,
				triIndices: triIdx,
			})
		}
	}

	// Assign headMesh triangles to the head ragdoll body (or create one anchored at the head joint)
	if headAttachment != nil && headMesh != nil {
		headBodyIdx := -1
		for bIdx, bp := range rawParts {
			if strings.EqualFold(bp.name, "head") {
				headBodyIdx = bIdx
				break
			}
		}
		if headBodyIdx < 0 {
			headJointIdx := 0
			for idx, j := range mesh.Joints {
				if strings.EqualFold(j.Name(), headJointName) || strings.EqualFold(j.Name(), "Head") {
					headJointIdx = idx
					break
				}
			}
			headBodyIdx = len(rawParts)
			rawParts = append(rawParts, ragdollBodyPart{
				name:        "head",
				anchorJoint: headJointIdx,
			})
			partTriCount = append(partTriCount, 0)
		}
		for sIdx, sub := range headMesh.SubMeshes {
			mat := headMaterials[sIdx]
			for _, triIdx := range sub.Tris {
				if triIdx[0] >= len(sub.Vertices) || triIdx[1] >= len(sub.Vertices) || triIdx[2] >= len(sub.Vertices) {
					continue
				}
				partTriCount[headBodyIdx]++
				allTriSpecs = append(allTriSpecs, ragdollTriSpec{
					partIdx:    headBodyIdx,
					mat:        mat,
					isHead:     true,
					subIdx:     sIdx,
					triIndices: triIdx,
				})
			}
		}
	}

	if len(allTriSpecs) == 0 {
		return nil, nil, 0, fmt.Errorf("no triangles in md5 ragdoll")
	}

	// 5. Filter to active bodies with >= 1 triangle and sort by ascending anchorJoint
	type activeBodyInfo struct {
		oldIdx      int
		name        string
		anchorJoint int
	}
	var activeBodies []activeBodyInfo
	for bIdx, bp := range rawParts {
		if partTriCount[bIdx] > 0 {
			activeBodies = append(activeBodies, activeBodyInfo{
				oldIdx:      bIdx,
				name:        bp.name,
				anchorJoint: bp.anchorJoint,
			})
		}
	}
	sort.SliceStable(activeBodies, func(i, j int) bool {
		return activeBodies[i].anchorJoint < activeBodies[j].anchorJoint
	})
	// Root part 0 is always anchored at model root (joint 0)
	activeBodies[0].anchorJoint = 0

	oldToNewPart := make(map[int]int, len(activeBodies))
	partTris := make([][]ragdollTriSpec, len(activeBodies))
	for newIdx, ab := range activeBodies {
		oldToNewPart[ab.oldIdx] = newIdx
	}
	for _, ts := range allTriSpecs {
		newIdx := oldToNewPart[ts.partIdx]
		ts.partIdx = newIdx
		partTris[newIdx] = append(partTris[newIdx], ts)
	}

	// 6. Build hierarchical Model3DLink parent connections using skeleton joint ancestry
	links := make([]config.Model3DLink, len(activeBodies))
	links[0] = config.Model3DLink{Parent: -1, Tag: ""}
	for i := 1; i < len(activeBodies); i++ {
		parentPart := 0
		currJ := activeBodies[i].anchorJoint
		// Walk up the MD5 joint tree to find the closest ancestor part p < i
		for currJ >= 0 && currJ < len(mesh.Joints) {
			found := -1
			for p := i - 1; p >= 0; p-- {
				if activeBodies[p].anchorJoint == currJ {
					found = p
					break
				}
			}
			if found >= 0 {
				parentPart = found
				break
			}
			nextJ := mesh.Joints[currJ].Parent()
			if nextJ == currJ {
				break
			}
			currJ = nextJ
		}
		tagName := fmt.Sprintf("tag_%s_%d", strings.ToLower(activeBodies[i].name), i)
		links[i] = config.Model3DLink{
			Parent: parentPart,
			Tag:    tagName,
		}
	}

	// 7. Evaluate all animation frames into world/model-space vertex positions and skeletons
	evaluateSkel := func(skel []*md5.Joint) rawSkelFrame {
		bodyPos := make([][]geometry.XYZ, len(mesh.SubMeshes))
		for sIdx, sub := range mesh.SubMeshes {
			computedVerts := make([]geometry.XYZ, len(sub.Vertices))
			for i, vert := range sub.Vertices {
				var p geometry.XYZ
				for w := 0; w < vert.WeightSpan; w++ {
					weight := sub.Weights[vert.WeightIdx+w]
					joint := skel[weight.JointId]
					rotated := physics.QuatRotate(joint.Orient, weight.Pos)
					p.X += (joint.Pos.X + rotated.X) * weight.Bias
					p.Y += (joint.Pos.Y + rotated.Y) * weight.Bias
					p.Z += (joint.Pos.Z + rotated.Z) * weight.Bias
				}
				computedVerts[i] = p
			}
			bodyPos[sIdx] = computedVerts
		}

		var headPos [][]geometry.XYZ
		if headAttachment != nil && headMesh != nil {
			headSkel := headAttachment.BuildHeadSkeleton(skel)
			headPos = make([][]geometry.XYZ, len(headMesh.SubMeshes))
			for sIdx, sub := range headMesh.SubMeshes {
				computedVerts := make([]geometry.XYZ, len(sub.Vertices))
				for i, vert := range sub.Vertices {
					var p geometry.XYZ
					for w := 0; w < vert.WeightSpan; w++ {
						weight := sub.Weights[vert.WeightIdx+w]
						joint := headSkel[weight.JointId]
						rotated := physics.QuatRotate(joint.Orient, weight.Pos)
						p.X += (joint.Pos.X + rotated.X) * weight.Bias
						p.Y += (joint.Pos.Y + rotated.Y) * weight.Bias
						p.Z += (joint.Pos.Z + rotated.Z) * weight.Bias
					}
					computedVerts[i] = p
				}
				headPos[sIdx] = computedVerts
			}
		}

		return rawSkelFrame{
			skel:    skel,
			bodyPos: bodyPos,
			headPos: headPos,
		}
	}

	var rawFrames []rawSkelFrame
	var actionNames []string
	var actionIntervals [][2]int

	actionKeys := []string{"idle", "walk", "run", "attack", "pain", "death"}
	for _, aKey := range actionKeys {
		anim, hasAnim := anims[aKey]
		if !hasAnim || anim == nil || len(anim.Frames) == 0 {
			continue
		}
		if aKey == "death" && len(anim.Frames) <= 1 {
			continue
		}
		startFrame := len(rawFrames)
		for f := 0; f < len(anim.Frames); f++ {
			skel := anim.BuildFrameSkeleton(f)
			rawFrames = append(rawFrames, evaluateSkel(skel))
		}
		endFrame := len(rawFrames) - 1
		actionNames = append(actionNames, aKey)
		actionIntervals = append(actionIntervals, [2]int{startFrame, endFrame})
	}

	if len(rawFrames) == 0 {
		skel := make([]*md5.Joint, len(mesh.Joints))
		for jIdx, j := range mesh.Joints {
			cp := *j
			skel[jIdx] = &cp
		}
		rawFrames = append(rawFrames, evaluateSkel(skel))
		actionNames = []string{"idle"}
		actionIntervals = [][2]int{{0, 0}}
	}

	// Synthesize ragdoll collapse death sequence when no multi-frame death animation exists
	hasDeath := false
	for _, aName := range actionNames {
		if aName == "death" {
			hasDeath = true
			break
		}
	}
	if !hasDeath && len(anims) > 0 && len(rawFrames) > 0 {
		floorZ := math.MaxFloat64
		for _, subVerts := range rawFrames[0].bodyPos {
			for _, v := range subVerts {
				if v.Z < floorZ {
					floorZ = v.Z
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
		startFrame := len(rawFrames)
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
					cp := *j
					skel[jIdx] = &cp
				}
			}

			tNorm := float64(f) / float64(deathFrames-1)
			theta := -(math.Pi * 0.5) * (tNorm * tNorm)
			half := theta * 0.5
			qFall := [4]float64{0, math.Sin(half), 0, math.Cos(half)}

			for _, j := range skel {
				j.Pos = physics.QuatRotate(qFall, j.Pos)
				j.Orient = physics.QuatNormalize(physics.QuatMultiplies(qFall, j.Orient))
			}

			rf := evaluateSkel(skel)
			minZ := math.MaxFloat64
			for _, subVerts := range rf.bodyPos {
				for _, v := range subVerts {
					if v.Z < minZ {
						minZ = v.Z
					}
				}
			}
			for _, subVerts := range rf.headPos {
				for _, v := range subVerts {
					if v.Z < minZ {
						minZ = v.Z
					}
				}
			}
			if minZ < math.MaxFloat64 {
				dz := floorZ - minZ
				for _, j := range rf.skel {
					j.Pos.Z += dz
				}
				for sIdx := range rf.bodyPos {
					for vIdx := range rf.bodyPos[sIdx] {
						rf.bodyPos[sIdx][vIdx].Z += dz
					}
				}
				for sIdx := range rf.headPos {
					for vIdx := range rf.headPos[sIdx] {
						rf.headPos[sIdx][vIdx].Z += dz
					}
				}
			}
			rawFrames = append(rawFrames, rf)
		}
		endFrame := len(rawFrames) - 1
		actionNames = append(actionNames, "death")
		actionIntervals = append(actionIntervals, [2]int{startFrame, endFrame})
	}

	// 8. Compute global centering offset and bounding radius from the assembled idle frames
	minB := geometry.XYZ{X: math.MaxFloat64, Y: math.MaxFloat64, Z: math.MaxFloat64}
	maxB := geometry.XYZ{X: -math.MaxFloat64, Y: -math.MaxFloat64, Z: -math.MaxFloat64}

	updateBounds := func(fIdx int) {
		rf := rawFrames[fIdx]
		for _, subVerts := range rf.bodyPos {
			for _, v := range subVerts {
				if v.X < minB.X {
					minB.X = v.X
				}
				if v.Y < minB.Y {
					minB.Y = v.Y
				}
				if v.Z < minB.Z {
					minB.Z = v.Z
				}
				if v.X > maxB.X {
					maxB.X = v.X
				}
				if v.Y > maxB.Y {
					maxB.Y = v.Y
				}
				if v.Z > maxB.Z {
					maxB.Z = v.Z
				}
			}
		}
		for _, subVerts := range rf.headPos {
			for _, v := range subVerts {
				if v.X < minB.X {
					minB.X = v.X
				}
				if v.Y < minB.Y {
					minB.Y = v.Y
				}
				if v.Z < minB.Z {
					minB.Z = v.Z
				}
				if v.X > maxB.X {
					maxB.X = v.X
				}
				if v.Y > maxB.Y {
					maxB.Y = v.Y
				}
				if v.Z > maxB.Z {
					maxB.Z = v.Z
				}
			}
		}
	}

	measuredIdle := false
	for aIdx, aName := range actionNames {
		if aName == "idle" || aName == "stand" {
			iv := actionIntervals[aIdx]
			for f := iv[0]; f <= iv[1]; f++ {
				updateBounds(f)
				measuredIdle = true
			}
		}
	}
	if !measuredIdle {
		updateBounds(0)
	}

	centerOffset := geometry.XYZ{
		X: -(minB.X + maxB.X) * 0.5,
		Y: -(minB.Y + maxB.Y) * 0.5,
		Z: -(minB.Z + maxB.Z) * 0.5,
	}
	dx := maxB.X - minB.X
	dy := maxB.Y - minB.Y
	radius := math.Max(dx, dy) * 0.5

	// 9. Build each articulated part's Model3DEntry in local joint space and populate parent Tags
	numParts := len(activeBodies)
	numFrames := len(rawFrames)
	parts := make([]*config.Model3DEntry, numParts)
	for pIdx := 0; pIdx < numParts; pIdx++ {
		entry := config.NewModel3DEntry(numFrames, actionNames, []string{"death", "die"})
		entry.ActionDefinitions = actionNames
		entry.ActionIntervals = actionIntervals
		for fIdx := 0; fIdx < numFrames; fIdx++ {
			entry.Frames[fIdx] = config.NewModel3DEntryFrame(nil)
		}
		parts[pIdx] = entry
	}

	unitX := geometry.XYZ{X: 1, Y: 0, Z: 0}
	unitY := geometry.XYZ{X: 0, Y: 1, Z: 0}
	unitZ := geometry.XYZ{X: 0, Y: 0, Z: 1}

	for fIdx, rf := range rawFrames {
		// Determine anchor position (in centered model space) and orientation for each part
		anchorPos := make([]geometry.XYZ, numParts)
		anchorQuat := make([][4]float64, numParts)

		// Part 0 is the root coordinate frame at centered origin (0,0,0) with identity orientation
		anchorPos[0] = geometry.XYZ{X: 0, Y: 0, Z: 0}
		anchorQuat[0] = [4]float64{0, 0, 0, 1}

		for pIdx := 1; pIdx < numParts; pIdx++ {
			jIdx := activeBodies[pIdx].anchorJoint
			j := rf.skel[jIdx]
			anchorPos[pIdx] = geometry.XYZ{
				X: j.Pos.X + centerOffset.X,
				Y: j.Pos.Y + centerOffset.Y,
				Z: j.Pos.Z + centerOffset.Z,
			}
			anchorQuat[pIdx] = j.Orient
		}

		// Populate attachment Tags on each parent part's frame
		for pIdx := 1; pIdx < numParts; pIdx++ {
			parentIdx := links[pIdx].Parent
			tagName := links[pIdx].Tag
			invParentQ := physics.QuatConjugate(anchorQuat[parentIdx])
			diff := geometry.XYZ{
				X: anchorPos[pIdx].X - anchorPos[parentIdx].X,
				Y: anchorPos[pIdx].Y - anchorPos[parentIdx].Y,
				Z: anchorPos[pIdx].Z - anchorPos[parentIdx].Z,
			}
			tagOrigin := physics.QuatRotate(invParentQ, diff)
			relQ := physics.QuatNormalize(physics.QuatMultiplies(invParentQ, anchorQuat[pIdx]))
			tagAxis := [3]geometry.XYZ{
				physics.QuatRotate(relQ, unitX),
				physics.QuatRotate(relQ, unitY),
				physics.QuatRotate(relQ, unitZ),
			}
			parts[parentIdx].Frames[fIdx].Tags[tagName] = config.NewModel3DTag(tagOrigin, tagAxis)
		}

		// Transform each part's triangles into its local joint coordinate system
		for pIdx := 0; pIdx < numParts; pIdx++ {
			specs := partTris[pIdx]
			frameTris := make([]config.Model3DEntryTriangle, len(specs))
			invQ := physics.QuatConjugate(anchorQuat[pIdx])
			aPos := anchorPos[pIdx]

			for tIdx, ts := range specs {
				tri := config.NewModel3DEntryTriangle(ts.mat)
				if !ts.isHead {
					sub := mesh.SubMeshes[ts.subIdx]
					verts := rf.bodyPos[ts.subIdx]
					for k := 0; k < 3; k++ {
						vi := ts.triIndices[k]
						vRaw := verts[vi]
						vCentered := geometry.XYZ{
							X: vRaw.X + centerOffset.X,
							Y: vRaw.Y + centerOffset.Y,
							Z: vRaw.Z + centerOffset.Z,
						}
						var vLocal geometry.XYZ
						if pIdx == 0 {
							vLocal = vCentered
						} else {
							rel := geometry.XYZ{
								X: vCentered.X - aPos.X,
								Y: vCentered.Y - aPos.Y,
								Z: vCentered.Z - aPos.Z,
							}
							vLocal = physics.QuatRotate(invQ, rel)
						}
						uv := sub.Vertices[vi]
						tri.Vertices[k] = config.Model3DEntryVertex{Pos: vLocal, U: uv.U, V: uv.V}
					}
				} else {
					sub := headMesh.SubMeshes[ts.subIdx]
					verts := rf.headPos[ts.subIdx]
					for k := 0; k < 3; k++ {
						vi := ts.triIndices[k]
						vRaw := verts[vi]
						vCentered := geometry.XYZ{
							X: vRaw.X + centerOffset.X,
							Y: vRaw.Y + centerOffset.Y,
							Z: vRaw.Z + centerOffset.Z,
						}
						var vLocal geometry.XYZ
						if pIdx == 0 {
							vLocal = vCentered
						} else {
							rel := geometry.XYZ{
								X: vCentered.X - aPos.X,
								Y: vCentered.Y - aPos.Y,
								Z: vCentered.Z - aPos.Z,
							}
							vLocal = physics.QuatRotate(invQ, rel)
						}
						uv := sub.Vertices[vi]
						tri.Vertices[k] = config.Model3DEntryVertex{Pos: vLocal, U: uv.U, V: uv.V}
					}
				}
				frameTris[tIdx] = tri
			}
			parts[pIdx].Frames[fIdx].Triangles = frameTris
		}
	}

	return parts, links, radius, nil
}

// BakeMD5Model evaluates an MD5Mesh (and optional attached headMesh) with optional animations and compiles it into a Config Model3DEntry.
func (t *Things) bakeMD5Model(mesh *md5.Mesh, headMesh *md5.Mesh, headJointName string, anims map[string]*md5.Anim) (*config.Model3DEntry, error) {
	if mesh == nil || len(mesh.SubMeshes) == 0 {
		return nil, fmt.Errorf("empty md5 mesh")
	}
	// Prepare body submesh materials
	subMaterials := make([]*config.Material, len(mesh.SubMeshes))
	for sIdx, sub := range mesh.SubMeshes {
		var mat *config.Material
		if len(sub.Shader) > 0 {
			if matDef := t.materials.Resolve(sub.Shader); matDef != nil {
				mat = ResolveConfigMaterial(matDef)
			}
		}
		if mat == nil {
			mat = config.NewConfigMaterial("md5.mesh", []string{sub.Shader}, config.MaterialKindSolid)
		}
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
			if len(sub.Shader) > 0 {
				if matDef := t.materials.Resolve(sub.Shader); matDef != nil {
					mat = ResolveConfigMaterial(matDef)
				}
			}
			if mat == nil {
				mat = config.NewConfigMaterial("md5.mesh", []string{sub.Shader}, config.MaterialKindSolid)
			}
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

					rotated := physics.QuatRotate(joint.Orient, weight.Pos)
					p.X += (joint.Pos.X + rotated.X) * weight.Bias
					p.Y += (joint.Pos.Y + rotated.Y) * weight.Bias
					p.Z += (joint.Pos.Z + rotated.Z) * weight.Bias
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
						rotated := physics.QuatRotate(joint.Orient, weight.Pos)
						p.X += (joint.Pos.X + rotated.X) * weight.Bias
						p.Y += (joint.Pos.Y + rotated.Y) * weight.Bias
						p.Z += (joint.Pos.Z + rotated.Z) * weight.Bias
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
					cp := *j
					skel[jIdx] = &cp
				}
			}

			tNorm := float64(f) / float64(deathFrames-1)
			theta := -(math.Pi * 0.5) * (tNorm * tNorm)
			half := theta * 0.5
			qFall := [4]float64{0, math.Sin(half), 0, math.Cos(half)}

			for _, j := range skel {
				j.Pos = physics.QuatRotate(qFall, j.Pos)
				j.Orient = physics.QuatNormalize(physics.QuatMultiplies(qFall, j.Orient))
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

// BakeLWO parses a LightWave Object (LWO2) binary model file and converts it into a Config Model3DEntry.
func (t *Things) bakeLWO(r io.Reader) (*config.Model3DEntry, error) {
	lwd := proc.NewLWO()
	if err := lwd.Parse(r); err != nil {
		return nil, err
	}

	if len(lwd.Points) == 0 || len(lwd.Polygons) == 0 {
		return nil, fmt.Errorf("no geometry found in LWO")
	}

	// Build materials for each tag
	tagMaterials := make([]*config.Material, len(lwd.Tags))
	for i, tagName := range lwd.Tags {
		if matDef := t.materials.Resolve(tagName); matDef != nil {
			if guiMat, _ := ResolveGuiMaterial(t.arc, t.materials, matDef, nil); guiMat != nil {
				tagMaterials[i] = guiMat
			} else {
				tagMaterials[i] = ResolveConfigMaterial(matDef)
			}
		}
		if tagMaterials[i] == nil {
			tagMaterials[i] = config.NewConfigMaterial("lwo.part", []string{tagName}, config.MaterialKindSolid)
		}
	}

	var triangles []config.Model3DEntryTriangle

	for polyIdx, poly := range lwd.Polygons {
		if len(poly) < 3 {
			continue
		}

		var mat *config.Material
		if tagIdx, ok := lwd.PolyTags[polyIdx]; ok && tagIdx >= 0 && tagIdx < len(tagMaterials) {
			mat = tagMaterials[tagIdx]
		}
		if mat == nil && len(tagMaterials) > 0 {
			mat = tagMaterials[0]
		}

		getVertex := func(vi int) config.Model3DEntryVertex {
			var p geometry.XYZ
			if vi >= 0 && vi < len(lwd.Points) {
				p = lwd.Points[vi]
			}
			var u, v float32
			if uv, ok := lwd.VMadUVs[proc.UVKey{Poly: polyIdx, Vert: vi}]; ok {
				u, v = uv[0], uv[1]
			} else if uv, ok = lwd.VMapUVs[vi]; ok {
				u, v = uv[0], uv[1]
			}
			return config.Model3DEntryVertex{Pos: p, U: u, V: v}
		}

		// Triangulate convex polygons (triangle fan)
		// Reverse winding (v0, v2, v1) due to coordinate system conversion (left-handed to right-handed)
		v0 := getVertex(poly[0])
		for i := 1; i < len(poly)-1; i++ {
			v1 := getVertex(poly[i])
			v2 := getVertex(poly[i+1])

			tri := config.NewModel3DEntryTriangle(mat)
			tri.Vertices[0] = v0
			tri.Vertices[1] = v2
			tri.Vertices[2] = v1
			triangles = append(triangles, tri)
		}
	}

	if len(triangles) == 0 {
		return nil, fmt.Errorf("no triangles generated from LWO")
	}

	cModel := config.NewModel3DEntry(1, []string{"idle"}, []string{"death", "die"})
	cModel.Frames[0] = config.NewModel3DEntryFrame(triangles)
	cModel.ActionDefinitions = []string{"idle"}
	cModel.ActionIntervals = [][2]int{{0, 0}}

	cModel.Center([]string{"stand", "idle", "legs_idle", "hover"})

	return cModel, nil
}
