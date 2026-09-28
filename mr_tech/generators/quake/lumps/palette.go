package lumps

import (
	"fmt"
	"io"
	"math"
)

// PaletteSize defines the fixed size of the color palette in bytes, commonly used for handling 256-color palettes.
const PaletteSize = 768

// NewPalette reads a palette of size PaletteSize from the provided reader and applies Gamma correction to its values.
// Returns the corrected palette as a byte slice or an error if the read fails or the size is incorrect.
func NewPalette(r io.Reader) ([]byte, error) {
	palette := make([]byte, PaletteSize)
	n, err := io.ReadFull(r, palette)
	if err != nil {
		return nil, fmt.Errorf("failed to read palette: %w", err)
	}
	if n != PaletteSize {
		return nil, fmt.Errorf("invalid palette size: read %d bytes, expected %d", n, PaletteSize)
	}
	const vidGamma = 0.8
	// Apply Quake standard Gamma correction
	for i := 0; i < len(palette); i++ {
		c := float64(palette[i]) / 255.0
		c = math.Pow(c, vidGamma)
		val := int(c * 255.0)
		if val > 255 {
			val = 255
		}
		palette[i] = byte(val)
	}
	return palette, nil
}
