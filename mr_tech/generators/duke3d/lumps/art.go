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

// TileAnim represents animation parameters for a tile in Build engine .ART files.
type TileAnim struct {
	NumFrames int  // Number of additional frames (0 = static, 1 = 2 frames, etc.)
	AnimType  int  // 0 = none, 1 = oscillating, 2 = forward, 3 = backward
	Speed     int  // Animation speed
	XOffset   int8 // Center X offset
	YOffset   int8 // Center Y offset
}

// ArtTile represents an individual tile of art with defined width, height, raw pixel data, and animation metadata.
type ArtTile struct {
	Width     int
	Height    int
	TileNum   int
	RawPixels []byte
	Anim      TileAnim
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
		val := picAnm[i]
		b0 := byte(val & 0xFF)
		b1 := int8((val >> 8) & 0xFF)
		b2 := int8((val >> 16) & 0xFF)
		b3 := byte((val >> 24) & 0xFF)
		tile.Anim = TileAnim{
			NumFrames: int(b0 & 63),
			AnimType:  int((b0 >> 6) & 3),
			Speed:     int(b3 & 15),
			XOffset:   b1,
			YOffset:   b2,
		}
		if _, err := io.ReadFull(r, tile.RawPixels); err != nil {
			return fmt.Errorf("reading raw tile %d pixels: %w", tileNum, err)
		}
		a.Tiles = append(a.Tiles, tile)
	}
	return nil
}
