package jedi

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Builder represents an entity responsible for constructing and configuring level structures with a specified scale.
type Builder struct {
	scaleFactor float64
}

// NewBuilder creates a new Builder instance and initializes its scale factor.
func NewBuilder() *Builder {
	return &Builder{scaleFactor: 2.5}
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

		const scaleW = 1.0
		const scaleH = 1.5
		const scaleLight = 0.09

		lightLevel := sector.LightLevel * scaleLight
		if lightLevel < 1.6 {
			lightLevel = 1.6
		}

		secId := strconv.Itoa(sector.Id)
		cSector := config.NewConfigSector(secId, lightLevel, config.LightKindAmbient, 0)

		// Quote altimetriche
		cSector.FloorY = -sector.FloorY
		cSector.CeilY = -sector.CeilingY

		//fmt.Println("---------------------------------------")
		//fmt.Println("SECTOR: ", cSector.FloorY, cSector.CeilY)

		if sector.FloorTexture >= 0 {
			texName := level.GetTexture(sector.FloorTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			cSector.Floor = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
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
			cSector.Ceil = config.NewConfigMaterial(names, animKind, scaleW, scaleH, 0, 0)
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
					cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
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
						cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindNone, scaleW, scaleH, 0, 0)
					} else if sector.CeilingY < adjSec.CeilingY {
						if wall.TopTexture >= 0 {
							texName := level.GetTexture(wall.TopTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
							cSeg.Upper = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
						} else {
							fmt.Println("MISSING TOP_TEXTURE")
						}
					}

					if sector.FloorY > adjSec.FloorY {
						if wall.BotTexture >= 0 {
							texName := level.GetTexture(wall.BotTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
							cSeg.Lower = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
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
				configPlayer = config.NewConfigPlayer(pos, 1, 10, 100, 1, 7)
				configPlayer.Radius = 1
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
	cr := config.NewConfigRoot(calibration, configSectors, configPlayer, nil, b.scaleFactor, textures)
	cr.Things = configThings
	cr.Vertices = globalVertices
	cr.Lights = lights

	return cr, nil
}

// CreateCoords creates a 3D point or vector with coordinates (x, -z, y) using the geometry.XYZ struct.
func CreateCoords(x, y, z float64) geometry.XYZ {
	//return geometry.XYZ{X: x, Y: -z, Z: -y}
	return geometry.XYZ{X: x, Y: z, Z: -y}
}
