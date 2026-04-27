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

// ParseBM legge un flusso di byte in formato BM e lo converte in un'immagine RGBA,
// mappando i valori a 8-bit sulla palette fornita in input.
func ParseBM(r io.Reader, palette [256]color.RGBA) (*image.RGBA, error) {
	var header BMHeader
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("errore nella lettura dell'header BM: %w", err)
	}

	// Verifica basic signature (i file validi iniziano con 'B', 'M')
	if header.Magic[0] != 'B' || header.Magic[1] != 'M' {
		return nil, fmt.Errorf("firma non valida per file BM: %s", string(header.Magic[:]))
	}

	width := int(header.SizeX)
	height := int(header.SizeY)

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("dimensioni immagine non valide: %dx%d", width, height)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Molte texture dei livelli in Dark Forces sono RAW (0), ma alcuni sprite sono compressi.
	// Restituiamo un errore gestito per le compressioni RLE non ancora implementate.
	if header.Compressed != 0 {
		return nil, fmt.Errorf("modalità di compressione BM %d non ancora supportata", header.Compressed)
	}

	// Estrazione dell'array di byte flat
	pixelData := make([]byte, width*height)
	n, err := io.ReadFull(r, pixelData)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("errore nella lettura dei pixel: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("nessun dato pixel trovato nel file")
	}

	// Come per i segmenti di Doom, l'orientamento in RAM spesso dipende dall'asse dominante del renderer.
	// Nel Jedi Engine, a seconda dei Flag, i pixel possono essere Row-Major o Column-Major.
	// Assumiamo una stesura lineare Column-Major tipica delle texture dei muri per i vecchi raycaster:
	isColumnMajor := (header.Flags & 1) != 0 // Euristica: spia i bit di flag, potresti doverla tarare

	idx := 0
	if isColumnMajor {
		// Scansione per colonne (es. pareti verticali)
		for x := 0; x < width; x++ {
			for y := 0; y < height; y++ {
				if idx >= len(pixelData) {
					break
				}
				mapPixelToRGBA(img, x, y, pixelData[idx], header.Transparent, palette)
				idx++
			}
		}
	} else {
		// Scansione per righe (es. flats, cielo, hud)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if idx >= len(pixelData) {
					break
				}
				mapPixelToRGBA(img, x, y, pixelData[idx], header.Transparent, palette)
				idx++
			}
		}
	}

	return img, nil
}

// mapPixelToRGBA traduce l'indice cromatico applicando o meno il canale alpha (A=0 per la trasparenza).
func mapPixelToRGBA(img *image.RGBA, x, y int, pIndex byte, tIndex uint8, pal [256]color.RGBA) {
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
