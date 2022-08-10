package lumps

import (
	"encoding/binary"
	"fmt"
	"os"
)

type Header struct {
	Magic        [4]byte
	NumLumps     int32
	InfoTableOfs int32
}

type LumpInfo struct {
	Filepos int64
	Size    int32
	Name    string
}

func NewLumpInfo(pos int64, size int32, name string) *LumpInfo{
	return &LumpInfo{
		Filepos: pos,
		Size:    size,
		Name:    name,
	}
}

func NewLumpInfos(f * os.File) ([]*LumpInfo, error) {
	header := &Header{}
	if err := binary.Read(f, binary.LittleEndian, header); err != nil { return nil, err }
	if string(header.Magic[:]) != "IWAD" { return nil, fmt.Errorf("bad magic: %s\n", header.Magic) }

	if err := Seek(f, int64(header.InfoTableOfs)); err != nil { return nil, err }

	type PrivateLumpInfo struct {
		Filepos int32
		Size    int32
		Name    [8]byte
	}
	lumpInfos := make([]*LumpInfo, header.NumLumps, header.NumLumps)
	for i := int32(0); i < header.NumLumps; i++ {
		p := &PrivateLumpInfo{}
		if err := binary.Read(f, binary.LittleEndian, p); err != nil { return nil, err }
		lumpInfos[i] = NewLumpInfo(int64(p.Filepos), p.Size, ToString(p.Name))
	}
	return lumpInfos, nil
}