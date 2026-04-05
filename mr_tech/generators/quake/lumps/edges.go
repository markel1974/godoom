package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// Edge represents a directed line segment between two vertices in 3D space.
type Edge struct {
	Vertex0 uint16
	Vertex1 uint16
}

// NewEdges reads and parses the edges lump data from a file.
func NewEdges(f *os.File, lumpInfo *LumpInfo) ([]*Edge, error) {
	var pEdge Edge
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pEdge))
	pEdges := make([]Edge, count)
	if err := binary.Read(f, binary.LittleEndian, pEdges); err != nil {
		return nil, err
	}
	edges := make([]*Edge, count)
	for idx, e := range pEdges {
		edges[idx] = &Edge{
			Vertex0: e.Vertex0,
			Vertex1: e.Vertex1,
		}
	}
	return edges, nil
}
