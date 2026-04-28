package jedi

import (
	"encoding/binary"
	"image/color"
	"io"
)

// ParsePAL estrae la palette VGA a 6-bit e la normalizza in RGBA a 8-bit.
func ParsePAL(r io.Reader) ([256]color.RGBA, error) {
	var pal [256]color.RGBA
	raw := make([]byte, 768)
	if _, err := io.ReadFull(r, raw); err != nil {
		return pal, err
	}
	for i := 0; i < 256; i++ {
		// Shift << 2 converte lo spazio VGA [0-63] nello spazio [0-255]
		pal[i] = color.RGBA{
			R: raw[i*3] << 2,
			G: raw[(i*3)+1] << 2,
			B: raw[(i*3)+2] << 2,
			A: 255, // La trasparenza è gestita dal parser BM/WAX, non dalla palette
		}
	}
	return pal, nil
}

// ColorMap contiene i 64 livelli di shading. 0 = Full Bright, 63 = Pitch Black.
type ColorMap struct {
	LightMaps [64][256]byte
}

func ParseCMP(r io.Reader) (*ColorMap, error) {
	cmp := &ColorMap{}
	// Il file CMP standard inizia con la mappa di illuminazione principale (16384 byte)
	if err := binary.Read(r, binary.LittleEndian, &cmp.LightMaps); err != nil {
		return nil, err
	}
	// (Omette l'estrazione delle mappe di tinting per i visori/gas/danni per brevità)
	return cmp, nil
}
