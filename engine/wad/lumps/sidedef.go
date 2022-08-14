package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

type SideDef struct {
	XOffset       int16
	YOffset       int16
	UpperTexture  string
	LowerTexture  string
	MiddleTexture string
	SectorRef     uint16
}

func NewSideDefs(f * os.File, lumpInfo *LumpInfo) ([]*SideDef, error) {
	type PrivateSideDef struct {
		XOffset       int16
		YOffset       int16
		UpperTexture  [8]byte
		LowerTexture  [8]byte
		MiddleTexture [8]byte
		SectorRef     uint16
	}
	var pSideDef PrivateSideDef
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSideDef))
	pSideDefs := make([]PrivateSideDef, count, count)
	if err := binary.Read(f, binary.LittleEndian, pSideDefs); err != nil {
		return nil, err
	}
	sideDef := make([]*SideDef, count, count)
	for idx, p := range pSideDefs {
		sideDef[idx] = &SideDef{
			XOffset:       p.XOffset,
			YOffset:       p.YOffset,
			UpperTexture:  ToString(p.UpperTexture),
			LowerTexture:  ToString(p.LowerTexture),
			MiddleTexture: ToString(p.MiddleTexture),
			SectorRef:     p.SectorRef,
		}
	}
	return sideDef, nil
}


func (s * SideDef) PrintTexture() string {
	return s.UpperTexture + " " + s.MiddleTexture + " " + s.LowerTexture
}