package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// Vertex represents a point in a 2D coordinate system with integer precision.
// XCoord specifies the horizontal position.
// YCoord specifies the vertical position.
type Vertex struct {
	XCoord int16
	YCoord int16
}

// NewVertexes reads vertex data from the given file and lump information and returns an array of vertex pointers or an error.
func NewVertexes(f *os.File, lumpInfo *LumpInfo) ([]*Vertex, error) {
	var pVertex Vertex
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pVertex))
	pVertexes := make([]Vertex, count, count)
	if err := binary.Read(f, binary.LittleEndian, pVertexes); err != nil {
		return nil, err
	}
	vertexes := make([]*Vertex, count, count)
	for idx, v := range pVertexes {
		vertexes[idx] = &Vertex{
			XCoord: v.XCoord,
			YCoord: v.YCoord,
		}
	}
	return vertexes, nil
}
