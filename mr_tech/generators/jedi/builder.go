package jedi

import (
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// LevAST represents the Abstract Syntax Tree (AST) of a level, containing a slice of sectors that define the level's structure.
type LevAST struct {
	Sectors []LevSector
}

// LevSector represents a sector in a level, defining its geometry, textures, light level, and associated walls.
type LevSector struct {
	Id             int
	FloorY         float64
	CeilingY       float64
	FloorTexture   string
	CeilingTexture string
	LightLevel     float64
	Vertices       []geometry.XY
	Walls          []LevWall
}

// LevWall represents a wall in a sector, defined by its vertex index, adjacency, and associated textures.
type LevWall struct {
	VertexIndex int
	Adjoin      int
	MidTexture  string
	TopTexture  string
	BotTexture  string
}

// Builder is a type for constructing configuration objects by interpreting level ASTs with a specified scale factor.
type Builder struct {
	scaleFactor float64
}

// NewJediBuilder initializes a Builder with a given scale factor for processing level configurations.
func NewJediBuilder(scale float64) *Builder {
	return &Builder{scaleFactor: scale}
}

// Build converte l'AST del livello e l'AST degli oggetti nel config.Root universale
func (b *Builder) Build(file string, levelNumber int) (*config.Root, error) {

	Decompress(file)

	//TODO READ
	var levAst *LevAST
	var objAst *ObjAST

	configSectors := make([]*config.Sector, 0, len(levAst.Sectors))
	totalVertices := 0
	for _, sec := range levAst.Sectors {
		totalVertices += len(sec.Walls)
	}
	globalVertices := make(geometry.Polygon, 0, totalVertices)

	// 1. INGESTIONE GEOMETRICA (.LEV)
	for _, levSec := range levAst.Sectors {
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
					if wall.MidTexture != "" {
						cSeg.Middle = config.NewConfigAnimation([]string{wall.MidTexture}, config.AnimationKindLoop, 1.0, 1.0)
					}
				} else {
					cSeg.Kind = config.SegmentUnknown
					adjSec := levAst.Sectors[wall.Adjoin]
					if levSec.CeilingY > adjSec.CeilingY && wall.TopTexture != "" {
						cSeg.Upper = config.NewConfigAnimation([]string{wall.TopTexture}, config.AnimationKindLoop, 1.0, 1.0)
					}
					if levSec.FloorY < adjSec.FloorY && wall.BotTexture != "" {
						cSeg.Lower = config.NewConfigAnimation([]string{wall.BotTexture}, config.AnimationKindLoop, 1.0, 1.0)
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
			// Mapping coordinate: X, Z planari, Y altitudine (scalate)
			pos := geometry.XYZ{X: obj.X, Y: -obj.Z, Z: obj.Y}
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
					//Sprite:   obj.Data, // Referenza al file WAX o FME da pre-caricare
					// Kind: config.ThingTypeEnemy / Item etc. verrebbe risolto qui tramite una mappa o un dizionario esterno
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
