package doom3

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/md5"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/textures"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Things manages the loading, caching, and instantiation of 3D entities (monsters, items, weapons, ammo) in Doom 3.
type Things struct {
	arc        proc.IPkArchive
	materials  *proc.Materials
	defs       *proc.Defs
	texManager *textures.Textures
	imgLoader  *proc.ImageLoader
	modelCache map[string]*config.Model3DEntry
}

// NewThings initializes and returns a new Things manager.
func NewThings(arc proc.IPkArchive, materials *proc.Materials, defs *proc.Defs, texManager *textures.Textures, imgLoader *proc.ImageLoader) *Things {
	return &Things{
		arc:        arc,
		materials:  materials,
		defs:       defs,
		texManager: texManager,
		imgLoader:  imgLoader,
		modelCache: make(map[string]*config.Model3DEntry),
	}
}

// Create creates a new Thing entity based on the parsed MapEntity properties.
func (t *Things) Create(ent *proc.MapEntity) (*config.Thing, error) {
	classname, ok := ent.GetProperty("classname")
	if !ok {
		return nil, fmt.Errorf("entity has no classname")
	}

	const difficulty = 1

	// In Doom 3, scripted monsters with teleport, hide, trigger, trigger_anim, num_cinematics, or spawner keys
	// are dormant spawn events and are not physically active in the world at map start.
	if ent.IsDormant() {
		return nil, fmt.Errorf("dormant entity: %s", classname)
	}
	if !ent.IsLevelAvailable(difficulty) {
		return nil, fmt.Errorf("difficulty filtered entity: %s", classname)
	}

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
		converter := NewRagdollConverter(t.arc, t.materials, t.imgLoader)
		md3, radius, err = converter.Convert(spec)
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

	if spec.Kind == config.ThingEnemyDef {
		if md3 != nil {
			return CreateConfigEnemyParts(classname, pos, md3, angle, spec.Mass, spec.Speed, 300.0), nil
		}
		return CreateConfigEnemy(classname, pos, cModel, angle, spec.Mass, spec.Speed, 300.0), nil
	}
	if keyId := d3KeyId(classname); keyId != "" || spec.Kind == config.ThingKeyDef {
		if keyId == "" {
			keyId = classname
		}
		return CreateConfigKey(classname, pos, cModel, angle, spec.Mass, spec.Speed, keyId), nil
	}
	return CreateConfigItem(classname, pos, cModel, angle, spec.Mass, spec.Speed), nil
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
		converter := NewMD5Converter(t.arc, t.materials)
		cModel, err = converter.Convert(mesh, nil, "", nil)
	} else if strings.HasSuffix(cleanPath, ".lwo") {
		converter := NewLWOConverter(t.arc, t.materials)
		cModel, err = converter.Convert(f)
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
