package lumps

import (
	"encoding/binary"
	"os"
)

type RGB struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

type Palette struct {
	Table [256]RGB
}

type PlayPal struct {
	Palettes [14]Palette
}

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
