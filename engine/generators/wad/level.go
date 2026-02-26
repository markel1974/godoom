package wad

import (
	lumps2 "github.com/markel1974/godoom/engine/generators/wad/lumps"
)

// subSectorBit is a bitmask used to identify whether a node in the BSP tree represents a sub-sector.
const subSectorBit = uint16(0x8000)

// Level represents a game level structure containing elements such as things, linedefs, sidedefs, vertexes, segments, subsectors, sectors, and nodes.
type Level struct {
	Things     []*lumps2.Thing
	LineDefs   []*lumps2.LineDef
	SideDefs   []*lumps2.SideDef
	Vertexes   []*lumps2.Vertex
	Segments   []*lumps2.Seg
	SubSectors []*lumps2.SubSector
	Sectors    []*lumps2.Sector
	Nodes      []*lumps2.Node
}

// GetSectorFromSubSector determines the sector ID corresponding to a given subsector ID and returns it with a success flag.
func (l *Level) GetSectorFromSubSector(subSectorId uint16) (uint16, bool) {
	if subSectorId < 0 || int(subSectorId) > len(l.SubSectors) {
		return 0, false
	}
	sSector := l.SubSectors[subSectorId]
	for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg+sSector.NumSegments; segIdx++ {
		seg := l.Segments[segIdx]
		lineDef := l.LineDefs[seg.LineDef]
		_, sideDef := l.SegmentSideDef(seg, lineDef)
		if sideDef != nil {
			return sideDef.SectorRef, true
		}
	}
	return 0, false
}

// SegmentSideDef identifies the correct SideDef for a segment based on its direction and the associated LineDef.
func (l *Level) SegmentSideDef(seg *lumps2.Seg, lineDef *lumps2.LineDef) (int16, *lumps2.SideDef) {
	if seg.Direction == 0 {
		return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight]
	}
	if lineDef.SideDefLeft == -1 {
		return 0, nil
	}
	return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
}

// SegmentOppositeSideDef determines the opposite side definition for a given segment based on its direction.
func (l *Level) SegmentOppositeSideDef(seg *lumps2.Seg, lineDef *lumps2.LineDef) (int16, *lumps2.SideDef) {
	if seg.Direction == 0 {
		if lineDef.SideDefLeft == -1 {
			return 0, nil
		}
		return lineDef.SideDefLeft, l.SideDefs[lineDef.SideDefLeft]
	}
	return lineDef.SideDefRight, l.SideDefs[lineDef.SideDefRight]
}
