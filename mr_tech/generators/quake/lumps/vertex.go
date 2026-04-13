package lumps

import (
	"encoding/binary"
	"io"
	"unsafe"
)

// Vertex represents a point in a 3D coordinate system with float precision.
// X, Y, and Z specify the position in 3D space.
type Vertex struct {
	X float32
	Y float32
	Z float32
}

// NewVertexes reads vertex data from the given file and lump information and returns an array of vertex pointers or an error.
func NewVertexes(r io.ReadSeeker, lumpInfo *LumpInfo) ([]*Vertex, error) {
	if err := Seek(r, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	var pVertex Vertex
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pVertex))
	pVertexes := make([]Vertex, count)

	if err := binary.Read(r, binary.LittleEndian, pVertexes); err != nil {
		return nil, err
	}

	vertexes := make([]*Vertex, count)
	for idx, v := range pVertexes {
		vertexes[idx] = &Vertex{X: v.X, Y: v.Y, Z: v.Z}
	}
	return vertexes, nil
}
