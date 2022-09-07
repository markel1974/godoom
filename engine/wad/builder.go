package wad

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/wad/lumps"
	"math"
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

			//if lineDef.HasFlag(lumps.TwoSided) {
				//if lineDef.HasFlag(lumps.Impassible) {
				//	continue
				//}
				//if lineDef.HasFlag(lumps.BlockMonsters) {
				//	continue
				//}
				//if lineDef.HasFlag(lumps.AlreadyOnMap) {
				//	continue
				//}
				//if lineDef.HasFlag(lumps.NotOnMap) {
				//	continue
				//}
			//}

			lower := sideDef.LowerTexture
			middle := sideDef.MiddleTexture
			upper := sideDef.UpperTexture

			if lower != "-" { b.textures[lower] = true }
			if middle != "-" { b.textures[middle] = true }
			if upper != "-" { b.textures[upper] = true }

			startXY := model.XY{X: float64(start.XCoord), Y: float64(-start.YCoord)}
			endXY := model.XY{X: float64(end.XCoord), Y: float64(-end.YCoord)}

			modelSegment := model.NewInputSegment(model.DefinitionVoid, startXY, endXY)
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

	//fmt.Println("-------- ACQUIRED -----------")
	//b.printSegments(miSectors[15].Segments)

	b.compileConvexHull(miSectors)

	b.compileSegmentRelations(miSectors)

	b.CompileNeighbors(miSectors)

	//for idx, _ := range miSectors {
	//	b.describeSegment(idx, miSectors)
	//}
	//b.describeSegment(38, miSectors)
	//b.describeSegment(103, miSectors)
	b.describeSegment(103, miSectors)
	b.describeSegment(96, miSectors)
	//b.describeSegment(103, miSectors)
	//b.describeSegment(2, miSectors)
	//b.describeSegment(8, miSectors)

	//os.Exit(-1)

	return miSectors
}

func (b * Builder) CompileNeighbors(miSectors []*model.InputSector)  {
	wallSectors := make(map[uint16]bool)

	for idx, miSector := range miSectors {
		if len(miSector.Segments) == 1 && miSector.Segments[0].Kind == model.DefinitionWall {
			wallSectors[uint16(idx)] = true
		}
	}

	for _, miSector := range miSectors {
		var segments []*model.InputSegment
		for _, s := range miSector.Segments {
			//if s.Kind == model.DefinitionWall || s.Kind == model.DefinitionValid {
			if s.Kind == model.DefinitionWall {
				segments = append(segments, s)
				continue
			}
			//duplicates := map[string]bool{}
			id, _ := strconv.Atoi(miSector.Id)
			res := b.bsp.FindOppositeSubSectorByPoints(uint16(id), s, wallSectors)

			/*
			for _, d := range res {
				if _, ok := duplicates[d.Neighbor]; ok {
					d.Kind = model.DefinitionUnknown
					d.Neighbor = ""
					fmt.Println(idx, s.Id, "DUPLICATES ARE NOT ALLOWED!!!!")
					continue
				}
				duplicates[d.Neighbor] = true
			}
			*/

			switch len(res) {
				case 0:	segments = append(segments, s)
				case 1: segments = append(segments, res[0])
				default: segments = append(segments, res...)
			}
		}

		/*
		if idx == 15 {
			fmt.Println("----------------- BEFORE ------------------")
			b.printSegments(miSectors[15].Segments)
			fmt.Println("----------------- AFTER ------------------")
			b.printSegments(segments)
			//os.Exit(1)
		}
		*/
		miSector.Segments = segments
	}
}


func (b * Builder) compileConvexHull(miSectors []*model.InputSector) {
	ch := model.NewConvexHull()
	for _, miSector := range miSectors {
		if len(miSector.Segments) <= 1 {
			continue
		}
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
				ns := model.NewInputSegment(model.DefinitionVoid, s.Start, s.End)
				ns.Tag = "missing"
				miSector.Segments = append(miSector.Segments, ns)
			}
		}
	}
}


func pointIsOnSegment(px float64, py float64, pz float64, x1 float64, y1 float64, z1 float64, x2 float64, y2 float64, z2 float64) bool {
	ab := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1) + (z2-z1)*(z2-z1))
	ap := math.Sqrt((px-x1)*(px-x1) + (py-y1)*(py-y1) + (pz-z1)*(pz-z1))
	pb := math.Sqrt((x2-px)*(x2-px) + (y2-py)*(y2-py) + (z2-pz)*(z2-pz))
	if ab == ap + pb {
		return true
	}
	return false
}

func (b * Builder) pointOnSegment(point model.XY, s1 model.XY, s2 model.XY) bool {
	return pointIsOnSegment(point.X, point.Y, 1.0, s1.X, s1.Y, 1.0, s2.X, s2.Y, 1.0)
}

func (b * Builder) compileSegmentRelations(miSectors []*model.InputSector) {
	type relation struct { to string; from string; s *model.InputSegment }
	relations := map[string][]*relation{}

	addRelation := func(to string, from string, s *model.InputSegment) {
		rel := &relation{ to: to, from: from, s: s }
		if k, ok := relations[to]; !ok {
			relations[to]=[]*relation{rel}
		} else {
			k = append(k, rel)
		}
	}

	for _, xSector := range miSectors {
		for _, xSegment := range xSector.Segments {
			if xSegment.Kind == model.DefinitionWall { continue	}
			partial := 0
			relation := 2
			for _, ySector := range miSectors {
				if ySector.Id == xSector.Id { continue }
				for _, ySegment := range ySector.Segments {
					relation = 0
					if  xSegment.Start.X == ySegment.End.X   && xSegment.Start.Y == ySegment.End.Y &&
						xSegment.End.X   == ySegment.Start.X && xSegment.End.Y   == ySegment.Start.Y {
						relation = 2
					}
					if relation < 2 {
						pStart := b.pointOnSegment(xSegment.Start, ySegment.End, ySegment.Start)
						pEnd := b.pointOnSegment(xSegment.End, ySegment.End, ySegment.Start)
						if pStart && pEnd {
							relation = 2
						} else if pStart || pEnd {
							relation = 1
							partial++
						}
					}
					if relation > 0 {
						addRelation(xSegment.Id, ySector.Id, ySegment)
						if relation == 2 {
							xSegment.Neighbor = ySector.Id
							xSegment.Kind = model.DefinitionValid
							partial = 0
							break
						}
					}
				}
				if relation == 2 { break }
			}
			/*
			if relation == 2 {
				totalFound++
			} else {
				test1 := make(map[uint16]bool)
				id, _ := strconv.Atoi(xSector.Id)
				b.bsp.findPointInSubSector(int16(xSegment.Start.X), int16(-xSegment.Start.Y), uint16(id), test1)
				b.bsp.findPointInSubSector(int16(xSegment.End.X), int16(-xSegment.End.Y), uint16(id), test1)

				if partial > 0 {
					totalPartial ++
				} else {
					totalNotFound ++
				}
				notFound[xSegment.Id] = xSegment
			}
			*/
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

func (b * Builder) describeSegment(targetSector int, miSectors []*model.InputSector) {
	fmt.Println("------------------", "DESCRIBE SECTOR", targetSector, "------------------")
	xy := miSectors[targetSector].Segments[0]
	var neighbors [] string
	//nodeIdx := b.bsp.findNodeSubSector(uint16(targetSector))

	for idx, tt := range miSectors[targetSector].Segments {
		neighbors = append(neighbors, tt.Neighbor)
		fmt.Println("SEGMENT:", idx)
		fmt.Println(idx, tt.Neighbor, "COORDS:", tt.Start.X, tt.Start.Y, tt.End.X, tt.End.Y, "TAG:", tt.Tag)
	}

	nodeIdx, _ := b.bsp.FindNode(int16(xy.Start.X), int16(-xy.Start.Y))

	var traverse[]uint16

	//b.bsp.describeLine2F()
	b.bsp.TraverseBsp(&traverse, int16(xy.Start.X), int16(-xy.Start.Y), nodeIdx)
	fmt.Println("NEIGHBORS:", neighbors)
	fmt.Println("TRAVERSE:", traverse)
}