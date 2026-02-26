package wad

import (
	"fmt"
	"math"
	"strconv"

	lumps2 "github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

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
func (bsp *BSP) FindSector(x int16, y int16) (uint16, uint16, *lumps2.Sector) {
	return bsp.findSector(x, y, bsp.root)
}

// findSector recursively traverses the BSP tree to identify the sector containing the given (x, y) coordinates.
func (bsp *BSP) findSector(x int16, y int16, idx uint16) (uint16, uint16, *lumps2.Sector) {
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
		return bsp.findSector(x, y, child)
	}
	return 0, 0, nil
}

// FindSubSector locates the sub-sector containing the given (x, y) coordinates and returns its ID and a success flag.
func (bsp *BSP) FindSubSector(x int16, y int16) (uint16, bool) {
	return bsp.findSubSector(x, y, bsp.root)
}

// Traverse traverses the BSP structure starting from the root to collect sub-sector indices into the container slice.
func (bsp *BSP) Traverse(container *[]uint16, x int16, y int16, root uint16) {
	bsp.traverse(container, x, y, root, root)
}

// traverseBsp recursively traverses the BSP tree to collect sub-sector IDs into the container, excluding the root ID.
func (bsp *BSP) traverse(container *[]uint16, x int16, y int16, root uint16, idx uint16) {
	if idx&subSectorBit == subSectorBit {
		if idx == 0xffff {
			return
		} else {
			subSectorId := (idx) & ^subSectorBit
			if subSectorId != root {
				*(container) = append(*(container), subSectorId)
			}
			return
		}
	}
	node := bsp.level.Nodes[idx]
	side := node.PointOnSide(x, y)
	sideIdx := node.Child[side]
	bsp.traverse(container, x, y, root, sideIdx)

	oppositeSide := side ^ 1
	oppositeSideIdx := node.Child[oppositeSide]
	bsp.traverse(container, x, y, root, oppositeSideIdx)
}

// findSubSector traverses the BSP tree recursively to locate the sub-sector containing the given (x, y) coordinates.
func (bsp *BSP) findSubSector(x int16, y int16, idx uint16) (uint16, bool) {
	//TODO WRONG IMPLEMENTATION!!!!!
	if idx&subSectorBit == subSectorBit {
		subSectorId := idx & ^subSectorBit
		return subSectorId, true
	}
	node := bsp.level.Nodes[idx]
	if child, ok := node.Intersect(x, y); ok {
		return bsp.findSubSector(x, y, child)
	}
	return 0, false
}

// FindSubSectorForced attempts to locate a sub-sector containing the point (x, y) by iterating over all leaf nodes.
// Returns the sub-sector index and a boolean indicating success.
func (bsp *BSP) FindSubSectorForced(x int16, y int16) (uint16, bool) {
	//TODO WRONG IMPLEMENTATION - DEVE FUNZIONARE PER FORZA CON INSIDE!!!!!

	for idx := 0; idx < len(bsp.leafNodes); idx++ {
		node := bsp.leafNodes[idx]
		if child, ok := node.Intersect(x, y); ok {
			return child & ^subSectorBit, true
		}

	}
	return 0, false
}

// FindNode traverses the BSP tree to locate the parent node index containing the point (x, y) and returns it if found.
func (bsp *BSP) FindNode(x int16, y int16) (uint16, bool) {
	return bsp.findNode(x, y, bsp.root, bsp.root)
}

// findNode recursively searches for a node in the BSP tree based on the given coordinates and node indices.
// Returns the parent node index and a boolean indicating if a valid node was found.
func (bsp *BSP) findNode(x int16, y int16, parentNodeIdx uint16, nodeIdx uint16) (uint16, bool) {
	if nodeIdx&subSectorBit == subSectorBit {
		return parentNodeIdx, true
	}
	node := bsp.level.Nodes[nodeIdx]
	if child, ok := node.Intersect(x, y); ok {
		return bsp.findNode(x, y, nodeIdx, child)
	}
	return 0, false
}

// FindRect traverses the BSP tree to locate the bounding box containing the specified point (x, y).
// Returns the found bounding box and a boolean indicating success.
func (bsp *BSP) FindRect(x int16, y int16) (lumps2.BBox, bool) {
	return bsp.findRect(x, y, lumps2.BBox{}, bsp.root)
}

// findRect traverses the BSP tree to locate the bounding box that contains the specified point (x, y).
func (bsp *BSP) findRect(x int16, y int16, b lumps2.BBox, nodeIdx uint16) (lumps2.BBox, bool) {
	if nodeIdx&subSectorBit == subSectorBit {
		return b, true
	}
	node := bsp.level.Nodes[nodeIdx]
	if node.BBox[0].Intersect(x, y) {
		return bsp.findRect(x, y, node.BBox[0], node.Child[0])
	}
	if node.BBox[1].Intersect(x, y) {
		return bsp.findRect(x, y, node.BBox[1], node.Child[1])
	}
	return lumps2.BBox{}, false
}

// traversePoint traverses the BSP tree to find sub-sectors based on a given point, excluding a specific sub-sector.
func (bsp *BSP) traversePoint(x1 int16, y1 int16, idx uint16, exclude uint16, res map[uint16]bool) {
	if idx&subSectorBit == subSectorBit {
		subSectorId := idx & ^subSectorBit
		if subSectorId != exclude {
			res[subSectorId] = true
		}
		return
	}
	node := bsp.level.Nodes[idx]
	if child, ok := node.Intersect(x1, y1); ok {
		bsp.traversePoint(x1, y1, child, exclude, res)
	}
	return
}

// findPointInSubSector identifies sub-sectors containing the specified point while excluding a given sub-sector.
func (bsp *BSP) findPointInSubSector(x1 int16, y1 int16, exclude uint16, res map[uint16]bool) {
	for _, n := range bsp.level.Nodes {
		if child, ok := n.Intersect(x1, y1); ok {
			bsp.traversePoint(x1, y1, child, exclude, res)
		}
	}
}

// FindOppositeSubSectorByPoints calculates opposite subsectors related to a given segment in the BSP tree structure.
// It considers margin adjustments, segments alignment, and potential wall sectors to generate a list of new input segments.
func (bsp *BSP) FindOppositeSubSectorByPoints(subSectorId uint16, is2 *model.ConfigSegment, wallSectors map[uint16]bool) []*model.ConfigSegment {
	const margin = 2
	x1 := int16(is2.Start.X)
	y1 := int16(is2.Start.Y)
	x2 := int16(is2.End.X)
	y2 := int16(is2.End.Y)
	xDif := float64(x2-x1) / 2
	yDif := float64(y2-y1) / 2
	//debug := subSectorId == 15 && x1 == 1992  && y1 == 2552 && x2 == 1784 && y2 == 2552
	//debug := subSectorId == 102
	debug := subSectorId == 96
	rl := bsp.describeLineF(float64(x1), float64(y1), float64(x2), float64(y2))
	rlStart := is2.Start //rl[0]
	rlEnd := is2.End     //rl[len(rl) - 1]
	//rlStart := rl[0]
	//rlEnd := rl[len(rl) - 1]

	if debug {
		fmt.Println("DEBUG IS STARTING")
	}

	factor := 1.0

	if x1 == x2 || y1 == y2 {
		factor = 5.0
		rl = rl[margin : len(rl)-margin]
	} else {

		//la linea deve essere ortogonale, non perpendicolare!!!!!!
		//funzionano anche le linee del rect del bbox...
		factor = 5.0
		//TODO COSA FARE PER I SEGMENTI OBLIQUI?
		//rl = []PointFloat{ rl[len(rl) / 2]}
		rl = rl[margin : len(rl)-margin]
	}

	var ret []*model.ConfigSegment

	addSegment := func(sId uint16, xy PointFloat) {
		id := strconv.Itoa(int(sId))
		kind := model.DefinitionJoin
		if _, ok := wallSectors[sId]; ok {
			id = "wall"
			kind = model.DefinitionWall
		}

		update := 0
		if len(ret) == 0 {
			update = 1
		} else if ret[len(ret)-1].Neighbor != id {
			prevSegment := ret[len(ret)-1]
			prevSegment.End.X = xy.X
			prevSegment.End.Y = xy.Y
			prevSegment.Tag += fmt.Sprintf(" - CREATED %0.f:%0.f", prevSegment.End.X-prevSegment.Start.X, prevSegment.End.Y-prevSegment.Start.Y)
			update = 2
		}
		if update > 0 {
			cloned := is2.Clone()
			cloned.Neighbor = id
			cloned.Kind = kind
			cloned.End = model.XY{}
			if update == 1 {
				cloned.Start.X = rlStart.X
				cloned.Start.Y = rlStart.Y
			} else if update == 2 {
				cloned.Start.X = xy.X
				cloned.Start.Y = xy.Y
			}
			ret = append(ret, cloned)
		}
	}

	for x := 0; x < len(rl); x++ {
		src := rl[x]
		a1 := src.X - (yDif * factor)
		b1 := src.Y + (xDif * factor)
		a2 := src.X + (yDif * factor)
		b2 := src.Y - (xDif * factor)

		if debug && x == 0 {
			fmt.Println("TEST")
		}

		perp := bsp.describeLineF(a1, b1, a2, b2)

		center := -1
		for idx, xy := range perp {
			if xy.X == src.X && xy.Y == src.Y {
				center = idx
			}
		}

		if center == -1 {
			center = len(perp) / 2
			//fmt.Println("CAN'T FIND CENTER!!!!!!", src.X, src.Y, "|", perp[center].X, perp[center].Y)
		}

		if debug {
			fmt.Println("------ PERP -------", x, "len:", len(rl), "[", x1, y1, x2, y2, "]", "[", perp[0].X, perp[0].Y, perp[len(perp)-1].X, perp[len(perp)-1].Y, "]")
			//fmt.Println("------ PERP -------", x, src)
			//for _, test := range perp {
			//	fmt.Println(test)
			//}
		}

		//center := len(perp) / 2
		d2 := center - 1

		for d1 := center; d1 < len(perp); d1++ {
			left := perp[d1]
			right := perp[d2]
			if leftSS, ok := bsp.FindSubSectorForced(int16(left.X), int16(-left.Y)); ok {
				if leftSS != subSectorId {
					if debug {
						fmt.Printf(" %0.f:%0.f %d - LEFT \n", left.X, left.Y, leftSS)
					}
					addSegment(leftSS, src)
					break
				} else {
					if debug {
						fmt.Printf(" %0.f:%0.f - SAME ON LEFT\n", left.X, left.Y)
					}
				}
			} else {
				if debug {
					fmt.Println("CRITICAL ERROR, NOT FOUND ON LEFT", left.X, -left.Y)
				}
			}
			if rightSS, ok := bsp.FindSubSectorForced(int16(right.X), int16(-right.Y)); ok {
				if rightSS != subSectorId {
					if debug {
						fmt.Printf("%0.f:%0.f %d - RIGHT\n", right.X, right.Y, rightSS)
					}
					addSegment(rightSS, src)
					break
				} else {
					if debug {
						fmt.Printf(" %0.f:%0.f - SAME ON RIGHT \n", right.X, right.Y)
					}
				}
			} else {
				if debug {
					fmt.Println("CRITICAL ERROR, NOT FOUND ON RIGHT", left.X, -left.Y)
				}
			}
			d2--
			if d2 < 0 {
				break
			}
		}
	}
	if len(ret) > 0 {
		lastSegment := ret[len(ret)-1]
		if lastSegment.Start.X == rlEnd.X && lastSegment.Start.Y == rlEnd.Y {
			ret = ret[:len(ret)-1]
		} else {
			lastSegment.End.X = rlEnd.X
			lastSegment.End.Y = rlEnd.Y
			lastSegment.Tag += fmt.Sprintf(" - CREATED %0.f:%0.f", lastSegment.End.X-lastSegment.Start.X, lastSegment.End.Y-lastSegment.Start.Y)
		}
	} else {
		fmt.Println("NOT FOUND!!!!")
	}

	if debug {
		//os.Exit(-1)
	}
	return ret
}

// FindOppositeSubSectorByHulls identifies opposite subsectors based on input segment hulls and specific wall sector mappings.
// It processes segment lines to detect adjacency and connection, managing overlaps and intersections through margin adjustments.
func (bsp *BSP) FindOppositeSubSectorByHulls(subSectorId uint16, is2 *model.ConfigSegment, wallSectors map[uint16]bool) []*model.ConfigSegment {
	const margin = 2
	x1 := int16(is2.Start.X)
	y1 := int16(is2.Start.Y)
	x2 := int16(is2.End.X)
	y2 := int16(is2.End.Y)
	debug := subSectorId == 96
	rl := bsp.describeLineF(float64(x1), float64(y1), float64(x2), float64(y2))
	rlStart := is2.Start
	rlEnd := is2.End

	if debug {
		fmt.Println("DEBUG IS STARTING")
	}

	rl = rl[margin : len(rl)-margin]

	var ret []*model.ConfigSegment

	addSegment := func(sId uint16, xy PointFloat) {
		id := strconv.Itoa(int(sId))
		kind := model.DefinitionJoin
		if _, ok := wallSectors[sId]; ok {
			id = "wall"
			kind = model.DefinitionWall
		}

		update := 0
		if len(ret) == 0 {
			update = 1
		} else if ret[len(ret)-1].Neighbor != id {
			prevSegment := ret[len(ret)-1]
			prevSegment.End.X = xy.X
			prevSegment.End.Y = xy.Y
			prevSegment.Tag += fmt.Sprintf(" - CREATED %0.f:%0.f", prevSegment.End.X-prevSegment.Start.X, prevSegment.End.Y-prevSegment.Start.Y)
			update = 2
		}
		if update > 0 {
			cloned := is2.Clone()
			cloned.Neighbor = id
			cloned.Kind = kind
			cloned.End = model.XY{}
			if update == 1 {
				cloned.Start.X = rlStart.X
				cloned.Start.Y = rlStart.Y
			} else if update == 2 {
				cloned.Start.X = xy.X
				cloned.Start.Y = xy.Y
			}
			ret = append(ret, cloned)
		}
	}

	for x := 0; x < len(rl); x++ {
		src := rl[x]

		addSegment(0, src)
	}

	if len(ret) > 0 {
		lastSegment := ret[len(ret)-1]
		if lastSegment.Start.X == rlEnd.X && lastSegment.Start.Y == rlEnd.Y {
			ret = ret[:len(ret)-1]
		} else {
			lastSegment.End.X = rlEnd.X
			lastSegment.End.Y = rlEnd.Y
			lastSegment.Tag += fmt.Sprintf(" - CREATED %0.f:%0.f", lastSegment.End.X-lastSegment.Start.X, lastSegment.End.Y-lastSegment.Start.Y)
		}
	} else {
		fmt.Println("NOT FOUND!!!!")
	}

	if debug {
		//os.Exit(-1)
	}
	return ret
}

// describeCircle generates the coordinates of points forming a circle centered at (x0, y0) with the given radius.
func (bsp *BSP) describeCircle(x0 float64, y0 float64, radius float64) []PointFloat {
	var res []PointFloat
	x := radius
	y := float64(0)
	radiusError := 1.0 - x
	for y <= x {
		res = append(res, PointFloat{x + x0, y + y0})
		res = append(res, PointFloat{x + x0, y + y0})
		res = append(res, PointFloat{y + x0, x + y0})
		res = append(res, PointFloat{-x + x0, y + y0})
		res = append(res, PointFloat{-y + x0, x + y0})
		res = append(res, PointFloat{-x + x0, -y + y0})
		res = append(res, PointFloat{-y + x0, -x + y0})
		res = append(res, PointFloat{x + x0, -y + y0})
		res = append(res, PointFloat{y + x0, -x + y0})
		y++
		if radiusError < 0 {
			radiusError += 2*y + 1
		} else {
			x--
			radiusError += 2 * (y - x + 1)
		}
	}
	return res
}

// describeLineF generates a list of points representing a line from (x0, y0) to (x1, y1) using Bresenham's algorithm.
func (bsp *BSP) describeLineF(x0 float64, y0 float64, x1 float64, y1 float64) []PointFloat {
	high := func(x0 float64, y0 float64, x1 float64, y1 float64) []PointFloat {
		var res []PointFloat
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
			res = append(res, PointFloat{X: math.Round(x), Y: math.Round(y)})
			if D > 0 {
				x = x + xi
				D = D + (2 * (dx - dy))
			} else {
				D = D + 2*dx
			}
		}
		return res
	}

	low := func(x0 float64, y0 float64, x1 float64, y1 float64) []PointFloat {
		var res []PointFloat
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
			res = append(res, PointFloat{X: math.Round(x), Y: math.Round(y)})
			if D > 0 {
				y = y + yi
				D = D + (2 * (dy - dx))
			} else {
				D = D + 2*dy
			}
		}
		return res
	}

	var res []PointFloat
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

	var r []PointFloat
	for x := len(res) - 1; x >= 0; x-- {
		r = append(r, res[x])
	}
	return r
}
