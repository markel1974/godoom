package lumps

import (
	"image/color"
	"io"

	"github.com/markel1974/godoom/mr_tech/generators/common"
)

// Palette represents a structure used to manage and manipulate color palettes for graphical applications.
type Palette struct {
}

// NewPalette creates and returns a new instance of the Palette struct.
func NewPalette() *Palette {
	return &Palette{}
}

// Parse reads VGA palette data from the provided io.Reader and converts it into an array of 256 color.RGBA values.
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

// ParseFromPCX extracts a 256-color RGBA palette from a PCX file provided via an io.ReadSeeker. Returns the palette or an error.
func (p *Palette) ParseFromPCX(r io.ReadSeeker) ([256]color.RGBA, error) {
	pcx := common.NewPCX()
	img, err := pcx.ParsePalette(r)
	return img, err
}
