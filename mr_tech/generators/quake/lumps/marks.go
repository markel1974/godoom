package lumps

import (
	"encoding/binary"
	"io"
)

type Marks struct {
	Count    int
	Surfaces []uint16
}

func NewMarks(rs io.ReadSeeker, lumpInfo *LumpInfo) (*Marks, error) {
	if err := Seek(rs, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	markCount := int(lumpInfo.Size) / 2
	markSurfaces := make([]uint16, markCount)
	if err := binary.Read(rs, binary.LittleEndian, markSurfaces); err != nil {
		return nil, err
	}
	marks := &Marks{
		Count:    markCount,
		Surfaces: markSurfaces,
	}
	return marks, nil
}
