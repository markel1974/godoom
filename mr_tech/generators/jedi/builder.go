package jedi

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Builder is a type for constructing configuration objects by interpreting level ASTs with a specified scale factor.
type Builder struct {
	scaleFactor float64
}

// NewJediBuilder initializes a Builder with a given scale factor for processing level configurations.
func NewJediBuilder(scale float64) *Builder {
	return &Builder{scaleFactor: scale}
}

// Build converte l'AST del livello e l'AST degli oggetti nel config.Root universale
// Definizione sequenziale dei livelli di Dark Forces (Index 1 = SECBASE)
var levelNames = []string{
	"SECBASE", "TALAY", "SEWERS", "TEST", "GROMAS", "DTENTION",
	"RAMSHEAD", "ROBOTICS", "NARSHADA", "JABBSHIP", "IMPCITY",
	"FUELSTAT", "EXECUTOR", "ARC",
}

// Build converte l'AST del livello e l'AST degli oggetti nel config.Root universale
func (b *Builder) Build(dir string, levelNumber int) (*config.Root, error) {
	d := NewGobHandler()
	if err := d.Parse(dir); err != nil {
		return nil, err
	}
	defer d.Close()
	if levelNumber < 1 || levelNumber > len(levelNames) {
		return nil, fmt.Errorf("indice di livello %d fuori scala (1-%d)", levelNumber, len(levelNames))
	}
	baseName := levelNames[levelNumber-1]

	// -------------------------------------------------------------------------
	// LETTURA PAYLOAD DAL VFS E PARSING DEGLI AST
	// -------------------------------------------------------------------------

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

	// -------------------------------------------------------------------------
	// COSTRUZIONE TOPOLOGIA (Unità Native LucasArts)
	// -------------------------------------------------------------------------

	configSectors := make([]*config.Sector, 0, len(level.Sectors))
	totalVertices := 0
	for _, sec := range level.Sectors {
		totalVertices += len(sec.Walls)
	}
	globalVertices := make(geometry.Polygon, 0, totalVertices)

	// 1. INGESTIONE GEOMETRICA (.LEV)
	for _, levSec := range level.Sectors {
		if levSec.Id < 0 {
			continue
		}
		secId := strconv.Itoa(levSec.Id)
		cSector := config.NewConfigSector(secId, levSec.LightLevel, config.LightKindAmbient, 0)

		// Quote altimetriche pure
		cSector.FloorY = levSec.FloorY
		cSector.CeilY = levSec.CeilingY

		if levSec.FloorTexture >= 0 {
			texName := level.GetTexture(levSec.FloorTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			cSector.Floor = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
		}
		if levSec.CeilingTexture >= 0 {
			texName := level.GetTexture(levSec.CeilingTexture)
			names := textures.AddTexture(d, bm, texName, colorPal)
			cSector.Ceil = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
		}

		wallCount := len(levSec.Walls)
		if wallCount > 0 {
			cSector.Segments = make([]*config.Segment, 0, wallCount)
			for i, wall := range levSec.Walls {
				v1 := levSec.Vertices[wall.VertexIndex]
				nextWall := levSec.Walls[(i+1)%wallCount]
				v2 := levSec.Vertices[nextWall.VertexIndex]

				globalVertices = append(globalVertices, v1)

				cSeg := config.NewConfigSegment(secId, config.SegmentUnknown, v1, v2)
				// Inversione asse Z (profondità planare) standardizzata per mr_tech, scalatura delegata al compilatore
				cSeg.Start.Y, cSeg.End.Y = -cSeg.Start.Y, -cSeg.End.Y

				if wall.Adjoin == -1 {
					cSeg.Kind = config.SegmentWall
					if wall.MidTexture >= 0 {
						texName := level.GetTexture(wall.MidTexture)
						names := textures.AddTexture(d, bm, texName, colorPal)
						cSeg.Middle = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
					}
				} else {
					cSeg.Kind = config.SegmentUnknown
					adjSec := level.Sectors[wall.Adjoin]
					if levSec.CeilingY > adjSec.CeilingY && wall.TopTexture >= 0 {
						texName := level.GetTexture(wall.TopTexture)
						names := textures.AddTexture(d, bm, texName, colorPal)
						cSeg.Upper = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
					}
					if levSec.FloorY < adjSec.FloorY && wall.BotTexture >= 0 {
						texName := level.GetTexture(wall.BotTexture)
						names := textures.AddTexture(d, bm, texName, colorPal)
						cSeg.Lower = config.NewConfigAnimation(names, config.AnimationKindLoop, 1.0, 1.0)
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
		if strings.ToUpper(obj.Class) == "SPIRIT" || strings.ToUpper(obj.Class) == "PLAYER" {
			if configPlayer == nil {
				configPlayer = config.NewConfigPlayer(pos, obj.Yaw, 10, 90, 1, 8)
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

func CreateCoords(x, y, z float64) geometry.XYZ {
	return geometry.XYZ{X: x, Y: -z, Z: y}
}
