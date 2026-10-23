package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// ArtHeader represents the header structure used in an art file, defining metadata about tiles and their properties.
type ArtHeader struct {
	ArtVersion        int32
	NumTiles          int32
	LocalTileArtStart int32
	LocalTileArtEnd   int32
}

// LoadPalette parses a VGA palette from a byte slice and returns a [256]color.RGBA array, scaling values appropriately.
func LoadPalette(data []byte) ([256]color.RGBA, error) {
	var pal [256]color.RGBA
	if len(data) < 768 {
		return pal, fmt.Errorf("palette too short: %d bytes", len(data))
	}
	for i := 0; i < 256; i++ {
		// VGA colors are 0..63; scale to 0..255 (multiply by 4 or shift << 2)
		r := data[i*3+0] << 2
		g := data[i*3+1] << 2
		b := data[i*3+2] << 2
		a := uint8(255)
		if i == 255 {
			a = 0 // In Build Engine, color 255 is the transparency key
		}
		pal[i] = color.RGBA{R: r, G: g, B: b, A: a}
	}
	return pal, nil
}

// LoadArt parses ART file data and populates textures with tiles using the provided palette and texture manager.
func LoadArt(data []byte, palette [256]color.RGBA, tex *Textures) error {
	r := bytes.NewReader(data)
	var hdr ArtHeader
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return fmt.Errorf("reading art header: %w", err)
	}

	numTiles := int(hdr.LocalTileArtEnd - hdr.LocalTileArtStart + 1)
	if numTiles <= 0 || numTiles > 4096 {
		return fmt.Errorf("invalid numtiles in art file: %d", numTiles)
	}

	tileSizesX := make([]int16, numTiles)
	if err := binary.Read(r, binary.LittleEndian, tileSizesX); err != nil {
		return fmt.Errorf("reading tilesizx: %w", err)
	}

	tileSizesY := make([]int16, numTiles)
	if err := binary.Read(r, binary.LittleEndian, tileSizesY); err != nil {
		return fmt.Errorf("reading tilesizy: %w", err)
	}

	// Skip picanm (numTiles * 4 bytes)
	picAnm := make([]int32, numTiles)
	if err := binary.Read(r, binary.LittleEndian, picAnm); err != nil {
		return fmt.Errorf("reading picanm: %w", err)
	}

	for i := 0; i < numTiles; i++ {
		w := int(tileSizesX[i])
		h := int(tileSizesY[i])
		tileNum := int(hdr.LocalTileArtStart) + i

		if w <= 0 || h <= 0 {
			continue
		}

		totalPixels := w * h
		rawPixels := make([]byte, totalPixels)
		if _, err := io.ReadFull(r, rawPixels); err != nil {
			return fmt.Errorf("reading raw tile %d pixels: %w", tileNum, err)
		}

		img := image.NewRGBA(image.Rect(0, 0, w, h))
		// Build engine stores tiles in Column-Major order: x*h + y
		for x := 0; x < w; x++ {
			colOffset := x * h
			for y := 0; y < h; y++ {
				colorIdx := rawPixels[colOffset+y]
				c := palette[colorIdx]
				img.SetRGBA(x, y, c)
			}
		}

		tex.AddTile(tileNum, img)
	}

	return nil
}
