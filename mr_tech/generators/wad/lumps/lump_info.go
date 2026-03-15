package lumps

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Header represents the file header containing metadata for lump management in a binary format.
// Magic is a 4-byte identifier used to validate the file type.
// NumLumps specifies the total number of lumps in the file.
// InfoTableOfs indicates the offset to the table containing lump information.
type Header struct {
	Magic        [4]byte
	NumLumps     int32
	InfoTableOfs int32
}

// LumpInfo represents metadata about a lump, including its file position, size, and name.
type LumpInfo struct {
	Filepos int64
	Size    int32
	Name    string
}

// NewLumpInfo creates and returns a new LumpInfo structure populated with the given file position, size, and name.
func NewLumpInfo(pos int64, size int32, name string) *LumpInfo {
	return &LumpInfo{
		Filepos: pos,
		Size:    size,
		Name:    name,
	}
}

// NewLumpInfos reads lump information from a WAD file and returns a slice of LumpInfo or an error if parsing fails.
func NewLumpInfos(f *os.File) ([]*LumpInfo, error) {
	header := &Header{}
	if err := binary.Read(f, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	if string(header.Magic[:]) != "IWAD" {
		return nil, fmt.Errorf("bad magic: %s\n", header.Magic)
	}

	if err := Seek(f, int64(header.InfoTableOfs)); err != nil {
		return nil, err
	}

	type PrivateLumpInfo struct {
		Filepos int32
		Size    int32
		Name    [8]byte
	}
	lumpInfos := make([]*LumpInfo, header.NumLumps, header.NumLumps)
	for i := int32(0); i < header.NumLumps; i++ {
		p := &PrivateLumpInfo{}
		if err := binary.Read(f, binary.LittleEndian, p); err != nil {
			return nil, err
		}
		lumpInfos[i] = NewLumpInfo(int64(p.Filepos), p.Size, ToString(p.Name))
	}
	return lumpInfos, nil
}
