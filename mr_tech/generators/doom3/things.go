package doom3

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/textures"
)

// Things manages the loading, caching, and instantiation of 3D entities (monsters, items, weapons, ammo) in Doom 3.
type Things struct {
	arc        IPkArchive
	materials  *Materials
	texManager *textures.Textures
	imgLoader  *ImageLoader
	modelCache map[string]*config.Model3DEntry
}

// NewThings initializes and returns a new Things manager.
func NewThings(arc IPkArchive, materials *Materials, texManager *textures.Textures, imgLoader *ImageLoader) *Things {
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

	mesh := NewMD5Mesh()
	if err := mesh.Parse(mf); err != nil {
		return nil, fmt.Errorf("parsing md5mesh %s: %w", cleanPath, err)
	}

	var headMesh *MD5Mesh
	if len(spec.HeadMesh) > 0 {
		cHeadPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(spec.HeadMesh, "\\", "/")))
		if hf, err := t.arc.Open(cHeadPath); err == nil {
			parsedHead := NewMD5Mesh()
			if err := parsedHead.Parse(hf); err == nil {
				headMesh = parsedHead
			}
		}
	}

	anims := make(map[string]*MD5Anim)
	loadAnim := func(actionKey, animPath string) {
		if len(animPath) == 0 {
			return
		}
		cAnimPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(animPath, "\\", "/")))
		if af, err := t.arc.Open(cAnimPath); err == nil {
			//defer af.Close()
			anim := NewMD5Anim()
			if err := anim.Parse(af); err == nil {
				anims[actionKey] = anim
			}
		}
	}

	loadAnim("idle", spec.IdleAnim)
	loadAnim("walk", spec.WalkAnim)
	loadAnim("run", spec.RunAnim)

	cModel, err := BakeMD5Model(mesh, headMesh, spec.HeadJoint, anims, t.materials)
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
		mesh := NewMD5Mesh()
		if mErr := mesh.Parse(f); mErr != nil {
			return nil, mErr
		}
		cModel, err = BakeMD5Model(mesh, nil, "", nil, t.materials)
	} else if strings.HasSuffix(cleanPath, ".lwo") {
		cModel, err = ParseLWO(f, t.materials)
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
func (t *Things) Create(ent *MapEntity) (*config.Thing, error) {
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
		if len(cModel.ActionDefinitions) > 0 {
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
