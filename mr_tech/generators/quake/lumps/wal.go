package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// WalHeader rappresenta i primi 100 byte di un file texture .wal (idTech 2).
type WalHeader struct {
	Name     [32]byte
	Width    uint32
	Height   uint32
	Offsets  [4]uint32 // Offset assoluti dal file start per i 4 Mipmap level (1/1, 1/2, 1/4, 1/8)
	AnimName [32]byte  // Nome della prossima frame texture (per switch, acqua e lava)
	Flags    uint32    // Proprietà fisiche e visive (es. SURF_SKY)
	Contents uint32
	Value    uint32
}

// WalTexture contiene i dati estratti e pronti per la pipeline del Texture Manager
type WalTexture struct {
	Header WalHeader
	Pixels []byte // Pixel grezzi indicizzati a 8-bit del Mipmap 0 (Full Res)
}

// ParseWal legge, decodifica e valida lo stream binario di una texture .wal
func ParseWal(rs io.ReadSeeker) (*WalTexture, error) {
	var header WalHeader
	if err := binary.Read(rs, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("errore lettura header .wal: %w", err)
	}

	if header.Offsets[0] == 0 {
		return nil, fmt.Errorf("file .wal corrotto: offset mipmap 0 nullo")
	}

	// Salto diretto al blocco dati del Mipmap a risoluzione massima
	if _, err := rs.Seek(int64(header.Offsets[0]), io.SeekStart); err != nil {
		return nil, fmt.Errorf("impossibile allineare il cursore al mipmap 0: %w", err)
	}

	// Le texture .wal usano 1 byte per pixel (indice palette).
	// La dimensione esatta è banalmente Width * Height
	size := int(header.Width * header.Height)
	pixels := make([]byte, size)

	if _, err := io.ReadFull(rs, pixels); err != nil {
		return nil, fmt.Errorf("impossibile estrarre i pixel del mipmap 0: %w", err)
	}

	return &WalTexture{
		Header: header,
		Pixels: pixels,
	}, nil
}

// GetAnimName restituisce la stringa per le texture animate a frame (es. +0button -> +1button)
func (w *WalTexture) GetAnimName() string {
	b := make([]byte, 0, 32)
	for _, v := range w.Header.AnimName {
		if v == 0 {
			break
		}
		b = append(b, v)
	}
	return strings.ToLower(string(b))
}
