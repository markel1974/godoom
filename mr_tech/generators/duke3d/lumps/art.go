package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// ArtHeader represents the metadata for an ART file, including version, tile counts, and tile index range.
type ArtHeader struct {
	ArtVersion        int32
	NumTiles          int32
	LocalTileArtStart int32
	LocalTileArtEnd   int32
}

// ArtTiles represents an individual tile of art with defined width, height, raw pixel data, and total pixel count.
type ArtTile struct {
	Width     int
	Height    int
	TileNum   int
	RawPixels []byte
}

func NewArtTile(w int, h int, tileNum int) *ArtTile {
	totalPixels := w * h
	return &ArtTile{
		Width:     w,
		Height:    h,
		TileNum:   tileNum,
		RawPixels: make([]byte, totalPixels),
	}
}

// Art represents a collection of ArtTiles along with metadata stored in the ArtHeader.
type Art struct {
	Header ArtHeader
	Tiles  []*ArtTile
}

// NewArt creates and returns a new instance of the Art struct.
func NewArt() *Art {
	return &Art{}
}

// Parse processes the given byte slice to extract art header, tile dimensions, and raw pixel data into the Art structure.
func (a *Art) Parse(data []byte) error {
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &a.Header); err != nil {
		return fmt.Errorf("reading art header: %w", err)
	}
	numTiles := int(a.Header.LocalTileArtEnd - a.Header.LocalTileArtStart + 1)
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
		tileNum := int(a.Header.LocalTileArtStart) + i
		if w <= 0 || h <= 0 {
			continue
		}
		tile := NewArtTile(w, h, tileNum)
		if _, err := io.ReadFull(r, tile.RawPixels); err != nil {
			return fmt.Errorf("reading raw tile %d pixels: %w", tileNum, err)
		}
		a.Tiles = append(a.Tiles, tile)
	}
	return nil
}
