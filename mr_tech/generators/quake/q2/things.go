package q2

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

// Things represents a collection of resources and utilities for managing textures and images within an archive.
type Things struct {
	arc        interfaces.IArchive
	texManager *lumps.Textures
	palette    [256]color.RGBA
	il         *ImageLoader
}

// NewThings creates and initializes a new Things instance with the provided archive, texture manager, and color palette.
func NewThings(arc interfaces.IArchive, texManager *lumps.Textures, palette [256]color.RGBA) *Things {
	return &Things{
		arc:        arc,
		texManager: texManager,
		palette:    palette,
		il:         NewImageLoader(arc, texManager, palette),
	}
}

// CreateThing initializes and configures a Thing entity based on its class, 3D position, and file path, returning it or an error.
func (th *Things) CreateThing(thingPath string, pos geometry.XYZ, classname string) (*config.Thing, error) {
	if len(thingPath) == 0 {
		return nil, fmt.Errorf("empty path for thing class %s", classname)
	}
	skinTargetIndex := 0
	kind := config.ThingEnemyDef
	var category string
	var definition string
	if c := strings.Split(classname, "_"); len(c) > 1 {
		category = c[0]
		definition = c[1]
	}
	//items := map[string]int{"armor1": 0, "armor2": 1, "armorInv": 2}

	switch category {
	case "item":
		kind = config.ThingItemDef
		if strings.HasSuffix(definition, "1") {
			skinTargetIndex = 0
		} else if strings.HasSuffix(definition, "2") {
			skinTargetIndex = 1
		} else if strings.HasSuffix(definition, "Inv") {
			skinTargetIndex = 2
		}
	case "weapon":
		kind = config.ThingItemDef
	case "ammo":
		kind = config.ThingItemDef
	//case "misc":
	//	kind = config.ThingItemDef
	case "enemy":
		kind = config.ThingEnemyDef
	case "monster":
		kind = config.ThingEnemyDef
	default:
		return nil, fmt.Errorf("unknown thing %s", classname)
	}

	rsMD2, err := th.arc.Open(thingPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", thingPath, err.Error())
	}

	md2 := lumps.NewMD2Resource()
	if err = md2.Parse(rsMD2); err != nil {
		return nil, fmt.Errorf("can't load MD2 %s: %s", classname, err.Error())
	}
	if skinTargetIndex >= len(md2.Skins.Names) {
		return nil, fmt.Errorf("skin index %d out of range for %s", skinTargetIndex, classname)
	}
	fileName := md2.Skins.Names[skinTargetIndex]
	if len(fileName) == 0 {
		return nil, fmt.Errorf("empty skin name for %s", classname)
	}
	materialName := fmt.Sprintf("%s_skin_%d", classname, skinTargetIndex)
	if err = th.il.Load(materialName, fileName); err != nil {
		return nil, fmt.Errorf("failed to load skin %s: %s", fileName, err.Error())
	}

	anim := config.NewConfigMaterial([]string{materialName}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)

	cModel := config.NewMD1(int(md2.Header.NumFrames), md2.Frames.FrameNames)
	skinW := float32(md2.Header.SkinWidth)
	skinH := float32(md2.Header.SkinHeight)

	for idx, f := range md2.Frames.Frames {
		triangles := make([]config.Model3DEntryTriangle, int(md2.Header.NumTris))
		for tIdx, tri := range md2.Triangles.Triangles {
			cTri := config.NewModel3DEntryTriangle(anim)
			for v := 0; v < 3; v++ {
				vx := tri.VertexIndices[v]
				if int(vx) >= len(f) {
					return nil, fmt.Errorf("invalid MD2 vertex index %d in triangle %d", vx, tIdx)
				}
				tcIndex := tri.STIndices[v]
				if int(tcIndex) >= len(md2.TexCoords.STS) {
					return nil, fmt.Errorf("invalid MD2 texture coordinate index %d in triangle %d", tcIndex, tIdx)
				}
				tc := md2.TexCoords.STS[tcIndex]
				s := float32(tc.S)
				t := float32(tc.T)
				nU := s / skinW
				nV := 1.0 - (t / skinH)
				cTri.Vertices[v] = config.Model3DEntryVertex{
					Pos: lumps.CreateXYZ(f[vx][0], f[vx][1], f[vx][2]),
					U:   nU,
					V:   nV,
				}
			}
			triangles[tIdx] = cTri
		}
		cFrame := config.NewModel3DEntryFrame(triangles)
		cModel.Frames[idx] = cFrame
	}
	thingCfg := th.doCreateConfigThing(classname, pos, kind, cModel, 0, 30.0, 16.0, 56, 600.0)
	return thingCfg, nil
}

// CreateThingBSP creates a new Thing instance from a BSP file, defining its geometry, materials, and position in the game world.
// It loads BSP models, textures, and raw faces, translating geometry into Model3DEntry format without animations.
// Parameters:
// bspPath - Path to the BSP file.
// position - The XYZ coordinates where the Thing will be placed.
// classname - The classification name for the entity.
// Returns: A pointer to the created Thing or an error if processing the BSP file fails.
func (th *Things) CreateThingBSP(bspPath string, position geometry.XYZ, classname string) (*config.Thing, error) {
	rs, err := th.arc.Open(bspPath)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %s", bspPath, err.Error())
	}
	reader := NewQ2BSPReader(th.arc, rs)
	if err = reader.Setup(); err != nil {
		return nil, err
	}
	bspModels, err := lumps.NewModels2(rs, reader.GetHeaders())
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
		animKind := TextInfo2ToMaterialKind(bspFace.Info)
		specificMaterial := config.NewConfigMaterial([]string{texName}, animKind, 1.0, 1.0, 0, 0)
		// Texture Manager handling for external BModels (Q3 vs Q1/Q2)
		if texes := texManager.Get([]string{texName}); len(texes) > 0 && texes[0] != nil {
			tWidth, tHeight, pixels := texes[0].RGBA()
			_ = th.registerPixelsRGBA(texName, tWidth, tHeight, pixels, false)
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

// doCreateConfigThing creates a Thing configuration with specified attributes, initializing logic based on its type.
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

// registerPixels registers a texture using palette-based pixel data with optional transparency and vertical inversion.
func (th *Things) registerPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return th.texManager.RegisterPixelsPalette(name, width, height, indices, th.palette, isTransparent, transIndex, invertY)
}

// registerPixelsRGBA registers an RGBA texture by providing its name, dimensions, pixel data, and an option to invert Y-axis.
func (th *Things) registerPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	return th.texManager.RegisterPixelsRGBA(name, width, height, pixels, invertY)
}
