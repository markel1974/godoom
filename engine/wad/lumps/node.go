package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

type Node struct {
	X     int16
	Y     int16
	DX    int16
	DY    int16
	BBox  [2]BBox
	Child [2]uint16
}

type BBox struct {
	Top    int16
	Bottom int16
	Left   int16
	Right  int16
}


func (b * BBox) Intersect(x int16, y int16) bool {
	return x >= b.Left && x <= b.Right && y >= b.Bottom && y <= b.Top
}

func (b * BBox) IntersectInside(x int16, y int16) bool {
	return x > b.Left && x < b.Right && y > b.Bottom && y < b.Top
}

func NewNodes(f * os.File, lumpInfo *LumpInfo) ([]*Node, error) {
	var pNode Node
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pNode))
	pNodes := make([]Node, count, count)
	if err := binary.Read(f, binary.LittleEndian, pNodes); err != nil {
		return nil, err
	}
	nodes := make([]*Node, count, count)
	for idx, n := range pNodes {
		nodes[idx] = &Node{
			X:     n.X,
			Y:     n.Y,
			DX:    n.DX,
			DY:    n.DY,
			BBox:  n.BBox,
			Child: n.Child,
		}
	}
	return nodes, nil
}


func (n * Node) Print() {
	/*
	Each node is 28 bytes in 14 <short> fields:

	(1)  X coordinate of partition line's start
	(2)  Y coordinate of partition line's start
	(3)  DX, change in X to end of partition line
	(4)  DY, change in Y to end of partition line

	If (1) to (4) equaled 64, 128, -64, -64, the partition line would go from (64, 128) to (0, 64).

	(5)  Y upper bound for right bounding-box.\
	(6)  Y lower bound                         All SEGS in right child of node
	(7)  X lower bound                         must be within this box.
	(8)  X upper bound                        /

	(9)  Y upper bound for left bounding box. \
	(10) Y lower bound                         All SEGS in left child of node
	(11) X lower bound                         must be within this box.
	(12) X upper bound                        /

	(13) a NODE or SSECTOR number for the right child. If bit 15 of this
	<short> is set, then the rest of the number represents the
	child SSECTOR. If not, the child is a recursed node.
	(14) a NODE or SSECTOR number for the left child.

	 */
}

func (n *Node) PointOnSide(x int16, y int16) int {
	dx := int(x) - int(n.X)
	dy := int(y) - int(n.Y)
	// Perp dot product:
	left := (int(n.DY) >> 16) * dx
	right := (int(n.DX) >> 16) * dy
	if right < left {
		// Point is on front side:
		return 0
	}
	// Point is on the back side:
	return 1
}

func (n * Node) Intersect(x int16, y int16) (uint16, bool) {
	if n.BBox[0].Intersect(x, y) {
		return n.Child[0], true
		//return bsp.findSubSector(x, y, int(node.Child[0]))
		//return 0
	}
	if n.BBox[1].Intersect(x, y) {
		return n.Child[1], true
		//return bsp.findSubSector(x, y, int(node.Child[1]))
	}
	return 0, false
}

func (n * Node) IntersectSegment(x1 int16, y1 int16, x2 int16, y2 int16) (uint16, bool) {
	if n.BBox[0].Intersect(x1, y1) && n.BBox[0].Intersect(x2, y2) {
		return n.Child[0], true
		//return bsp.findSubSector(x, y, int(node.Child[0]))
		//return 0
	}
	if n.BBox[1].Intersect(x1, y1) && n.BBox[1].Intersect(x2, y2) {
		return n.Child[1], true
		//return bsp.findSubSector(x, y, int(node.Child[1]))
	}
	return 0, false
}


func (n * Node) IntersectInside(x int16, y int16) (uint16, bool) {
	if n.BBox[0].IntersectInside(x, y) {
		return n.Child[0], true
		//return bsp.findSubSector(x, y, int(node.Child[0]))
		//return 0
	}
	if n.BBox[1].IntersectInside(x, y) {
		return n.Child[1], true
		//return bsp.findSubSector(x, y, int(node.Child[1]))
	}
	return 0, false
}