package lumps

import (
	"encoding/binary"
	"os"
)

// RGB represents a color in the RGB color model with 8-bit intensity values for red, green, and blue components.
type RGB struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

// Palette represents a fixed-size collection of 256 RGB color entries used for color mapping and lookups.
type Palette struct {
	Table [256]RGB
}

// PlayPal represents the collection of color palettes used for rendering graphics in Doom-engine WAD files.
type PlayPal struct {
	Palettes [14]Palette
}

// NewPlayPal reads the PLAYPAL lump data from the given file and initializes a new PlayPal structure.
func NewPlayPal(f *os.File, lumpInfo *LumpInfo) (*PlayPal, error) {
	if err := Seek(f, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	playPal := &PlayPal{}
	if err := binary.Read(f, binary.LittleEndian, playPal); err != nil {
		return nil, err
	}
	return playPal, nil
}
