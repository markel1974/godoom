package lumps

import (
	"fmt"
	"io"
)

// PaletteSize definisce la dimensione fissa della palette di Quake (256 colori * 3 canali RGB).
const PaletteSize = 768

// NewPalette estrae la mappa dei colori (256 triplette RGB) da uno stream dati.
func NewPalette(r io.Reader) ([]byte, error) {
	palette := make([]byte, PaletteSize)

	// Utilizziamo ReadFull per garantire che lo stream contenga esattamente i 768 byte necessari
	n, err := io.ReadFull(r, palette)
	if err != nil {
		return nil, fmt.Errorf("failed to read palette: %w", err)
	}
	if n != PaletteSize {
		return nil, fmt.Errorf("invalid palette size: read %d bytes, expected %d", n, PaletteSize)
	}

	return palette, nil
}
