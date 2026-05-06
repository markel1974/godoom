package jedi

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

const aspectRatio = 1.6

const scaleX = 10.0
const scaleY = 10.0
const scaleZ = 1.0

const scaleSectorH = 8.0

const scaleTextureW = 1.0 //0.1
const scaleTextureH = 1.0 //0.2

const scaleLight = 0.11
const scaleLightFalloff = 40

const playerHeight = 6.0 * scaleSectorH
const playerRadius = 2.5
const playerSpeed = 1800
const playerMass = 40

const gForce = 9.8 * 8

type IArchive interface {
	Parse(dir string) error

	GetLevels() []string

	SetLevel(levelNumber int) error

	GetPayload(name string) ([]byte, error)

	GetTextures() *Textures

	GetLevel() *Level

	GetEntities() *Entities

	AddTexture(texName string) ([]string, error)

	AddRawTexture(texName string, sizeX int, sizeY int, pixels []byte)

	Close() error
}

// Builder represents an entity responsible for constructing and configuring level structures with a specified scale.
type Builder struct {
}

// NewBuilder creates a new Builder instance and initializes its scale factor.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build constructs and returns a *config.Root object by parsing geometry, entities, and textures from a specified directory.
// It validates the level index, processes sector topology, and integrates player and object configurations.
// Returns an error if the input directory is invalid, files are missing, or parsing fails.
func (b *Builder) Build(mode int, dir string, levelNumber int) (*config.Root, error) {
	var archive IArchive
	if mode >= 1 {
		archive = NewArchiveLab()
	} else {
		archive = NewArchiveGob()
	}

	levelNumber -= 1
	if levelNumber <= 0 {
		levelNumber = 0
	}

	if err := archive.Parse(dir); err != nil {
		return nil, err
	}
	defer archive.Close()

	if err := archive.SetLevel(levelNumber); err != nil {
		return nil, err
	}

	level := archive.GetLevel()
	entities := archive.GetEntities()

	configSectors := make([]*config.Sector, 0, len(level.Sectors))
	totalVertices := 0
	for _, sec := range level.Sectors {
		totalVertices += len(sec.Walls)
	}
	globalVertices := make(geometry.Polygon, 0, totalVertices)
	var lights []*config.Light

	for _, sector := range level.Sectors {
		if len(sector.Id) == 0 {
			continue
		}

		lightLevel := sector.LightLevel * scaleLight
		if lightLevel < 2.2 {
			lightLevel = 2.2
		}
		lightFalloff := lightLevel * scaleLightFalloff

		cSector := config.NewConfigSector(sector.Id, lightLevel, config.LightKindAmbient, lightFalloff)

		// Quote altimetriche
		cSector.FloorY = -sector.FloorY * scaleSectorH
		cSector.CeilY = -sector.CeilingY * scaleSectorH

		//fmt.Println("---------------------------------------")
		//fmt.Println("SECTOR: ", cSector.FloorY, cSector.CeilY)

		if sector.FloorTexture >= 0 {
			texName := level.GetTexture(sector.FloorTexture)
			names, _ := archive.AddTexture(texName)
			cSector.Floor = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
		} else {
			fmt.Println("MISSING FLOOR_TEXTURE")
		}

		if sector.CeilingTexture >= 0 {
			texName := level.GetTexture(sector.CeilingTexture)
			names, _ := archive.AddTexture(texName)
			animKind := config.MaterialKindLoop
			if sector.IsSky() {
				animKind = config.MaterialKindSky
				cSector.Light.Kind = config.LightKindOpenAir
			}
			cSector.Ceil = config.NewConfigMaterial(names, animKind, scaleTextureW, scaleTextureH, 0, 0)
		} else {
			fmt.Println("MISSING CEILING_TEXTURE")
		}

		wallCount := len(sector.Walls)
		if wallCount > 0 {
			cSector.Segments = make([]*config.Segment, 0, wallCount)
			for _, wall := range sector.Walls {
				if wall.LeftVertex < 0 || wall.RightVertex < 0 || wall.LeftVertex >= len(sector.Vertices) || wall.RightVertex >= len(sector.Vertices) {
					fmt.Println("INVALID VERTEX")
					continue
				}
				v1 := sector.Vertices[wall.LeftVertex]
				v2 := sector.Vertices[wall.RightVertex]

				globalVertices = append(globalVertices, v1)
				globalVertices = append(globalVertices, v2)

				cSeg := config.NewConfigSegment(sector.Id, config.SegmentWall, v1, v2)

				//if wall.Light > 0 {
				//	pos := geometry.XYZ{X: v1.X, Y: sector.CeilingY, Z: -v1.Y}
				//	light := config.NewConfigLight(pos, float64(wall.Light), config.LightKindSpot, 50)
				//	lights = append(lights, light)
				//}

				// Inversione asse Z (profondità planare)
				//cSeg.Start.Y, cSeg.End.Y = -cSeg.Start.Y, -cSeg.End.Y

				if wall.Adjoin == -1 {
					if wall.MidTexture < 0 {
						fmt.Println("WARNING MISSING MID_TEXTURE")
						continue
					}
					texName := level.GetTexture(wall.MidTexture)
					names, _ := archive.AddTexture(texName)
					cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
				} else {
					if wall.Adjoin >= len(level.Sectors) {
						fmt.Println("INVALID ADJOIN")
						continue
					}
					adjSec := level.Sectors[wall.Adjoin]
					cSeg.Kind = config.SegmentUnknown
					if sector.IsSky() && adjSec.IsSky() {
						topTexture := wall.TopTexture
						texName := level.GetTexture(topTexture)
						_, _ = archive.AddTexture(texName)
						cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindNone, scaleTextureW, scaleTextureH, 0, 0)
					} else if sector.CeilingY < adjSec.CeilingY {
						topTexture := wall.TopTexture
						if topTexture >= 0 {
							texName := level.GetTexture(topTexture)
							names, _ := archive.AddTexture(texName)
							cSeg.Upper = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						} else {
							fmt.Println("MISSING TOP_TEXTURE")
						}
					}

					if sector.FloorY > adjSec.FloorY {
						botTexture := wall.BotTexture
						if botTexture >= 0 {
							texName := level.GetTexture(botTexture)
							names, _ := archive.AddTexture(texName)
							cSeg.Lower = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						} else {
							fmt.Println("MISSING BOTTOM_TEXTURE")
						}
					}
				}
				cSector.Segments = append(cSector.Segments, cSeg)
			}
		}

		configSectors = append(configSectors, cSector)
	}

	var configThings []*config.Thing
	var configPlayer *config.Player

	for _, obj := range entities.Objects {
		if len(obj.Name) == 0 {
			continue
		}
		pos := CreateCoords(obj.X, obj.Y, obj.Z)
		key := strings.ToUpper(strings.TrimSpace(obj.Name))
		switch key {
		case "SPIRIT", "PLAYER":
			if configPlayer == nil {
				configPlayer = b.buildPlayer(pos)
			}
		case "STARTPOS": //TODO
		case "BALLSTAR": //TODO
		case "FLAGSTAR": //TODO
		case "DOCSTART": //TODO
		case "SIGNODD2": //TODO
		case "GBPLATE": //TODO
		default:
			data, err := archive.GetPayload(obj.Name + ".ITM")
			if err != nil {
				fmt.Printf("Warning: could not load ITM %s: %v\n", obj.Name, err)
				continue
			}
			item := NewItem()
			err = item.Parse(bytes.NewReader(data))
			if err != nil {
				fmt.Printf("Error parsing ITM %s: %v\n", obj.Name, err)
				continue
			}
			if len(item.Anim) == 0 {
				continue
			}
			targetName := strings.ToUpper(item.Anim)
			if strings.Contains(targetName, ".NWX") {
				cThing, err := b.NWXToThing(item.Anim, archive, pos)
				if err != nil {
					fmt.Printf("Error parsing %s: %v\n", item.Anim, err)
					continue
				}
				configThings = append(configThings, cThing)
			} else if strings.Contains(targetName, ".3DO") {
				cThing, err := b.ThreedoToThing(targetName, pos, archive)
				if err != nil {
					fmt.Printf("Error parsing %s: %v\n", item.Anim, err)
					continue
				}
				configThings = append(configThings, cThing)
			}

		}
	}

	for _, obj := range entities.Objects {
		if len(obj.Class) == 0 {
			continue
		}
		pos := CreateCoords(obj.X, obj.Y, obj.Z)
		key := strings.ToUpper(strings.TrimSpace(obj.Class))
		switch key {
		case "SPIRIT", "PLAYER":
			if configPlayer == nil {
				configPlayer = b.buildPlayer(pos)
			}
		case "SPRITE":
			dataIdx, _ := strconv.Atoi(obj.Data)
			if dataIdx < 0 || dataIdx >= len(entities.Waxes) {
				fmt.Printf("Warning: sprite index not valid: %s\n", obj.Data)
				continue
			}
			fileName := entities.Waxes[dataIdx]
			cThing, err := b.WAXToThing(fileName, archive, pos)
			if err != nil {
				fmt.Printf("Error parsing %s: %v\n", fileName, err)
				continue
			}
			configThings = append(configThings, cThing)
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
			cThing, err := b.ThreedoToThing(fileName, pos, archive)
			if err != nil {
				fmt.Printf("Error parsing %s: %v\n", fileName, err)
				continue
			}
			configThings = append(configThings, cThing)
		case "SAFE":
		default:
			return nil, fmt.Errorf("unsupported object class: %s", key)
		}
	}

	calibration := config.NewConfigCalibration(false, 0, 0, 0, 0, 0, 0, true)
	calibration.AspectRatio = aspectRatio
	scaleFactor := geometry.XYZ{X: scaleX, Y: scaleY, Z: scaleZ}
	cr := config.NewConfigRoot(calibration, configSectors, configPlayer, nil, scaleFactor, archive.GetTextures())
	cr.Things = configThings
	cr.Vertices = globalVertices
	cr.Lights = lights

	return cr, nil
}

func (b *Builder) buildPlayer(pos geometry.XYZ) *config.Player {
	player := config.NewConfigPlayer(pos, 1.0, playerMass, playerSpeed, playerRadius, playerHeight)
	playerLogic := common.NewPlayer()
	player.OnCollision = playerLogic.OnCollision
	player.OnImpact = playerLogic.OnImpact
	player.GForce = gForce
	player.JumpForce = 1000

	player.Flash.ZFar = 8192
	player.Flash.Factor = 0.02
	player.Flash.Falloff = 1500
	player.Flash.OffsetX = 0.2
	player.Flash.OffsetY = 0.1
	player.Bobbing.SwayScale = 2.0
	player.Bobbing.SwayOffsetX = 50
	player.Bobbing.SwayOffsetY = -0.9

	player.Bobbing.MaxAmplitudeX = playerHeight * 0.2
	player.Bobbing.MaxAmplitudeY = playerHeight * 0.2
	player.Bobbing.StrideLength = 0.0008 // FREQUENZA: 1000 * 0.0007 = 0.7 rad/frame.
	player.Bobbing.IdleAmpX = 0.9        // Respiro
	player.Bobbing.IdleAmpY = 0.9
	player.Bobbing.IdleDrift = 0.01
	player.Bobbing.SpeedLerp = 0.30 // Reattività istantanea alla velocità
	player.Bobbing.AmpLerp = 0.20
	player.Bobbing.ImpactMax = 1000.0
	player.Bobbing.ImpactScale = 0.02   // ATTERRAGGIO: 1000 * 0.02 = 20 unità di scuotimento verticale
	player.Bobbing.SpringTension = 0.20 // Molla più rigida (ritorno rapido)
	player.Bobbing.SpringDamping = 0.80
	player.Bobbing.TiltAmp = 0.05

	return player
}

func (b *Builder) createConfigThing(classname string, pos geometry.XYZ, kind config.ThingType, angle, mass, radius, height, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, kind, mass, radius, height, speed)
	thingCfg.GForce = gForce
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

func (b *Builder) ThreedoToThing(fileName string, pos geometry.XYZ, archive IArchive) (*config.Thing, error) {
	threedoData, err := archive.GetPayload(fileName)
	if err != nil {
		return nil, err
	}
	threedoObj := NewThreedo()
	if err := threedoObj.Parse(bytes.NewReader(threedoData)); err != nil {
		return nil, err
	}
	id := fmt.Sprintf("%s_%s", "3DO", fileName)

	var allTriangles []config.MD1Triangle

	texMap := make(map[string]*config.Material)
	for _, obj := range threedoObj.Objects {
		var material *config.Material
		if obj.TextureIdx >= 0 && obj.TextureIdx < len(threedoObj.Textures) {
			texName := threedoObj.Textures[obj.TextureIdx]
			var ok bool
			if material, ok = texMap[texName]; !ok {
				tNames, _ := archive.AddTexture(texName)
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
				tri := config.NewMD1Triangle(material)
				tri.Vertices[0] = config.MD1Vertex{Pos: v0, U: float32(uv0[0]), V: float32(uv0[1])}
				tri.Vertices[1] = config.MD1Vertex{Pos: v1, U: float32(uv1[0]), V: float32(uv1[1])}
				tri.Vertices[2] = config.MD1Vertex{Pos: v2, U: float32(uv2[0]), V: float32(uv2[1])}
				allTriangles = append(allTriangles, tri)
			}
		}
	}

	// Create a single-frame MD1
	cModel := config.NewMD1(1, []string{"stand"})
	cModel.Frames[0] = config.NewMD1Frame(allTriangles)
	//id := fmt.Sprintf("%s_%s", "3DO", fileName)
	cThing := b.createConfigThing(id, pos, config.ThingItemDef, 0, 40, 10, 50, 0)
	cThing.MD1 = cModel
	return cThing, nil
}

func (b *Builder) WAXToThing(fileName string, archive IArchive, pos geometry.XYZ) (*config.Thing, error) {
	waxData, err := archive.GetPayload(fileName)
	if err != nil {
		return nil, fmt.Errorf("could not load %s: %v\n", fileName, err)
	}
	waxData, err = DecompressPayload(waxData)
	if err != nil {
		return nil, fmt.Errorf("could not decompress %s: %v\n", fileName, err)
	}
	//fmt.Printf("Decompression Success: %s\n", fileName)
	//os.Exit(1)
	wax := NewWAX()
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
			for _, view := range act.GetViews() {
				if view == nil {
					continue
				}
				for _, cell := range view.GetCells() {
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
				archive.AddRawTexture(texId, sizeX, sizeY, cell.GetPixels())
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

func (b *Builder) NWXToThing(fileName string, archive IArchive, pos geometry.XYZ) (*config.Thing, error) {
	waxData, err := archive.GetPayload(fileName)
	if err != nil {
		return nil, fmt.Errorf("could not load %s: %v\n", fileName, err)
	}
	//fmt.Printf("Decompression Success: %s\n", fileName)
	//os.Exit(1)
	wax := NewNWX()
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
			for _, view := range act.GetViews() {
				if view == nil {
					continue
				}
				for _, cell := range view.GetCells() {
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
				archive.AddRawTexture(texId, sizeX, sizeY, cell.GetPixels())
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

// CreateCoords creates a 3D point or vector with coordinates (x, -z, y) using the geometry.XYZ struct.
func CreateCoords(x, y, z float64) geometry.XYZ {
	//return geometry.XYZ{X: x, Y: -z, Z: -y}
	return geometry.XYZ{X: x, Y: z, Z: -y}
}
