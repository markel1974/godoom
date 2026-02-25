package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type Image struct {
	Width  int
	Height int
	Pixels []byte
}

func NewImage(f *os.File, lumpInfo *LumpInfo, transparentPaletteIndex byte) (*Image, error) {
	if err := Seek(f, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	lump := make([]byte, lumpInfo.Size, lumpInfo.Size)
	n, err := f.Read(lump)
	if err != nil {
		return nil, err
	}
	if n != int(lumpInfo.Size) {
		return nil, fmt.Errorf("truncated lump")
	}
	reader := bytes.NewBuffer(lump[0:])
	var header PictureHeader
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	//if header.Width > 4096 || header.Height > 4096 {
	//	continue
	//}
	offsets := make([]int32, header.Width, header.Width)
	if err := binary.Read(reader, binary.LittleEndian, offsets); err != nil {
		return nil, err
	}
	size := int(header.Width) * int(header.Height)
	pixels := make([]byte, size, size)
	for y := 0; y < int(header.Height); y++ {
		for x := 0; x < int(header.Width); x++ {
			pixels[y*int(header.Width)+x] = transparentPaletteIndex
		}
	}
	for columnIndex, offset := range offsets {
		for {
			rowStart := lump[offset]
			offset += 1
			if rowStart == 255 {
				break
			}
			numPixels := lump[offset]
			offset += 1
			offset += 1 /* Padding */
			for i := 0; i < int(numPixels); i++ {
				pixelOffset := (int(rowStart)+i)*int(header.Width) + columnIndex
				pixels[pixelOffset] = lump[offset]
				offset += 1
			}
			offset += 1 /* Padding */
		}
	}
	return &Image{Width: int(header.Width), Height: int(header.Height), Pixels: pixels}, nil
}
