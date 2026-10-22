package jedi

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/jedi/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// aspectRatio defines the proportional relationship between width and height, commonly used in graphics and media contexts.
const aspectRatio = 1.6

// scaleFactor is a constant used to define a uniform scaling multiplier in various calculations.
const scaleFactor = 11.0

// scaleX represents the horizontal scaling factor derived from the base value defined by scaleFactor.
const scaleX = scaleFactor

// scaleY defines the vertical scaling factor for transformations, derived from the base value of scaleFactor.
const scaleY = scaleFactor

// scaleZ represents the scaling factor applied along the Z-axis, derived from the predefined scaleFactor constant.
const scaleZ = scaleFactor

// scaleSectorH represents a height scaling factor derived from the base scale factor.
const scaleSectorH = scaleFactor * 1

const scaleTextureW = 1.0 // scaleTextureW represents the horizontal scaling factor for a texture in the rendering process.
const scaleTextureH = 1.0 // scaleTextureH defines the horizontal scaling factor for texture rendering, set to a default value of 1.0.

// scaleLight represents the scaling factor applied for light intensity adjustments in specific calculations.
const scaleLight = 0.11

// scaleLightFalloff defines the falloff factor for light scaling, influencing the attenuation of light intensity over distance.
const scaleLightFalloff = 40

// playerHeight defines the height of the player model in the game, scaled proportionally to the sector height.
const playerHeight = 6.0 * scaleSectorH

// playerRadius defines the radius of the player entity, used for collision detection and spatial calculations.
const playerRadius = 2.5

// playerSpeed defines the movement speed of the player, measured in units per second.
const playerSpeed = 1800

// playerMass defines the default mass of the player in the game, measured in consistent in-game units.
const playerMass = 40

const GForce = 9.8 * 50

const JumpForce = GForce * 8

type EngineMode int

const (
	EngineModeJedi EngineMode = iota
	EngineModeOutlaws
)

// Builder provides a flexible mechanism for constructing complex objects step-by-step.
type Builder struct {
}

// NewBuilder initializes and returns a pointer to a new Builder instance.
func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) computeLight(sector *lumps.Sector) (float64, float64) {
	lightLevel := sector.LightLevel * scaleLight
	if lightLevel < 2.2 {
		lightLevel = 2.2
	}
	lightFalloff := lightLevel * scaleLightFalloff
	return lightLevel, lightFalloff
}

// Build constructs a game configuration based on the provided mode, directory, and level number. Returns a config.Root or error.
func (b *Builder) Build(fs common.IFileSystem, eMode EngineMode, dir string, levelNumber int) (*config.Root, error) {
	var archive lumps.IArchive
	if eMode == EngineModeOutlaws {
		archive = lumps.NewArchiveLab(fs)
	} else {
		archive = lumps.NewArchiveGob(fs)
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

	configSectors, globalVertices, lights, err := b.buildWorld(archive, eMode)
	if err != nil {
		return nil, err
	}

	configPlayer, configThings, err := b.buildEntities(archive, eMode)
	if err != nil {
		return nil, err
	}

	calibration := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	calibration.AspectRatio = aspectRatio
	sf := geometry.XYZ{X: scaleX, Y: scaleY, Z: scaleZ}
	tex := archive.GetTextures()
	cr := config.NewConfigRoot(calibration, configSectors, configPlayer, nil, sf, tex)
	cr.Things = configThings
	cr.Vertices = globalVertices
	cr.Lights = lights

	crosshair := common.NewSimpleCrosshair()
	tex.AddDirect(crosshair.GetTexture())
	cr.Crosshair = crosshair.Create()

	return cr, nil
}

// buildWorld parses the level data from the archive, constructs sectors, vertices, lights, and returns them with any errors.
func (b *Builder) buildWorld(archive lumps.IArchive, eMode EngineMode) ([]*config.Sector, geometry.Polygon, []*config.Light, error) {
	level := archive.GetLevel()
	type secWallKey struct{ secIdx, wallIdx int }
	suppressUpperPortals := make(map[secWallKey]bool)

	if eMode == EngineModeOutlaws {
		for _, sec := range level.Sectors {
			for _, w := range sec.Walls {
				if w.DAdjoin >= 0 && w.DAdjoin < len(level.Sectors) && w.Adjoin >= 0 && w.Adjoin < len(level.Sectors) {
					upperSecIdx := w.Adjoin
					upperSec := level.Sectors[upperSecIdx]
					v1 := sec.Vertices[w.LeftVertex]
					v2 := sec.Vertices[w.RightVertex]
					for uwIdx, uw := range upperSec.Walls {
						if uw.LeftVertex >= 0 && uw.RightVertex >= 0 && uw.LeftVertex < len(upperSec.Vertices) && uw.RightVertex < len(upperSec.Vertices) {
							uv1 := upperSec.Vertices[uw.LeftVertex]
							uv2 := upperSec.Vertices[uw.RightVertex]
							if (uv1 == v2 && uv2 == v1) || (uv1 == v1 && uv2 == v2) {
								suppressUpperPortals[secWallKey{secIdx: upperSecIdx, wallIdx: uwIdx}] = true
							}
						}
					}
				}
			}
		}
	}

	configSectors := make([]*config.Sector, 0, len(level.Sectors))
	totalVertices := 0
	for _, sec := range level.Sectors {
		totalVertices += len(sec.Walls)
	}
	globalVertices := make(geometry.Polygon, 0, totalVertices)
	var lights []*config.Light

	for sIdx, sector := range level.Sectors {
		if len(sector.Id) == 0 {
			continue
		}

		secFloorY := sector.FloorY

		if eMode == EngineModeOutlaws {
			if v, ok := sector.GetFlag(1, 0x80000000); ok && v != 0 {
				bestFY := sector.FloorY
				foundAdj := false
				for _, w := range sector.Walls {
					if w.Adjoin >= 0 && w.Adjoin < len(level.Sectors) {
						adjSec := level.Sectors[w.Adjoin]
						if !foundAdj || adjSec.FloorY > bestFY {
							bestFY = adjSec.FloorY
							foundAdj = true
						}
					}
				}
				if foundAdj {
					secFloorY = bestFY
				}
			}
		}

		lightLevel, lightFalloff := b.computeLight(sector)

		cSector := config.NewConfigSector(sector.Id, lightLevel, config.LightKindAmbient, lightFalloff)
		// Quote altimetriche
		cSector.FloorY = -secFloorY * scaleSectorH
		cSector.CeilY = -sector.CeilingY * scaleSectorH

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
			cSector.Segments = make([]*config.Segment, wallCount)
			for wallIdx, wall := range sector.Walls {
				if wall.LeftVertex < 0 || wall.RightVertex < 0 || wall.LeftVertex >= len(sector.Vertices) || wall.RightVertex >= len(sector.Vertices) {
					fmt.Println("INVALID VERTEX")
					continue
				}
				v1 := sector.Vertices[wall.LeftVertex]
				v2 := sector.Vertices[wall.RightVertex]

				globalVertices = append(globalVertices, v1)
				globalVertices = append(globalVertices, v2)

				cSeg := config.NewConfigSegment(sector.Id, config.SegmentUnknown, v1, v2)

				if sector.SlopedFloor != nil {
					cSeg.SlopedFloorRef = wallIdx == sector.SlopedFloor.WallIndex
				}

				if sector.SlopedCeiling != nil {
					cSeg.SlopedCeilingRef = wallIdx == sector.SlopedCeiling.WallIndex
				}

				isSuppressedUpperPortal := suppressUpperPortals[secWallKey{secIdx: sIdx, wallIdx: wallIdx}]
				if wall.Adjoin == -1 || isSuppressedUpperPortal {
					texIdx := wall.MidTexture
					if texIdx < 0 || isDefaultTexture(level.GetTexture(texIdx)) {
						if wall.TopTexture >= 0 && !isDefaultTexture(level.GetTexture(wall.TopTexture)) {
							texIdx = wall.TopTexture
						} else if wall.BotTexture >= 0 && !isDefaultTexture(level.GetTexture(wall.BotTexture)) {
							texIdx = wall.BotTexture
						}
					}
					if texIdx >= 0 {
						texName := level.GetTexture(texIdx)
						if !isDefaultTexture(texName) {
							cSeg.Kind = config.SegmentWall
							names, _ := archive.AddTexture(texName)
							cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						}
					} else if isSuppressedUpperPortal {
						cSeg.Kind = config.SegmentWall
					} else {
						fmt.Println("WARNING MISSING MID_TEXTURE")
						continue
					}
				} else {
					targetSecIdx := wall.Adjoin
					if eMode == EngineModeOutlaws {
						if wall.DAdjoin >= 0 && wall.DAdjoin < len(level.Sectors) {
							upperSec := level.Sectors[wall.Adjoin]
							lowerSec := level.Sectors[wall.DAdjoin]
							if math.Abs(sector.FloorY-lowerSec.FloorY) <= math.Abs(sector.FloorY-upperSec.FloorY) {
								targetSecIdx = wall.DAdjoin
							}
						}
					}

					if targetSecIdx >= len(level.Sectors) {
						fmt.Println("INVALID ADJOIN")
						continue
					}
					adjSec := level.Sectors[targetSecIdx]

					topTexIdx := wall.TopTexture
					if topTexIdx < 0 || isDefaultTexture(level.GetTexture(topTexIdx)) {
						if wall.DAdjoin >= 0 {
							if wall.BotTexture >= 0 && !isDefaultTexture(level.GetTexture(wall.BotTexture)) {
								topTexIdx = wall.BotTexture
							} else if wall.MidTexture >= 0 && !isDefaultTexture(level.GetTexture(wall.MidTexture)) {
								topTexIdx = wall.MidTexture
							}
						}
					}
					if topTexIdx >= 0 && (!sector.IsSky() || !adjSec.IsSky()) {
						texName := level.GetTexture(topTexIdx)
						if !isDefaultTexture(texName) {
							names, _ := archive.AddTexture(texName)
							cSeg.Upper = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						}
					}

					isStepUp := true
					if eMode == EngineModeOutlaws {
						isStepUp = adjSec.FloorY < sector.FloorY-0.01
					}
					if isStepUp && wall.BotTexture >= 0 {
						texName := level.GetTexture(wall.BotTexture)
						if !isDefaultTexture(texName) {
							names, _ := archive.AddTexture(texName)
							cSeg.Lower = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						}
					}

					// Only portal walls with bit 0 set (force middle texture on adjoin: windows, grates, bars) have middle material
					if wall.MidTexture >= 0 && (wall.Flags&1 != 0) {
						texName := level.GetTexture(wall.MidTexture)
						if !isDefaultTexture(texName) {
							names, _ := archive.AddTexture(texName)
							cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						}
					}
				}
				cSector.Segments[wallIdx] = cSeg
			}
		}
		if sector.SlopedFloor != nil {
			cSector.SlopedFloorGradient = sector.SlopedFloor.GetGradient()
		}
		if sector.SlopedCeiling != nil {
			cSector.SlopedCeilingGradient = sector.SlopedCeiling.GetGradient()
		}
		configSectors = append(configSectors, cSector)
	}
	return configSectors, globalVertices, lights, nil
}

// buildEntities parses and constructs player and entity configurations from the given archive, returning Player, Things, or an error.
func (b *Builder) buildEntities(archive lumps.IArchive, _ EngineMode) (*config.Player, []*config.Thing, error) {
	var cThings []*config.Thing
	var cPlayer *config.Player
	things := NewThings(archive)
	entities := archive.GetEntities()

	for _, obj := range entities.Objects {
		if len(obj.Name) == 0 {
			continue
		}
		pos := CreateCoords(obj.X, obj.Y, obj.Z)
		key := strings.ToUpper(strings.TrimSpace(obj.Name))
		switch key {
		case "SPIRIT", "PLAYER":
			if cPlayer == nil {
				cPlayer = b.buildPlayer(pos)
			}
		case "STARTPOS": //TODO
		case "BALLSTAR": //TODO
		case "FLAGSTAR": //TODO
		case "DOCSTART": //TODO
		case "SIGNODD2": //TODO
		case "GBPLATE": //TODO
		default:
			data, dErr := archive.GetPayload(obj.Name + ".ITM")
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
				cThing, err := things.NWXToThing(item.Anim, config.ThingEnemyDef, pos)
				if err != nil {
					fmt.Printf("Error parsing %s: %v\n", item.Anim, err)
					continue
				}
				cThings = append(cThings, cThing)
			} else if strings.Contains(targetName, ".3DO") {
				cThing, err := things.ThreedoToThing(targetName, config.ThingItemDef, pos)
				if err != nil {
					fmt.Printf("Error parsing %s: %v\n", item.Anim, err)
					continue
				}
				cThings = append(cThings, cThing)
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
			if cPlayer == nil {
				cPlayer = b.buildPlayer(pos)
			}
		case "SPRITE":
			dataIdx, _ := strconv.Atoi(obj.Data)
			if dataIdx < 0 || dataIdx >= len(entities.Waxes) {
				fmt.Printf("Warning: sprite index not valid: %s\n", obj.Data)
				continue
			}
			fileName := entities.Waxes[dataIdx]
			cThing, err := things.WAXToThing(fileName, config.ThingEnemyDef, pos)
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
			cThing, err := things.ThreedoToThing(fileName, config.ThingEnemyDef, pos)
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

	return cPlayer, cThings, nil
}

// buildPlayer initializes and returns a configured Player instance with specified position and predefined attributes.
func (b *Builder) buildPlayer(pos geometry.XYZ) *config.Player {
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

// CreateCoords generates a geometry.XYZ object from the provided x, y, and z coordinates with modified Y and Z values.
func CreateCoords(x, y, z float64) geometry.XYZ {
	//return geometry.XYZ{X: x, Y: -z, Z: -y}
	return geometry.XYZ{X: x, Y: z, Z: -y}
}

func isDefaultTexture(texName string) bool {
	return len(texName) == 0 || strings.EqualFold(texName, "DEFAULT.PCX") || strings.EqualFold(texName, "DEFAULT.BM")
}
