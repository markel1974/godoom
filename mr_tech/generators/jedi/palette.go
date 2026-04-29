package jedi

import (
	"image/color"
	"io"
)

type Palette struct {
}

func NewPalette() *Palette {
	return &Palette{}
}

// Parse estrae la palette VGA a 6-bit e la normalizza in RGBA a 8-bit.
func (p *Palette) Parse(r io.Reader) ([256]color.RGBA, error) {
	var pal [256]color.RGBA
	raw := make([]byte, 768)
	if _, err := io.ReadFull(r, raw); err != nil {
		return pal, err
	}
	for i := 0; i < 256; i++ {
		// Shift << 2 converte lo spazio VGA [0-63] nello spazio [0-255]
		pal[i] = color.RGBA{
			R: raw[i*3] << 2,
			G: raw[(i*3)+1] << 2,
			B: raw[(i*3)+2] << 2,
			A: 255, // La trasparenza è gestita dal parser BM/WAX, non dalla palette
		}
	}
	return pal, nil
}
