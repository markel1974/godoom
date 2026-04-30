package jedi

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

const aspectRatio = 1.5

const scaleX = 1.0
const scaleY = 1.0
const scaleZ = 1.0

const scaleSectorH = 1.0

const scaleTextureW = 0.1
const scaleTextureH = 0.2 * scaleSectorH //0.6

const scaleLight = 0.07
const scaleLightFalloff = 1.3

const playerHeight = 6.0 * scaleSectorH
const playerRadius = 2.5
const playerSpeed = 300
const playerMass = 8

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
func (b *Builder) Build(dir string, levelNumber int) (*config.Root, error) {
	d := NewGobHandler()
	if err := d.Parse(dir); err != nil {
		return nil, err
	}
	defer d.Close()
	levels := d.GetLevels()
	levelNumber -= 1
	if levelNumber <= 0 {
		levelNumber = 0
	}
	if levelNumber >= len(levels) {
		return nil, fmt.Errorf("level %d not found (total levels %d)", levelNumber, len(levels)-1)
	}
	baseName := levels[levelNumber]
	levelName := baseName + ExtLevel

	levelData, err := d.GetPayload(levelName)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", levelName, err)
	}

	level := NewLevel()
	if err = level.Parse(bytes.NewReader(levelData)); err != nil {
		return nil, fmt.Errorf("syntax error in %s: %w", levelName, err)
	}

	entitiesName := baseName + ".O"
	entitiesData, err := d.GetPayload(entitiesName)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", entitiesName, err)
	}

	entities := NewEntities()
	if err = entities.Parse(bytes.NewReader(entitiesData)); err != nil {
		return nil, err
	}

	palData, err := d.GetPayload(entities.LevelName + ".PAL")
	if err != nil {
		palData, err = d.GetPayload("SECBASE.PAL")
		if err != nil {
			return nil, fmt.Errorf("master palette non found: %w", err)
		}
	}

	palette := NewPalette()
	colorPal, err := palette.Parse(bytes.NewReader(palData))
	if err != nil {
		return nil, fmt.Errorf("error parsing palette: %w", err)
	}

	bm := NewBM()

	textures := NewTextures()

	configSectors := make([]*config.Sector, 0, len(level.Sectors))
	totalVertices := 0
	for _, sec := range level.Sectors {
		totalVertices += len(sec.Walls)
	}
	globalVertices := make(geometry.Polygon, 0, totalVertices)
	var lights []*config.Light

	for _, sector := range level.Sectors {
		if sector.Id < 0 {
			continue
		}

		lightLevel := sector.LightLevel * scaleLight
		lightFalloff := sector.LightLevel * scaleLightFalloff

		secId := strconv.Itoa(sector.Id)
		cSector := config.NewConfigSector(secId, lightLevel, config.LightKindAmbient, lightFalloff)

		// Quote altimetriche
		cSector.FloorY = -sector.FloorY * scaleSectorH
		cSector.CeilY = -sector.CeilingY * scaleSectorH

		//fmt.Println("---------------------------------------")
		//fmt.Println("SECTOR: ", cSector.FloorY, cSector.CeilY)

		if sector.FloorTexture >= 0 {
			texName := level.GetTexture(sector.FloorTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			cSector.Floor = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
		} else {
			fmt.Println("MISSING FLOOR_TEXTURE")
		}

		if sector.CeilingTexture >= 0 {
			texName := level.GetTexture(sector.CeilingTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
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

				cSeg := config.NewConfigSegment(secId, config.SegmentWall, v1, v2)

				//if wall.Light > 0 {
				//	pos := geometry.XYZ{X: v1.X, Y: sector.CeilingY, Z: -v1.Y}
				//	light := config.NewConfigLight(pos, float64(wall.Light), config.LightKindSpot, 50)
				//	lights = append(lights, light)
				//}

				// Inversione asse Z (profondità planare)
				//cSeg.Start.Y, cSeg.End.Y = -cSeg.Start.Y, -cSeg.End.Y

				if wall.Adjoin < 0 {
					if wall.MidTexture < 0 {
						fmt.Println("WARNING MISSING MID_TEXTURE")
						continue
					}
					texName := level.GetTexture(wall.MidTexture)
					names := textures.AddTexture(d, bm, texName, colorPal)
					cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
				} else {
					cSeg.Kind = config.SegmentUnknown
					if wall.Adjoin >= len(level.Sectors) {
						fmt.Println("INVALID ADJOIN")
						continue
					}
					adjSec := level.Sectors[wall.Adjoin]
					if sector.IsSky() && adjSec.IsSky() {
						texName := level.GetTexture(wall.TopTexture)
						_ = textures.AddTexture(d, bm, texName, colorPal)
						cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindNone, scaleTextureW, scaleTextureH, 0, 0)
					} else if sector.CeilingY < adjSec.CeilingY {
						if wall.TopTexture >= 0 {
							texName := level.GetTexture(wall.TopTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
							cSeg.Upper = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						} else {
							fmt.Println("MISSING TOP_TEXTURE")
						}
					}

					if sector.FloorY > adjSec.FloorY {
						if wall.BotTexture >= 0 {
							texName := level.GetTexture(wall.BotTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
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
		pos := CreateCoords(obj.X, obj.Y, obj.Z)
		key := CleanKey(obj.Class)
		if key == "SPIRIT" || key == "PLAYER" {
			if configPlayer == nil {
				configPlayer = b.buildPlayer(pos)
			}
		} else {
			//TODO
			/*
				cThing := &config.Thing{
					Id:       obj.Class + "_" + strconv.Itoa(i),
					Position: pos,
					Angle:    obj.Yaw,
				}
				configThings = append(configThings, cThing)

			*/
		}
	}

	calibration := config.NewConfigCalibration(false, 0, 0, 0, 0, 0, 0, true)
	calibration.AspectRatio = aspectRatio
	scaleFactor := geometry.XYZ{X: scaleX, Y: scaleY, Z: scaleZ}
	cr := config.NewConfigRoot(calibration, configSectors, configPlayer, nil, scaleFactor, textures)
	cr.Things = configThings
	cr.Vertices = globalVertices
	cr.Lights = lights

	return cr, nil
}

func (b *Builder) buildPlayer(pos geometry.XYZ) *config.Player {
	player := config.NewConfigPlayer(pos, 1.0, playerMass, playerSpeed, playerRadius, playerHeight)
	player.GForce = 9.8 * 7
	player.JumpForce = 1500

	player.Flash.ZFar = 8192
	player.Flash.Factor = 0.02
	player.Flash.Falloff = 2000
	player.Flash.OffsetX = 0.2
	player.Flash.OffsetY = 0.1
	player.Bobbing.SwayScale = 2.0
	player.Bobbing.SwayOffsetX = 3
	player.Bobbing.SwayOffsetY = -0.9
	player.Bobbing.MaxAmplitudeX = 2.0 // ESCURSIONE MASSIMA: 12 unità (circa il 20% dell'altezza player)
	player.Bobbing.MaxAmplitudeY = 2.5
	player.Bobbing.StrideLength = 0.0015 // FREQUENZA: 1000 * 0.0007 = 0.7 rad/frame.
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

// CreateCoords creates a 3D point or vector with coordinates (x, -z, y) using the geometry.XYZ struct.
func CreateCoords(x, y, z float64) geometry.XYZ {
	//return geometry.XYZ{X: x, Y: -z, Z: -y}
	return geometry.XYZ{X: x, Y: z, Z: -y}
}
