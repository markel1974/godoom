package lumps

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"
)

// Palettes represents a collection of 256-color palettes and allows managing palette data and active indices.
type Palettes struct {
	ret      IRetriever
	colorPal [][256]color.RGBA
}

// NewPalettes initializes a Palettes instance with the specified number of color palettes.
func NewPalettes(ret IRetriever) *Palettes {
	return &Palettes{
		ret:      ret,
		colorPal: nil,
	}
}

// Setup initializes the palette collection by loading and parsing palettes from the provided list of file names.
// Returns an error if any palette file cannot be loaded or parsed correctly.
func (pl *Palettes) Setup(p []string) error {
	pl.colorPal = make([][256]color.RGBA, len(p))
	for idx, pal := range p {
		// Prendiamo la prima (la master per le texture di base)
		palName := CleanName(pal)
		if !strings.HasSuffix(palName, PCXSuffix) {
			palName += PCXSuffix
		}
		palData, err := pl.ret.GetPayload(palName)
		if err != nil {
			return fmt.Errorf("can't open palette %s: %v", palName, err)
		}
		palette := NewPalette()
		colorPal, err := palette.ParseFromPCX(bytes.NewReader(palData))
		if err != nil {
			return fmt.Errorf("error while parsing palette %w", err)
		}
		pl.colorPal[idx] = colorPal
	}
	return nil
}

// Get retrieves the active color palette based on the current palette index.
// Returns an array of 256 color.RGBA values and an error if the index is invalid.
func (pl *Palettes) Get(palIndex int) ([256]color.RGBA, error) {
	if palIndex < 0 || palIndex >= len(pl.colorPal) {
		return [256]color.RGBA{}, fmt.Errorf("invalid palette index %d", palIndex)
	}
	return pl.colorPal[palIndex], nil
}
