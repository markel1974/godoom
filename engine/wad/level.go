package wad

import (
	"github.com/markel1974/godoom/engine/wad/lumps"
	"math"
)

type Level struct {
	Things     []*lumps.Thing
	LineDefs   []*lumps.LineDef
	SideDefs   []*lumps.SideDef
	Vertexes   []*lumps.Vertex
	Segments   []*lumps.Seg
	SubSectors []*lumps.SubSector
	Sectors    []*lumps.Sector
	Nodes      []*lumps.Node
}


func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func swap(a int, b int) (int, int) {
	return b, a
}

func (l *Level) FindSector(x int16, y int16) (int16, int, *lumps.Sector) {
	return l.findSector(x, y, len(l.Nodes) - 1)
}

func (l *Level) FindSubSectorByLine(x1 int, y1 int, x2 int, y2 int) (int16, int16){
	//length := math.Sqrt((float64(x2 - x1) * float64(x2 - x1)) + (float64(y2 - y1) * float64(y2 - y1)))
	xt := int(math.Round(float64(x1 + x2) / 2))
	yt := int(math.Round(float64(y1 + y2) / 2))

	rt := l.describeCircle(xt, yt, 1)
	a := -1
	b := -1

	found := 0

	for _, c := range rt {
		if _, subSector, ok := l.findSubSector(int16(c.X), int16(c.Y), len(l.Nodes) - 1); ok {
			if a == - 1 {
				a = subSector
				found++
				continue
			}
			if b == -1 {
				if subSector != a {
					b = subSector
					found++
				}
				continue
			}
			if subSector != a && subSector != b {
				found++
			}
		}
	}

	switch found {
	case 0:
		return -1, -1
	case 1:
		//TODO IN QUESTO CASO SCEGLIERE LA RETTA MIGLIORE ALL'INTERNO DEL SETTORE
		return int16(a), int16(a)
	case 2:
		return int16(a), int16(b)
	default:
		//TODO TROPPI RISULTATI,  MIGLIORARE LA RICERCA
		return int16(a), int16(b)
	}
}


func (l *Level) findSector(x int16, y int16, idx int) (int16, int, *lumps.Sector) {
	const subSectorBit = int(0x8000)
	if idx & subSectorBit == subSectorBit {
		idx = int(uint16(idx) & ^uint16(subSectorBit))
		sSector := l.SubSectors[idx]
		for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg + sSector.NumSegments; segIdx++ {
			seg := l.Segments[segIdx]
			lineDef := l.LineDefs[seg.LineNum]
			_, sideDef := l.SegmentSideDef(seg, lineDef)
			if sideDef != nil {
				return sideDef.SectorRef, idx, l.Sectors[sideDef.SectorRef]
			}
			_, oppositeSideDef := l.SegmentOppositeSideDef(seg, lineDef)
			if oppositeSideDef != nil {
				return oppositeSideDef.SectorRef, idx, l.Sectors[oppositeSideDef.SectorRef]
			}
		}
	}
	node := l.Nodes[idx]
	if node.BBox[0].Intersect(x, y) {
		return l.findSector(x, y, int(node.Child[0]))
	}
	if node.BBox[1].Intersect(x, y) {
		return l.findSector(x, y, int(node.Child[1]))
	}
	return 0, 0, nil
}

func (l *Level) findSubSector(x int16, y int16, subSectorId int) (int, int, bool) {
	const subSectorBit = int(0x8000)
	if subSectorId & subSectorBit == subSectorBit {
		subSectorId = int(uint16(subSectorId) & ^uint16(subSectorBit))
		sSector := l.SubSectors[subSectorId]
		sector := -1
		for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg + sSector.NumSegments; segIdx++ {
			seg := l.Segments[segIdx]
			lineDef := l.LineDefs[seg.LineNum]
			_, sideDef := l.SegmentSideDef(seg, lineDef)
			if sideDef != nil {
				sector = int(sideDef.SectorRef)
			}
		}
		return sector, subSectorId, true
	}
	node := l.Nodes[subSectorId]
	if node.BBox[0].Intersect(x, y) {
		return l.findSubSector(x, y, int(node.Child[0]))
	}
	if node.BBox[1].Intersect(x, y) {
		return l.findSubSector(x, y, int(node.Child[1]))
	}
	return -1, -1, false
}

func (l *Level) SegmentSideDef(seg *lumps.Seg, lineDef *lumps.LineDef) (int16, *lumps.SideDef) {
	if seg.SegmentSide == 0 { return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight] }
	if lineDef.SideDefLeft == -1 { return 0, nil }
	return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
}

func (l *Level) SegmentOppositeSideDef(seg *lumps.Seg, lineDef *lumps.LineDef) (int16, *lumps.SideDef) {
	if seg.SegmentSide == 0 {
		if lineDef.SideDefLeft == -1 { return 0, nil }
		return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
	}
	return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight]
}

func (l *Level) describeLine(x1 int, y1 int, x2 int, y2 int) (int, int, int, int) {
	const stepCount = 1
	steep := abs(y2-y1) > abs(x2-x1)
	if steep { x1, y1 = swap(x1, y1); x2, y2 = swap(x2, y2) }
	if x1 > x2 { x1, x2 = swap(x1, x2); y1, y2 = swap(y1, y2) }
	var yStep int
	if y1 < y2 { yStep = stepCount } else { yStep = -stepCount }
	outX := (x1 + x2) / 2
	outY := (y1 + y2) / 2
	return outX + yStep, outY + yStep, outX - yStep, outY - yStep
}


/*
func (l * Level) describeCircle(x0 int, y0 int, radius int) []XY{
	var res []XY
	x := radius
	y := 0
	err := 0

	for ;x >= y; {
		res = append(res, XY{float64(x0 + x), float64(y0 + y)})
		res = append(res, XY{float64(x0 + y), float64(y0 + x)})
		res = append(res, XY{float64(x0 - y), float64(y0 + x)})
		res = append(res, XY{float64(x0 - x), float64(y0 + y)})
		res = append(res, XY{float64(x0 - x), float64(y0 - y)})
		res = append(res, XY{float64(x0 - y), float64(y0 - x)})
		res = append(res, XY{float64(x0 + y), float64(y0 - x)})
		res = append(res, XY{float64(x0 + x), float64(y0 - y)})
		if err <= 0 {
			y += 1
			err += 2*y + 1
		}
		if err > 0 {
			x -= 1
			err -= 2 * x + 1
		}
	}
	return res
}
*/

func (l * Level) describeCircle(x0 int, y0 int, radius int) []XY {
	var res []XY
	x := radius
	y := 0
	radiusError := 1 - x
	for ;y <= x; {
		res = append(res, XY{float64( x + x0),float64( y + y0) })
		res = append(res, XY{float64( x + x0),float64( y + y0) })
		res = append(res, XY{float64( y + x0),float64( x + y0) })
		res = append(res, XY{float64(-x + x0),float64( y + y0) })
		res = append(res, XY{float64(-y + x0),float64( x + y0) })
		res = append(res, XY{float64(-x + x0),float64(-y + y0) })
		res = append(res, XY{float64(-y + x0),float64(-x + y0) })
		res = append(res, XY{float64( x + x0),float64(-y + y0) })
		res = append(res, XY{float64( y + x0),float64(-x + y0) })
		y++
		if radiusError < 0 {
			radiusError += 2 * y + 1
		} else {
			x--
			radiusError += 2 * (y - x + 1)
		}
	}
	return res
}