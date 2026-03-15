package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// SubSector represents a partition of a sector in a level, defined by a number of segments and a starting segment index.
type SubSector struct {
	NumSegments int16
	StartSeg    int16
}

// NewSubSectors reads and parses subsectors from a file based on the provided lump information, returning a slice of SubSector pointers.
func NewSubSectors(f *os.File, lumpInfo *LumpInfo) ([]*SubSector, error) {
	var pSubSector SubSector
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSubSector))
	pSubSectors := make([]SubSector, count, count)
	if err := binary.Read(f, binary.LittleEndian, pSubSectors); err != nil {
		return nil, err
	}
	subSectors := make([]*SubSector, count, count)
	for idx, s := range pSubSectors {
		subSectors[idx] = &SubSector{
			NumSegments: s.NumSegments,
			StartSeg:    s.StartSeg,
		}
	}
	return subSectors, nil
}
