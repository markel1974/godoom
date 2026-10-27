package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// ArtHeader represents the header metadata of an ART file, including version and tile range information.
type ArtHeader struct {
	ArtVersion        int32
	NumTiles          int32
	LocalTileArtStart int32
	LocalTileArtEnd   int32
}

// ArtTile represents a single tile in an art file, including its dimensions, pixel data, and optional animation metadata.
type ArtTile struct {
	Width     int
	Height    int
	TileNum   int
	RawPixels []byte
	Anim      *ArtTileAnim
}

// NewArtTile creates and initializes a new ArtTile with the given width, height, and tile number.
func NewArtTile(w int, h int, tileNum int) *ArtTile {
	totalPixels := w * h
	return &ArtTile{
		Width:     w,
		Height:    h,
		TileNum:   tileNum,
		RawPixels: make([]byte, totalPixels),
	}
}

// ArtTileAnim defines animation properties for a tile, including frame count, type, speed, and offsets for positioning.
type ArtTileAnim struct {
	NumFrames int  // Number of additional frames (0 = static, 1 = 2 frames, etc.)
	AnimType  int  // 0 = none, 1 = oscillating, 2 = forward, 3 = backward
	Speed     int  // Animation speed
	XOffset   int8 // Center X offset
	YOffset   int8 // Center Y offset
}

// NewArtTileAnim creates and initializes an ArtTileAnim with the specified frame count, animation type, speed, and offsets.
func NewArtTileAnim(NumFrames int, AnimType int, Speed int, XOffset int8, YOffset int8) *ArtTileAnim {
	return &ArtTileAnim{
		NumFrames: NumFrames,
		AnimType:  AnimType,
		Speed:     Speed,
		XOffset:   XOffset,
		YOffset:   YOffset,
	}
}

// Art represents a container for art assets, including a header and a collection of tiles.
type Art struct {
	Header ArtHeader
	Tiles  []*ArtTile
}

// NewArt initializes and returns a new instance of the Art structure.
func NewArt() *Art {
	return &Art{}
}

// Parse parses binary data into an Art structure, decoding header, tiles, and animation metadata.
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
		val := picAnm[i]
		b0 := byte(val & 0xFF)
		b1 := int8((val >> 8) & 0xFF)
		b2 := int8((val >> 16) & 0xFF)
		b3 := byte((val >> 24) & 0xFF)
		numFrames := int(b0 & 63)
		animType := int((b0 >> 6) & 3)
		speed := int(b3 & 15)
		xOffset := b1
		yOffset := b2
		tile.Anim = NewArtTileAnim(numFrames, animType, speed, xOffset, yOffset)

		if _, err := io.ReadFull(r, tile.RawPixels); err != nil {
			return fmt.Errorf("reading raw tile %d pixels: %w", tileNum, err)
		}
		a.Tiles = append(a.Tiles, tile)
	}
	return nil
}
