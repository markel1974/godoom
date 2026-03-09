package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type Post struct {
	RowStart int
	Pixels   []byte
}

type Column struct {
	Posts []Post
}

type Image struct {
	Width   int
	Height  int
	Columns []Column
}

func NewImage(f *os.File, lumpInfo *LumpInfo, _ byte) (*Image, error) {
	if err := Seek(f, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	lump := make([]byte, lumpInfo.Size)
	n, err := f.Read(lump)
	if err != nil || n != int(lumpInfo.Size) {
		return nil, fmt.Errorf("read error or truncated lump")
	}

	reader := bytes.NewBuffer(lump)
	var header PictureHeader
	if err = binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	offsets := make([]int32, header.Width)
	if err = binary.Read(reader, binary.LittleEndian, offsets); err != nil {
		return nil, err
	}

	columns := make([]Column, header.Width)

	for x, offset := range offsets {
		var posts []Post
		for {
			rowStart := int(lump[offset])
			offset++
			if rowStart == 255 {
				break
			}
			numPixels := int32(lump[offset])
			offset += 2 // Salta numPixels e byte di padding iniziale

			pixels := make([]byte, numPixels)
			copy(pixels, lump[offset:offset+numPixels])
			posts = append(posts, Post{RowStart: rowStart, Pixels: pixels})

			offset += numPixels + 1 // Salta i pixel e il byte di padding finale
		}
		columns[x] = Column{Posts: posts}
	}

	return &Image{Width: int(header.Width), Height: int(header.Height), Columns: columns}, nil
}
