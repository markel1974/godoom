package jedi

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// BMHeader rappresenta la struttura dell'header di un file .BM nel Jedi Engine.
// L'header è fisso a 32 byte.
type BMHeader struct {
	Magic       [4]byte  // Magico, tipicamente inizia con "BM"
	SizeX       uint16   // Larghezza dell'immagine
	SizeY       uint16   // Altezza dell'immagine
	Idc         uint16   // Identificatore/Modalità
	Flags       uint16   // Flag (determina orientamento, es. row-major vs column-major)
	Transparent uint8    // Indice del colore trasparente nella palette
	LogSizeY    uint8    // Esponente di 2 per altezze di texture (utile per il wrapping)
	Compressed  uint16   // Modalità di compressione: 0 = Raw, 1/2 = RLE (Run-Length Encoding)
	DataSize    uint32   // Dimensione in byte della porzione dati
	Padding     [12]byte // Spazio riservato/padding per allineamento a 32 byte
}

func NewHeader() *BMHeader {
	return &BMHeader{}
}

// ParseBM legge un flusso di byte in formato BM e lo converte in un'immagine RGBA,
// mappando i valori a 8-bit sulla palette fornita in input.
func ParseBM(r io.Reader, palette [256]color.RGBA) ([]*image.RGBA, error) {
	header := NewHeader()
	if err := binary.Read(r, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	img, err := header.Decode(r, palette)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// Decode estrae uno o più frame da un file .BM, restituendo un array di image.RGBA.
// Il numero di frame è dedotto dinamicamente dalla dimensione del payload decompresso.
func (bm *BMHeader) Decode(r io.Reader, palette [256]color.RGBA) ([]*image.RGBA, error) {
	if bm.Magic[0] != 'B' || bm.Magic[1] != 'M' {
		return nil, fmt.Errorf("firma non valida per file BM: %s", string(bm.Magic[:]))
	}
	width := int(bm.SizeX)
	height := int(bm.SizeY)
	frameSize := width * height
	if frameSize <= 0 {
		return nil, fmt.Errorf("dimensioni immagine non valide: %dx%d", width, height)
	}
	var pixelData []byte

	switch bm.Compressed {
	case 0:
		// Lettura RAW lineare (singolo o multi-frame)
		data, err := io.ReadAll(r)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		pixelData = data

	case 1, 2:
		// Decodifica RLE dinamica per stream continui
		compData, err := io.ReadAll(r)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		out := make([]byte, 0, frameSize)
		for i := 0; i < len(compData); {
			cmd := int8(compData[i])
			i++
			if cmd >= 0 {
				count := int(cmd) + 1
				if i+count > len(compData) {
					count = len(compData) - i
				}
				out = append(out, compData[i:i+count]...)
				i += count
			} else if cmd != -128 {
				count := int(-cmd) + 1
				if i < len(compData) {
					val := compData[i]
					i++
					for j := 0; j < count; j++ {
						out = append(out, val)
					}
				}
			}
		}
		// Gestione specifica RLE0 (Compressed == 2):
		// Se il packer ha troncato lo stream omettendo l'ultimo blocco di pixel trasparenti,
		// riempiamo il buffer fino al limite corretto del frameSize.
		if bm.Compressed == 2 && len(out)%frameSize != 0 {
			padding := frameSize - (len(out) % frameSize)
			for j := 0; j < padding; j++ {
				out = append(out, bm.Transparent)
			}
		}
		pixelData = out
	default:
		return nil, fmt.Errorf("algoritmo di compressione %d non supportato", bm.Compressed)
	}

	numFrames := len(pixelData) / frameSize
	if numFrames == 0 {
		return nil, fmt.Errorf("dati insufficienti per l'estrazione di un singolo frame")
	}

	isColumnMajor := (bm.Idc & 1) != 0
	frames := make([]*image.RGBA, numFrames)

	for f := 0; f < numFrames; f++ {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		offset := f * frameSize
		idx := 0

		if isColumnMajor {
			// Offset extraction per pareti / switch (Idc == 1, 3)
			for x := 0; x < width; x++ {
				for y := 0; y < height; y++ {
					bm.mapPixelToRGBA(img, x, y, pixelData[offset+idx], bm.Transparent, palette)
					idx++
				}
			}
		} else {
			// Offset extraction per UI / flats (Idc == 0, 2)
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					bm.mapPixelToRGBA(img, x, y, pixelData[offset+idx], bm.Transparent, palette)
					idx++
				}
			}
		}
		frames[f] = img
	}

	return frames, nil
}

// mapPixelToRGBA traduce l'indice cromatico applicando o meno il canale alpha (A=0 per la trasparenza).
func (bm *BMHeader) mapPixelToRGBA(img *image.RGBA, x, y int, pIndex byte, tIndex uint8, pal [256]color.RGBA) {
	c := pal[pIndex]
	if pIndex == tIndex {
		c.A = 0 // Colore bucato (es: grate o staccionate)
	} else {
		c.A = 255
	}
	img.SetRGBA(x, y, c)
}

/*
pal := loadJediPalette("SECBASE.PAL")
file, _ := os.Open("OAK.BM")
imgRGBA, _ := jedi.ParseBM(file, pal)

// E l'inserimento nel tuo sistema generico del generatore
texHandler.Add("OAK", imgRGBA)
*/
