package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

type Seg struct {
	VertexStart   int16
	VertexEnd     int16
	Bams          int16
	LineNum       int16
	SegmentSide   int16
	SegmentOffset int16
}


func NewSegments(f * os.File, lumpInfo *LumpInfo) ([]*Seg, error) {
	var pSeg Seg
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSeg))
	pSegments := make([]Seg, count, count)
	if err := binary.Read(f, binary.LittleEndian, pSegments); err != nil {
		return nil, err
	}
	segments := make([]*Seg, count, count)
	for idx, s := range pSegments {
		segments[idx] = &Seg{
			VertexStart:   s.VertexStart,
			VertexEnd:     s.VertexEnd,
			Bams:          s.Bams,
			LineNum:       s.LineNum,
			SegmentSide:   s.SegmentSide,
			SegmentOffset: s.SegmentOffset,
		}
	}
	return segments, nil
}

