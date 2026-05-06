package jedi

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// PCXHeader represents the standard 128-byte ZSoft PCX header.
type PCXHeader struct {
	Manufacturer uint8
	Version      uint8
	Encoding     uint8
	BitsPerPixel uint8
	XMin         uint16
	YMin         uint16
	XMax         uint16
	YMax         uint16
	HDpi         uint16
	VDpi         uint16
	Colormap     [48]byte
	Reserved     uint8
	NPlanes      uint8
	BytesPerLine uint16
	PaletteInfo  uint16
	HScreenSize  uint16
	VScreenSize  uint16
	Filler       [54]byte
}

// PCX handles the decoding of PCX image files.
type PCX struct{}

// NewPCX initializes and returns a new PCX parser instance.
func NewPCX() *PCX {
	return &PCX{}
}

// Parse decodes a PCX file from the reader. If the file contains an embedded VGA palette,
// it overrides the defaultPalette provided. Returns a slice containing the single parsed frame.
func (p *PCX) Parse(r io.Reader, defaultPalette [256]color.RGBA) (*image.RGBA, error) {
	var header PCXHeader
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.Manufacturer != 10 {
		return nil, fmt.Errorf("invalid PCX signature: %d", header.Manufacturer)
	}

	width := int(header.XMax - header.XMin + 1)
	height := int(header.YMax - header.YMin + 1)
	bytesPerLine := int(header.BytesPerLine)
	expectedSize := bytesPerLine * height

	// Leggiamo tutto in memoria per decompressione RLE ad alta velocità
	rawData, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	imgData := make([]byte, 0, expectedSize)
	idx := 0

	// Decompressione RLE
	for len(imgData) < expectedSize && idx < len(rawData) {
		b := rawData[idx]
		idx++
		if (b & 0xC0) == 0xC0 {
			count := int(b & 0x3F)
			if idx >= len(rawData) {
				break
			}
			val := rawData[idx]
			idx++
			for i := 0; i < count; i++ {
				imgData = append(imgData, val)
			}
		} else {
			imgData = append(imgData, b)
		}
	}
	// Override della palette se il flag 0x0C è presente a -769 byte dalla fine.
	pal := defaultPalette
	if len(rawData)-idx >= 769 && rawData[len(rawData)-769] == 0x0C {
		palOffset := len(rawData) - 768
		for i := 0; i < 256; i++ {
			pal[i] = color.RGBA{
				R: rawData[palOffset+(i*3)],
				G: rawData[palOffset+(i*3)+1],
				B: rawData[palOffset+(i*3)+2],
				A: 255,
			}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dataIdx := y*bytesPerLine + x
			if dataIdx >= len(imgData) {
				break
			}
			pIndex := imgData[dataIdx]
			c := pal[pIndex]
			if pIndex == 0 {
				c.A = 0
			} else {
				c.A = 255
			}
			img.SetRGBA(x, height-1-y, c)
			//img.SetRGBA(width-1-x, height-1-y, c)
		}
	}
	return img, nil
}

func (p *PCX) ParsePalette(r io.ReadSeeker) ([256]color.RGBA, error) {
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
