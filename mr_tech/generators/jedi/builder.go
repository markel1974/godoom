package jedi

import (
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

type LevAST struct {
	Sectors []LevSector
}

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

type LevWall struct {
	VertexIndex int
	Adjoin      int
	MidTexture  string
	TopTexture  string
	BotTexture  string
}

type Builder struct {
	scaleFactor float64
}

func NewJediBuilder(scale float64) *Builder {
	return &Builder{scaleFactor: scale}
}

func (b *Builder) Build(ast *LevAST) (*config.Root, error) {
	configSectors := make([]*config.Sector, 0, len(ast.Sectors))

	totalVertices := 0
	for _, sec := range ast.Sectors {
		totalVertices += len(sec.Walls)
	}
	globalVertices := make(geometry.Polygon, 0, totalVertices)

	for _, levSec := range ast.Sectors {
		if levSec.Id < 0 {
			continue // Evita instanziazione di ghost sectors derivati dal parser
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
		if wallCount == 0 {
			continue
		}

		cSector.Segments = make([]*config.Segment, 0, wallCount)

		for i, wall := range levSec.Walls {
			v1 := levSec.Vertices[wall.VertexIndex]
			nextWall := levSec.Walls[(i+1)%wallCount]
			v2 := levSec.Vertices[nextWall.VertexIndex]

			globalVertices = append(globalVertices, v1)

			cSeg := config.NewConfigSegment(secId, config.SegmentUnknown, v1, v2)
			cSeg.Start.Y, cSeg.End.Y = -cSeg.Start.Y, -cSeg.End.Y

			if wall.Adjoin == -1 {
				cSeg.Kind = config.SegmentWall
				if wall.MidTexture != "" {
					cSeg.Middle = config.NewConfigAnimation([]string{wall.MidTexture}, config.AnimationKindLoop, 1.0, 1.0)
				}
			} else {
				cSeg.Kind = config.SegmentUnknown
				adjSec := ast.Sectors[wall.Adjoin]

				if levSec.CeilingY > adjSec.CeilingY && wall.TopTexture != "" {
					cSeg.Upper = config.NewConfigAnimation([]string{wall.TopTexture}, config.AnimationKindLoop, 1.0, 1.0)
				}
				if levSec.FloorY < adjSec.FloorY && wall.BotTexture != "" {
					cSeg.Lower = config.NewConfigAnimation([]string{wall.BotTexture}, config.AnimationKindLoop, 1.0, 1.0)
				}
			}
			cSector.Segments = append(cSector.Segments, cSeg)
		}
		configSectors = append(configSectors, cSector)
	}

	cr := config.NewConfigRoot(nil, configSectors, nil, nil, b.scaleFactor, nil)
	cr.Vertices = globalVertices

	return cr, nil
}
