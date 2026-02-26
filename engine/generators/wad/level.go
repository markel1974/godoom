package wad

import (
	"github.com/markel1974/godoom/engine/generators/wad/lumps"
)

// Level represents a game level structure containing elements such as things, linedefs, sidedefs, vertexes, segments, subsectors, sectors, and nodes.
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

// GetSectorFromSubSector determines the sector ID corresponding to a given subsector ID and returns it with a success flag.
func (l *Level) GetSectorFromSubSector(subSectorId uint16) (uint16, bool) {
	sideDef := l.GetSideDefFromSubSector(subSectorId)
	if sideDef == nil {
		return 0, false
	}
	return sideDef.SectorRef, true
}

// GetSideDefFromSubSector determines the sector ID corresponding to a given subsector ID and returns it with a success flag.
func (l *Level) GetSideDefFromSubSector(subSectorId uint16) *lumps.SideDef {
	if subSectorId < 0 || int(subSectorId) > len(l.SubSectors) {
		return nil
	}
	sSector := l.SubSectors[subSectorId]
	for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg+sSector.NumSegments; segIdx++ {
		seg := l.Segments[segIdx]
		lineDef := l.LineDefs[seg.LineDef]
		_, sideDef := l.SegmentSideDef(seg, lineDef)
		if sideDef != nil {
			return sideDef
		}
	}
	return nil
}

// SegmentSideDef identifies the correct SideDef for a segment based on its direction and the associated LineDef.
func (l *Level) SegmentSideDef(seg *lumps.Seg, lineDef *lumps.LineDef) (int16, *lumps.SideDef) {
	if seg.Direction == 0 {
		return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight]
	}
	if lineDef.SideDefLeft == -1 {
		return 0, nil
	}
	return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
}

// SegmentOppositeSideDef determines the opposite side definition for a given segment based on its direction.
func (l *Level) SegmentOppositeSideDef(seg *lumps.Seg, lineDef *lumps.LineDef) (int16, *lumps.SideDef) {
	if seg.Direction == 0 {
		if lineDef.SideDefLeft == -1 {
			return 0, nil
		}
		return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
	}
	return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight]
}
