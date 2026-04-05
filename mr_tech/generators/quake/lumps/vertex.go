package lumps

import (
	"encoding/binary"
	"os"
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
func NewVertexes(f *os.File, lumpInfo *LumpInfo) ([]*Vertex, error) {
	// IMPORTANTE: Assicurarsi che il chiamante (il parser BSP) abbia già fatto
	// il Seek all'offset corretto indicato dal lumpInfo prima di chiamare questa funzione!

	var pVertex Vertex
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pVertex))
	pVertexes := make([]Vertex, count)

	if err := binary.Read(f, binary.LittleEndian, pVertexes); err != nil {
		return nil, err
	}

	vertexes := make([]*Vertex, count)
	for idx, v := range pVertexes {
		// Ricreiamo i puntatori proprio come facevi per Doom
		vertexes[idx] = &Vertex{
			X: v.X,
			Y: v.Y,
			Z: v.Z,
		}
	}
	return vertexes, nil
}
