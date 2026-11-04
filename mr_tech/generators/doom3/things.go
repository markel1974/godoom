package doom3

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/md5"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/generators/quake/textures"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Things manages the loading, caching, and instantiation of 3D entities (monsters, items, weapons, ammo) in Doom 3.
type Things struct {
	arc        proc.IPkArchive
	materials  *proc.Materials
	texManager *textures.Textures
	imgLoader  *proc.ImageLoader
	modelCache map[string]*config.Model3DEntry
}

// NewThings initializes and returns a new Things manager.
func NewThings(arc proc.IPkArchive, materials *proc.Materials, texManager *textures.Textures, imgLoader *proc.ImageLoader) *Things {
	return &Things{
		arc:        arc,
		materials:  materials,
		texManager: texManager,
		imgLoader:  imgLoader,
		modelCache: make(map[string]*config.Model3DEntry),
	}
}

// loadAnimatedMonster loads and caches a skinned, animated MD5 model with idle, walk, and run cycles.
func (t *Things) loadAnimatedMonster(spec Doom3ThingSpec) (*config.Model3DEntry, error) {
	cleanPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(spec.ModelPath, "\\", "/")))
	cacheKey := cleanPath
	if len(spec.HeadMesh) > 0 {
		cacheKey += "#" + strings.TrimSpace(strings.ToLower(strings.ReplaceAll(spec.HeadMesh, "\\", "/")))
	}
	if cached, ok := t.modelCache[cacheKey]; ok {
		return cached, nil
	}

	mf, err := t.arc.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("opening md5mesh %s: %w", cleanPath, err)
	}
	//defer mf.Close()

	mesh := md5.NewMesh()
	if err := mesh.Parse(mf); err != nil {
		return nil, fmt.Errorf("parsing md5mesh %s: %w", cleanPath, err)
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
			//defer af.Close()
			anim := md5.NewAnim()
			if err := anim.Parse(af); err == nil {
				anims[actionKey] = anim
			}
		}
	}

	loadAnim("idle", spec.IdleAnim)
	loadAnim("walk", spec.WalkAnim)
	loadAnim("run", spec.RunAnim)

	cModel, err := t.bakeMD5Model(mesh, headMesh, spec.HeadJoint, anims)
	if err != nil {
		return nil, fmt.Errorf("baking md5 model %s: %w", cleanPath, err)
	}

	// Preload all texture maps (diffuse, normal, specular)
	if t.imgLoader != nil && len(cModel.Frames) > 0 {
		for _, tri := range cModel.Frames[0].Triangles {
			if tri.Material != nil {
				for _, texName := range tri.Material.Textures {
					t.imgLoader.LoadMaterialMaps(texName)
				}
			}
		}
	}

	t.modelCache[cacheKey] = cModel
	return cModel, nil
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
	if ent.IsDormant() {
		return nil, fmt.Errorf("dormant entity: %s", classname)
	}

	origin, err := ent.GetOrigin()
	if err != nil {
		return nil, fmt.Errorf("entity %s origin error: %w", classname, err)
	}

	angle := ent.GetAngle()

	spec, hasSpec := GetDoom3EntitySpec(classname)
	if !hasSpec {
		return nil, fmt.Errorf("unhandled entity classname: %s", classname)
	}

	// Entity might override model explicitly
	modelPath := spec.ModelPath
	if explicitModel, hasModel := ent.GetProperty("model"); hasModel {
		expLower := strings.ToLower(explicitModel)
		if strings.HasSuffix(expLower, ".md5mesh") || strings.HasSuffix(expLower, ".lwo") {
			modelPath = explicitModel
		}
	}

	var cModel *config.Model3DEntry
	if spec.Kind == config.ThingEnemyDef && strings.HasSuffix(strings.ToLower(modelPath), ".md5mesh") {
		cModel, err = t.loadAnimatedMonster(spec)
	} else {
		cModel, err = t.loadModel(modelPath)
	}
	if err != nil {
		return nil, fmt.Errorf("loading model for %s (%s): %w", classname, modelPath, err)
	}

	height := spec.Height
	radius := spec.Radius
	if cModel != nil && len(cModel.Frames) > 0 {
		minX, minY, minZ := math.Inf(1), math.Inf(1), math.Inf(1)
		maxX, maxY, maxZ := math.Inf(-1), math.Inf(-1), math.Inf(-1)
		hasV := false
		for _, tri := range cModel.Frames[0].Triangles {
			for _, v := range tri.Vertices {
				hasV = true
				if v.Pos.X < minX {
					minX = v.Pos.X
				}
				if v.Pos.X > maxX {
					maxX = v.Pos.X
				}
				if v.Pos.Y < minY {
					minY = v.Pos.Y
				}
				if v.Pos.Y > maxY {
					maxY = v.Pos.Y
				}
				if v.Pos.Z < minZ {
					minZ = v.Pos.Z
				}
				if v.Pos.Z > maxZ {
					maxZ = v.Pos.Z
				}
			}
		}
		if hasV {
			modelH := maxZ - minZ
			if modelH > 1.0 {
				height = modelH
			}
			modelR := math.Max(maxX-minX, maxY-minY) * 0.5
			if modelR > 1.0 {
				radius = modelR
			}
		}
	}

	thingCfg := config.NewConfigThing(classname, origin, angle, spec.Kind, spec.Mass, radius, height, spec.Speed)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel

	if spec.Kind == config.ThingEnemyDef {
		actions := config.NewEnemyAnimations()
		if cModel != nil && len(cModel.ActionDefinitions) > 0 {
			ConvertAnimation(actions, cModel.ActionDefinitions, cModel.ActionIntervals)
		}
		if len(actions.Content[config.EnemyStateIdle].Indices) == 0 {
			actions.Add(config.EnemyStateIdle, 0, 1)
			actions.Add(config.EnemyStateWalk, 0, 1)
			actions.Add(config.EnemyStateRun, 0, 1)
		}

		thingCfg.JumpForce = JumpForce
		thingCfg.Impulse = 3.0
		thingCfg.WakeupDistance = 300.0
		_ = common.NewEnemy(thingCfg, actions)
	} else if spec.Kind == config.ThingItemDef {
		_ = common.NewItem(thingCfg)
	}

	return thingCfg, nil
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
				mat = CreateConfigMaterial(matDef)
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
					mat = CreateConfigMaterial(matDef)
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

	bakeFrameWithSkel := func(skel []md5.Joint) config.Model3DEntryFrame {
		var frameTriangles []config.Model3DEntryTriangle
		// Bake body submeshes
		for sIdx, sub := range mesh.SubMeshes {
			mat := subMaterials[sIdx]
			computedVerts := make([]geometry.XYZ, len(sub.Vertices))
			for i, vert := range sub.Vertices {
				var p geometry.XYZ
				for w := 0; w < vert.WeightSpan; w++ {
					weight := sub.Weights[vert.WeightIdx+w]
					joint := skel[weight.Joint]

					rotated := md5.QuatRotate(joint.Orient, weight.Pos)
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
						joint := headSkel[weight.Joint]
						rotated := md5.QuatRotate(joint.Orient, weight.Pos)
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

	// Process actions in standard order: idle, walk, run
	actionKeys := []string{"idle", "walk", "run"}
	for _, aKey := range actionKeys {
		anim, hasAnim := anims[aKey]
		if !hasAnim || anim == nil || len(anim.Frames) == 0 {
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

	cModel := config.NewModel3DEntry(len(allFrames), actionNames)
	for i, f := range allFrames {
		cModel.Frames[i] = f
	}
	cModel.ActionDefinitions = actionNames
	cModel.ActionIntervals = actionIntervals

	// Center the model in X, Y, and Z so that its center is at (0, 0, 0).
	// In Godoom, rendering places the model at (center.x, center.z, -center.y).
	// Centering ensures that the feet touch the floor at pos.Z and the head reaches pos.Z + height.
	cModel.Center()

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
			tagMaterials[i] = CreateConfigMaterial(matDef)
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
			} else if uv, ok := lwd.VMapUVs[vi]; ok {
				u, v = uv[0], uv[1]
			}
			return config.Model3DEntryVertex{
				Pos: p,
				U:   u,
				V:   v,
			}
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

	cModel := config.NewModel3DEntry(1, []string{"idle"})
	cModel.Frames[0] = config.NewModel3DEntryFrame(triangles)
	cModel.ActionDefinitions = []string{"idle"}
	cModel.ActionIntervals = [][2]int{{0, 0}}

	cModel.Center()

	return cModel, nil
}
