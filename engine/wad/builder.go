package wad

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/wad/lumps"
	"os"
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
	textures map[string]bool
	level    *Level
	bsp      *BSP
}

func NewBuilder() * Builder {
	return &Builder{
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

	sectors := b.scanSubSectors()

	for textureId := range b.textures {
		if texture, err := b.w.GetTextureImage(textureId); err != nil {
			fmt.Println(textureId, err.Error())
		} else {
			fmt.Println(textureId, texture.Rect)
		}
	}

	p1 := b.level.Things[1]
	position := model.XY{ X : float64(p1.X), Y : float64(p1.Y) }

	playerSectorId, playerSSectorId, playerSector := b.bsp.FindSector(p1.X, p1.Y)
	//TEST
	//playerSSectorId = 44
	//position.X = 1520 + 5
	//position.Y = -3168 + 5
	position.Y = -position.Y

	out, _ := json.Marshal(sectors[playerSSectorId])
	//out, _ := json.Marshal(b.cfg[1])
	fmt.Println(string(out))

	fmt.Println(playerSector, playerSectorId, playerSSectorId)

	cfg := &model.Input{DisableLoop: true, ScaleFactor: scaleFactor, Sectors: sectors, Player: &model.InputPlayer{ Position: position, Angle: float64(p1.Angle), Sector: strconv.Itoa(int(playerSSectorId)) }}

	return cfg, nil
}

func (b * Builder) scanSubSectors() []*model.InputSector {
	miSectors := make([]*model.InputSector, len(b.level.SubSectors))
	b.textures = make(map[string]bool)

	for subSectorId := uint16(0); int(subSectorId) < len(b.level.SubSectors); subSectorId ++ {
		subSector := b.level.SubSectors[subSectorId]
		segment := b.level.Segments[subSector.StartSeg]
		lineDef := b.level.LineDefs[int(segment.LineDef)]
		_, sideDef := b.level.SegmentSideDef(segment, lineDef)
		if sideDef == nil { continue }
		sectorId := sideDef.SectorRef
		miSector := b.getConfigSector(miSectors, sectorId, b.level.Sectors[sectorId], subSectorId)

		for segmentId := subSector.StartSeg; segmentId < subSector.StartSeg + subSector.NumSegments; segmentId++ {
			segment := b.level.Segments[segmentId]
			lineDef := b.level.LineDefs[int(segment.LineDef)]
			_, sideDef := b.level.SegmentSideDef(segment, lineDef)
			if sideDef == nil { continue }
			start := b.level.Vertexes[segment.VertexStart]
			end := b.level.Vertexes[segment.VertexEnd]

			lower := sideDef.LowerTexture
			middle := sideDef.MiddleTexture
			upper := sideDef.UpperTexture

			if lower != "-" { b.textures[lower] = true }
			if middle != "-" { b.textures[middle] = true }
			if upper != "-" { b.textures[upper] = true }

			startXY := model.XY{X: float64(start.XCoord), Y: float64(-start.YCoord)}
			endXY := model.XY{X: float64(end.XCoord), Y: float64(-end.YCoord)}

			modelSegment := &model.InputSegment{ Tag: "", Neighbor: "", Start: startXY, End: endXY}

			wall := false
			if !lineDef.HasFlag(lumps.TwoSided) {
				wall = middle != "-"
			}
			if wall {
				modelSegment.Kind = model.DefinitionWall
				modelSegment.Neighbor = "wall"
			}
			tag := "Id: " + modelSegment.Neighbor + " (" + lineDef.PrintBits() + " | "
			if wall { tag += "wall" } else { tag += sideDef.PrintTexture() }
			tag += ")"
			modelSegment.Tag = tag
			modelSegment.Upper = upper
			modelSegment.Middle = middle
			modelSegment.Lower = lower
			miSector.Segments = append(miSector.Segments, modelSegment)
		}
	}

	fmt.Println("-------- ACQUIRED -----------")
	b.printSegments(miSectors[15].Segments)

	b.compileConvexHull(miSectors)

	b.CompileNeighbors(miSectors)

	return miSectors
}

func (b * Builder) CompileNeighbors(miSectors []*model.InputSector)  {
	for idx, miSector := range miSectors {
		var segments []*model.InputSegment
		for _, s := range miSector.Segments {
			if s.Kind == model.DefinitionWall {
				segments = append(segments, s)
				continue
			}
			id, _ := strconv.Atoi(miSector.Id)
			_, _, res := b.bsp.FindOppositeSubSectorByPoints(uint16(id), s)
			switch len(res) {
				case 0:	segments = append(segments, s)
				case 1: segments = append(segments, res[0])
				default: segments = append(segments, res...)
			}
		}

		if idx == 15 {
			fmt.Println("----------------- BEFORE ------------------")
			b.printSegments(miSectors[15].Segments)
			fmt.Println("----------------- AFTER ------------------")
			b.printSegments(segments)
			//os.Exit(1)
		}
		miSector.Segments = segments
	}

	/*
	for idx, miSector := range miSectors {
		for _, s := range miSector.Segments {
			if s.Kind != model.DefinitionWall {
				id, _ := strconv.Atoi(miSector.Id)
				oppositeSubSector, state, _ := b.bsp.FindOppositeSubSectorByPoints(uint16(id), s)
				if state >= 0 {
					s.Kind = model.DefinitionValid
					s.Neighbor = strconv.Itoa(int(oppositeSubSector))
				} else if state == -2 {
					//VOID
					s.Neighbor = strconv.Itoa(int(oppositeSubSector))
					s.Kind = model.DefinitionVoid
				} else {
					s.Kind = model.DefinitionUnknown
					//m.Kind = model.DefinitionWall
					//m.Neighbor = "wall"
				}
				//b.SetNeighbor(uint16(id), s)
			}
		}
		if idx == 15 {
			fmt.Println("--------------------------", "BEFORE", "--------------------------")
			for i, test := range miSectors[idx].Segments{
				fmt.Println(i, "[", test.Neighbor, "]", test.Start, test.End, test.Tag)
			}
			os.Exit(1)
		}
	}

	 */
}


func (b * Builder) compileConvexHull(miSectors []*model.InputSector) {
	ch := model.NewConvexHull()
	for _, miSector := range miSectors {
		var chs []*model.CHSegment
		for _, s := range miSector.Segments {
			c := model.NewCHSegment(miSector.Id, s, s.Start, s.End)
			chs = append(chs, c)
		}
		miSector.Segments = nil
		for _, s := range ch.Create(miSector.Id, chs) {
			if s.Data != nil {
				miSector.Segments = append(miSector.Segments, s.Data.(*model.InputSegment))
			} else {
				ns := &model.InputSegment{ Tag: "Missing", Neighbor: "", Start: s.Start, End: s.End, Kind: model.DefinitionVoid }
				miSector.Segments = append(miSector.Segments, ns)
			}
		}
	}
}

func (b * Builder) getConfigSector(cfg []*model.InputSector, sectorId uint16, sector *lumps.Sector, subSectorId uint16) * model.InputSector{
	c := cfg[subSectorId]
	if c == nil {
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
		cfg[subSectorId] = c
	}
	return c
}

func (b * Builder) printSegments(miSegments []*model.InputSegment) {
	for i, test := range miSegments{
		fmt.Println(i, "[", test.Neighbor, "]", test.Start.X, test.Start.Y, test.End.X, test.End.Y, test.Tag)
	}
}