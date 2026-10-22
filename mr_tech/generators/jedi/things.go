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

// Things represents a data handler for managing game objects and resources using the provided IArchive implementation.
type Things struct {
	archive lumps.IArchive
	eMode   EngineMode
}

// NewThings creates and initializes a new instance of Things using the provided IArchive for resource management.
func NewThings(archive lumps.IArchive, eMode EngineMode) *Things {
	return &Things{
		archive: archive,
		eMode:   eMode,
	}
}

// Build parses and constructs player and entity configurations from the given archive, returning Player, Things, or an error.
func (b *Things) Build() (*config.Player, []*config.Thing, error) {
	var cThings []*config.Thing
	var cPlayer *config.Player
	entities := b.archive.GetEntities()

	if b.eMode == EngineModeOutlaws {
		for _, obj := range entities.Objects {
			if len(obj.Name) == 0 {
				continue
			}
			pos := CreateCoords(obj.X, obj.Y, obj.Z)
			key := strings.ToUpper(strings.TrimSpace(obj.Name))
			switch key {
			case "SPIRIT", "PLAYER":
				if cPlayer == nil {
					cPlayer = b.createPlayer(pos)
				}
			case "STARTPOS": //TODO
			case "BALLSTAR": //TODO
			case "FLAGSTAR": //TODO
			case "DOCSTART": //TODO
			case "SIGNODD2": //TODO
			case "GBPLATE": //TODO
			default:
				data, dErr := b.archive.GetPayload(obj.Name + ".ITM")
				if dErr != nil {
					fmt.Printf("Warning: could not load ITM %s: %v\n", obj.Name, dErr)
					continue
				}
				item := lumps.NewItem()
				iErr := item.Parse(bytes.NewReader(data))
				if iErr != nil {
					fmt.Printf("Error parsing ITM %s: %v\n", obj.Name, iErr)
					continue
				}
				if len(item.Anim) == 0 {
					continue
				}
				targetName := strings.ToUpper(item.Anim)
				if strings.Contains(targetName, ".NWX") {
					cThing, err := b.nwxToThing(item.Anim, config.ThingEnemyDef, pos)
					if err != nil {
						fmt.Printf("Error parsing %s: %v\n", item.Anim, err)
						continue
					}
					cThings = append(cThings, cThing)
				} else if strings.Contains(targetName, ".3DO") {
					cThing, err := b.threedoToThing(targetName, config.ThingItemDef, pos)
					if err != nil {
						fmt.Printf("Error parsing %s: %v\n", item.Anim, err)
						continue
					}
					cThings = append(cThings, cThing)
				}
			}
		}
	}

	if b.eMode == EngineModeJedi {
		for _, obj := range entities.Objects {
			if len(obj.Class) == 0 {
				continue
			}
			pos := CreateCoords(obj.X, obj.Y, obj.Z)
			key := strings.ToUpper(strings.TrimSpace(obj.Class))
			switch key {
			case "SPIRIT", "PLAYER":
				if cPlayer == nil {
					cPlayer = b.createPlayer(pos)
				}
			case "SPRITE":
				dataIdx, _ := strconv.Atoi(obj.Data)
				if dataIdx < 0 || dataIdx >= len(entities.Waxes) {
					fmt.Printf("Warning: sprite index not valid: %s\n", obj.Data)
					continue
				}
				fileName := entities.Waxes[dataIdx]
				cThing, err := b.waxToThing(fileName, config.ThingEnemyDef, pos)
				if err != nil {
					fmt.Printf("Error parsing %s: %v\n", fileName, err)
					continue
				}
				cThings = append(cThings, cThing)
			case "FRAME":
				//fmt.Println("---------------- FRAME ------------")
				//fmt.Println(obj)
			case "3D":
				dataIdx, _ := strconv.Atoi(obj.Data)
				if dataIdx < 0 || dataIdx >= len(entities.Threedos) {
					fmt.Printf("Warning: 3DO/POD index not valid: %s\n", obj.Data)
					continue
				}
				fileName := entities.Threedos[dataIdx]
				cThing, err := b.threedoToThing(fileName, config.ThingEnemyDef, pos)
				if err != nil {
					fmt.Printf("Error parsing %s: %v\n", fileName, err)
					continue
				}
				cThings = append(cThings, cThing)
			case "SAFE":
			case "SOUND":

			default:
				return nil, nil, fmt.Errorf("unsupported object class: %s", key)
			}
		}
	}

	return cPlayer, cThings, nil
}

// WAXToThing converts a WAX file into a Thing object with animations or static visuals, defined at the given position.
func (b *Things) waxToThing(fileName string, kind config.ThingType, pos geometry.XYZ) (*config.Thing, error) {
	waxData, err := b.archive.GetPayload(fileName)
	if err != nil {
		return nil, fmt.Errorf("could not load %s: %v\n", fileName, err)
	}
	waxData, err = lumps.DecompressPayload(waxData)
	if err != nil {
		return nil, fmt.Errorf("could not decompress %s: %v\n", fileName, err)
	}
	wax := lumps.NewWAX()
	err = wax.Parse(fileName, bytes.NewReader(waxData))
	if err != nil {
		return nil, fmt.Errorf("error parsing WAX %s: %v\n", fileName, err)
	}
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
	cThing := b.createConfigThing(id, pos, kind, 0, 50, 3, 50, 400)
	cThing.MultiSprite = multiSprite
	return cThing, nil
}

// NWXToThing converts NWX animation data into a config.Thing by parsing its contents and creating associated materials.
func (b *Things) nwxToThing(fileName string, kind config.ThingType, pos geometry.XYZ) (*config.Thing, error) {
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
	cThing := b.createConfigThing(id, pos, kind, 0, 50, 3, 50, 400)
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
func (b *Things) threedoToThing(fileName string, kind config.ThingType, pos geometry.XYZ) (*config.Thing, error) {
	threedoData, tErr := b.archive.GetPayload(fileName)
	if tErr != nil {
		return nil, tErr
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
	cThing := b.createConfigThing(id, pos, kind, 0, 40, 10, 50, 0)
	cThing.Model3DEntry = cModel
	return cThing, nil
}

// buildPlayer initializes and returns a configured Player instance with specified position and predefined attributes.
func (b *Things) createPlayer(pos geometry.XYZ) *config.Player {
	player := config.NewConfigPlayer(pos, 1.0, playerMass, playerSpeed, playerRadius, playerHeight)
	_ = common.NewPlayer(player)
	player.JumpForce = JumpForce
	player.GForce = GForce
	player.Impulse = 0.13 //mod

	player.Flash.ZFar = 8192
	player.Flash.Intensity = 0.02
	player.Flash.Falloff = 2000
	player.Flash.OffsetX = 0.2
	player.Flash.OffsetY = 0.1
	player.Bobbing.SwayScale = 1.0 //mod
	player.Bobbing.SwayOffsetX = 50
	player.Bobbing.SwayOffsetY = -0.9
	player.Bobbing.MaxAmplitudeX = 5.0 // MAXIMUM EXCURSION: 12 units (approx. 20% of player height)
	player.Bobbing.MaxAmplitudeY = 5.5
	player.Bobbing.StrideLength = 0.07 //mod //0.0015 // FREQUENCY: 1000 * 0.0007 = 0.7 rad/frame.
	//breathing
	player.Bobbing.IdleAmpX = 0.9  //mod 0.9
	player.Bobbing.IdleAmpY = 0.9  //mod 0.9
	player.Bobbing.IdleDrift = 3.0 //2.0 - 0.01

	player.Bobbing.SpeedLerp = 0.30 // Instant reactivity to speed
	player.Bobbing.AmpLerp = 0.20
	player.Bobbing.ImpactMax = 1000.0
	player.Bobbing.ImpactScale = 0.02   // LANDING: 1000 * 0.02 = 20 units of vertical shake
	player.Bobbing.SpringTension = 0.20 // Stiffer spring (faster return)
	player.Bobbing.SpringDamping = 0.80
	player.Bobbing.TiltAmp = 0.05

	return player
}
