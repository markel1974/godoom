package jedi

import (
	"fmt"
	"image/color"
	"io"
)

type Palette struct {
}

func NewPalette() *Palette {
	return &Palette{}
}

// Parse estrae la palette VGA a 6-bit e la normalizza in RGBA a 8-bit.
func (p *Palette) Parse(r io.Reader) ([256]color.RGBA, error) {
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

func (p *Palette) ParseFromPCX(r io.ReadSeeker) ([256]color.RGBA, error) {
	var pal [256]color.RGBA

	// Saltiamo agli ultimi 769 byte del file
	if _, err := r.Seek(-769, io.SeekEnd); err != nil {
		return pal, fmt.Errorf("impossibile cercare la fine del file PCX: %w", err)
	}

	// Leggiamo il byte indicatore
	var indicator [1]byte
	if _, err := io.ReadFull(r, indicator[:]); err != nil {
		return pal, err
	}

	// 0x0C (12 in decimale) è il flag standard che annuncia la presenza di una palette a 256 colori
	if indicator[0] != 0x0C {
		return pal, fmt.Errorf("firma della palette PCX non valida, atteso 0x0C, trovato 0x%02X", indicator[0])
	}

	// Leggiamo i 768 byte di dati RGB
	raw := make([]byte, 768)
	if _, err := io.ReadFull(r, raw); err != nil {
		return pal, err
	}

	// Costruiamo l'array RGBA
	for i := 0; i < 256; i++ {
		pal[i] = color.RGBA{
			R: raw[i*3],
			G: raw[(i*3)+1],
			B: raw[(i*3)+2],
			A: 255, // Trasparenza solida di default
		}
	}

	return pal, nil
}
