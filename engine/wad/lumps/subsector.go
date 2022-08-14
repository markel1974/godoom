package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

type SubSector struct {
	NumSegments int16
	StartSeg    int16
}


func NewSubSectors(f * os.File, lumpInfo *LumpInfo) ([]*SubSector, error) {
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