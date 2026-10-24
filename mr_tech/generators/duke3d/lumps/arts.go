package lumps

import (
	"fmt"
	"image"
	"image/color"
)

// Arts represents a collection of textures and a color palette used for rendering graphics.
type Arts struct {
	palette [256]color.RGBA
	tex     *Textures
}

// NewArts creates a new Arts instance using the provided palette and texture manager.
func NewArts(palette [256]color.RGBA, tex *Textures) *Arts {
	return &Arts{
		palette: palette,
		tex:     tex,
	}
}

// Load processes the given binary art data, decodes tiles, and stores them in the texture manager using the palette colors.
func (arts *Arts) Load(data []byte) error {
	art := NewArt()
	if err := art.Parse(data); err != nil {
		return fmt.Errorf("parsing art: %w", err)
	}

	for _, tile := range art.Tiles {
		if tile == nil {
			continue
		}
		img := image.NewRGBA(image.Rect(0, 0, tile.Width, tile.Height))
		// Build engine stores tiles in Column-Major order: x*h + y
		for x := 0; x < tile.Width; x++ {
			colOffset := x * tile.Height
			for y := 0; y < tile.Height; y++ {
				colorIdx := tile.RawPixels[colOffset+y]
				c := arts.palette[colorIdx]
				img.SetRGBA(x, y, c)
			}
		}
		arts.tex.AddTile(tile.TileNum, img)
	}
	return nil
}
