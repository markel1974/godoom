package wad

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/wad/lumps"
	"os"
	"sort"
	"strconv"
)

//http://www.gamers.org/dhs/helpdocs/dmsp1666.html
//http://doomwiki.org/
//https://github.com/penberg/godoom
//https://github.com/mausimus/rtdoom/blob/master/rtdoom/Projection.cpp

//https://github.com/gamescomputersplay/wad2pic/blob/main/wad2pic.py
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
	w        *WAD
	cfg      map[uint16]*model.InputSector
	textures map[string]bool
	level * Level
	bsp * BSP
}

func NewBuilder() * Builder {
	return &Builder{
		cfg:      nil,
		textures: nil,
		level:    nil,
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
	var err error
	b.level, err = b.w.GetLevel(levelName)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	b.bsp = NewBsp(b.level)

	b.scanSubSectors()

	for textureId := range b.textures {
		if texture, err := b.w.GetTextureImage(textureId); err != nil {
			fmt.Println(textureId, err.Error())
		} else {
			fmt.Println(textureId, texture.Rect)
		}
	}

	//stubs := NewFooBsp()
	//stubs.Verify(b.level, b.bsp)
	//os.Exit(-1)
	//stubs.Print()
	//os.Exit(-1)

	var sectors []*model.InputSector
	for _, c := range b.cfg {
		sectors = append(sectors, c)
	}

	sort.SliceStable(sectors, func(i, j int) bool {
		a, _ := strconv.Atoi(sectors[i].Id)
		b, _ := strconv.Atoi(sectors[j].Id)
		return a < b
	})

	for _, s := range sectors {
		for _, n := range s.Segments {
			n.Start.Y = -n.Start.Y
			n.End.Y = -n.End.Y
		}
	}

	p1 := b.level.Things[1]
	position := model.XY{
		X : float64(p1.X),
		Y : float64(p1.Y),
	}

	playerSectorId, playerSSectorId, playerSector := b.bsp.FindSector(p1.X, p1.Y)
	//TEST
	//playerSSectorId = 44
	//position.X = 1520 + 5
	//position.Y = -3168 + 5
	position.Y = -position.Y

	out, _ := json.Marshal(b.cfg[playerSSectorId])
	//out, _ := json.Marshal(b.cfg[1])
	fmt.Println(string(out))

	fmt.Println(playerSector, playerSectorId, playerSSectorId)

	cfg := &model.Input{DisableFix : false, ScaleFactor: scaleFactor, Sectors: sectors, Player: &model.InputPlayer{ Position: position, Angle: float64(p1.Angle), Sector: strconv.Itoa(int(playerSSectorId)) }}

	return cfg, nil
}




func (b * Builder) scanSubSectors() {
	b.cfg = make(map[uint16]*model.InputSector)
	b.textures = make(map[string]bool)

	for subSectorId := uint16(0); int(subSectorId) < len(b.level.SubSectors); subSectorId ++ {
		subSector := b.level.SubSectors[subSectorId]
		segment := b.level.Segments[subSector.StartSeg]
		lineDef := b.level.LineDefs[int(segment.LineDef)]
		_, sideDef := b.level.SegmentSideDef(segment, lineDef)
		if sideDef == nil { continue }
		sectorId := sideDef.SectorRef
		sector := b.level.Sectors[sectorId]

		endSegmentId := subSector.StartSeg + subSector.NumSegments

		for segmentId := subSector.StartSeg; segmentId < endSegmentId; segmentId++ {
			segment := b.level.Segments[segmentId]

			lineDef := b.level.LineDefs[int(segment.LineDef)]
			_, sideDef := b.level.SegmentSideDef(segment, lineDef)
			if sideDef == nil { continue }
			start := b.level.Vertexes[segment.VertexStart]
			end := b.level.Vertexes[segment.VertexEnd]

			/*
			if subSectorId == 117 {
				fmt.Println("------------", subSectorId)
				fmt.Println("segmentId", segmentId, segmentId - subSector.StartSeg)
				fmt.Println("Segment Offset:", segment.Offset)
				fmt.Println("Segment Angle:", segment.BAM)
				fmt.Println("Start:", start)
				fmt.Println("End:", end)
				fmt.Println(lineDef.Flags, "->", lineDef.PrintBits())
				fmt.Println(sideDef.PrintTexture())
				//fmt.Println(start.XCoord, ",", -start.YCoord)
			}
			*/

			lower := sideDef.LowerTexture
			middle := sideDef.MiddleTexture
			upper := sideDef.UpperTexture

			if lower != "-" { b.textures[lower] = true }
			if middle != "-" { b.textures[middle] = true }
			if upper != "-" { b.textures[upper] = true }

			startXY := model.XY{X: float64(start.XCoord), Y: float64(start.YCoord)}
			endXY := model.XY{X: float64(end.XCoord), Y: float64(end.YCoord)}

			modelSegment := &model.InputSegment{ Tag: "", Neighbor: "", Start: startXY, End: endXY}

			wall := false
			if !lineDef.HasFlag(lumps.TwoSided) {
				wall = middle != "-"
			}

			if wall {
				modelSegment.Kind = model.DefinitionWall
				modelSegment.Neighbor = "wall"
			} else {
				b.SetNeighbor(subSectorId, modelSegment)
			}

			tag := "Id: " + modelSegment.Neighbor + " (" + lineDef.PrintBits() + " | "
			if wall { tag += "wall" } else { tag += sideDef.PrintTexture() }
			tag += ")"
			modelSegment.Tag = tag
			modelSegment.Upper = upper
			modelSegment.Middle = middle
			modelSegment.Lower = lower
			current := b.getConfigSector(sectorId, sector, subSectorId, lineDef)

			if len(current.Segments) > 0 {
				prev := current.Segments[len(current.Segments) - 1]
				if prev.End.X != modelSegment.Start.X && prev.End.Y != modelSegment.End.Y {
					missingSegment := &model.InputSegment{ Tag: "Missing", Neighbor: prev.Neighbor, Start: prev.End, End: modelSegment.Start}
					b.SetNeighbor(subSectorId, missingSegment)
					//missingSegment.Kind = model.DefinitionVoid
					current.Segments = append(current.Segments, missingSegment)
				}
			}

			current.Segments = append(current.Segments, modelSegment)
		}
	}

	//os.Exit(-1)
}

func (b * Builder) getConfigSector(sectorId uint16, sector *lumps.Sector, subSectorId uint16, ld *lumps.LineDef) * model.InputSector{
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
			Segments:     nil,
			Tag:          strconv.Itoa(int(sectorId)),
		}
		b.cfg[subSectorId] = c
	}
	return c
}

func (b * Builder) SetNeighbor(subSectorId uint16, m *model.InputSegment)  {
	x1 := int16(m.Start.X)
	y1 := int16(m.Start.Y)
	x2 := int16(m.End.X)
	y2 := int16(m.End.Y)


	/*
	//m.Kind = model.DefinitionValid
	_, oppositeSubSector, state := b.bsp.FindOppositeSubSectorByLine(subSectorId, x1, y1, x2, y2)
	if state >= 0 {
		m.Kind = model.DefinitionValid
		m.Neighbor = strconv.Itoa(int(oppositeSubSector))
	} else if state == -2 {
		m.Kind = model.DefinitionVoid
		m.Neighbor = strconv.Itoa(int(subSectorId))
	} else if state == -1 {
		oppositeSubSector, state := b.bsp.FindOppositeSubSectorByPoints(subSectorId, x1, y1, x2, y2)
		if state >= 0 {
			m.Kind = model.DefinitionValid
			m.Neighbor = strconv.Itoa(int(oppositeSubSector))
		} else if state == -2 {
			m.Kind = model.DefinitionVoid
			m.Neighbor = strconv.Itoa(int(subSectorId))
		} else {
			//UNDEFINED!
		}
	}
	*/

	oppositeSubSector, state := b.bsp.FindOppositeSubSectorByPoints(subSectorId, x1, y1, x2, y2)
	if state >= 0 {
		m.Kind = model.DefinitionValid
		m.Neighbor = strconv.Itoa(int(oppositeSubSector))
	} else if state == -2 {
		//VOID
		m.Neighbor = strconv.Itoa(int(subSectorId))
		m.Kind = model.DefinitionVoid
	} else {
		m.Kind = model.DefinitionUnknown
		//m.Kind = model.DefinitionWall
		//m.Neighbor = "wall"
	}
}