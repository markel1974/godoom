package common

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

func Hexdump(data []byte) {
	const width = 16
	for i := 0; i < len(data); i += width {
		// Offset
		fmt.Printf("%08x  ", i)

		// Hex values (16 bytes per riga)
		for j := 0; j < width; j++ {
			if i+j < len(data) {
				fmt.Printf("%02x ", data[i+j])
			} else {
				fmt.Printf("   ")
			}
		}

		// ASCII representation
		fmt.Printf(" |")
		for j := 0; j < width; j++ {
			if i+j < len(data) {
				c := data[i+j]
				if c >= 32 && c <= 126 {
					fmt.Printf("%c", c)
				} else {
					fmt.Printf(".")
				}
			} else {
				fmt.Printf(" ")
			}
		}
		fmt.Printf("|\n")
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
