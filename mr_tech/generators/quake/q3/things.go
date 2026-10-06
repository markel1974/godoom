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
	basePath := thingPath
	partName := thingPath
	if lastSlash := strings.LastIndex(thingPath, "/"); lastSlash != -1 {
		basePath = thingPath[:lastSlash+1]
		partName = thingPath[lastSlash+1:]
	}
	partName = strings.Replace(partName, ".md3", "", 1)
	cModel, err := t.loadMD3Part(partName, basePath)
	if err != nil {
		return nil, fmt.Errorf("can't load Model3D %s: %s", classname, err.Error())
	}

	il := NewImageLoader(t.arc, t.texManager, t.shaders)
	// Load materials for the Model3D model
	for _, frame := range cModel.Frames {
		for _, tri := range frame.Triangles {
			if tri.Material != nil && len(tri.Material.Frames) > 0 {
				texName := strings.TrimSpace(strings.ToLower(tri.Material.Frames[0]))
				if lErr := il.Load(texName, false); lErr != nil {
					fmt.Printf("warning: %s\n", lErr.Error())
				}
			}
		}
	}

	thingCfg := t.doCreate(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)

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
	bspModels, err := lumps.NewModels3(rs, reader.GetHeaders())
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
	// Geometry translation into agnostic Model3DEntry, collect all triangles in this single frame
	var allTriangles []config.Model3DEntryTriangle
	for _, bspFace := range rawFaces {
		// RETRIEVAL OF SPECIFIC TEXTURE
		texName := bspFace.TexName
		animKind := config.MaterialKindLoop
		if bspFace.Info.IsSky() {
			animKind = config.MaterialKindSky
		}
		specificMaterial := config.NewConfigMaterial([]string{texName}, animKind, 1.0, 1.0, 0, 0)
		specificMaterial.Shader = texName
		texNameLC := strings.ToLower(texName)

		t.shaders.MaterialBind(texNameLC, specificMaterial)

		// Texture Manager handling for external BModels (Q3 vs Q1/Q2)
		if texes := texManager.Get([]string{texName}); len(texes) > 0 && texes[0] != nil {
			tw, th, pixels := texes[0].RGBA()
			_ = t.texManager.RegisterPixelsRGBA(texName, tw, th, pixels, true)
		}
		rawTriangles := lumps.TriangulateConvex3d(bspFace.Points)
		// Assignment of pre-calculated UVs from IBSPReader
		for _, rawTri := range rawTriangles {
			tri := config.NewModel3DEntryTriangle(specificMaterial)
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
				tri.Vertices[k] = config.Model3DEntryVertex{Pos: pos, U: u, V: v}
			}
			allTriangles = append(allTriangles, tri)
		}
	}
	// BSPs do not have vertex-morphing animations, 1 single frame
	model3d := config.NewMD1(1, []string{"default"})
	model3d.Frames[0] = config.NewModel3DEntryFrame(allTriangles)
	thingCfg := t.doCreate(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// createConfigThing creates and configures a new Thing entity based on the provided parameters and its type.
func (t *Things) doCreate(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.Model3DEntry, angle, mass, radius, height, speed float64) *config.Thing {
	const gForce = 9.8 * 14
	thingCfg := config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
	thingCfg.GForce = gForce
	thingCfg.Model3DEntry = cModel
	if thingCfg.Kind == config.ThingEnemyDef {
		var actions []string
		if thingCfg.Model3DEntry != nil {
			actions = thingCfg.Model3DEntry.ActionDefinitions
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

// CreatePlayer loads and assembles a multi-part Quake 3 player model.
func (t *Things) CreatePlayer(basePath string, pos geometry.XYZ, classname string) (*config.Thing, error) {
	const idleDef = "idle"

	loadMaterial := func(il *ImageLoader, part *config.Model3DEntry) {
		for _, frame := range part.Frames {
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

	lower, err := t.loadMD3Part("lower", basePath)
	if err != nil {
		return nil, err
	}
	upper, err := t.loadMD3Part("upper", basePath)
	if err != nil {
		return nil, err
	}
	head, err := t.loadMD3Part("head", basePath)
	if err != nil {
		return nil, err
	}

	il := NewImageLoader(t.arc, t.texManager, t.shaders)

	loadMaterial(il, lower)
	loadMaterial(il, upper)
	loadMaterial(il, head)

	const defaultWeapon = "machinegun"
	const defaultWeaponPath = "models/weapons2/machinegun/"

	weapon, _ := t.loadMD3Part(defaultWeapon, defaultWeaponPath)
	if weapon != nil {
		loadMaterial(il, weapon)
	}

	parts := []*config.Model3DEntry{lower, upper, head, weapon}
	links := []config.Model3DLink{
		{-1, ""},
		{0, "tag_torso"},
		{1, "tag_head"},
		{1, "tag_weapon"},
	}
	md3 := config.NewMD3(parts, links)

	if rsAnim, err := t.arc.Open(basePath + "animation.cfg"); err == nil {
		if animCfg, err := lumps.NewAnimConfig3(rsAnim); err == nil {
			// Map animations to intervals based on the parsed file
			for _, anim := range animCfg.Animations {
				if strings.HasPrefix(anim.Name, "BOTH_") {
					md3.Parts[0].ActionDefinitions = append(md3.Parts[0].ActionDefinitions, anim.Name)
					md3.Parts[0].ActionIntervals = append(md3.Parts[0].ActionIntervals, [2]int{anim.FirstFrame, anim.FirstFrame + anim.NumFrames - 1})
					md3.Parts[1].ActionDefinitions = append(md3.Parts[1].ActionDefinitions, anim.Name)
					md3.Parts[1].ActionIntervals = append(md3.Parts[1].ActionIntervals, [2]int{anim.FirstFrame, anim.FirstFrame + anim.NumFrames - 1})
				} else if strings.HasPrefix(anim.Name, "TORSO_") {
					md3.Parts[1].ActionDefinitions = append(md3.Parts[1].ActionDefinitions, anim.Name)
					md3.Parts[1].ActionIntervals = append(md3.Parts[1].ActionIntervals, [2]int{anim.FirstFrame, anim.FirstFrame + anim.NumFrames - 1})
				} else if strings.HasPrefix(anim.Name, "LEGS_") {
					// Apply LegsOffset to correct the frame index for lower.md3
					adjustedFirst := anim.FirstFrame - animCfg.LegsOffset
					md3.Parts[0].ActionDefinitions = append(md3.Parts[0].ActionDefinitions, anim.Name)
					md3.Parts[0].ActionIntervals = append(md3.Parts[0].ActionIntervals, [2]int{adjustedFirst, adjustedFirst + anim.NumFrames - 1})
				}
			}
		}
	}

	// Fallback if animation.cfg is missing or empty
	if len(md3.Parts[0].ActionDefinitions) == 0 {
		md3.Parts[0].ActionDefinitions = []string{idleDef}
		md3.Parts[0].ActionIntervals = [][2]int{{0, len(lower.Frames) - 1}}
		md3.Parts[1].ActionDefinitions = []string{idleDef}
		md3.Parts[1].ActionIntervals = [][2]int{{0, len(upper.Frames) - 1}}
	}

	// Head has no specific animations in Q3, just loop frame 0
	md3.Parts[2].ActionDefinitions = []string{idleDef}
	md3.Parts[2].ActionIntervals = [][2]int{{0, 0}}

	if weapon != nil {
		md3.Parts[3].ActionDefinitions = []string{idleDef}
		md3.Parts[3].ActionIntervals = [][2]int{{0, 0}}
	}

	md3.ActionMaps = make([][]int, 4)

	md3.ActionMaps[0] = make([]int, len(md3.Parts[0].ActionDefinitions))
	for idx := range md3.Parts[0].ActionDefinitions {
		md3.ActionMaps[0][idx] = idx
	}

	md3.ActionMaps[1] = make([]int, len(md3.Parts[0].ActionDefinitions))
	for idx, lowerName := range md3.Parts[0].ActionDefinitions {
		upperIdx := 0
		nameLower := strings.ToLower(lowerName)
		found := false

		for ui, uName := range md3.Parts[1].ActionDefinitions {
			if strings.ToLower(uName) == nameLower {
				upperIdx = ui
				found = true
				break
			}
		}

		if !found {
			if strings.HasPrefix(nameLower, "both_") {
				torsoName := strings.Replace(nameLower, "both_", "torso_", 1)
				for ui, uName := range md3.Parts[1].ActionDefinitions {
					if strings.ToLower(uName) == torsoName {
						upperIdx = ui
						found = true
						break
					}
				}
			} else if strings.HasPrefix(nameLower, "legs_") {
				torsoName := "torso_stand"
				for ui, uName := range md3.Parts[1].ActionDefinitions {
					if strings.ToLower(uName) == torsoName {
						upperIdx = ui
						found = true
						break
					}
				}
			}
		}

		md3.ActionMaps[1][idx] = upperIdx
	}

	md3.ActionMaps[2] = make([]int, len(md3.Parts[0].ActionDefinitions))
	md3.ActionMaps[3] = make([]int, len(md3.Parts[0].ActionDefinitions))

	// We pass md3.Parts[0] to doCreate so that it can extract the ActionDefinitions for the enemy logic.
	// We'll set Model3DEntry to nil afterwards since this is an Model3D model.
	thingCfg := t.doCreate(classname, pos, config.ThingEnemyDef, md3.Parts[0], 0, 30.0, 16.0, 56, 600.0)
	thingCfg.Model3DEntry = nil
	thingCfg.Model3D = md3

	return thingCfg, nil
}

// MD3ToConfig converts an Model3D model into an Model3DEntry configuration, applying scaling and texture mapping if provided.
func (t *Things) loadMD3Part(partName, basePath string) (*config.Model3DEntry, error) {
	// Scale Model3D vertices (which are short ints) to float
	const md3Scale = 1.0 / 64.0

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
		return nil, fmt.Errorf("can't open Model3D %s: %s", md3Path, err.Error())
	}
	md3 := lumps.NewMD3Resource()
	res, err := md3.Parse(rsMd3)
	if err != nil {
		return nil, err
	}
	cfg := config.NewMD1(int(res.Header.NumFrames), res.FrameNames)
	if res.Header.NumTags > 0 && res.Header.OfsTags > 0 {
		for i := 0; i < int(res.Header.NumFrames); i++ {
			for j := 0; j < int(res.Header.NumTags); j++ {
				tag := res.Tags[i*int(res.Header.NumTags)+j]
				tagName := lumps.FromNullTerminatingString(tag.Name[:])
				// Model3D tags don't seem to be scaled by 1/64, but let's check later, wait, Model3D tags coordinates are float32, so no md3Scale needed!
				cfg.Frames[i].Tags[tagName] = geometry.XYZ{
					X: float64(tag.Origin[0]),
					Y: float64(tag.Origin[1]),
					Z: float64(tag.Origin[2]),
				}
			}
		}
	}

	for s := 0; s < int(res.Header.NumSurfaces); s++ {
		surf := res.Surfaces[s]

		// Find the material (use the first shader as base material)
		var material *config.Material
		surfName := lumps.FromNullTerminatingString(surf.Header.Name[:])
		var shaderName string
		if skinMap != nil {
			if texPath, ok := skinMap[surfName]; ok {
				shaderName = texPath
			}
		}

		if len(shaderName) == 0 && len(surf.Shaders) > 0 {
			shaderName = lumps.FromNullTerminatingString(surf.Shaders[0].Name[:])
			shaderName = strings.ReplaceAll(shaderName, "\\", "/")
			if len(shaderName) > 0 && !strings.Contains(shaderName, "/") {
				shaderName = basePath + shaderName
			}
		}
		if len(shaderName) > 0 {
			animKind := config.MaterialKindLoop
			texNameLC := strings.ToLower(shaderName)
			if t.shaders.IsLiquid(texNameLC) {
				animKind = config.MaterialKindLiquid
			}
			material = config.NewConfigMaterial([]string{shaderName}, animKind, 1.0, 1.0, 0, 0)
		}

		// Assemble triangles for each frame
		for i := 0; i < int(surf.Header.NumFrames); i++ {
			for t := 0; t < int(surf.Header.NumTriangles); t++ {
				var configTri config.Model3DEntryTriangle
				configTri.Material = material

				for k := 0; k < 3; k++ {
					vIndex := surf.Triangles[t].Indexes[k]

					// The 'vertices' array contains vertices of all frames concatenated
					globVIndex := (i * int(surf.Header.NumVerts)) + int(vIndex)
					v := surf.Vertices[globVIndex]
					uv := surf.TexCoords[vIndex]

					configTri.Vertices[k] = config.Model3DEntryVertex{
						Pos: geometry.XYZ{
							X: float64(v.Coord[0]) * md3Scale,
							Y: float64(v.Coord[1]) * md3Scale,
							Z: float64(v.Coord[2]) * md3Scale,
						},
						U: uv.St[0],
						V: uv.St[1],
					}
				}
				cfg.Frames[i].Triangles = append(cfg.Frames[i].Triangles, configTri)
			}
		}
	}
	return cfg, nil
}
