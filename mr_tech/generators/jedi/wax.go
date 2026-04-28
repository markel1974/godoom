package jedi

import (
	"encoding/binary"
	"io"
)

// WaxHeader represents the structure of the header used in a WAX file format.
// It includes the magic number, version, and sequence offsets.
type WaxHeader struct {
	Magic      [4]byte // "WAX "
	Version    uint32  // Tipicamente 0x00010001
	SeqOffsets [32]uint32
}

// WaxSequence represents a data structure containing padding bytes and frame offsets for sequential processing.
type WaxSequence struct {
	Padding      [16]byte
	FrameOffsets [32]uint32
}

// WaxFrame represents a frame structure used for positioning and rendering graphical elements.
// Contains metadata such as insertion point, flipping, cell offset, size dimensions, and padding.
type WaxFrame struct {
	InsertX    int32
	InsertY    int32
	Flip       int32
	CellOffset uint32
	UnitWidth  uint32
	UnitHeight uint32
	Pad        [8]byte
}

// WaxCellHeader represents the metadata header for a WaxCell structure, providing layout and compression details.
type WaxCellHeader struct {
	SizeX      uint32
	SizeY      uint32
	Compressed uint32
	DataSize   uint32
	ColOffsets uint32 // Offset interno all'array di colonne
	Padding    [12]byte
}

// ParseWaxCell reads and parses a WaxCellHeader from the given io.ReadSeeker, starting at the specified offset.
func ParseWaxCell(r io.ReadSeeker, offset int64) (*WaxCellHeader, error) {
	r.Seek(offset, io.SeekStart)
	header := &WaxCellHeader{}
	err := binary.Read(r, binary.LittleEndian, header)
	// (Implementazione decompressore Column-Major con Color-Key transparent omission)
	return header, err
}
