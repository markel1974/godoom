package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

type Sector struct {
	FloorHeight   int16
	CeilingHeight int16
	FloorPic      string
	CeilingPic    string
	LightLevel    int16
	SpecialSector int16
	Tag           int16
}


func NewSectors(f *os.File, lumpInfo *LumpInfo) ([]*Sector, error) {
	type privateSector struct {
		FloorHeight   int16
		CeilingHeight int16
		FloorPic      [8]byte
		CeilingPic    [8]byte
		LightLevel    int16
		SpecialSector int16
		Tag           int16
	}
	var pSector privateSector
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSector))
	pSectors := make([]privateSector, count, count)
	if err := binary.Read(f, binary.LittleEndian, pSectors); err != nil {
		return nil, err
	}
	sectors := make([]*Sector, count, count)
	for idx, p := range pSectors {
		sectors[idx] = &Sector{
			FloorHeight:   p.FloorHeight,
			CeilingHeight: p.CeilingHeight,
			FloorPic:      ToString(p.FloorPic),
			CeilingPic:    ToString(p.CeilingPic),
			LightLevel:    p.LightLevel,
			SpecialSector: p.SpecialSector,
			Tag:           p.Tag,
		}
	}
	return sectors, nil
}

