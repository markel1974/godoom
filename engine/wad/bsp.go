package wad

import (
	"fmt"
	"github.com/markel1974/godoom/engine/wad/lumps"
	"math"
	"sort"
)

type BSP struct {
	level      * Level
	root       uint16
}

func swapF(a float64, b float64) (float64, float64) {
	return b, a
}

func NewBsp(level * Level) *BSP {
	return &BSP{
		level: level,
		root:  uint16(len(level.Nodes) - 1),
	}
}

func (bsp * BSP) FindSector(x int16, y int16) (uint16, uint16, *lumps.Sector) {
	return bsp.findSector(x, y, bsp.root)
}

func (bsp * BSP) findSector(x int16, y int16, idx uint16) (uint16, uint16, *lumps.Sector) {
	if idx & subSectorBit == subSectorBit {
		idx = idx & ^subSectorBit
		sSector := bsp.level.SubSectors[idx]
		for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg + sSector.NumSegments; segIdx++ {
			seg := bsp.level.Segments[segIdx]
			lineDef := bsp.level.LineDefs[seg.LineDef]
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
	if child, ok := node.Intersect(x, y); ok {
		return bsp.findSector(x, y, child)
	}
	return 0, 0, nil
}

func (bsp * BSP) FindSubSector(x int16, y int16) (uint16, uint16, bool) {
	return bsp.findSubSector(x, y, bsp.root)
}

func (bsp * BSP) TraverseBsp(x int16, y int16, opposite bool) uint16 {
	return bsp.traverseBsp(x, y, opposite, bsp.root)
}

func (bsp* BSP) traverseBsp(x int16, y int16, opposite bool, idx uint16) uint16 {
	if idx & subSectorBit == subSectorBit {
		subSectorId := (idx) & ^subSectorBit
		return subSectorId
	}
	node := bsp.level.Nodes[idx]
	side := node.PointOnSide(x, y)

	if !opposite {
		sideIdx := node.Child[side]
		return bsp.traverseBsp(x, y, opposite, sideIdx)
	} else {
		oppositeSide := side ^ 1
		oppositeSideIdx := node.Child[oppositeSide]
		return bsp.traverseBsp(x, y, opposite, oppositeSideIdx)
	}
}

func (bsp* BSP) findSubSector(x int16, y int16, idx uint16) (uint16, uint16, bool) {
	if idx & subSectorBit == subSectorBit {
		subSectorId := idx & ^subSectorBit
		sectorId, _ := bsp.level.GetSectorFromSubSector(subSectorId)
		return sectorId, subSectorId, true
	}
	node := bsp.level.Nodes[idx]
	if child, ok := node.Intersect(x, y); ok {
		return bsp.findSubSector(x, y, child)
	}
	return 0, 0, false
}



func (bsp * BSP) FindOppositeSubSectorByPoints(subSectorId uint16, x1 int16, y1 int16, x2 int16, y2 int16) (uint16, int) {
	//TODO TROVARE LE LINEE PERPENDICOLARI AGLI ESTREMI DELLA RETTA
	//Calcolare 2 px avanti e dietro su tutte e due le rette\
	//bsp.Test(x1, y1, x2, y2, x1, x2)
	//bsp.Test(x1, y1, x2, y2, x2, y2)

	radius := 2
	//offset := 0
	//rOffset := radius + offset

	rl := bsp.describeLine(float64(x1), float64(y1), float64(x2), float64(y2))
	//if len(rl) < (radius * 2) + 2 {
	//	return 0, -1
	//}

	//rl = rl[rOffset : len(rl) - rOffset]

	out := make(map[uint16]int)

	for _, p := range rl {
		bsp.findSubSectorByPoint(int16(math.Round(p.X)), int16(math.Round(p.Y)), radius, out)
	}

	if len(out) == 0 {
		return 0, -1
	}

	if _, ok := out[subSectorId]; ok {
		delete(out, subSectorId)
		if len(out) == 0 {
			return subSectorId, -2
		}
	}

	if len(out) == 1 { for k := range out { return k, 0 } }

	type result struct{ ss uint16; count int }
	var r[] result
	for k, v := range out { r = append(r, result{ss:k, count: v}) }
	sort.SliceStable(r, func(i, j int) bool { return r[i].count > r[j].count })
	return r[0].ss, 0
}

func (bsp * BSP) findSubSectorByPoint(cx int16, cy int16, radius int, out map[uint16]int) {
	rt := bsp.describeCircle(float64(cx), float64(cy), float64(radius))
	for _, c := range rt {
		if _, ss, ok := bsp.findSubSector(int16(math.Round(c.X)), int16(math.Round(c.Y)), bsp.root); ok {
			if v, ok := out[ss]; ok {
				out[ss] = v + 1
			} else {
				out[ss] = 1
			}
		}
	}
}

func (bsp * BSP) FindOppositeSubSectorByLine(subSector uint16, x1 int16, y1 int16, x2 int16, y2 int16) (uint16, uint16, int) {
	//length := math.Sqrt((float64(x2 - x1) * float64(x2 - x1)) + (float64(y2 - y1) * float64(y2 - y1)))
	//cx := int(math.Round(float64(x1 + x2) / 2))
	//cy := int(math.Round(float64(y1 + y2) / 2))
	oppositeSector := uint16(0)
	oppositeSubSector := uint16(0)
	state := -1

	//tests := []float64 {0.5, 0.6, 0.4, 0.7, 0.3}
	tests := []float64 {0.5}
	for _, k := range tests {
		cx := int16(math.Round(float64(x1) + k * (float64(x2) - float64(x1))))
		cy := int16(math.Round(float64(y1) + k * (float64(y2) - float64(y1))))
		oppositeSector, oppositeSubSector, state = bsp.findOppositeSubSectorByLine(subSector, cx, cy)
		if state >= 0 {
			break
		}
	}
	return oppositeSector, oppositeSubSector, state
}

func (bsp * BSP) findOppositeSubSectorByLine(subSector uint16, cx int16, cy int16) (uint16, uint16, int) {
	rt := bsp.describeCircle(float64(cx), float64(cy), 2)
	resultSector := uint16(0)
	resultSubSector := uint16(0)
	state := -1
	multi := 0
	count := 0
	for _, c := range rt {
		if sector, ss, ok := bsp.findSubSector(int16(c.X), int16(c.Y), bsp.root); ok {
			count ++
			if ss != subSector {
				if state == -1 {
					resultSector = sector
					resultSubSector = ss
					state = 0
					continue
				}
				if resultSubSector != ss {
					multi++
				}
			}
		}
	}
	if multi > 0 {
		state = -3
	} else if state == - 1 && count > 0 {
		state = -2
	}
	return resultSector, resultSubSector, state
}

func (bsp* BSP) describeCircle(x0 float64, y0 float64, radius float64) []XY {
	var res []XY
	x := radius
	y := float64(0)
	radiusError := 1.0 - x
	for ;y <= x; {
		res = append(res, XY{ x + x0, y + y0 })
		res = append(res, XY{ x + x0, y + y0 })
		res = append(res, XY{ y + x0, x + y0 })
		res = append(res, XY{-x + x0, y + y0 })
		res = append(res, XY{-y + x0, x + y0 })
		res = append(res, XY{-x + x0,-y + y0 })
		res = append(res, XY{-y + x0,-x + y0 })
		res = append(res, XY{ x + x0,-y + y0 })
		res = append(res, XY{ y + x0,-x + y0 })
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

func (bsp * BSP) describeLine(x1 float64, y1 float64, x2 float64, y2 float64) []XY {
	var res []XY
	steep := math.Abs(y2-y1) > math.Abs(x2-x1)
	if steep { x1, y1 = swapF(x1, y1); x2, y2 = swapF(x2, y2) }

	if x1 > x2 { x1, x2 = swapF(x1, x2); y1, y2 = swapF(y1, y2) }
	dx := x2 - x1
	dy := math.Abs(y2 - y1)
	errorDx := dx / 2.0
	var yStep float64
	if y1 < y2 { yStep = 1 } else { yStep = -1 }
	y := y1
	maxX := x2
	for x := x1; x <= maxX; x++ {
		if steep {
			res = append(res, XY{ X: y, Y: x })
		} else {
			res = append(res, XY{ X: x, Y: y })
		}
		errorDx -= dy
		if errorDx < 0 {
			y += yStep
			errorDx += dx
		}
	}
	return res
}



func (bsp * BSP) Test(x1 int, y1 int, x2 int, y2 int, cx int, cy int) int {
	//TODO TROVARE LE LINEE PERPENDICOLARI AGLI ESTREMI DELLA RETTA
	//Calcolare 2 px avanti e dietro su tutte e due le rette\
	div := float64(x2)-float64(x1)
	slope := 0.0
	if div != 0 {
		slope = (float64(y2) - float64(y1)) / (float64(x2) - float64(x1))
		slope = -1 / slope
	}

	//% Point slope formula (y-yp) = slope * (x-xp)
	//% y = slope * (x - midX) + midY
	//% Compute y at some x, for example at x=300
	x := 300.0
	y := slope * (x - float64(cx)) + float64(cy)
	//plot([x, midX], [y, midY], 'bo-', 'LineWidth', 2);

	fmt.Println(x, y)
	return -1
}




/*
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
*/
/*
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
*/