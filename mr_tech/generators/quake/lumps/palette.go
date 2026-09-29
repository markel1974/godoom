package lumps

import (
	"image/color"
	"io"
	"math"

	"github.com/markel1974/godoom/mr_tech/generators/common"
)

// Palette represents a color palette, typically used for managing and parsing 256-color VGA palettes in various formats.
type Palette struct {
	vidGamma float64
}

// NewPalette creates and returns a new instance of the Palette struct.
func NewPalette(vidGamma float64) *Palette {
	return &Palette{
		vidGamma: vidGamma,
	}
}

// Parse reads a palette from the given io.ReadSeeker and returns it as an array of 256 color.RGBA entries.
func (p *Palette) Parse(reader io.ReadSeeker) ([256]color.RGBA, error) {
	var pal [256]color.RGBA
	raw := make([]byte, 768)
	if _, err := io.ReadFull(reader, raw); err != nil {
		return pal, err
	}

	for i := 0; i < len(pal); i++ {
		r := p.computeGamma(raw[i*3], p.vidGamma)
		g := p.computeGamma(raw[(i*3)+1], p.vidGamma)
		b := p.computeGamma(raw[(i*3)+2], p.vidGamma)
		pal[i] = color.RGBA{R: r, G: g, B: b, A: 255}
	}
	return pal, nil
}

// ParseFromPCX reads a PCX file stream and extracts a 256-color RGBA palette. Returns the palette and any encountered error.
func (p *Palette) ParseFromPCX(reader io.ReadSeeker) ([256]color.RGBA, error) {
	pcx := common.NewPCX()
	pal, err := pcx.ParsePalette(reader)
	if err != nil {
		return [256]color.RGBA{}, err
	}
	for i := 0; i < len(pal); i++ {
		r := p.computeGamma(pal[i].R, p.vidGamma)
		g := p.computeGamma(pal[i].G, p.vidGamma)
		b := p.computeGamma(pal[i].B, p.vidGamma)
		pal[i] = color.RGBA{R: r, G: g, B: b, A: 255}
	}
	return pal, nil
}

// computeGamma applies gamma correction to an input value and returns the adjusted value as a uint8.
func (p *Palette) computeGamma(in uint8, gamma float64) uint8 {
	if gamma == 0 || gamma == 1 {
		return in
	}
	c := float64(in) / 255.0
	c = math.Pow(c, gamma)
	val := int(c * 255.0)
	if val > 255 {
		val = 255
	}
	return uint8(val)
}
