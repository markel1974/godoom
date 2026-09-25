package common

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// TGAHeader represents the header structure of a TGA (Truevision Graphics Adapter) image file.
type TGAHeader struct {
	IDLength        uint8
	ColorMapType    uint8
	ImageType       uint8
	ColorMapOrigin  uint16
	ColorMapLength  uint16
	ColorMapDepth   uint8
	XOrigin         uint16
	YOrigin         uint16
	Width           uint16
	Height          uint16
	PixelDepth      uint8
	ImageDescriptor uint8
}

// tgaParseColor converts a pixel buffer into a color.RGBA value based on the given bytes per pixel.
func tgaParseColor(bytesPerPixel int, pixelBuf []byte) color.RGBA {
	switch bytesPerPixel {
	case 1:
		v := pixelBuf[0]
		return color.RGBA{R: v, G: v, B: v, A: 255}
	case 2:
		val := uint16(pixelBuf[0]) | (uint16(pixelBuf[1]) << 8)
		b := uint8((val & 0x001F) << 3)
		g := uint8(((val >> 5) & 0x001F) << 3)
		rC := uint8(((val >> 10) & 0x001F) << 3)
		a := uint8(255)
		if (val & 0x8000) != 0 {
			a = 255
		} // Just to be safe, assume opaque for 16-bit unless specific alpha logic is needed
		return color.RGBA{R: rC, G: g, B: b, A: a}
	case 3:
		return color.RGBA{R: pixelBuf[2], G: pixelBuf[1], B: pixelBuf[0], A: 255}
	case 4:
		return color.RGBA{R: pixelBuf[2], G: pixelBuf[1], B: pixelBuf[0], A: pixelBuf[3]}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}

// tgaReadPixel reads a single pixel from a TGA image file based on the given parameters and returns its color value.
func tgaReadPixel(r io.Reader, bytesPerPixel int, pixelBuf []byte, colorMap []color.RGBA, cmapOrigin int) (color.RGBA, error) {
	if _, err := io.ReadFull(r, pixelBuf[:bytesPerPixel]); err != nil {
		return color.RGBA{}, err
	}

	if len(colorMap) > 0 {
		var idx int
		if bytesPerPixel == 1 {
			idx = int(pixelBuf[0])
		} else if bytesPerPixel == 2 {
			idx = int(uint16(pixelBuf[0]) | (uint16(pixelBuf[1]) << 8))
		}
		idx -= cmapOrigin
		if idx >= 0 && idx < len(colorMap) {
			return colorMap[idx], nil
		}
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}, nil
	}

	return tgaParseColor(bytesPerPixel, pixelBuf), nil
}

// DecodeTGA decodes a TGA image from the provided io.Reader and returns it as an image.Image object or an error.
func DecodeTGA(r io.Reader) (image.Image, error) {
	var header TGAHeader

	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.IDLength > 0 {
		if _, err := io.CopyN(io.Discard, r, int64(header.IDLength)); err != nil {
			return nil, err
		}
	}

	supportedTypes := map[uint8]bool{1: true, 2: true, 3: true, 9: true, 10: true, 11: true}
	if !supportedTypes[header.ImageType] {
		return nil, fmt.Errorf("unsupported TGA image type: %d", header.ImageType)
	}

	if header.PixelDepth != 8 && header.PixelDepth != 16 && header.PixelDepth != 24 && header.PixelDepth != 32 {
		return nil, fmt.Errorf("unsupported TGA pixel depth: %d", header.PixelDepth)
	}

	var colorMap []color.RGBA
	switch header.ColorMapType {
	case 0: //nothing to do
	case 1:
		if header.ColorMapDepth != 15 && header.ColorMapDepth != 16 && header.ColorMapDepth != 24 && header.ColorMapDepth != 32 {
			return nil, fmt.Errorf("unsupported TGA colormap depth: %d", header.ColorMapDepth)
		}
		cMapBpp := int(header.ColorMapDepth) / 8
		if header.ColorMapDepth == 15 {
			cMapBpp = 2
		}
		colorMapBytes := int(header.ColorMapLength) * cMapBpp
		cMapData := make([]byte, colorMapBytes)
		if _, err := io.ReadFull(r, cMapData); err != nil {
			return nil, err
		}

		colorMap = make([]color.RGBA, header.ColorMapLength)
		for i := 0; i < int(header.ColorMapLength); i++ {
			colorMap[i] = tgaParseColor(cMapBpp, cMapData[i*cMapBpp:(i+1)*cMapBpp])
		}
	default:
		return nil, fmt.Errorf("unsupported ColorMapType: %d", header.ColorMapType)
	}

	width := int(header.Width)
	height := int(header.Height)
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid TGA dimensions: %dx%d", width, height)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bytesPerPixel := (int(header.PixelDepth) + 7) / 8
	if bytesPerPixel == 0 {
		bytesPerPixel = 1
	}

	pixelBuf := make([]byte, bytesPerPixel)

	// TGA usually stores pixels upside down (origin at bottom-left)
	// If ImageDescriptor bit 5 is set, origin is top-left
	flipY := (header.ImageDescriptor & 0x20) == 0
	flipX := (header.ImageDescriptor & 0x10) != 0
	cMapOrigin := int(header.ColorMapOrigin)

	switch header.ImageType {
	case 1, 2, 3:
		for y := 0; y < height; y++ {
			destY := y
			if flipY {
				destY = height - 1 - y
			}
			for x := 0; x < width; x++ {
				c, err := tgaReadPixel(r, bytesPerPixel, pixelBuf, colorMap, cMapOrigin)
				if err != nil {
					return nil, err
				}
				destX := x
				if flipX {
					destX = width - 1 - x
				}
				img.SetRGBA(destX, destY, c)
			}
		}

	case 9, 10, 11:
		x, y := 0, 0
		var rleHeader [1]byte
		for y < height {
			if _, err := io.ReadFull(r, rleHeader[:]); err != nil {
				break
			}
			packetHeader := rleHeader[0]
			count := int(packetHeader&0x7F) + 1

			if packetHeader&0x80 != 0 { // Run-length packet
				c, err := tgaReadPixel(r, bytesPerPixel, pixelBuf, colorMap, cMapOrigin)
				if err != nil {
					return nil, err
				}
				for i := 0; i < count; i++ {
					destY := y
					if flipY {
						destY = height - 1 - y
					}
					destX := x
					if flipX {
						destX = width - 1 - x
					}
					img.SetRGBA(destX, destY, c)
					x++
					if x >= width {
						x = 0
						y++
					}
				}
			} else { // Raw packet
				for i := 0; i < count; i++ {
					c, err := tgaReadPixel(r, bytesPerPixel, pixelBuf, colorMap, cMapOrigin)
					if err != nil {
						return nil, err
					}
					destY := y
					if flipY {
						destY = height - 1 - y
					}
					destX := x
					if flipX {
						destX = width - 1 - x
					}
					img.SetRGBA(destX, destY, c)
					x++
					if x >= width {
						x = 0
						y++
					}
				}
			}
		}
	}
	// --- Post-processing Alpha Channel Fix (TEST MODE) ---
	// Forcing alpha to 255 for ALL 32-bit images to test if missing textures appear
	//if bytesPerPixel == 4 {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.RGBAAt(x, y)
			c.A = 255
			img.SetRGBA(x, y, c)
		}
	}
	//}

	return img, nil
}

// EncodeTGA encodes an image in TGA format and writes it to the provided io.Writer.
// The image must be a valid image.Image and will be encoded as an uncompressed 32-bit BGRA image.
func EncodeTGA(w io.Writer, img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 || width > 65535 || height > 65535 {
		return fmt.Errorf("invalid TGA dimensions: %dx%d", width, height)
	}

	header := TGAHeader{
		IDLength:        0,
		ColorMapType:    0,
		ImageType:       2, // Uncompressed TrueColor
		ColorMapOrigin:  0,
		ColorMapLength:  0,
		ColorMapDepth:   0,
		XOrigin:         0,
		YOrigin:         0,
		Width:           uint16(width),
		Height:          uint16(height),
		PixelDepth:      32,   // Always 32-bit RGBA for simplicity
		ImageDescriptor: 0x28, // Top-left origin (bit 5) + 8-bit alpha (bits 0-3)
	}

	if err := binary.Write(w, binary.LittleEndian, &header); err != nil {
		return err
	}

	rowBuf := make([]byte, width*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		idx := 0
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			// TGA is BGRA
			rowBuf[idx] = c.B
			rowBuf[idx+1] = c.G
			rowBuf[idx+2] = c.R
			rowBuf[idx+3] = c.A
			idx += 4
		}
		if _, err := w.Write(rowBuf); err != nil {
			return err
		}
	}

	return nil
}
