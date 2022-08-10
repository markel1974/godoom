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
	Child [2]int16
}

type BBox struct {
	Top    int16
	Bottom int16
	Left   int16
	Right  int16
}


func (b * BBox) Intersect(x int16, y int16) bool {
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
