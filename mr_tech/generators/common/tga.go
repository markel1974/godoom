package common

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

func DecodeTGA(r io.Reader) (image.Image, error) {
	var header struct {
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

	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.IDLength > 0 {
		if _, err := io.CopyN(io.Discard, r, int64(header.IDLength)); err != nil {
			return nil, err
		}
	}

	if header.ColorMapType != 0 {
		return nil, fmt.Errorf("color-mapped TGA not supported")
	}

	if header.ImageType != 2 && header.ImageType != 10 {
		return nil, fmt.Errorf("unsupported TGA image type: %d", header.ImageType)
	}

	if header.PixelDepth != 24 && header.PixelDepth != 32 {
		return nil, fmt.Errorf("unsupported TGA pixel depth: %d", header.PixelDepth)
	}

	width := int(header.Width)
	height := int(header.Height)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bytesPerPixel := int(header.PixelDepth / 8)

	pixelBuf := make([]byte, bytesPerPixel)

	// TGA usually stores pixels upside down (origin at bottom-left)
	// If ImageDescriptor bit 5 is set, origin is top-left
	flipY := (header.ImageDescriptor & 0x20) == 0

	readPixel := func() (color.RGBA, error) {
		if _, err := io.ReadFull(r, pixelBuf); err != nil {
			return color.RGBA{}, err
		}
		// TGA is BGRA
		b, g, rC := pixelBuf[0], pixelBuf[1], pixelBuf[2]
		a := uint8(255)
		if bytesPerPixel == 4 {
			a = pixelBuf[3]
		}
		return color.RGBA{R: rC, G: g, B: b, A: a}, nil
	}

	if header.ImageType == 2 { // Uncompressed
		for y := 0; y < height; y++ {
			destY := y
			if flipY {
				destY = height - 1 - y
			}
			for x := 0; x < width; x++ {
				c, err := readPixel()
				if err != nil {
					return nil, err
				}
				img.SetRGBA(x, destY, c)
			}
		}
	} else if header.ImageType == 10 { // RLE
		x, y := 0, 0
		var rleHeader [1]byte
		for y < height {
			if _, err := io.ReadFull(r, rleHeader[:]); err != nil {
				break
			}
			packetHeader := rleHeader[0]
			count := int(packetHeader&0x7F) + 1

			if packetHeader&0x80 != 0 { // Run-length packet
				c, err := readPixel()
				if err != nil {
					return nil, err
				}
				for i := 0; i < count; i++ {
					destY := y
					if flipY {
						destY = height - 1 - y
					}
					img.SetRGBA(x, destY, c)
					x++
					if x >= width {
						x = 0
						y++
					}
				}
			} else { // Raw packet
				for i := 0; i < count; i++ {
					c, err := readPixel()
					if err != nil {
						return nil, err
					}
					destY := y
					if flipY {
						destY = height - 1 - y
					}
					img.SetRGBA(x, destY, c)
					x++
					if x >= width {
						x = 0
						y++
					}
				}
			}
		}
	}

	return img, nil
}
