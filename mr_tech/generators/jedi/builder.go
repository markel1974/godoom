package jedi

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// levelNames holds the list of level identifiers used as base names for accessing level-specific configurations and data.
var __levelNames = []string{
	"SECBASE", "TALAY", "SEWERS", "TEST", "GROMAS", "DTENTION",
	"RAMSHEAD", "ROBOTICS", "NARSHADA", "JABBSHIP", "IMPCITY",
	"FUELSTAT", "EXECUTOR", "ARC",
}

// Builder represents an entity responsible for constructing and configuring level structures with a specified scale.
type Builder struct {
	scaleFactor float64
}

// NewBuilder creates a new Builder instance and initializes its scale factor.
func NewBuilder() *Builder {
	return &Builder{scaleFactor: 0.9}
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
	if levelNumber < 1 || levelNumber > len(__levelNames) {
		return nil, fmt.Errorf("indice di livello %d fuori scala (1-%d)", levelNumber, len(__levelNames))
	}
	baseName := __levelNames[levelNumber-1]

	levelData, err := d.GetPayload(baseName + ".LEV")
	if err != nil {
		return nil, fmt.Errorf("geometria %s.LEV mancante: %w", baseName, err)
	}

	level := NewLevel()
	if err = level.Parse(bytes.NewReader(levelData)); err != nil {
		return nil, fmt.Errorf("errore sintattico in %s.LEV: %w", baseName, err)
	}

	entitiesData, err := d.GetPayload(baseName + ".O")
	if err != nil {
		return nil, err
	}

	entities := NewEntities()
	if err = entities.Parse(bytes.NewReader(entitiesData)); err != nil {
		return nil, err
	}

	palData, err := d.GetPayload(entities.LevelName + ".PAL")
	if err != nil {
		palData, err = d.GetPayload("SECBASE.PAL")
		if err != nil {
			return nil, fmt.Errorf("master palette non trovata: %w", err)
		}
	}

	palette := NewPalette()
	colorPal, err := palette.Parse(bytes.NewReader(palData))
	if err != nil {
		return nil, fmt.Errorf("errore parsing palette VGA: %w", err)
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
		secId := strconv.Itoa(sector.Id)
		cSector := config.NewConfigSector(secId, sector.LightLevel, config.LightKindAmbient, 0)

		// Quote altimetriche pure
		cSector.FloorY = sector.FloorY
		cSector.CeilY = sector.CeilingY

		//fmt.Println("---------------------------------------")
		//fmt.Println("SECTOR: ", cSector.FloorY, cSector.CeilY)

		if sector.FloorTexture >= 0 {
			texName := level.GetTexture(sector.FloorTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			cSector.Floor = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
		} else {
			fmt.Println("MISSING FLOOR_TEXTURE")
		}
		if sector.CeilingTexture >= 0 {
			texName := level.GetTexture(sector.CeilingTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			cSector.Ceil = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
		} else {
			fmt.Println("MISSING CEILING_EXTURE")
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
				//globalVertices = append(globalVertices, v2)

				cSeg := config.NewConfigSegment(secId, config.SegmentWall, v1, v2)
				//cSeg.Id = strconv.Itoa(wall.Id)
				// Inversione asse Z (profondità planare) standardizzata per mr_tech
				cSeg.Start.Y, cSeg.End.Y = -cSeg.Start.Y, -cSeg.End.Y
				//fmt.Println("SEGMENT ", cSeg.Start, cSeg.End)
				if wall.MidTexture >= 0 {
					texName := level.GetTexture(wall.MidTexture)
					names := textures.AddTexture(d, bm, texName, colorPal)
					cSeg.Middle = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
				} else {
					fmt.Println("MISSING MID_TEXTURE")
				}
				if wall.Adjoin >= 0 {
					cSeg.Kind = config.SegmentUnknown
					if wall.Adjoin < len(level.Sectors) {
						adjSec := level.Sectors[wall.Adjoin]
						if sector.CeilingY > adjSec.CeilingY && wall.TopTexture >= 0 {
							texName := level.GetTexture(wall.TopTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
							cSeg.Upper = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
						}
						if sector.FloorY < adjSec.FloorY && wall.BotTexture >= 0 {
							texName := level.GetTexture(wall.BotTexture)
							names := textures.AddTexture(d, bm, texName, colorPal)
							cSeg.Lower = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
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
				configPlayer = config.NewConfigPlayer(pos, 1, 10, 100, 1, 8)
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
	return geometry.XYZ{X: x, Y: -z, Z: y}
	//return geometry.XYZ{X: x, Y: y, Z: z}
}
