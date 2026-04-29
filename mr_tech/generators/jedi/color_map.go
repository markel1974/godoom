package jedi

import (
	"encoding/binary"
	"io"
)

// ColorMap contiene i 64 livelli di shading. 0 = Full Bright, 63 = Pitch Black.
type ColorMap struct {
	LightMaps [64][256]byte
}

// NewColorMap creates and returns a new instance of ColorMap with default values.
func NewColorMap() *ColorMap {
	return &ColorMap{}
}

// Parse reads and populates the LightMaps field of the ColorMap from the provided io.Reader.
// It expects the data to be in Little Endian binary format and returns an error if the reading fails.
func (cmp *ColorMap) Parse(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &cmp.LightMaps); err != nil {
		return err
	}
	// (Omette l'estrazione delle mappe di tinting per i visori/gas/danni per brevità)
	return nil
}
