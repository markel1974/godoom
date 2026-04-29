package jedi

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// levelNames holds the list of level identifiers used as base names for accessing level-specific configurations and data.
//var __levelNames = []string{
//	"SECBASE", "TALAY", "SEWERS", "TEST", "GROMAS", "DTENTION",
//	"RAMSHEAD", "ROBOTICS", "NARSHADA", "JABBSHIP", "IMPCITY",
//	"FUELSTAT", "EXECUTOR", "ARC",
//}

// Builder represents an entity responsible for constructing and configuring level structures with a specified scale.
type Builder struct {
	scaleFactor float64
}

// NewBuilder creates a new Builder instance and initializes its scale factor.
func NewBuilder() *Builder {
	return &Builder{scaleFactor: 1.0}
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
	if levelNumber > len(levels) {
		return nil, fmt.Errorf("level %d not found (%d)", levelNumber, len(levels))
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

	for _, sector := range level.Sectors {
		if sector.Id < 0 {
			continue
		}

		const scaleW = 0.5
		const scaleH = 0.5

		secId := strconv.Itoa(sector.Id)
		cSector := config.NewConfigSector(secId, sector.LightLevel*0.3, config.LightKindAmbient, 0)

		// Quote altimetriche pure
		cSector.FloorY = -sector.FloorY
		cSector.CeilY = -sector.CeilingY

		//fmt.Println("---------------------------------------")
		//fmt.Println("SECTOR: ", cSector.FloorY, cSector.CeilY)

		isSky := sector.IsSky()

		if sector.FloorTexture >= 0 {
			texName := level.GetTexture(sector.FloorTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			cSector.Floor = config.NewConfigAnimation(names, config.AnimationKindLoop, scaleW, scaleH)
		} else {
			fmt.Println("MISSING FLOOR_TEXTURE")
		}

		if sector.CeilingTexture >= 0 {
			texName := level.GetTexture(sector.CeilingTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			animKind := config.AnimationKindLoop
			if isSky {
				animKind = config.AnimationKindSky
				cSector.Light.Kind = config.LightKindOpenAir
			}
			cSector.Ceil = config.NewConfigAnimation(names, animKind, scaleW, scaleH)
		} else {
			fmt.Println("MISSING CEILING_TEXTURE")
		}

		wallCount := len(sector.Walls)
		if wallCount > 0 {
			cSector.Segments = make([]*config.Segment, 0, wallCount)
			for _, wall := range sector.Walls {
				if wall.LeftVertex < 0 || wall.RightVertex < 0 {
					fmt.Println("INVALID VERTEX")
					continue
				}
				v1 := sector.Vertices[wall.LeftVertex]
				v2 := sector.Vertices[wall.RightVertex]

				globalVertices = append(globalVertices, v1)
				globalVertices = append(globalVertices, v2)

				cSeg := config.NewConfigSegment(secId, config.SegmentWall, v2, v1)

				// Inversione asse Z (profondità planare) standardizzata per mr_tech
				cSeg.Start.Y, cSeg.End.Y = -cSeg.Start.Y, -cSeg.End.Y

				if wall.MidTexture >= 0 {
					texName := level.GetTexture(wall.MidTexture)
					names := textures.AddTexture(d, bm, texName, colorPal)
					cSeg.Middle = config.NewConfigAnimation(names, config.AnimationKindLoop, scaleW, scaleH)
				} else if wall.Adjoin < 0 {
					fmt.Println("MISSING MID_TEXTURE")
				}

				if wall.Adjoin >= 0 {
					if wall.Adjoin < len(level.Sectors) {
						// Corretto: per mr_tech i portali devono essere SegmentUnknown per l'attraversamento
						cSeg.Kind = config.SegmentUnknown

						adjSec := level.Sectors[wall.Adjoin]
						adjIsSky := adjSec.IsSky()

						if isSky && adjIsSky {
							cSeg.Upper = config.NewConfigAnimation(nil, config.AnimationKindNone, scaleW, scaleH)
						} else if sector.CeilingY < adjSec.CeilingY && wall.TopTexture >= 0 {
							// UPPER TEXTURE STANDARD (uno dei due settori è un interno chiuso)
							texName := level.GetTexture(wall.TopTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
							cSeg.Upper = config.NewConfigAnimation(names, config.AnimationKindLoop, scaleW, scaleH)
						}

						// --- 2. GESTIONE LOWER WALL ---
						if sector.FloorY > adjSec.FloorY && wall.BotTexture >= 0 {
							texName := level.GetTexture(wall.BotTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
							cSeg.Lower = config.NewConfigAnimation(names, config.AnimationKindLoop, scaleW, scaleH)
						}
					} else {
						fmt.Println("INVALID ADJOIN")
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

	return cr, nil
}

// CreateCoords creates a 3D point or vector with coordinates (x, -z, y) using the geometry.XYZ struct.
func CreateCoords(x, y, z float64) geometry.XYZ {
	return geometry.XYZ{X: x, Y: -z, Z: -y}
	//return geometry.XYZ{X: x, Y: y, Z: z}
}
