package lumps

import (
	"encoding/binary"
	"io"
)

// NewSurfEdges reads an array of int32 representing directed edge indices.
// A negative value indicates the edge's vertices should be read in reverse order (Winding Order).
func NewSurfEdges(rs io.ReadSeeker, lumpInfo *LumpInfo) ([]int32, error) {
	if err := Seek(rs, lumpInfo.Filepos); err != nil {
		return nil, err
	}

	count := int(lumpInfo.Size) / 4
	surfEdges := make([]int32, count)

	if err := binary.Read(rs, binary.LittleEndian, surfEdges); err != nil {
		return nil, err
	}

	return surfEdges, nil
}
