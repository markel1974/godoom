package lumps

import (
	"encoding/binary"
	"os"
)

// Flat represents a flat texture consisting of raw byte data, typically used in Doom-engine WAD files.
type Flat struct {
	Data []byte
}

// NewFlat reads flat texture data from the specified lump in the WAD file and returns a Flat instance.
func NewFlat(f *os.File, lumpInfo *LumpInfo) (*Flat, error) {
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
