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

	// 2. Estrazione e Parsing Entità (.O)
	var objAst *ObjAST
	if objData, err := d.GetPayload(baseName + ".O"); err == nil {
		objAst, err = ParseObjects(bytes.NewReader(objData))
		if err != nil {
			return nil, fmt.Errorf("errore sintattico in %s.O: %w", baseName, err)
		}
	} else {
		fmt.Printf("Warning: file %s.O non trovato nel VFS, livello privo di entità.\n", baseName)
	}

	// -------------------------------------------------------------------------
	// COSTRUZIONE TOPOLOGIA
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
		cSector.FloorY = levSec.FloorY / b.scaleFactor
		cSector.CeilY = levSec.CeilingY / b.scaleFactor

		if levSec.FloorTexture != "" {
			cSector.Floor = config.NewConfigAnimation([]string{levSec.FloorTexture}, config.AnimationKindLoop, 1.0, 1.0)
		}
		if levSec.CeilingTexture != "" {
			cSector.Ceil = config.NewConfigAnimation([]string{levSec.CeilingTexture}, config.AnimationKindLoop, 1.0, 1.0)
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
				// Inversione asse Z (profondità planare) standardizzata per mr_tech
				cSeg.Start.Y, cSeg.End.Y = -cSeg.Start.Y, -cSeg.End.Y

				if wall.Adjoin == -1 {
					cSeg.Kind = config.SegmentWall
					if wall.MidTexture != -1 {
						texName := level.GetTexture(wall.MidTexture)
						cSeg.Middle = config.NewConfigAnimation([]string{texName}, config.AnimationKindLoop, 1.0, 1.0)
					}
				} else {
					cSeg.Kind = config.SegmentUnknown
					adjSec := level.Sectors[wall.Adjoin]
					if levSec.CeilingY > adjSec.CeilingY && wall.TopTexture != -1 {
						texName := level.GetTexture(wall.TopTexture)
						cSeg.Upper = config.NewConfigAnimation([]string{texName}, config.AnimationKindLoop, 1.0, 1.0)
					}
					if levSec.FloorY < adjSec.FloorY && wall.BotTexture != -1 {
						texName := level.GetTexture(wall.BotTexture)
						cSeg.Lower = config.NewConfigAnimation([]string{texName}, config.AnimationKindLoop, 1.0, 1.0)
					}
				}
				cSector.Segments = append(cSector.Segments, cSeg)
			}
		}
		configSectors = append(configSectors, cSector)
	}

	// 2. INGESTIONE ENTITÀ (.O)
	var configThings []*config.Thing
	var configPlayer *config.Player

	if objAst != nil {
		for i, obj := range objAst.Objects {
			// Mapping coordinate: X, Z planari, Y altitudine (con ripristino della scalatura)
			pos := geometry.XYZ{
				X: obj.X / b.scaleFactor,
				Y: -obj.Z / b.scaleFactor,
				Z: obj.Y / b.scaleFactor,
			}

			// In Dark Forces "SPIRIT" è il marker per lo start del giocatore o la telecamera
			if strings.ToUpper(obj.Class) == "SPIRIT" || strings.ToUpper(obj.Class) == "PLAYER" {
				// Il primo player trovato vince (supporto limitato al single-player per i .O standard)
				if configPlayer == nil {
					configPlayer = config.NewConfigPlayer(pos, obj.Yaw, 10, 90, 1, 8)
				}
			} else {
				cThing := &config.Thing{
					Id:       obj.Class + "_" + strconv.Itoa(i),
					Position: pos,
					Angle:    obj.Yaw,
					//Sprite:   obj.Data,
				}
				configThings = append(configThings, cThing)
			}
		}
	}

	cr := config.NewConfigRoot(nil, configSectors, configPlayer, nil, b.scaleFactor, nil)
	cr.Things = configThings // Aggancio esplicito
	cr.Vertices = globalVertices

	return cr, nil
}
