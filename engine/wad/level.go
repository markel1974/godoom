package wad

import (
	"github.com/markel1974/godoom/engine/wad/lumps"
)


const subSectorBit = uint16(0x8000)

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


func (l * Level) GetSectorFromSubSector(subSectorId uint16) (uint16, bool){
	if subSectorId < 0 || int(subSectorId) > len(l.SubSectors) {
		return  0, false
	}
	sSector := l.SubSectors[subSectorId]
	for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg + sSector.NumSegments; segIdx++ {
		seg := l.Segments[segIdx]
		lineDef := l.LineDefs[seg.LineDef]
		_, sideDef := l.SegmentSideDef(seg, lineDef)
		if sideDef != nil {
			return sideDef.SectorRef, true
		}
	}
	return 0, false
}

func (l *Level) SegmentSideDef(seg *lumps.Seg, lineDef *lumps.LineDef) (int16, *lumps.SideDef) {
	if seg.Direction == 0 { return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight] }
	if lineDef.SideDefLeft == -1 { return 0, nil }
	return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
}

func (l *Level) SegmentOppositeSideDef(seg *lumps.Seg, lineDef *lumps.LineDef) (int16, *lumps.SideDef) {
	if seg.Direction == 0 {
		if lineDef.SideDefLeft == -1 { return 0, nil }
		return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
	}
	return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight]
}
