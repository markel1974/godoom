package doom3

import (
	"fmt"
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

// loadModel loads and caches a 3D model (.md5mesh or .lwo) from the PK4 archive.
func (t *Things) loadModel(modelPath string) (*config.Model3DEntry, error) {
	cleanPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(modelPath, "\\", "/")))
	if cached, ok := t.modelCache[cleanPath]; ok {
		return cached, nil
	}

	f, err := t.arc.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("opening model %s: %w", cleanPath, err)
	}

	var cModel *config.Model3DEntry
	if strings.HasSuffix(cleanPath, ".md5mesh") {
		cModel, err = ParseMD5Mesh(f, t.materials)
	} else if strings.HasSuffix(cleanPath, ".lwo") {
		cModel, err = ParseLWO(f, t.materials)
	} else {
		return nil, fmt.Errorf("unsupported model format for %s", cleanPath)
	}

	if err != nil {
		return nil, fmt.Errorf("parsing model %s: %w", cleanPath, err)
	}

	// Preload all texture maps (diffuse, normal, specular) for all materials referenced by the model
	if t.imgLoader != nil {
		for _, frame := range cModel.Frames {
			for _, tri := range frame.Triangles {
				if tri.Material != nil {
					for _, texName := range tri.Material.Textures {
						t.imgLoader.LoadMaterialMaps(texName)
					}
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

	cModel, err := t.loadModel(modelPath)
	if err != nil {
		return nil, fmt.Errorf("loading model for %s (%s): %w", classname, modelPath, err)
	}

	thingCfg := config.NewConfigThing(classname, origin, angle, spec.Kind, spec.Mass, spec.Radius, spec.Height, spec.Speed)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel

	if spec.Kind == config.ThingEnemyDef {
		actions := config.NewEnemyAnimations()
		actions.Add(config.EnemyStateIdle, 0, 1)
		actions.Add(config.EnemyStateWalk, 0, 1)
		actions.Add(config.EnemyStateRun, 0, 1)

		thingCfg.JumpForce = JumpForce
		thingCfg.Impulse = 3.0
		thingCfg.WakeupDistance = 300.0
		_ = common.NewEnemy(thingCfg, actions)
	} else if spec.Kind == config.ThingItemDef {
		_ = common.NewItem(thingCfg)
	}

	return thingCfg, nil
}
