package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// Seg represents a segment in a level, defined by start and end vertices, direction, offset, and other spatial data.
type Seg struct {
	VertexStart int16
	VertexEnd   int16
	BAM         int16 //Binary Angle Measurement
	LineDef     int16
	Direction   int16
	Offset      int16
}

// NewSegments reads segment data from the given file based on LumpInfo and returns a slice of Seg pointers or an error.
func NewSegments(f *os.File, lumpInfo *LumpInfo) ([]*Seg, error) {
	var pSeg Seg
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSeg))
	pSegments := make([]Seg, count, count)
	if err := binary.Read(f, binary.LittleEndian, pSegments); err != nil {
		return nil, err
	}
	segments := make([]*Seg, count, count)
	for idx, s := range pSegments {
		segments[idx] = &Seg{
			VertexStart: s.VertexStart,
			VertexEnd:   s.VertexEnd,
			BAM:         s.BAM,
			LineDef:     s.LineDef,
			Direction:   s.Direction,
			Offset:      s.Offset,
		}
	}
	return segments, nil
}
