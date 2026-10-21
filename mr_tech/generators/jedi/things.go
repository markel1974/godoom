package jedi

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/jedi/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

type Things struct {
	archive lumps.IArchive
}

func NewThings(archive lumps.IArchive) *Things {
	return &Things{
		archive: archive,
	}
}

// WAXToThing converts a WAX file into a Thing object with animations or static visuals, defined at the given position.
func (b *Things) WAXToThing(fileName string, pos geometry.XYZ) (*config.Thing, error) {
	waxData, err := b.archive.GetPayload(fileName)
	if err != nil {
		return nil, fmt.Errorf("could not load %s: %v\n", fileName, err)
	}
	waxData, err = lumps.DecompressPayload(waxData)
	if err != nil {
		return nil, fmt.Errorf("could not decompress %s: %v\n", fileName, err)
	}
	//fmt.Printf("Decompression Success: %s\n", fileName)
	//os.Exit(1)
	wax := lumps.NewWAX()
	err = wax.Parse(fileName, bytes.NewReader(waxData))
	if err != nil {
		return nil, fmt.Errorf("error parsing WAX %s: %v\n", fileName, err)
	}

	/*
		var frameTextureNames []string
		for _, act := range wax.GetActions() {
			if act == nil {
				continue
			}
			for _, nodes := range act.GetViews() {
				if nodes == nil {
					continue
				}
				for _, cell := range nodes.GetCells() {
					texId := cell.GetId()
					sizeX, sizeY := cell.GetSize()
					textures.AddRawTexture(texId, sizeX, sizeY, cell.GetPixels(), colorPal)
					frameTextureNames = append(frameTextureNames, texId)
				}
			}
		}
		material := config.NewConfigMaterial(frameTextureNames, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
		id := fmt.Sprintf("%s_%s", "SPRITE", fileName)
		cThing := b.createConfigThing(id, pos, config.ThingEnemyDef, 0, 50, 3, 50, 400)
		cThing.Sprite = config.NewSprite(material)
		configThings = append(configThings, cThing)

	*/

	multiSprite := config.NewMultiSprite()
	for _, act := range wax.GetActions() {
		if act == nil {
			continue
		}
		for _, view := range act.GetViews() {
			if view == nil || len(view.GetCells()) == 0 {
				continue
			}
			var tn []string
			for _, cell := range view.GetCells() {
				texId := cell.GetId()
				sizeX, sizeY := cell.GetSize()
				b.archive.AddRawTexture(texId, sizeX, sizeY, cell.GetPixels(), false)
				tn = append(tn, texId)
			}
			material := config.NewConfigMaterial(tn, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
			multiSprite.Add(material)
		}
	}

	// Creiamo il materiale animato (o statico se 1 solo frame)
	id := fmt.Sprintf("%s_%s", "SPRITE", fileName)
	cThing := b.createConfigThing(id, pos, config.ThingEnemyDef, 0, 50, 3, 50, 400)
	cThing.MultiSprite = multiSprite
	return cThing, nil
}

// NWXToThing converts NWX animation data into a config.Thing by parsing its contents and creating associated materials.
func (b *Things) NWXToThing(fileName string, pos geometry.XYZ) (*config.Thing, error) {
	waxData, err := b.archive.GetPayload(fileName)
	if err != nil {
		return nil, fmt.Errorf("could not load %s: %v\n", fileName, err)
	}
	wax := lumps.NewNWX()
	err = wax.Parse(fileName, bytes.NewReader(waxData))
	if err != nil {
		fmt.Printf("error parsing WAX %s: %v\n", fileName, err)
	}
	multiSprite := config.NewMultiSprite()
	counter := 0
	if wax.GetSequencer() != nil {
		for _, action := range wax.GetActions() {
			if action == nil {
				continue
			}
			var tn []string
			for _, node := range action.GetNodes() {
				if node == nil {
					continue
				}
				cell := node.GetCell()
				if cell == nil {
					continue
				}
				var pixels []byte
				args := []string{strconv.Itoa(int(node.GetIndex()))}
				if node.GetFlipX() {
					args = append(args, "FlipX")
					pixels = cell.GetFlippedPixels()
				} else {
					pixels = cell.GetPixels()
				}
				texId := fmt.Sprintf("%s_%s", fileName, strings.Join(args, "_"))
				sizeX, sizeY := cell.GetSize()
				b.archive.AddRawTexture(texId, sizeX, sizeY, pixels, false)
				tn = append(tn, texId)
				//TargetIndex[ClosestAngleBucket][CurrentFrame]
			}
			if len(tn) > 0 {
				material := config.NewConfigMaterial(tn, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
				multiSprite.Add(material)
				counter += len(tn)
			}
		}
	}

	id := fmt.Sprintf("%s_%s", "SPRITE", fileName)
	cThing := b.createConfigThing(id, pos, config.ThingEnemyDef, 0, 50, 3, 50, 400)
	if counter > 0 {
		cThing.MultiSprite = multiSprite
	}
	return cThing, nil
}

// createConfigThing initializes a Thing with the specified attributes such as position, type, dimensions, and behavior logic.
func (b *Things) createConfigThing(classname string, pos geometry.XYZ, kind config.ThingType, angle, mass, radius, height, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
	thingCfg.GForce = GForce
	if thingCfg.Kind == config.ThingEnemyDef {
		actions := config.NewEnemyAnimations()
		if thingCfg.Model3DEntry != nil {
			ConvertAnimation(actions, thingCfg.Model3DEntry.ActionDefinitions, thingCfg.Model3DEntry.ActionIntervals)
		}
		_ = common.NewEnemy(thingCfg, actions)
		thingCfg.WakeupDistance = 400
	} else {
		_ = common.NewItem(thingCfg)
	}
	return thingCfg
}

// ThreedoToThing converts a 3DO model file into a Thing object using the provided position and archive for resources.
func (b *Things) ThreedoToThing(fileName string, pos geometry.XYZ) (*config.Thing, error) {
	threedoData, err := b.archive.GetPayload(fileName)
	if err != nil {
		return nil, err
	}
	threedoObj := lumps.NewThreedo()
	if err := threedoObj.Parse(bytes.NewReader(threedoData)); err != nil {
		return nil, err
	}
	id := fmt.Sprintf("%s_%s", "3DO", fileName)

	var allTriangles []config.Model3DEntryTriangle

	texMap := make(map[string]*config.Material)
	for _, obj := range threedoObj.Objects {
		var material *config.Material
		if obj.TextureIdx >= 0 && obj.TextureIdx < len(threedoObj.Textures) {
			texName := threedoObj.Textures[obj.TextureIdx]
			var ok bool
			if material, ok = texMap[texName]; !ok {
				tNames, _ := b.archive.AddTexture(texName)
				material = config.NewConfigMaterial(tNames, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
				texMap[texName] = material
			}
		}

		// Iterate over the quads (or N-gons) in this object
		for qIdx, quad := range obj.Quads {
			pLen := len(quad.VertexIndices)
			if pLen < 3 {
				continue
			}
			// Ensure we have matching texture coordinates if the fill type uses them
			hasUVs := quad.Fill == "TEXTURE" && qIdx < len(obj.TexQuads) && len(obj.TexQuads[qIdx].TexVertIndices) == pLen
			// Triangle Fan triangulation (anchored at vertex 0)
			for i := 1; i < pLen-1; i++ {
				// 1. Get physical positions
				v0 := obj.Vertices[quad.VertexIndices[0]]
				v1 := obj.Vertices[quad.VertexIndices[i]]
				v2 := obj.Vertices[quad.VertexIndices[i+1]]
				// 2. Get UV coordinates (default to 0.0 if not a textured face)
				var uv0, uv1, uv2 [2]float64
				if hasUVs {
					uv0 = obj.TexVertices[obj.TexQuads[qIdx].TexVertIndices[0]]
					uv1 = obj.TexVertices[obj.TexQuads[qIdx].TexVertIndices[i]]
					uv2 = obj.TexVertices[obj.TexQuads[qIdx].TexVertIndices[i+1]]
				}
				tri := config.NewModel3DEntryTriangle(material)
				tri.Vertices[0] = config.Model3DEntryVertex{Pos: v0, U: float32(uv0[0]), V: float32(uv0[1])}
				tri.Vertices[1] = config.Model3DEntryVertex{Pos: v1, U: float32(uv1[0]), V: float32(uv1[1])}
				tri.Vertices[2] = config.Model3DEntryVertex{Pos: v2, U: float32(uv2[0]), V: float32(uv2[1])}
				allTriangles = append(allTriangles, tri)
			}
		}
	}

	// Create a single-frame Model3DEntry
	cModel := config.NewModel3DEntry(1, []string{"stand"})
	cModel.Frames[0] = config.NewModel3DEntryFrame(allTriangles)
	//id := fmt.Sprintf("%s_%s", "3DO", fileName)
	cThing := b.createConfigThing(id, pos, config.ThingItemDef, 0, 40, 10, 50, 0)
	cThing.Model3DEntry = cModel
	return cThing, nil
}
