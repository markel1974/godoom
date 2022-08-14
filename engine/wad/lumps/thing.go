package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

type Thing struct {
	X       int16
	Y       int16
	Angle   int16
	Type    int16
	Options int16
}



func NewThings(f * os.File, lumpInfo *LumpInfo) ([]*Thing, error) {
	var pThing Thing
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pThing))
	pThings := make([]Thing, count, count)
	if err := binary.Read(f, binary.LittleEndian, pThings); err != nil {
		return nil, err
	}
	things := make([]*Thing, count, count)
	for idx, t := range pThings {
		things[idx] = &Thing{
			X:       t.X,
			Y:       t.Y,
			Angle:   t.Angle,
			Type:    t.Type,
			Options: t.Options,
		}
	}
	return things, nil
}