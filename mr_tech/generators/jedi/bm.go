package jedi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

// BMHeader represents the header structure of a BM file format, including metadata and configuration details.
type BMHeader struct {
	Magic       [4]byte  // 0x00 - Magico ("BM" + \x1E\x00)
	SizeX       uint16   // 0x04 - Larghezza
	SizeY       uint16   // 0x06 - Altezza
	Idc         uint16   // 0x08 - Identificatore (0=UI/Flat, 8=Column-Major, etc.)
	LogSizeY    uint8    // 0x0A - Esponente base 2 per tiling verticale
	Transparent uint8    // 0x0B - 0 = Solido, > 0 = Indice 0 bucato
	Unknown     uint16   // 0x0C - Padding/Riservato
	Compressed  uint16   // 0x0E - Compressione (0=RAW, 1/2=RLE)
	DataSize    uint32   // 0x10 - Dimensione del payload
	Padding     [12]byte // 0x14 - Spazio riservato a riempimento dei 32 byte
}

// NewBMHeader creates and initializes a new BMHeader instance with default values.
func NewBMHeader() *BMHeader {
	return &BMHeader{}
}

// BM represents a type used for parsing and handling custom bitmap image formats.
type BM struct {
}

// NewBM initializes and returns a new instance of the BM struct.
func NewBM() *BM {
	return &BM{}
}

// Parse reads data from the provided io.Reader, decodes it using the specified palette, and returns a slice of RGBA images.
func (b *BM) Parse(r io.Reader, palette [256]color.RGBA) ([]*image.RGBA, error) {
	header := NewBMHeader()
	if err := binary.Read(r, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	img, err := header.Decode(r, palette)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// Decode parses image data from the provided reader using the BMHeader properties and palette, returning decoded frames or an error.
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
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, err
		}
		pixelData = data

	case 1, 2:
		// Decodifica RLE dinamica per stream continui
		compData, err := io.ReadAll(r)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
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

	isColumnMajor := bm.Idc >= 8
	//isColumnMajor := (bm.Idc & 1) != 0
	//isColumnMajor := bm.Idc != 0
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

// mapPixelToRGBA maps a pixel with a palette index to its corresponding RGBA value in the image, applying transparency rules.
func (bm *BMHeader) mapPixelToRGBA(img *image.RGBA, x, y int, pIndex byte, tIndex uint8, pal [256]color.RGBA) {
	c := pal[pIndex]

	// bm.Transparent è il flag nativo letto dall'header del file .BM.
	// Se la texture è contrassegnata come trasparente E il pixel matcha l'indice chiave (solitamente 0), buca.
	// Altrimenti, rendi il pixel solido, salvando il nero assoluto sui muri.
	if bm.Transparent > 0 && pIndex == tIndex {
		c.A = 0
	} else {
		c.A = 255
	}

	img.SetRGBA(x, y, c)
}
