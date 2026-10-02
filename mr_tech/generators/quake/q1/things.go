package q1

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Things represents a container that manages archive access, texture resources, and a shared color palette.
type Things struct {
	arc        interfaces.IArchive
	texManager *lumps.Textures
	palette    [256]color.RGBA
}

// NewThings creates and initializes a new Things object using the provided archive, texture manager, and palette data.
func NewThings(arc interfaces.IArchive, texManager *lumps.Textures, palette [256]color.RGBA) *Things {
	return &Things{
		arc:        arc,
		texManager: texManager,
		palette:    palette,
	}
}

// CreateThing creates a new game entity (thing) based on the given position and classname.
// It returns a pointer to the created thing configuration or an error if the creation fails.
func (th *Things) CreateThing(pos geometry.XYZ, classname string) (*config.Thing, error) {
	thingPath := GetModelFileName(classname)
	if len(thingPath) == 0 {
		return nil, fmt.Errorf("unknown thing %s", classname)
	}

	skinTargetIndex := 0
	kind := config.ThingEnemyDef
	var category string
	var definition string
	if c := strings.Split(classname, "_"); len(c) > 1 {
		category = c[0]
		definition = c[1]
	}
	items := map[string]int{"armor1": 0, "armor2": 1, "armorInv": 2}
	switch category {
	case "item":
		kind = config.ThingItemDef
		if skinTIndex, ok := items[definition]; ok {
			skinTargetIndex = skinTIndex
		}
	case "weapon":
		kind = config.ThingItemDef
	case "enemy":
		kind = config.ThingEnemyDef
	case "monster":
		kind = config.ThingEnemyDef
	default:
		return nil, fmt.Errorf("unknown thing %s", classname)
	}
	rsMd1, err := th.arc.Open(thingPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", thingPath, err.Error())
	}
	md1 := lumps.NewMD1Resource()
	if err = md1.Parse(rsMd1); err != nil {
		return nil, fmt.Errorf("can't load MDL %s: %s\n", classname, err.Error())
	}
	if skinTargetIndex >= len(md1.Skins) {
		return nil, fmt.Errorf("no skin found for %s", classname)
	}
	skin := md1.Skins[skinTargetIndex]
	skinName := fmt.Sprintf("%s_skin_%d", classname, skinTargetIndex)

	if err = th.texManager.RegisterPixelsPalette(skinName, int(md1.Header.SkinWidth), int(md1.Header.SkinHeight), skin.Data, th.palette, false, 255, false); err != nil {
		return nil, fmt.Errorf("Warning: texture %s error: %s\n", skinName, err.Error())
	}
	anim := config.NewConfigMaterial([]string{skinName}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)

	cModel := config.NewMD1(int(md1.Header.NumFrames), md1.FrameNames)
	for idx, f := range md1.Frames {
		triangles := make([]config.Model3DEntryTriangle, int(md1.Header.NumTris))
		skinW := float32(md1.Header.SkinWidth)
		skinH := float32(md1.Header.SkinHeight)
		for tIdx, tri := range md1.Triangles {
			cTri := config.NewModel3DEntryTriangle(anim)
			for v := 0; v < 3; v++ {
				vx := tri.Vertices[v]
				tc := md1.TexCoords[vx]
				s := float32(tc.S)
				t := float32(tc.T)
				if tri.FacesFront == 0 && tc.OnSeam != 0 {
					s += skinW / 2.0
				}
				nU := s / skinW
				nV := 1.0 - (t / skinH)
				cTri.Vertices[v] = config.Model3DEntryVertex{Pos: lumps.CreateXYZ(f[vx][0], f[vx][1], f[vx][2]), U: nU, V: nV}
			}
			triangles[tIdx] = cTri
		}
		cFrame := config.NewModel3DEntryFrame(triangles)
		cModel.Frames[idx] = cFrame
	}

	thingCfg := th.doCreateConfigThing(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)

	return thingCfg, nil
}

// CreateInternalBModel generates a Thing with a 3D model from BSP face data, position, and classname configuration.
func (th *Things) CreateInternalBModel(rawFaces []*lumps.RawFace, position geometry.XYZ, classname string) (*config.Thing, error) {
	return nil, fmt.Errorf("warning disabled for now")
	// Geometry translation into agnostic Model3DEntry, collect all triangles in this single frame
	var allTriangles []config.Model3DEntryTriangle
	for _, bspFace := range rawFaces {
		// RETRIEVAL OF SPECIFIC TEXTURE
		texName := bspFace.TexName
		animKind := config.MaterialKindLoop
		velX, velY := 0.0, 0.0
		if bspFace.IsSky {
			animKind = config.MaterialKindSky
			velX, velY = 0.05, 0.05
		} else if len(bspFace.TexName) > 0 && bspFace.TexName[0] == '*' {
			animKind = config.MaterialKindLiquid
		}
		specificMaterial := config.NewConfigMaterial([]string{texName}, animKind, 1.0, 1.0, velX, velY)
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
	thingCfg := th.doCreateConfigThing(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)

	const heavyMass = 1000.0
	// Setup standard interactions based on classname
	if strings.HasPrefix(classname, "func_door") {
		thingCfg.Kind = config.ThingDoorDef
		thingCfg.Mass = heavyMass
	} else if strings.HasPrefix(classname, "func_plat") {
		thingCfg.Kind = config.ThingPlatformDef
		thingCfg.Mass = heavyMass
	} else if strings.HasPrefix(classname, "func_train") {
		thingCfg.Kind = config.ThingPlatformDef
		thingCfg.Mass = heavyMass
	} else if strings.HasPrefix(classname, "func_button") {
		thingCfg.Kind = config.ThingButtonDef
	}

	return thingCfg, nil
}

// CreateThingBSP loads a BSP file, processes its geometry, and creates a Thing with the specified position and classname.
func (th *Things) CreateThingBSP(bspPath string, position geometry.XYZ, classname string) (*config.Thing, error) {
	rs, err := th.arc.Open(bspPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", bspPath, err.Error())
	}
	reader := NewQ1BSPReader(th.arc, rs)
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
	// Geometry translation into agnostic Model3DEntry, collect all triangles in this single frame
	var allTriangles []config.Model3DEntryTriangle
	for _, bspFace := range rawFaces {
		// RETRIEVAL OF SPECIFIC TEXTURE
		texName := bspFace.TexName
		animKind := config.MaterialKindLoop
		velX, velY := 0.0, 0.0
		if bspFace.IsSky {
			animKind = config.MaterialKindSky
			velX, velY = 0.05, 0.05
		} else if len(bspFace.TexName) > 0 && bspFace.TexName[0] == '*' {
			animKind = config.MaterialKindLiquid
		}
		specificMaterial := config.NewConfigMaterial([]string{texName}, animKind, 1.0, 1.0, velX, velY)
		// Texture Manager handling for external BModels (Q3 vs Q1/Q2)
		if texes := texManager.Get([]string{texName}); len(texes) > 0 && texes[0] != nil {
			tWidth, tHeight, pixels := texes[0].RGBA()
			_ = th.texManager.RegisterPixelsRGBA(texName, tWidth, tHeight, pixels, false)
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
	thingCfg := th.doCreateConfigThing(classname, position, config.ThingItemDef, model3d, 0.0, 16.0, 16.0, 32.0, 0.0)
	return thingCfg, nil
}

// doCreateConfigThing creates a Thing configuration based on provided parameters and assigns logic for enemy or item behavior.
func (th *Things) doCreateConfigThing(classname string, pos geometry.XYZ, kind config.ThingType, cModel *config.Model3DEntry, angle, mass, radius, height, speed float64) *config.Thing {
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
