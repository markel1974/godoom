package q3

import (
	"fmt"
	_ "image/jpeg"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	common2 "github.com/markel1974/godoom/mr_tech/generators/quake/textures"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Things represents a structure that manages archives, shaders, and texture managers for a graphical context.
type Things struct {
	arc        interfaces.IArchive
	shaders    *Shaders
	texManager *common2.Textures
}

// NewThings initializes and returns a new Things instance with the provided archive, shaders, and texture manager.
func NewThings(arc interfaces.IArchive, shaders *Shaders, texManager *common2.Textures) *Things {
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
		specificMaterial := config.NewConfigMaterial("thing", []string{texName}, animKind)
		specificMaterial.Name = texName
		texNameLC := strings.ToLower(texName)

		t.shaders.MaterialBind(texNameLC, specificMaterial)

		// Texture Manager handling for external BModels (Q3 vs Q1/Q2)
		if texes := texManager.GetFrames([]string{texName}); len(texes) > 0 && texes[0] != nil {
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
	model3d := config.NewModel3DEntry(1, []string{"default"})
	model3d.Frames[0] = config.NewModel3DEntryFrame(allTriangles)
	thingCfg := t.doCreate(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// createConfigThing creates and configures a new Thing entity based on the provided parameters and its type.
func (t *Things) doCreate(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.Model3DEntry, angle, mass, radius, height, speed float64) *config.Thing {
	const wakeupDistance = 300
	var thingCfg *config.Thing
	if kind == config.ThingEnemyDef {
		actions := config.NewEnemyAnimations()
		if cModel != nil {
			ConvertAnimation(actions, cModel.ActionDefinitions, cModel.ActionIntervals)
		}
		thingCfg = config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
		thingCfg.GForce = GForce
		thingCfg.Model3DEntry = cModel
		thingCfg.JumpForce = JumpForce
		thingCfg.WakeupDistance = wakeupDistance
		_ = common.NewEnemy(thingCfg, actions)
	} else {
		//cModel.DropToFloor()
		thingCfg = config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
		thingCfg.GForce = GForce
		thingCfg.Model3DEntry = cModel
		_ = common.NewItem(thingCfg)
	}
	return thingCfg
}

func (t *Things) doCreateParts(classname string, pos geometry.XYZ, kind config.ThingType, md3 *config.Model3D, angle, mass, radius, height, speed float64) *config.Thing {
	const wakeupDistance = 300
	thingCfg := config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
	thingCfg.Model3D = md3
	thingCfg.GForce = GForce
	if thingCfg.Kind == config.ThingEnemyDef {
		actions := config.NewEnemyAnimations()
		if md3 != nil && len(md3.ActionDefinitions) > 0 {
			ConvertAnimation(actions, md3.ActionDefinitions, md3.ActionIntervals)
		}
		_ = common.NewEnemy(thingCfg, actions)
		thingCfg.JumpForce = JumpForce
		thingCfg.WakeupDistance = wakeupDistance
	} else {
		_ = common.NewItem(thingCfg)
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

	// Reset any dummy actions generated from raw MD3 frame names
	lower.ActionDefinitions = nil
	lower.ActionIntervals = nil
	upper.ActionDefinitions = nil
	upper.ActionIntervals = nil
	head.ActionDefinitions = nil
	head.ActionIntervals = nil
	if weapon != nil {
		weapon.ActionDefinitions = nil
		weapon.ActionIntervals = nil
	}

	var globalActionNames []string
	var globalActionIntervals [][2]int

	if rsAnim, err := t.arc.Open(basePath + "animation.cfg"); err == nil {
		if animCfg, err := lumps.NewAnimConfig3(rsAnim); err == nil {
			// Map animations to intervals based on the parsed file
			for _, anim := range animCfg.Animations {
				globalActionNames = append(globalActionNames, anim.Name)
				globalActionIntervals = append(globalActionIntervals, [2]int{anim.FirstFrame, anim.FirstFrame + anim.NumFrames - 1})

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
	if len(globalActionNames) == 0 {
		globalActionNames = []string{idleDef}
		globalActionIntervals = [][2]int{{0, len(lower.Frames) - 1}}
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

	md3.ActionDefinitions = globalActionNames
	md3.ActionIntervals = globalActionIntervals

	defaultLegsIdle := 0
	for li, lName := range md3.Parts[0].ActionDefinitions {
		if strings.EqualFold(lName, "legs_idle") {
			defaultLegsIdle = li
			break
		}
	}

	defaultTorsoStand := 0
	for ui, uName := range md3.Parts[1].ActionDefinitions {
		if strings.EqualFold(uName, "torso_stand") {
			defaultTorsoStand = ui
			break
		}
	}

	numGlobal := len(globalActionNames)
	md3.ActionMaps = make([][]int, 4)
	md3.ActionMaps[0] = make([]int, numGlobal)
	md3.ActionMaps[1] = make([]int, numGlobal)
	md3.ActionMaps[2] = make([]int, numGlobal)
	md3.ActionMaps[3] = make([]int, numGlobal)

	for idx, gName := range globalActionNames {
		// Part 0 (Legs / Lower)
		lowerIdx := -1
		for li, lName := range md3.Parts[0].ActionDefinitions {
			if strings.EqualFold(lName, gName) {
				lowerIdx = li
				break
			}
		}
		if lowerIdx == -1 {
			if strings.HasPrefix(strings.ToLower(gName), "torso_") {
				lowerIdx = defaultLegsIdle
			} else {
				lowerIdx = 0
			}
		}
		md3.ActionMaps[0][idx] = lowerIdx

		// Part 1 (Torso / Upper)
		upperIdx := -1
		for ui, uName := range md3.Parts[1].ActionDefinitions {
			if strings.EqualFold(uName, gName) {
				upperIdx = ui
				break
			}
		}
		if upperIdx == -1 {
			if strings.HasPrefix(strings.ToLower(gName), "legs_") {
				upperIdx = defaultTorsoStand
			} else {
				upperIdx = 0
			}
		}
		md3.ActionMaps[1][idx] = upperIdx

		// Part 2 (Head) & Part 3 (Weapon)
		md3.ActionMaps[2][idx] = 0
		md3.ActionMaps[3][idx] = 0
	}

	thingCfg := t.doCreateParts(classname, pos, config.ThingEnemyDef, md3, 0, 30.0, 16.0, 56, 600.0)

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
	cfg := config.NewModel3DEntry(int(res.Header.NumFrames), nil)
	cfg.ActionDefinitions = []string{"idle"}
	cfg.ActionIntervals = [][2]int{{0, int(res.Header.NumFrames) - 1}}
	if res.Header.NumTags > 0 && res.Header.OfsTags > 0 {
		for i := 0; i < int(res.Header.NumFrames); i++ {
			for j := 0; j < int(res.Header.NumTags); j++ {
				tag := res.Tags[i*int(res.Header.NumTags)+j]
				tagName := lumps.FromNullTerminatingString(tag.Name[:])
				axis := [3]geometry.XYZ{
					{X: float64(tag.Axis[0][0]), Y: float64(tag.Axis[0][1]), Z: float64(tag.Axis[0][2])},
					{X: float64(tag.Axis[1][0]), Y: float64(tag.Axis[1][1]), Z: float64(tag.Axis[1][2])},
					{X: float64(tag.Axis[2][0]), Y: float64(tag.Axis[2][1]), Z: float64(tag.Axis[2][2])},
				}
				if axis[0].X == 0 && axis[0].Y == 0 && axis[0].Z == 0 &&
					axis[1].X == 0 && axis[1].Y == 0 && axis[1].Z == 0 &&
					axis[2].X == 0 && axis[2].Y == 0 && axis[2].Z == 0 {
					axis[0] = geometry.XYZ{X: 1, Y: 0, Z: 0}
					axis[1] = geometry.XYZ{X: 0, Y: 1, Z: 0}
					axis[2] = geometry.XYZ{X: 0, Y: 0, Z: 1}
				}
				cfg.Frames[i].Tags[tagName] = config.Model3DTag{
					Origin: geometry.XYZ{
						X: float64(tag.Origin[0]),
						Y: float64(tag.Origin[1]),
						Z: float64(tag.Origin[2]),
					},
					Axis: axis,
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
			material = config.NewConfigMaterial("md3.part", []string{shaderName}, animKind)
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
