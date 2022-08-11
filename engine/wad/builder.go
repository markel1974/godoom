package wad

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/wad/lumps"
	"math"
	"os"
	"sort"
	"strconv"
)

//http://www.gamers.org/dhs/helpdocs/dmsp1666.html
//http://doomwiki.org/
//https://github.com/penberg/godoom
const scaleFactor = 10.0








type Point struct {
	X int16
	Y int16
}

type XY struct {
	X float64
	Y float64
}

type Point3 struct {
	X float64
	Y float64
	Z float64
	U float64
	V float64
}

func MakePoint3F(x, y, z, u, v float64) Point3{
	return Point3{ X:x, Y:y, Z:z, U:u, V: v }
}

func MakePoint3(x, y, z, u, v int16) Point3{
	return MakePoint3F(float64(x), float64(y), float64(z), float64(u), float64(v))
}


type Builder struct {
	w   *WAD
	cfg map[int16]*model.InputSector
}

func NewBuilder() * Builder {
	return &Builder{
		cfg : make(map[int16]*model.InputSector),
		//cfg : &config.Config{Sectors: nil, Player: &config.Player{}},
	}
}

func (b * Builder) Setup(wadFile string, levelNumber int) (*model.Input, error) {
	b.w = New()
	if err := b.w.Load(wadFile); err != nil {
		return nil, err
	}
	levelNames := b.w.GetLevels()
	if len(levelNames) == 0 {
		return nil,errors.New("error: No levels found")
	}
	levelIdx := levelNumber - 1
	if levelIdx >= len(levelNames) {
		return nil, errors.New(fmt.Sprintf("error: No such level number %d", levelNumber))
	}
	levelName := levelNames[levelIdx]
	fmt.Printf("Loading level %s ...\n", levelName)
	level, err := b.w.GetLevel(levelName)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	b.createSubSector(level)

	for _, c := range b.cfg {
		for idx := 0; idx < len(c.Neighbors); idx++ {
			curr := c.Neighbors[idx]
			var next *model.InputNeighbor
			if idx < len(c.Neighbors)-1 {
				next = c.Neighbors[idx+1]
			} else {
				next = c.Neighbors[0]
			}

			if curr.Neighbor != "wall" {
				id, _ := strconv.Atoi(c.Id)
				opposite := b.getOppositeSubSectorByLine(level, int16(id), int16(curr.X), int16(curr.Y), int16(next.X), int16(next.Y))
				curr.Neighbor = opposite
				//fmt.Println(opposite)
			}
		}
	}

	//os.Exit(-1)

	var sectors []*model.InputSector
	for _, c := range b.cfg {
		for idx := 0; idx < len(c.Neighbors); idx++ {
			curr := c.Neighbors[idx]
			curr.X = math.Abs(curr.X)
			curr.Y = math.Abs(curr.Y)
		}
		sectors = append(sectors, c)
	}

	sort.SliceStable(sectors, func(i, j int) bool {
		a, _ := strconv.Atoi(sectors[i].Id)
		b, _ := strconv.Atoi(sectors[j].Id)
		return a < b
	})

	p1 := level.Things[1]
	position := model.XY{
		X : float64(p1.XPosition),
		Y : float64(p1.YPosition),
	}

	playerSectorId, playerSSectorId, playerSector := level.FindSector(p1.XPosition, p1.YPosition)
	//TEST
	//playerSSectorId = 44
	//position.X = 1520 + 5
	//position.Y = -3168 + 5
	position.X = math.Abs(position.X)
	position.Y = math.Abs(position.Y)

	out, _ := json.Marshal(b.cfg[int16(playerSSectorId)])
	//out, _ := json.Marshal(b.cfg[1])
	fmt.Println(string(out))

	fmt.Println(playerSector, playerSectorId, playerSSectorId)

	cfg := &model.Input{ScaleFactor: scaleFactor, Sectors: sectors, Player: &model.InputPlayer{ Position: position, Angle: float64(p1.Angle), Sector: strconv.Itoa(int(playerSSectorId)) }}

	return cfg, nil
}

func (b * Builder) createSubSector(level *Level) {
	for subSectorId := int16(0); subSectorId < int16(len(level.SubSectors)); subSectorId ++ {
		subSector := level.SubSectors[subSectorId]
		segment := level.Segments[subSector.StartSeg]
		lineDef := level.LineDefs[int(segment.LineNum)]
		_, sideDef := level.SegmentSideDef(segment, lineDef)
		if sideDef == nil { continue }
		sectorId := sideDef.SectorRef
		sector := level.Sectors[sectorId]

		endSegmentId := subSector.StartSeg + subSector.NumSegments
		for segmentId := subSector.StartSeg; segmentId < endSegmentId; segmentId++ {
			segment := level.Segments[segmentId]
			lineDef := level.LineDefs[int(segment.LineNum)]
			_, sideDef := level.SegmentSideDef(segment, lineDef)
			if sideDef == nil { continue }

			start := level.Vertexes[segment.VertexStart]
			end := level.Vertexes[segment.VertexEnd]

			neighborStart := &model.InputNeighbor{ Tag: "", Neighbor: "", XY: model.XY{X: float64(start.XCoord), Y: float64(start.YCoord)}}
			neighborEnd := &model.InputNeighbor{ Tag: "", Neighbor: "", XY: model.XY{X: float64(end.XCoord), Y: float64(end.YCoord)}}

			//transparent := sideDef.LowerTexture == "-" && sideDef.MiddleTexture == "-" && sideDef.UpperTexture == "-"
			wall := sideDef.LowerTexture == "-" && sideDef.MiddleTexture != "-" && sideDef.UpperTexture == "-"
			current := b.getConfigSector(sectorId, sector, subSectorId, lineDef)
			add := true
			if len(current.Neighbors) > 0 {
				last := current.Neighbors[len(current.Neighbors) - 1]
				if last.X == neighborStart.X && last.Y == neighborStart.Y {
					neighborStart = last
					add = false
				} else {
					// la definizione last - neighborStart
					// viene realizzata in fase di Setup
				}
			}
			if wall {
				neighborStart.Neighbor = "wall"
			}
			tag := "--" + neighborStart.Neighbor + "(" + lineDef.PrintBits() + ")" + "[" + sideDef.LowerTexture + sideDef.MiddleTexture + sideDef.UpperTexture + "]"
			neighborStart.Tag = tag
			if add {
				current.Neighbors = append(current.Neighbors, neighborStart)
			}
			current.Neighbors = append(current.Neighbors, neighborEnd)
		}
	}

	//os.Exit(-1)
}


func (b * Builder) getOppositeSubSectorByLine(level * Level, subSectorId int16, x1 int16, y1 int16, x2 int16, y2 int16) string {
	alpha, beta := level.FindSubSectorByLine(int(x1), int(y1), int(x2), int(y2))
	best := alpha; if best == -1 { best = beta }
	out := int16(-2)
	if alpha == subSectorId {
		out = beta
	} else if beta == subSectorId {
		out = alpha
	}
	if out == -1 {
		out = best
	}
	switch out {
	case -2: return "unknown"
	case -1: return "wall"
	default: return strconv.Itoa(int(out))
	}
}

/*
func (b * Builder) getOppositeSubSectorByLine(level * Level, subSectorId int16, x1 int16, y1 int16, x2 int16, y2 int16) string {
	alpha, beta := level.FindSubSectorByLine(int(x1), int(y1), int(x2), int(y2))
	best := alpha; if best == -1 { best = beta }
	out := int16(-2)
	if alpha == subSectorId {
		out = beta
	} else if beta == subSectorId {
		out = alpha
	}
	if out == -1 {
		out = best
	}
	switch out {
	case -2: return "unknown"
	case -1: return "wall"
	default: return strconv.Itoa(int(out))
	}
}

 */

func (b * Builder) getConfigSector(sectorId int16, sector *lumps.Sector, subSectorId int16, ld *lumps.LineDef) * model.InputSector{
	c, ok := b.cfg[subSectorId]
	if !ok {
		c = &model.InputSector{
			Id:           strconv.Itoa(int(subSectorId)),
			Ceil:         float64(sector.CeilingHeight) / 5,
			Floor:        float64(sector.FloorHeight) / 5,
			Textures:     true,
			WallTexture:  "wall2.ppm",
			LowerTexture: "wall.ppm",
			UpperTexture: "wall3.ppm",
			FloorTexture: "floor.ppm",
			CeilTexture:  "ceil.ppm",
			Neighbors:    nil,
			Tag:          strconv.Itoa(int(sectorId)),
		}
		b.cfg[subSectorId] = c
	}
	return c
}