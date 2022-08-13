package wad

import (
	"github.com/markel1974/godoom/engine/wad/lumps"
	"math"
)

type BSP struct {
	level      * Level
	root       int
}



func NewBsp(level * Level) *BSP {
	return &BSP{
		level: level,
		root:  len(level.Nodes) - 1,
	}
}

func (bsp * BSP) FindSector(x int16, y int16) (int16, int, *lumps.Sector) {
	return bsp.findSector(x, y, bsp.root)
}

func (bsp * BSP) FindSubSector(x int16, y int16) (int, int, bool) {
	return bsp.findSubSector(x, y, bsp.root)
}

func (bsp * BSP) TraverseBsp(x int16, y int16, opposite bool) int {
	return bsp.traverseBsp(x, y, opposite, bsp.root)
}

func (bsp * BSP) findSector(x int16, y int16, idx int) (int16, int, *lumps.Sector) {
	const subSectorBit = int(0x8000)
	if idx & subSectorBit == subSectorBit {
		idx = int(uint16(idx) & ^uint16(subSectorBit))
		sSector := bsp.level.SubSectors[idx]
		for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg + sSector.NumSegments; segIdx++ {
			seg := bsp.level.Segments[segIdx]
			lineDef := bsp.level.LineDefs[seg.LineNum]
			_, sideDef := bsp.level.SegmentSideDef(seg, lineDef)
			if sideDef != nil {
				return sideDef.SectorRef, idx, bsp.level.Sectors[sideDef.SectorRef]
			}
			_, oppositeSideDef := bsp.level.SegmentOppositeSideDef(seg, lineDef)
			if oppositeSideDef != nil {
				return oppositeSideDef.SectorRef, idx, bsp.level.Sectors[oppositeSideDef.SectorRef]
			}
		}
	}
	node := bsp.level.Nodes[idx]
	if node.BBox[0].Intersect(x, y) {
		return bsp.findSector(x, y, int(node.Child[0]))
	}
	if node.BBox[1].Intersect(x, y) {
		return bsp.findSector(x, y, int(node.Child[1]))
	}
	return 0, 0, nil
}

func (bsp* BSP) findSubSector(x int16, y int16, subSectorId int) (int, int, bool) {
	if subSectorId & subSectorBit == subSectorBit {
		subSectorId = int(uint16(subSectorId) & ^uint16(subSectorBit))
		sSector := bsp.level.SubSectors[subSectorId]
		sector := -1
		for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg + sSector.NumSegments; segIdx++ {
			seg := bsp.level.Segments[segIdx]
			lineDef := bsp.level.LineDefs[seg.LineNum]
			_, sideDef := bsp.level.SegmentSideDef(seg, lineDef)
			if sideDef != nil {
				sector = int(sideDef.SectorRef)
			}
		}
		return sector, subSectorId, true
	}
	node := bsp.level.Nodes[subSectorId]
	if node.BBox[0].Intersect(x, y) {
		return bsp.findSubSector(x, y, int(node.Child[0]))
	}
	if node.BBox[1].Intersect(x, y) {
		return bsp.findSubSector(x, y, int(node.Child[1]))
	}
	return -1, -1, false
}

func (bsp* BSP) traverseBsp(x int16, y int16, opposite bool, idx int) int {
	if idx & subSectorBit == subSectorBit {
		if idx == -1 {
			return -1
		} else {
			subSectorId := int(uint16(idx) & ^uint16(subSectorBit))
			return subSectorId
		}
	}
	node := bsp.level.Nodes[idx]
	side := bsp.pointOnSide(x, y, node)

	if !opposite {
		sideIdx := int(node.Child[side])
		return bsp.traverseBsp(x, y, opposite, sideIdx)
	} else {
		oppositeSide := side ^ 1
		oppositeSideIdx := int(node.Child[oppositeSide])
		return bsp.traverseBsp(x, y, opposite, oppositeSideIdx)
	}
}

func (bsp * BSP) FindSubSectorByLine(x1 int, y1 int, x2 int, y2 int) (int16, int16){
	//length := math.Sqrt((float64(x2 - x1) * float64(x2 - x1)) + (float64(y2 - y1) * float64(y2 - y1)))
	xt := int(math.Round(float64(x1 + x2) / 2))
	yt := int(math.Round(float64(y1 + y2) / 2))

	rt := bsp.describeCircle(xt, yt, 1)
	a := -1
	b := -1

	found := 0

	for _, c := range rt {
		if _, subSector, ok := bsp.findSubSector(int16(c.X), int16(c.Y), bsp.root); ok {
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

func (bsp * BSP) FindOppositeSubSectorByLine(subSector int16, x1 int, y1 int, x2 int, y2 int) (int16, int16, float64) {
	length := math.Sqrt((float64(x2 - x1) * float64(x2 - x1)) + (float64(y2 - y1) * float64(y2 - y1)))
	//cx := int(math.Round(float64(x1 + x2) / 2))
	//cy := int(math.Round(float64(y1 + y2) / 2))

	oppositeSector := int16(-1)
	oppositeSubSector := int16(-1)

	test := []float64 {0.5, 0.6, 0.4, 0.7, 0.3}

	for _, k := range test {
		cx := int(math.Round(float64(x1) + k * (float64(x2) - float64(x1))))
		cy := int(math.Round(float64(y1) + k * (float64(y2) - float64(y1))))
		oppositeSector, oppositeSubSector =  bsp.findOppositeSubSectorByLine(subSector, cx, cy)
		if oppositeSubSector >= 0 {
			break
		}
	}
	return oppositeSector, oppositeSubSector, length
}

func (bsp * BSP) findOppositeSubSectorByLine(subSector int16, cx int, cy int) (int16, int16) {
	rt := bsp.describeCircle(cx, cy, 2)
	resultSector := int16(-1)
	resultSubSector := int16(-1)
	multi := 0
	count := 0
	for _, c := range rt {
		if sector, ss, ok := bsp.findSubSector(int16(c.X), int16(c.Y), bsp.root); ok {
			count ++
			if ss != int(subSector) {
				if resultSubSector == -1 {
					resultSector = int16(sector)
					resultSubSector = int16(ss)
					continue
				}
				if resultSubSector != int16(ss) {
					multi++
				}
			}
		}
	}
	if multi > 0 {
		resultSubSector = -3
	} else if resultSubSector == - 1 && count > 0 {
		resultSubSector = -2
	}
	return resultSector, resultSubSector
}


//TODO REMOVE....
func (bsp * BSP) BruteForceLineDef(startX int16, startY int16, endX int16, endY int16) (int16, *lumps.SideDef) {
	for subSectorId := int16(0); subSectorId < int16(len(bsp.level.SubSectors)); subSectorId++ {
		subSector := bsp.level.SubSectors[subSectorId]

		endSegmentId := subSector.StartSeg + subSector.NumSegments
		for segmentId := subSector.StartSeg; segmentId < endSegmentId; segmentId++ {
			segment := bsp.level.Segments[segmentId]
			lineDef := bsp.level.LineDefs[int(segment.LineNum)]
			_, sideDef := bsp.level.SegmentSideDef(segment, lineDef)
			if sideDef == nil {
				continue
			}

			start := bsp.level.Vertexes[segment.VertexStart]
			end := bsp.level.Vertexes[segment.VertexEnd]

			if start.XCoord == startX && start.YCoord == startY && end.XCoord == endX && end.YCoord == endY {
				return subSectorId, sideDef
			}
		}
	}
	return -1, nil
}

func (bsp* BSP) pointOnSide(x int16, y int16, node *lumps.Node) int {
	dx := int(x) - int(node.X)
	dy := int(y) - int(node.Y)
	// Perp dot product:
	left := (int(node.DY) >> 16) * dx
	right := (int(node.DX) >> 16) * dy
	if right < left {
		// Point is on front side:
		return 0
	}
	// Point is on the back side:
	return 1
}

func (bsp* BSP) describeCircle(x0 int, y0 int, radius int) []XY {
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

func (bsp * BSP) describeLine(x1 int, y1 int, x2 int, y2 int) []XY {
	var res []XY
	steep := abs(y2-y1) > abs(x2-x1)
	if steep { x1, y1 = swap(x1, y1); x2, y2 = swap(x2, y2) }

	if x1 > x2 { x1, x2 = swap(x1, x2); y1, y2 = swap(y1, y2) }
	dx := x2 - x1
	dy := abs(y2 - y1)
	errorDx := dx / 2.0
	var yStep int
	if y1 < y2 { yStep = 1 } else { yStep = -1 }
	y := y1
	maxX := x2
	for x := x1; x <= maxX; x++ {
		if steep {
			res = append(res, XY{ X: float64(y), Y: float64(x) })
		} else {
			res = append(res, XY{ X: float64(x), Y: float64(y) })
		}
		errorDx -= dy
		if errorDx < 0 {
			y += yStep
			errorDx += dx
		}
	}
	return res
}

/*

func (bsp * BSP) Test(x1 int, y1 int, x2 int, y2 int, cx int, cy int) int {
	//The slope m of the p1-p2 line is given by:
	m := (y2-y1)/(x2-x1)
	//Then the equation of the line perpendicular to p1-p2 passing through p3 is:
	//(y-y3)/(x-x3) = -1/m

	//Rearranging gives:
	//x = (y3-y)*m + x3
	//Therefore:
	qy1 := -3
	qy2 := +3

	findX1 := (cy - qy1) * m + cx
	findX2 := (cy - qy2) * m + cx
}

 */