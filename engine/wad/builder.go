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

	var sectors []*model.InputSector
	for _, c := range b.cfg {
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
	playerSSectorId = 44
	position.X = 1520 + 5
	position.Y = -3168 + 5

	out, _ := json.Marshal(b.cfg[int16(playerSSectorId)])
	//out, _ := json.Marshal(b.cfg[1])
	fmt.Println(string(out))

	fmt.Println(playerSector, playerSectorId, playerSSectorId)

	cfg := &model.Input{ScaleFactor: scaleFactor, Sectors: sectors, Player: &model.InputPlayer{ Position: position, Angle: float64(p1.Angle), Sector: strconv.Itoa(int(playerSSectorId)) }}

	return cfg, nil
}


//level--> SECTOR
//SECTOR --> SUBSECTOR
//....
/*
SSECTOR stands for sub-sector. These divide up all the SECTORS into
convex polygons. They are then referenced through the NODES resources.
There will be (number of nodes + 1) ssectors.
Each ssector is 4 bytes in 2 <short> fields:

(1) This many SEGS are in this SSECTOR...
(2) ...starting with this SEG number

The segs in ssector 0 should be segs 0 through x, then ssector 1
contains segs x+1 through y, ssector 2 containg segs y+1 to z, etc.
*/

type SContainer struct {
	SegmentId      int16
	LineNum        int16
	Line           *lumps.LineDef
	SubSectors     int16
}
type LineContainer struct {
	Container     []*SContainer
}

func (b * Builder) createSectorCache (level *Level) map[int16] *LineContainer {
	var t = make(map[int16] *LineContainer)

	for subSectorId := int16(0); subSectorId < int16(len(level.SubSectors)); subSectorId ++ {
		subSector := level.SubSectors[subSectorId]
		for segmentId := subSector.StartSeg; segmentId < subSector.StartSeg + subSector.NumSegments; segmentId++ {
			segment := level.Segments[segmentId]
			if segment == nil { continue }
			lineDef := level.LineDefs[int(segment.LineNum)]
			if lineDef == nil { continue}
			//(7) "left" SIDEDEF, if this line adjoins 2 SECTORS. Otherwise, it is
			//equal to -1 (FFFF hex).

			//if !lineDef.HasFlag(lumps.TwoSided) { continue }
			var sectorId int16
			if _, sideDef := level.SegmentSideDef(segment, lineDef); sideDef != nil {
				sectorId = sideDef.SectorRef
			} else {
				continue
			}
			//fmt.Println("R:", lineDef.SideDefLeft, lineDef.SideDefRight)
			sc := &SContainer{SegmentId: segmentId, LineNum: segment.LineNum, Line: lineDef, SubSectors: subSectorId }
			if z, ok := t[sectorId]; ok {
				z.Container = append(z.Container, sc)
			} else {
				t[sectorId] = &LineContainer{ Container: []*SContainer{sc}}
			}
		}
	}
	return t
}

func (b * Builder) createSubSector(level *Level) {
	t := b.createSectorCache(level)

	same := 0
	different := 0
	unknown := 0
	unmatch := 0
	total := 0

	for subSectorId := int16(0); subSectorId < int16(len(level.SubSectors)); subSectorId ++ {
		subSector := level.SubSectors[subSectorId]
		if subSector == nil { continue }
		segment := level.Segments[subSector.StartSeg]
		if segment == nil { continue }
		lineDef := level.LineDefs[int(segment.LineNum)]
		if lineDef == nil { continue}
		_, sideDef := level.SegmentSideDef(segment, lineDef)
		if sideDef == nil { continue }
		sectorId := sideDef.SectorRef
		sector := level.Sectors[sectorId]
		if sector == nil { continue }

		endSegmentId := subSector.StartSeg + subSector.NumSegments
		for segmentId := subSector.StartSeg; segmentId < endSegmentId; segmentId++ {
			segment := level.Segments[segmentId]
			lineDef := level.LineDefs[int(segment.LineNum)]
			_, sideDef := level.SegmentSideDef(segment, lineDef)
			if sideDef == nil { continue }

			current := b.getConfigSector(sectorId, sector, subSectorId, lineDef)
			start := level.Vertexes[segment.VertexStart]
			end := level.Vertexes[segment.VertexEnd]

			neighborId := "wall"

			/*
			if lineDef.HasFlag(lumps.TwoSided) {
				sAId, ssAId, sBId, ssBId, _ := level.FindSubSectorByLine(int(start.XCoord), int(start.YCoord), int(end.XCoord), int(end.YCoord))
				fmt.Println()
				fmt.Println("---------------", sectorId, subSectorId)

				if ssAId == subSectorId {
					neighborId = strconv.Itoa(int(ssBId))
					fmt.Println("Target", sAId, "B", sBId)
					fmt.Println("SSA:", ssAId, "SSB:", ssBId)
				} else if ssBId == subSectorId {
					neighborId = strconv.Itoa(int(ssAId))
					fmt.Println("A", sAId, "B", sBId)
					fmt.Println("SSA:", ssAId, "SSB:", ssBId)
				} else {
					fmt.Println("CAN'T FIND")
				}
			}
			*/


			if !lineDef.HasFlag(lumps.TwoSided)  {
				neighborId = "wall"
				//fmt.Println("------------")
				//fmt.Println(lineDef.PrintBits())
				//fmt.Println(sideDef.LowerTexture, sideDef.MiddleTexture, sideDef.UpperTexture)
			} else {
				total ++
				count := 0
				_, oppositeSideDef := level.SegmentOppositeSideDef(segment, lineDef)
				if oppositeSideDef != nil {
					if ld, ok := t[oppositeSideDef.SectorRef]; ok {
						for _, z := range ld.Container {
							if z.LineNum == segment.LineNum {
								neighborId = strconv.Itoa(int(z.SubSectors))
								count++
							}
						}
					}
				}

				if count != 1 {
					neighborId = "wall"
					ssAId, ssBId, ok := level.FindSubSectorByLine(int(start.XCoord), int(start.YCoord), int(end.XCoord), int(end.YCoord))
					if ok {
						if ssAId == subSectorId {
							if count == 1 && neighborId != strconv.Itoa(int(ssBId)) {
								different++
							} else {
								same++
							}
							neighborId = strconv.Itoa(int(ssBId))
						} else if ssBId == subSectorId {
							if count == 1 && neighborId != strconv.Itoa(int(ssAId)) {
								different++
							} else {
								same++
							}
							neighborId = strconv.Itoa(int(ssAId))
						} else {
							unmatch++
							//fmt.Println("CAN'T FIND")
						}
					} else {
						unknown++
					}
				}
			}

			neighborStart := &model.InputNeighbor{Id: neighborId, XY: model.XY{X: math.Abs(float64(start.XCoord)), Y: math.Abs(float64(start.YCoord))}}
			neighborEnd := &model.InputNeighbor{Id: neighborId, XY: model.XY{X: math.Abs(float64(end.XCoord)), Y: math.Abs(float64(end.YCoord))}}
			current.Neighbors = append(current.Neighbors, neighborStart)
			current.Neighbors = append(current.Neighbors, neighborEnd)
		}
	}

	fmt.Println("TOTAL:", total)
	fmt.Println("SAME:", same)
	fmt.Println("DIFFERENT:", different)
	fmt.Println("UNMATCH:", unmatch)
	fmt.Println("UNKNOWN:", unknown)
	//os.Exit(-1)
}

func (b * Builder) getConfigSector(sectorId int16, sector *lumps.Sector, subSectorId int16, ld *lumps.LineDef) * model.InputSector{
	c, ok := b.cfg[subSectorId]
	if !ok {
		c = &model.InputSector{
			Id:           strconv.Itoa(int(subSectorId)),
			Ceil:         float64(sector.CeilingHeight) / 3,
			Floor:        float64(sector.FloorHeight) / 3,
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




/*
func (b * Builder) createSegment(level *Level, segment * Seg, current * model.InputSector) {
	//func (b * Builder) createSegment(level *Level, subSectorId int16, segmentId int16) {
	//In general, being a convex polygon is the goal of a ssector.
	//Convex  means a line connecting any two points that are inside the polygon will be completely contained in the polygon.
	//segment := level.Segments[segmentId]

	//lineDef := level.LineDefs[int(segment.LineNum)]
	//_, sideDef := b.segmentSideDef(level, segment, lineDef)
	//if sideDef == nil { return }

	start := level.Vertexes[segment.VertexStart]
	end := level.Vertexes[segment.VertexEnd]

	//upperTexture := sideDef.UpperTexture
	//middleTexture := sideDef.MiddleTexture
	//lowerTexture := sideDef.LowerTexture

	//_, oppositeSideDef := b.segmentOppositeSideDef(level, segment, lineDef)
	//sector := level.Sectors[sideDef.SectorRef]

	neighborId := "wall"
	//if oppositeSideDef != nil {
		//z := b.findSubSector(level, start.XCoord, start.YCoord, len(level.Nodes)-1)
		//fmt.Println("RESULT", sideDef.SectorRef, z, oppositeSideDef.SectorRef)
		//Il neighborId deve essere necessariamente il subsector!
		//neighborId = strconv.Itoa(int(oppositeSideDef.SectorRef))
	//}

	//current := b.getConfigSector(subSectorId, sector)
	//neighborId = strconv.Itoa(int(oppositeSideDef.SectorRef))
	neighborStart := &model.InputNeighbor{Id: neighborId, XY: model.XY{X: float64(start.XCoord) / scaleFactor, Y: -float64(start.YCoord) / scaleFactor}}
	neighborEnd := &model.InputNeighbor{Id: neighborId, XY: model.XY{X: float64(end.XCoord) / scaleFactor, Y: -float64(end.YCoord) / scaleFactor}}

	addStart := true


	//if len(current.Neighbors) > 0 {
	//	last := current.Neighbors[len(current.Neighbors) - 1]
	//	if last.X == neighborStart.X && last.Y == neighborStart.Y {
	//		addStart = false
	//		fmt.Println("Start already added")
	//	}}
	if addStart {
		current.Neighbors = append(current.Neighbors, neighborStart)
	}
	current.Neighbors = append(current.Neighbors, neighborEnd)
}

*/