package lumps

import (
	"fmt"
	"image"
	"image/color"
)

// Arts represents a collection of raw art tiles and a color palette used for decoding textures on demand.
type Arts struct {
	palette [256]color.RGBA
	tiles   map[int]*ArtTile
}

// NewArts creates a new Arts instance using the provided palette.
func NewArts(palette [256]color.RGBA) *Arts {
	return &Arts{
		palette: palette,
		tiles:   make(map[int]*ArtTile),
	}
}

// Load processes the given binary art data and stores the raw tiles in memory for on-demand decoding.
func (arts *Arts) Load(data []byte) error {
	art := NewArt()
	if err := art.Parse(data); err != nil {
		return fmt.Errorf("parsing art: %w", err)
	}

	for _, tile := range art.Tiles {
		if tile != nil && tile.Width > 0 && tile.Height > 0 {
			arts.tiles[tile.TileNum] = tile
		}
	}
	return nil
}

// CreateImage decodes a single tile into an RGBA image using the palette colors in Build engine column-major order.
func (arts *Arts) CreateImage(picNum int) *image.RGBA {
	tile, ok := arts.tiles[picNum]
	if !ok || tile == nil || tile.Width <= 0 || tile.Height <= 0 {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, tile.Width, tile.Height))
	hasTransparent := false
	// Build engine stores tiles in Column-Major order: x*h + y
	for x := 0; x < tile.Width; x++ {
		colOffset := x * tile.Height
		for y := 0; y < tile.Height; y++ {
			colorIdx := tile.RawPixels[colOffset+y]
			if colorIdx == 255 {
				hasTransparent = true
			}
			c := arts.palette[colorIdx]
			img.SetRGBA(x, y, c)
		}
	}

	if hasTransparent {
		w, h := tile.Width, tile.Height
		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				if img.RGBAAt(x, y).A == 0 {
					if x > 0 && img.RGBAAt(x-1, y).A > 0 {
						nc := img.RGBAAt(x-1, y)
						img.SetRGBA(x, y, color.RGBA{R: nc.R, G: nc.G, B: nc.B, A: 0})
					} else if x < w-1 && img.RGBAAt(x+1, y).A > 0 {
						nc := img.RGBAAt(x+1, y)
						img.SetRGBA(x, y, color.RGBA{R: nc.R, G: nc.G, B: nc.B, A: 0})
					} else if y > 0 && img.RGBAAt(x, y-1).A > 0 {
						nc := img.RGBAAt(x, y-1)
						img.SetRGBA(x, y, color.RGBA{R: nc.R, G: nc.G, B: nc.B, A: 0})
					} else if y < h-1 && img.RGBAAt(x, y+1).A > 0 {
						nc := img.RGBAAt(x, y+1)
						img.SetRGBA(x, y, color.RGBA{R: nc.R, G: nc.G, B: nc.B, A: 0})
					}
				}
			}
		}
	}

	return img
}

// HasTile checks if a tile with the specified picNum exists in the collection.
func (arts *Arts) HasTile(picNum int) bool {
	_, ok := arts.tiles[picNum]
	return ok
}

// GetTiles returns the map of art tiles where the key is the tile number and the value is a pointer to an ArtTile.
func (arts *Arts) GetTiles() map[int]*image.RGBA {
	out := make(map[int]*image.RGBA)
	for k := range arts.tiles {
		out[k] = arts.CreateImage(k)
	}
	return out
}
