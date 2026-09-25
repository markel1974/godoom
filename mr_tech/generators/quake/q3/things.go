package q3

import (
	"fmt"
	_ "image/jpeg"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Things represents a structure that manages archives, shaders, and texture managers for a graphical context.
type Things struct {
	arc        interfaces.IArchive
	shaders    *Shaders
	texManager *lumps.Textures
}

// NewThings initializes and returns a new Things instance with the provided archive, shaders, and texture manager.
func NewThings(arc interfaces.IArchive, shaders *Shaders, texManager *lumps.Textures) *Things {
	return &Things{
		arc:        arc,
		shaders:    shaders,
		texManager: texManager,
	}
}

// Create creates a new Thing entity based on its position and classname, returning the configured Thing or an error.
func (t *Things) Create(thingPath string, pos geometry.XYZ, classname string) (*config.Thing, error) {
	if len(thingPath) == 0 {
		return nil, fmt.Errorf("unknown thing %s", classname)
	}

	kind := config.ThingEnemyDef
	var category string
	if c := strings.Split(classname, "_"); len(c) > 1 {
		category = c[0]
	}
	switch category {
	case "item", "ammo", "holdable":
		kind = config.ThingItemDef
	case "weapon":
		kind = config.ThingItemDef
	case "enemy":
		kind = config.ThingEnemyDef
	case "monster":
		kind = config.ThingEnemyDef
	default:
		return nil, fmt.Errorf("unknown thing %s", classname)
	}
	rsMd3, err := t.arc.Open(thingPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", thingPath, err.Error())
	}

	md3 := lumps.NewMD3Resource()
	basePath := thingPath
	if lastSlash := strings.LastIndex(thingPath, "/"); lastSlash != -1 {
		basePath = thingPath[:lastSlash+1]
	}

	cModel, err := md3.Parse(rsMd3, t.texManager, basePath, nil)
	if err != nil {
		return nil, fmt.Errorf("can't load MD3 %s: %s", classname, err.Error())
	}

	il := NewImageLoader(t.arc, t.texManager)
	// Load materials for the MD3 model
	for _, frame := range cModel.Frames {
		for _, tri := range frame.Triangles {
			if tri.Material != nil && len(tri.Material.Frames) > 0 {
				texName := tri.Material.Frames[0]
				if lErr := il.Load(texName, false); lErr != nil {
					fmt.Printf("warning: %s\n", lErr.Error())
				}
			}
		}
	}

	thingCfg := doCreate(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)

	return thingCfg, nil
}

// CreateBSP reads a BSP file, extracts its models, and generates a Thing configuration from the extracted data.
func (t *Things) CreateBSP(bspPath string, position geometry.XYZ, classname string) (*config.Thing, error) {
	rs, err := t.arc.Open(bspPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", bspPath, err.Error())
	}
	reader := NewQ3BSPReader(t.arc, rs)
	if err = reader.Setup(); err != nil {
		return nil, err
	}
	bspModels, err := reader.GetModels()
	if err != nil {
		return nil, fmt.Errorf("failed to get models from %s: %v", bspPath, err)
	}
	if len(bspModels) == 0 {
		return nil, fmt.Errorf("no model found in %s", bspPath)
	}
	rawFaces, err := reader.GetRawFaces(0)
	if err != nil {
		return nil, err
	}
	texManager := reader.GetTextures()
	// Geometry translation into agnostic MD1, collect all triangles in this single frame
	var allTriangles []config.MD1Triangle
	for _, bspFace := range rawFaces {
		// RETRIEVAL OF SPECIFIC TEXTURE
		texName := bspFace.TexName
		animKind := config.MaterialKindLoop
		if bspFace.IsSky {
			animKind = config.MaterialKindSky
		}
		specificMaterial := config.NewConfigMaterial([]string{texName}, animKind, 1.0, 1.0, 0, 0)
		specificMaterial.Shader = texName

		texNameLC := strings.ToLower(texName)
		if t.shaders.IsAdditive(texNameLC) {
			specificMaterial.BlendMode = config.BlendModeAdditive
		}
		// Texture Manager handling for external BModels (Q3 vs Q1/Q2)
		if texes := texManager.Get([]string{texName}); len(texes) > 0 && texes[0] != nil {
			tw, th, pixels := texes[0].RGBA()
			_ = t.texManager.RegisterPixelsRGBA(texName, tw, th, pixels, true)
		}
		rawTriangles := lumps.TriangulateConvex3d(bspFace.Points)
		// Assignment of pre-calculated UVs from IBSPReader
		for _, rawTri := range rawTriangles {
			tri := config.NewMD1Triangle(specificMaterial)
			for k := 0; k < 3; k++ {
				pos := rawTri[k]
				u, v := float32(0.0), float32(0.0)
				// Find corresponding UV index for vertex
				for idx, pt := range bspFace.Points {
					if pt.X == pos.X && pt.Y == pos.Y && pt.Z == pos.Z {
						if len(bspFace.UVs) > idx {
							u = float32(bspFace.UVs[idx][0])
							v = float32(bspFace.UVs[idx][1])
						}
						break
					}
				}
				tri.Vertices[k] = config.MD1Vertex{Pos: pos, U: u, V: v}
			}
			allTriangles = append(allTriangles, tri)
		}
	}
	// BSPs do not have vertex-morphing animations, 1 single frame
	model3d := config.NewMD1(1, []string{"default"})
	model3d.Frames[0] = config.NewMD1Frame(allTriangles)
	thingCfg := doCreate(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// createConfigThing creates and configures a new Thing entity based on the provided parameters and its type.
func doCreate(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.MD1, angle, mass, radius, height, speed float64) *config.Thing {
	const gForce = 9.8 * 14
	thingCfg := config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
	thingCfg.GForce = gForce
	thingCfg.MD1 = cModel
	if thingCfg.Kind == config.ThingEnemyDef {
		var actions []string
		if thingCfg.MD1 != nil {
			actions = thingCfg.MD1.ActionDefinitions
		}
		enemyLogic := common.NewEnemy(actions, 300)
		thingCfg.OnThinking = enemyLogic.OnThinking
		thingCfg.OnCollision = enemyLogic.OnCollision
		thingCfg.OnImpact = enemyLogic.OnImpact
		thingCfg.WakeUpDistance = 400
	} else {
		itemLogic := common.NewItem()
		thingCfg.OnCollision = itemLogic.OnCollision
		thingCfg.OnImpact = itemLogic.OnImpact
	}
	return thingCfg
}

// loadMD3Part loads a single MD3 part (lower, upper, or head) and its corresponding skin file.
func (t *Things) loadMD3Part(basePath, partName string) (*config.MD1, error) {
	md3Path := basePath + partName + ".md3"
	skinPath := basePath + partName + "_default.skin"

	var skinMap map[string]string
	if rsSkin, err := t.arc.Open(skinPath); err == nil {
		skin := lumps.NewSkin(rsSkin)
		skinMap, err = skin.Parse()
		if err != nil {
			return nil, fmt.Errorf("can't parse skin %s: %s", skinPath, err.Error())
		}
	}

	rsMd3, err := t.arc.Open(md3Path)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", md3Path, err.Error())
	}

	md3 := lumps.NewMD3Resource()
	return md3.Parse(rsMd3, t.texManager, basePath, skinMap)
}

// CreatePlayer loads and assembles a multi-part Quake 3 player model.
func (t *Things) CreatePlayer(basePath string, pos geometry.XYZ, classname string) (*config.Thing, error) {
	lower, err := t.loadMD3Part(basePath, "lower")
	if err != nil {
		return nil, err
	}
	upper, err := t.loadMD3Part(basePath, "upper")
	if err != nil {
		return nil, err
	}
	head, err := t.loadMD3Part(basePath, "head")
	if err != nil {
		return nil, err
	}

	il := NewImageLoader(t.arc, t.texManager)
	// Load machinegun as the default weapon
	weapon, _ := t.loadMD3Part("models/weapons2/machinegun/", "machinegun")
	for _, frame := range lower.Frames {
		for _, tri := range frame.Triangles {
			if tri.Material != nil && len(tri.Material.Frames) > 0 {
				texName := tri.Material.Frames[0]
				if lErr := il.Load(texName, false); lErr != nil {
					fmt.Printf("warning: %s\n", lErr.Error())
				}
			}
		}
	}
	for _, frame := range upper.Frames {
		for _, tri := range frame.Triangles {
			if tri.Material != nil && len(tri.Material.Frames) > 0 {
				texName := tri.Material.Frames[0]
				if lErr := il.Load(texName, false); lErr != nil {
					fmt.Printf("warning: %s\n", lErr.Error())
				}
			}
		}
	}
	for _, frame := range head.Frames {
		for _, tri := range frame.Triangles {
			if tri.Material != nil && len(tri.Material.Frames) > 0 {
				texName := tri.Material.Frames[0]
				if lErr := il.Load(texName, false); lErr != nil {
					fmt.Printf("warning: %s\n", lErr.Error())
				}
			}
		}
	}

	if weapon != nil {
		for _, frame := range weapon.Frames {
			for _, tri := range frame.Triangles {
				if tri.Material != nil && len(tri.Material.Frames) > 0 {
					texName := tri.Material.Frames[0]
					if lErr := il.Load(texName, false); lErr != nil {
						fmt.Printf("warning: %s\n", lErr.Error())
					}
				}
			}
		}
	}

	md3 := config.NewMD3(lower, upper, head, weapon)

	if rsAnim, err := t.arc.Open(basePath + "animation.cfg"); err == nil {
		if animCfg, err := lumps.ParseAnimCfg(rsAnim); err == nil {
			// Map animations to intervals based on the parsed file
			for _, anim := range animCfg.Animations {
				if strings.HasPrefix(anim.Name, "BOTH_") {
					md3.Lower.ActionDefinitions = append(md3.Lower.ActionDefinitions, anim.Name)
					md3.Lower.ActionIntervals = append(md3.Lower.ActionIntervals, [2]int{anim.FirstFrame, anim.FirstFrame + anim.NumFrames - 1})
					md3.Upper.ActionDefinitions = append(md3.Upper.ActionDefinitions, anim.Name)
					md3.Upper.ActionIntervals = append(md3.Upper.ActionIntervals, [2]int{anim.FirstFrame, anim.FirstFrame + anim.NumFrames - 1})
				} else if strings.HasPrefix(anim.Name, "TORSO_") {
					md3.Upper.ActionDefinitions = append(md3.Upper.ActionDefinitions, anim.Name)
					md3.Upper.ActionIntervals = append(md3.Upper.ActionIntervals, [2]int{anim.FirstFrame, anim.FirstFrame + anim.NumFrames - 1})
				} else if strings.HasPrefix(anim.Name, "LEGS_") {
					// Apply LegsOffset to correct the frame index for lower.md3
					adjustedFirst := anim.FirstFrame - animCfg.LegsOffset
					md3.Lower.ActionDefinitions = append(md3.Lower.ActionDefinitions, anim.Name)
					md3.Lower.ActionIntervals = append(md3.Lower.ActionIntervals, [2]int{adjustedFirst, adjustedFirst + anim.NumFrames - 1})
				}
			}
		}
	}

	// Fallback if animation.cfg is missing or empty
	if len(md3.Lower.ActionDefinitions) == 0 {
		md3.Lower.ActionDefinitions = []string{"idle"}
		md3.Lower.ActionIntervals = [][2]int{{0, len(lower.Frames) - 1}}
		md3.Upper.ActionDefinitions = []string{"idle"}
		md3.Upper.ActionIntervals = [][2]int{{0, len(upper.Frames) - 1}}
	}

	// Head has no specific animations in Q3, just loop frame 0
	md3.Head.ActionDefinitions = []string{"idle"}
	md3.Head.ActionIntervals = [][2]int{{0, 0}}

	if weapon != nil {
		md3.Weapon.ActionDefinitions = []string{"idle"}
		md3.Weapon.ActionIntervals = [][2]int{{0, 0}}
	}

	// We pass md3.Lower to doCreate so that it can extract the ActionDefinitions for the enemy logic.
	// We'll set MD1 to nil afterwards since this is an MD3 model.
	thingCfg := doCreate(classname, pos, config.ThingEnemyDef, md3.Lower, 0, 30.0, 16.0, 56, 600.0)
	thingCfg.MD1 = nil
	thingCfg.MD3 = md3

	return thingCfg, nil
}
