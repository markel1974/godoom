package common

import (
	"bufio"
	"os"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// PPMToTexture reads a PPM (P6) texture file, dynamically parses its header for dimensions, and populates pixel values.
func PPMToTexture(filename string, name string, idx uint32, emissive bool) (*textures.Texture, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	br := bufio.NewReader(file)

	// Helper to read next token (ignoring whitespace and # comments)
	readToken := func() (string, error) {
		var token []byte
		inComment := false
		for {
			b, err := br.ReadByte()
			if err != nil {
				return "", err
			}
			if inComment {
				if b == '\n' || b == '\r' {
					inComment = false
				}
				continue
			}
			if b == '#' {
				inComment = true
				continue
			}
			if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
				if len(token) > 0 {
					// We successfully read a token and hit whitespace, so stop here.
					return string(token), nil
				}
				// Skip leading whitespaces
				continue
			}
			token = append(token, b)
		}
	}

	// 1. Magic Number
	magic, err := readToken()
	if err != nil {
		return nil, err
	}
	if magic != "P6" {
		return nil, os.ErrInvalid
	}

	// 2. Width
	widthStr, err := readToken()
	if err != nil {
		return nil, err
	}
	var width int
	for _, c := range widthStr {
		width = width*10 + int(c-'0')
	}

	// 3. Height
	heightStr, err := readToken()
	if err != nil {
		return nil, err
	}
	var height int
	for _, c := range heightStr {
		height = height*10 + int(c-'0')
	}

	// 4. Max Color Value
	maxColorStr, err := readToken()
	if err != nil {
		return nil, err
	}
	_ = maxColorStr // typically "255"

	// Create the texture with dynamic dimensions
	tex := textures.NewTexture(name, idx, width, height, emissive)

	var r, g, b byte
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if r, err = br.ReadByte(); err != nil {
				return nil, err
			}
			if g, err = br.ReadByte(); err != nil {
				return nil, err
			}
			if b, err = br.ReadByte(); err != nil {
				return nil, err
			}
			rgba := int(r)<<24 | int(g)<<16 | int(b)<<8 | 255
			tex.Set(x, y, rgba)
		}
	}

	return tex, nil
}
