package common

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

func Hexdump(data []byte) {
	HexdumpF(os.Stdout, data)
}
func HexdumpF(writer io.Writer, data []byte) {
	const width = 16
	for i := 0; i < len(data); i += width {
		// Offset
		_, _ = fmt.Fprintf(writer, "%08x  ", i)

		// Hex values (16 bytes per riga)
		for j := 0; j < width; j++ {
			if i+j < len(data) {
				_, _ = fmt.Fprintf(writer, "%02x ", data[i+j])
			} else {
				_, _ = fmt.Fprintf(writer, "   ")
			}
		}

		// ASCII representation
		_, _ = fmt.Fprintf(writer, " |")
		for j := 0; j < width; j++ {
			if i+j < len(data) {
				c := data[i+j]
				if c >= 32 && c <= 126 {
					_, _ = fmt.Fprintf(writer, "%c", c)
				} else {
					_, _ = fmt.Fprintf(writer, ".")
				}
			} else {
				_, _ = fmt.Fprintf(writer, " ")
			}
		}
		_, _ = fmt.Fprintf(writer, "|\n")
	}
}

func SaveImage(name string, img image.Image) error {
	out, err := os.Create(name + ".png")
	if err != nil {
		return err
	}
	defer out.Close()
	if err = png.Encode(out, img); err != nil {
		return err
	}
	return nil
}
