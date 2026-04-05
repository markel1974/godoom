package lumps

import (
	"encoding/binary"
	"os"
)

// NewSurfEdges reads an array of int32 representing directed edge indices.
// A negative value indicates the edge's vertices should be read in reverse order (Winding Order).
func NewSurfEdges(f *os.File, lumpInfo *LumpInfo) ([]int32, error) {
	count := int(lumpInfo.Size) / 4
	surfEdges := make([]int32, count)

	if err := binary.Read(f, binary.LittleEndian, surfEdges); err != nil {
		return nil, err
	}

	return surfEdges, nil
}
