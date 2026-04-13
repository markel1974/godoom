package lumps

import (
	"encoding/binary"
	"io"
	"unsafe"
)

// Node represents a binary space partitioning node in Quake 3D space.
// It uses a reference to a plane to split the space into two halves (children).
type Node struct {
	PlaneID   uint32   // Index into the PLANES lump (Lump 1)
	Children  [2]int16 // If >= 0: Node index. If < 0: Bitwise inverted Leaf index (~child)
	Mins      [3]int16 // Bounding box min (X, Y, Z) for rapid culling
	Maxs      [3]int16 // Bounding box max (X, Y, Z) for rapid culling
	FirstFace uint16   // Index into the FACES lump
	NumFaces  uint16   // Number of faces associated with this node
}

// NewNodes reads node data from the provided file and returns a slice of pointers to Node structures.
func NewNodes(rs io.ReadSeeker, lumpInfo *LumpInfo) ([]*Node, error) {
	if err := Seek(rs, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	var pNode Node
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pNode))
	pNodes := make([]Node, count)

	if err := binary.Read(rs, binary.LittleEndian, pNodes); err != nil {
		return nil, err
	}

	nodes := make([]*Node, count)
	for idx, n := range pNodes {
		nodes[idx] = &Node{
			PlaneID:   n.PlaneID,
			Children:  n.Children,
			Mins:      n.Mins,
			Maxs:      n.Maxs,
			FirstFace: n.FirstFace,
			NumFaces:  n.NumFaces,
		}
	}
	return nodes, nil
}

// IsChildLeaf checks if a specific child index represents a leaf node.
// In Quake, a negative child index signifies a leaf.
func IsChildLeaf(child int16) bool {
	return child < 0
}

// GetChildLeafIndex converts a negative child index into the actual leaf index.
// It applies a bitwise NOT (complemento a 1) to resolve the correct index in the leaves array.
func GetChildLeafIndex(child int16) int {
	return int(^child)
}

// Intersect3D checks if a 3D point (x, y, z) lies strictly inside the node's bounding box.
func (n *Node) Intersect3D(x, y, z int16) bool {
	return x >= n.Mins[0] && x <= n.Maxs[0] &&
		y >= n.Mins[1] && y <= n.Maxs[1] &&
		z >= n.Mins[2] && z <= n.Maxs[2]
}
