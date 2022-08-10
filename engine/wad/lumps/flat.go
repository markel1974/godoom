package lumps

import (
	"encoding/binary"
	"os"
)

type Flat struct {
	Data []byte
}

func NewFlat(f * os.File, lumpInfo *LumpInfo) (*Flat, error){
	if err := Seek(f, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	size := 4096
	data := make([]byte, size, size)
	if err := binary.Read(f, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	return &Flat{Data: data}, nil
}