package wad

import "C"
import (
	"github.com/markel1974/godoom/engine/wad/lumps"
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

func (l *Level) FindSubSectorByLine(x1 int, y1 int, x2 int, y2 int) (int16, int16, bool){
	//TODO WRONG!!!!!
	lx, ly, rx, ry := l.drawLine(x2, y1, x1, y2)
	ssAId, ok := l.findSubSector(int16(lx), int16(ly), len(l.Nodes) - 1)
	if !ok { return -1, -1, false }
	ssBId, ok := l.findSubSector(int16(rx), int16(ry), len(l.Nodes) - 1)
	if !ok { return -1, -1, false }

	/*
	xt := (x1 + x2) / 2
	yt := (y1 + y2) / 2
	ssAId, ok := l.findSubSector(int16(xt), int16(yt), len(l.Nodes) - 1)
	if !ok {
		ssAId, ok = l.findSubSector(int16(xt), int16(yt), len(l.Nodes) - 1)
		return -1, -1, false
	}
	ssBId, ok := l.findSubSector(int16(x1), int16(yt), len(l.Nodes) - 1)
	if !ok {
		return -1, -1, false
	}
	*/

	return int16(ssAId), int16(ssBId), true
}

func (l *Level) drawLine(x1 int, y1 int, x2 int, y2 int) (int, int, int, int) {
	/*
	const stepCount = 1
	steep := abs(y2-y1) > abs(x2-x1)
	if steep { x1, y1 = swap(x1, y1); x2, y2 = swap(x2, y2) }
	if x1 > x2 { x1, x2 = swap(x1, x2); y1, y2 = swap(y1, y2) }
	var yStep int
	if y1 < y2 { yStep = stepCount } else { yStep = -stepCount }
	*/
	yStep := 0
	outX := (x1 + x2) / 2
	outY := (y1 + y2) / 2
	return outX + yStep, outY + yStep, outX - yStep, outY - yStep
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

func (l *Level) findSubSector(x int16, y int16, subSectorId int) (int, bool) {
	const subSectorBit = int(0x8000)
	if subSectorId & subSectorBit == subSectorBit {
		subSectorId = int(uint16(subSectorId) & ^uint16(subSectorBit))
		return subSectorId, true
	}
	node := l.Nodes[subSectorId]
	if node.BBox[0].Intersect(x, y) {
		return l.findSubSector(x, y, int(node.Child[0]))
	}
	if node.BBox[1].Intersect(x, y) {
		return l.findSubSector(x, y, int(node.Child[1]))
	}
	return 0, false
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
