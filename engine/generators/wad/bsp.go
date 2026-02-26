package wad

import (
	"math"

	lumps2 "github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// subSectorBit is a bitmask used to identify whether a node in the BSP tree represents a sub-sector.
const subSectorBit = uint16(0x8000)

// BSP represents a Binary Space Partitioning data structure used for spatial management in level geometry.
type BSP struct {
	level     *Level
	root      uint16
	leafNodes []*lumps2.Node
}

// NewBsp constructs and returns a new BSP instance initialized with the provided Level.
func NewBsp(level *Level) *BSP {
	var nodes []*lumps2.Node
	for idx := 0; idx < len(level.Nodes); idx++ {
		node := level.Nodes[idx]
		if node.Child[0]&subSectorBit == subSectorBit {
			nodes = append(nodes, node)
		}
		if node.Child[1]&subSectorBit == subSectorBit {
			nodes = append(nodes, node)
		}
	}
	return &BSP{
		level:     level,
		root:      uint16(len(level.Nodes) - 1),
		leafNodes: nodes,
	}
}

// FindSector identifies the sector containing the given coordinates (x, y) and returns sector reference, node index, and sector data.
//func (bsp *BSP) FindSector(x int16, y int16) (uint16, uint16, *lumps2.Sector) {
//	return bsp.findSector(x, y, bsp.root)
//}

// FindSector recursively traverses the BSP tree to identify the sector containing the given (x, y) coordinates.
func (bsp *BSP) FindSector(x int16, y int16, idx uint16) (uint16, uint16, *lumps2.Sector) {
	if idx&subSectorBit == subSectorBit {
		idx = idx & ^subSectorBit
		sSector := bsp.level.SubSectors[idx]
		for segIdx := sSector.StartSeg; segIdx < sSector.StartSeg+sSector.NumSegments; segIdx++ {
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
		return bsp.FindSector(x, y, child)
	}
	return 0, 0, nil
}

// Traverse traverses a BSP tree, splits input polygons, and associates them with subsectors in the output map.
func (bsp *BSP) Traverse(level *Level, nodeIdx uint16, poly Polygons, out map[uint16]Polygons) {
	if nodeIdx&0x8000 != 0 {
		ssIdx := nodeIdx &^ 0x8000
		out[ssIdx] = poly
		return
	}
	node := level.Nodes[nodeIdx]
	front, back := PolygonSplit(poly, node.X, node.Y, node.DX, node.DY)
	if len(front) > 0 {
		bsp.Traverse(level, node.Child[0], front, out)
	}
	if len(back) > 0 {
		bsp.Traverse(level, node.Child[1], back, out)
	}
}

// DescribeLineF generates a list of points representing a line from (x0, y0) to (x1, y1) using Bresenham's algorithm.
func (bsp *BSP) DescribeLineF(x0 float64, y0 float64, x1 float64, y1 float64) Polygons {
	high := func(x0 float64, y0 float64, x1 float64, y1 float64) Polygons {
		var res Polygons
		dx := x1 - x0
		dy := y1 - y0
		xi := float64(1)
		if dx < 0 {
			xi = -1
			dx = -dx
		}
		D := (2 * dx) - dy
		x := x0
		for y := y0; y <= y1; y++ {
			res = append(res, model.XY{X: math.Round(x), Y: math.Round(y)})
			if D > 0 {
				x = x + xi
				D = D + (2 * (dx - dy))
			} else {
				D = D + 2*dx
			}
		}
		return res
	}

	low := func(x0 float64, y0 float64, x1 float64, y1 float64) Polygons {
		var res Polygons
		dx := x1 - x0
		dy := y1 - y0
		yi := float64(1)
		if dy < 0 {
			yi = -1
			dy = -dy
		}
		D := (2 * dy) - dx
		y := y0
		for x := x0; x <= x1; x++ {
			res = append(res, model.XY{X: math.Round(x), Y: math.Round(y)})
			if D > 0 {
				y = y + yi
				D = D + (2 * (dy - dx))
			} else {
				D = D + 2*dy
			}
		}
		return res
	}

	var res Polygons
	reverse := false
	if math.Abs(y1-y0) < math.Abs(x1-x0) {
		if x0 > x1 {
			reverse = true
			res = low(x1, y1, x0, y0)
		} else {
			res = low(x0, y0, x1, y1)
		}
	} else {
		if y0 > y1 {
			reverse = true
			res = high(x1, y1, x0, y0)
		} else {
			res = high(x0, y0, x1, y1)
		}
	}

	if !reverse {
		return res
	}

	var r Polygons
	for x := len(res) - 1; x >= 0; x-- {
		r = append(r, res[x])
	}
	return r
}
