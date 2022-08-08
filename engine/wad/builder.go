package wad

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/markel1974/godoom/engine/model"
	"image"
	"image/color"
	"os"
	"sort"
	"strconv"
)

//http://www.gamers.org/dhs/helpdocs/dmsp1666.html
//http://doomwiki.org/
//https://github.com/penberg/godoom
const scaleFactor = 10.0



/*
type lineDef struct { start *Vertex; end *Vertex; subSectorId string }
func lineDefHash(start *Vertex, end *Vertex) string {
	startX := strconv.Itoa(int(start.XCoord))
	startY := strconv.Itoa(int(start.YCoord))
	endX := strconv.Itoa(int(end.XCoord))
	endY := strconv.Itoa(int(end.YCoord))
	return startX + "|" + startY + "=>" + endX + "|" + endY
}*/



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

	playerSectorId, playerSSectorId, playerSector := b.findSector(level, p1.XPosition, p1.YPosition, len(level.Nodes)-1)
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
func (b * Builder) createSubSector(level *Level) {
	//CACHE
	/*
	t := make(map[string] lineDef)
	for subSectorId := int16(0); subSectorId < int16(len(level.SubSectors)); subSectorId ++ {
		subSector := level.SubSectors[subSectorId]
		for segmentId := subSector.StartSeg; segmentId < subSector.StartSeg + subSector.NumSegments; segmentId++ {
			segment := level.Segments[segmentId]
			start := level.Vertexes[segment.VertexStart]
			end := level.Vertexes[segment.VertexEnd]
			t[lineDefHash(start, end)] = lineDef{start: start, end: end, subSectorId: strconv.Itoa(int(subSectorId)) }
		}
	}
	*/

	for subSectorId := int16(0); subSectorId < int16(len(level.SubSectors)); subSectorId ++ {
		subSector := level.SubSectors[subSectorId]
		if subSector == nil { continue }
		segment := level.Segments[subSector.StartSeg]
		if segment == nil { continue }
		lineDef := level.LineDefs[int(segment.LineNum)]
		if lineDef == nil { continue}
		_, sideDef := b.segmentSideDef(level, segment, lineDef)
		if sideDef == nil { continue }
		sector := level.Sectors[sideDef.SectorRef]
		if sector == nil { continue }

		endSegmentId := subSector.StartSeg + subSector.NumSegments
		for segmentId := subSector.StartSeg; segmentId < endSegmentId; segmentId++ {
			segment := level.Segments[segmentId]
			lineDef := level.LineDefs[int(segment.LineNum)]
			_, sideDef := b.segmentSideDef(level, segment, lineDef)
			if sideDef == nil { continue }
			//_, oppositeSideDef := b.segmentOppositeSideDef(level, segment, lineDef)
			//if oppositeSideDef == nil {
			//	return
			//}

			//impassible 0
			//Block Monsters
			//twoSided
			//upperUnpegged
			//lowerUnpegged
			//secret
			//blockSound
			//NotOnMap
			//Already on Map
			//if (lineDef.Flags >> 2) & 1 == 0 {
			//	continue
			//}

			//fmt.Println(lineDef.Tag)

			current := b.getConfigSector(subSectorId, sector, lineDef)
			start := level.Vertexes[segment.VertexStart]
			end := level.Vertexes[segment.VertexEnd]

			//fmt.Println(lineDef.Flags & 0x3)
			//fmt.Println(sideDef.MiddleTexture, sideDef.UpperTexture, sideDef.LowerTexture)

			//upperTexture := sideDef.UpperTexture
			//middleTexture := sideDef.MiddleTexture
			//lowerTexture := sideDef.LowerTexture

			//if  ld, ok := t[lineDefHash(end, start)]; ok {
			//	neighborId = ld.subSectorId
			//}

			neighborId := "wall"
			neighborStart := &model.InputNeighbor{Id: neighborId, XY: model.XY{X: -float64(start.XCoord), Y: float64(start.YCoord)}}
			neighborEnd := &model.InputNeighbor{Id: neighborId, XY: model.XY{X: -float64(end.XCoord), Y: float64(end.YCoord)}}
			current.Neighbors = append(current.Neighbors, neighborStart)
			current.Neighbors = append(current.Neighbors, neighborEnd)
			//b.createSegment(level, segment, current)
		}
	}
}

func (b * Builder) segmentSideDef(level *Level, seg *Seg, lineDef *LineDef) (int16, *SideDef) {
	if seg.SegmentSide == 0 { return lineDef.SideDefRight, level.SideDefs[lineDef.SideDefRight] }
	if lineDef.SideDefLeft == -1 { return 0, nil }
	return lineDef.SideDefLeft, level.SideDefs[lineDef.SideDefLeft]
}

func (b * Builder) segmentOppositeSideDef(level *Level, seg *Seg, lineDef *LineDef) (int16, *SideDef) {
	if seg.SegmentSide == 0 {
		if lineDef.SideDefLeft == -1 { return 0, nil }
		return lineDef.SideDefLeft, level.SideDefs[lineDef.SideDefLeft]
	}
	return lineDef.SideDefRight, level.SideDefs[lineDef.SideDefRight]
}

func (b * Builder) loadTexture(wad *WAD, textureName string) (*image.RGBA, error) {
	texture, ok := wad.GetTexture(textureName)
	if !ok {
		return nil, errors.New("unknown texture " + textureName)
	}
	if texture.Header == nil {
		return nil, nil
	}
	bounds := image.Rect(0, 0, int(texture.Header.Width), int(texture.Header.Height))
	rgba := image.NewRGBA(bounds)
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride")
	}
	for _, patch := range texture.Patches {
		img, ok := wad.GetImage(patch.PNameNumber)
		if !ok {
			return nil, errors.New(fmt.Sprintf("unknown patch %d for %s", patch.PNameNumber, textureName))
		}
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				pixel := img.Pixels[y*img.Width+x]
				var alpha uint8
				if pixel == wad.transparentPaletteIndex {
					alpha = 0
				} else {
					alpha = 255
				}
				rgb := wad.playPal.Palettes[0].Table[pixel]
				rgba.Set(int(patch.XOffset) + x, int(patch.YOffset) + y, color.RGBA{R: rgb.Red, G: rgb.Green, B: rgb.Blue, A: alpha})
			}
		}
	}
	return rgba, nil

	/*
	var texId uint32
	gl.GenTextures(1, &texId)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	return texId, nil
	*/
}


func (b * Builder) getConfigSector(id int16, sector *Sector, ld * LineDef) * model.InputSector{
	c, ok := b.cfg[id]
	if !ok {
		c = &model.InputSector{
			Id:           strconv.Itoa(int(id)),
			Ceil:         float64(sector.CeilingHeight) / 3,
			Floor:        float64(sector.FloorHeight) / 3,
			Textures:     true,
			WallTexture:  "wall2.ppm",
			LowerTexture: "wall.ppm",
			UpperTexture: "wall3.ppm",
			FloorTexture: "floor.ppm",
			CeilTexture:  "ceil.ppm",
			Neighbors:    nil,
		}
		b.cfg[id] = c
	}
	return c
}


func (b * Builder) findSector(level *Level, x int16, y int16, idx int) (int16, int, *Sector) {
	const subSectorBit = int(0x8000)

	if idx & subSectorBit == subSectorBit {
		idx = int(uint16(idx) & ^uint16(subSectorBit))
		sSector := level.SubSectors[idx]
		for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg + sSector.NumSegments; segIdx++ {
			seg := level.Segments[segIdx]
			lineDef := level.LineDefs[seg.LineNum]
			_, sideDef := b.segmentSideDef(level, seg, lineDef)
			if sideDef != nil {
				return sideDef.SectorRef, idx, level.Sectors[sideDef.SectorRef]
			}
			_, oppositeSideDef := b.segmentOppositeSideDef(level, seg, lineDef)
			if oppositeSideDef != nil {
				return oppositeSideDef.SectorRef, idx, level.Sectors[oppositeSideDef.SectorRef]
			}
		}
	}
	node := level.Nodes[idx]
	if b.intersects(x, y, &node.BBox[0]) {
		return b.findSector(level, x, y, int(node.Child[0]))
	}
	if b.intersects(x, y, &node.BBox[1]) {
		return b.findSector(level, x, y, int(node.Child[1]))
	}
	return 0, 0, nil
}

func (b * Builder) intersects(x int16, y int16, bbox *BBox) bool {
	return x > bbox.Left && x < bbox.Right && y > bbox.Bottom && y <=bbox.Top
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