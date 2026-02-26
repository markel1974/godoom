package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// Node represents a binary space partitioning node with partition line coordinates, bounding boxes, and child references.
type Node struct {
	X     int16
	Y     int16
	DX    int16
	DY    int16
	BBox  [2]BBox
	Child [2]uint16
}

// BBox represents a rectangular bounding box defined by its top, bottom, left, and right edges.
type BBox struct {
	Top    int16
	Bottom int16
	Left   int16
	Right  int16
}

// Intersect determines if the point (x, y) lies within the bounding box, including its edges.
func (b *BBox) Intersect(x int16, y int16) bool {
	return x >= b.Left && x <= b.Right && y >= b.Bottom && y <= b.Top
}

// IntersectInside checks if a point (x, y) is strictly inside the bounding box, excluding boundaries.
func (b *BBox) IntersectInside(x int16, y int16) bool {
	return x > b.Left && x < b.Right && y > b.Bottom && y < b.Top
}

// NewNodes reads node data from the provided file and returns a slice of pointers to Node structures or an error.
func NewNodes(f *os.File, lumpInfo *LumpInfo) ([]*Node, error) {
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

// Print outputs the contents of a Node, including its partition line and bounding box details.
func (n *Node) Print() {
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

// PointOnSide determines whether the given point (x, y) lies on the front (0) or back (1) side of the partition line.
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

// Intersect checks if the given (x, y) point lies within either of the bounding boxes of the node.
// It returns the child node index and a boolean indicating if an intersection was found.
func (n *Node) Intersect(x int16, y int16) (uint16, bool) {
	if n.BBox[0].Intersect(x, y) {
		//fmt.Println(n.BBox[0].Top, n.BBox[0].Left, n.BBox[0].Bottom, n.BBox[0].Right)
		return n.Child[0], true
		//return bsp.findSubSector(x, y, int(node.Child[0]))
		//return 0
	}
	if n.BBox[1].Intersect(x, y) {
		//fmt.Println(n.BBox[1].Top, n.BBox[1].Left, n.BBox[1].Bottom, n.BBox[1].Right)
		return n.Child[1], true
		//return bsp.findSubSector(x, y, int(node.Child[1]))
	}
	return 0, false
}

// IntersectSegment checks if a line segment defined by (x1, y1) and (x2, y2) intersects the bounding boxes of the node.
// Returns the child index of the intersecting bounding box and a boolean indicating intersection status.
func (n *Node) IntersectSegment(x1 int16, y1 int16, x2 int16, y2 int16) (uint16, bool) {
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

// IntersectInside checks if the given (x, y) coordinates lie strictly inside the bounding boxes of the node and returns the corresponding child index along with a boolean indicating success.
func (n *Node) IntersectInside(x int16, y int16) (uint16, bool) {
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
