package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

type Vertex struct {
	XCoord int16
	YCoord int16
}



func NewVertexes(f * os.File, lumpInfo *LumpInfo) ([]*Vertex, error) {
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
