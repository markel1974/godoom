package lumps

import (
	"bytes"
	"fmt"
	"image/color"
)

// Palettes represents a collection of 256-color palettes and allows managing palette data and active indices.
type Palettes struct {
	colorPal      [][256]color.RGBA
	colorPalIndex int
}

// NewPalettes initializes a Palettes instance with the specified number of color palettes.
func NewPalettes(size int) *Palettes {
	return &Palettes{
		colorPalIndex: 0,
		colorPal:      make([][256]color.RGBA, size),
	}
}

// Set updates the palette at a given index with the provided palette data. Returns an error if the index is invalid or parsing fails.
func (pl *Palettes) Set(idx int, palData []byte) error {
	if idx < 0 || idx >= len(pl.colorPal) {
		return fmt.Errorf("invalid palette index %d", idx)
	}
	palette := NewPalette()
	colorPal, err := palette.ParseFromPCX(bytes.NewReader(palData))
	if err != nil {
		return fmt.Errorf("error while parsing palette %w", err)
	}
	pl.colorPal[idx] = colorPal
	return nil
}

// Get retrieves the active color palette based on the current palette index.
// Returns an array of 256 color.RGBA values and an error if the index is invalid.
func (pl *Palettes) Get() ([256]color.RGBA, error) {
	if pl.colorPalIndex < 0 || pl.colorPalIndex >= len(pl.colorPal) {
		return [256]color.RGBA{}, fmt.Errorf("invalid palette index %d", pl.colorPalIndex)
	}
	return pl.colorPal[pl.colorPalIndex], nil
}

// SetIndex sets the active palette index to the specified value if it is within the valid range of available palettes.
func (pl *Palettes) SetIndex(idx int) {
	if idx < 0 || idx >= len(pl.colorPal) {
		return
	}
	pl.colorPalIndex = idx
}
