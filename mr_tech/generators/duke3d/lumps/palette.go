package lumps

import (
	"fmt"
	"image/color"
)

// Palette represents a fixed-size collection of 256 RGBA color values.
type Palette struct {
	Data [256]color.RGBA
}

// NewPalette creates and initializes a new instance of the Palette struct with default values.
func NewPalette() *Palette {
	return &Palette{}
}

// Parse processes a byte slice to populate the palette with 256 color entries, each defined as an RGBA value.
func (p *Palette) Parse(data []byte) error {
	if len(data) < 768 {
		return fmt.Errorf("palette too short: %d bytes", len(data))
	}
	for i := 0; i < 256; i++ {
		// VGA colors are 0..63; scale to 0..255 (multiply by 4 or shift << 2)
		r := data[i*3+0] << 2
		g := data[i*3+1] << 2
		b := data[i*3+2] << 2
		a := uint8(255)
		if i == 255 {
			r = 0
			g = 0
			b = 0
			a = 0 // In Build Engine, color 255 is the transparency key
		}
		p.Data[i] = color.RGBA{R: r, G: g, B: b, A: a}
	}
	return nil
}

// GetData returns the palette data as an array of 256 color.RGBA values.
func (p *Palette) GetData() [256]color.RGBA {
	return p.Data
}
