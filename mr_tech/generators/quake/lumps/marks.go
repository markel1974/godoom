package lumps

import (
	"encoding/binary"
	"io"
)

// Marks encapsulates information about the count of surfaces and their indices in a level's data.
// Count specifies the number of surfaces.
// Surfaces holds the indices of surfaces as a slice of uint16.
type Marks struct {
	Count    int
	Surfaces []uint16
}

// NewMarks initializes and returns a Marks object by reading data from the provided ReadSeeker and LumpInfo metadata.
// Returns an error if seeking or data reading fails.
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
