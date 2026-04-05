package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// MipTexture rappresenta una texture di Quake con i suoi 4 livelli di mipmap precalcolati.
type MipTexture struct {
	Name   string
	Width  uint32
	Height uint32
	Pixels [][]byte // Array di 4 byte-slices (livello 0, 1, 2, 3)
}

// NewMipTextures legge il lump TEXTURES e decodifica tutte le texture contenute.
func NewMipTextures(f *os.File, lumpInfo *LumpInfo) ([]*MipTexture, error) {
	if _, err := f.Seek(lumpInfo.Filepos, os.SEEK_SET); err != nil {
		return nil, err
	}

	lump := make([]byte, lumpInfo.Size)
	n, err := f.Read(lump)
	if err != nil || n != int(lumpInfo.Size) {
		return nil, fmt.Errorf("read error or truncated texture lump")
	}

	reader := bytes.NewReader(lump)

	// Legge il numero di texture nel dizionario
	var numTextures int32
	if err := binary.Read(reader, binary.LittleEndian, &numTextures); err != nil {
		return nil, err
	}

	// Legge gli offset per ogni texture
	offsets := make([]int32, numTextures)
	if err := binary.Read(reader, binary.LittleEndian, offsets); err != nil {
		return nil, err
	}

	textures := make([]*MipTexture, numTextures)

	for i, offset := range offsets {
		// Se l'offset è -1, la texture è un frame di animazione gestito esternamente
		if offset == -1 {
			continue
		}

		headerOffset := int(offset)

		// Estrae il nome (fino a 16 caratteri null-terminated)
		nameBytes := lump[headerOffset : headerOffset+16]
		name := strings.TrimRight(string(nameBytes), "\x00")

		// Estrae larghezza e altezza
		width := binary.LittleEndian.Uint32(lump[headerOffset+16 : headerOffset+20])
		height := binary.LittleEndian.Uint32(lump[headerOffset+20 : headerOffset+24])

		// Estrae gli offset dei 4 livelli di mipmap (relativi all'inizio di questa singola MipTexture)
		var mipOffsets [4]uint32
		for j := 0; j < 4; j++ {
			start := headerOffset + 24 + (j * 4)
			mipOffsets[j] = binary.LittleEndian.Uint32(lump[start : start+4])
		}

		// Estrae i pixel per ciascuno dei 4 livelli di mipmap
		pixels := make([][]byte, 4)
		for j := 0; j < 4; j++ {
			// Ogni livello di mipmap dimezza la risoluzione
			mipWidth := width >> j
			mipHeight := height >> j
			size := mipWidth * mipHeight

			start := headerOffset + int(mipOffsets[j])
			pixels[j] = make([]byte, size)
			copy(pixels[j], lump[start:start+int(size)])
		}

		textures[i] = &MipTexture{
			Name:   name,
			Width:  width,
			Height: height,
			Pixels: pixels,
		}
	}

	return textures, nil
}
