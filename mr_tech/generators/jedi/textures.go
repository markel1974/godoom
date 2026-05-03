package jedi

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// cleanId processes the input string by trimming whitespace and converting it to uppercase.
func cleanId(id string) string {
	return strings.TrimSpace(strings.ToUpper(id))
}

// Textures manages a collection of 2D textures and provides caching for efficient retrieval.
type Textures struct {
	resources map[string]*textures.Texture
	cache     map[string][]string
}

// NewTextures creates and returns a new instance of the Textures struct with initialized resource and cache maps.
func NewTextures() *Textures {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
		cache:     make(map[string][]string),
	}
	return t
}

// AddTexture adds a texture by parsing bitmap data and applying a palette, caching the result for future requests.
func (t *Textures) AddTexture(d *GobHandler, bm *BM, texName string, palette [256]color.RGBA) []string {
	v, ok := t.cache[texName]
	if ok {
		return v
	}
	bmData, err := d.GetPayload(texName)
	if err != nil {
		fmt.Printf("payload %s not found: %v\n", texName, err)
		return nil
	}
	images, err := bm.Parse(bytes.NewReader(bmData), palette)
	if err != nil {
		fmt.Printf("decode error %s: %v\n", texName, err)
		return nil
	}
	out := make([]string, len(images))
	for counter, img := range images {
		name := fmt.Sprintf("%s__FRAME__%d", texName, counter)
		t.add(name, img)
		out[counter] = name
	}
	t.cache[texName] = out
	return out
}

// AddRawTexture creates a texture from raw indexed pixel data, applies a palette, and stores it in the texture manager.
func (t *Textures) AddRawTexture(name string, width, height int, indexedPixels []byte, palette [256]color.RGBA) string {
	if _, exists := t.resources[name]; exists {
		return name
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			// NOTA SULLA TOPOLOGIA: Nel parser WAX che abbiamo scritto,
			// abbiamo salvato i dati nel buffer come Column-Major (x * height + y).
			// Se hai deciso di cambiarlo in Row-Major, usa: pixelPos := y*width + x
			pixelPos := y*width + x //x*height + y
			// Controllo di sicurezza bounds
			if pixelPos >= len(indexedPixels) {
				continue
			}
			colorIdx := indexedPixels[pixelPos]
			var c color.RGBA
			if colorIdx == 0 {
				// Nel Jedi Engine, il colore 0 è rigorosamente il color-key della trasparenza
				c = color.RGBA{R: 0, G: 0, B: 0, A: 0}
			} else {
				// Recuperiamo il colore dalla palette master
				c = palette[colorIdx]
				// Assicuriamoci che il canale Alpha sia completamente opaco
				c.A = 255
			}
			img.SetRGBA(x, y, c)
		}
	}

	// Passiamo il risultato al tuo metodo helper che si occuperà di fare
	// il Y-Flip e il packing a 32-bit (int(c.R)<<24...) per il tuo engine.
	t.add(name, img)

	return name
}

// add creates and adds a new texture to the resources map, initializing it with RGBA pixel data from the source image.
func (t *Textures) add(srcId string, src *image.RGBA) *textures.Texture {
	size := src.Bounds().Size()
	id := len(t.resources)
	emissive := false
	if len(srcId) > 0 && srcId[0] == '*' || srcId[0] == '+' {
		emissive = true
	}
	texture := textures.NewTexture(srcId, uint32(id), size.X, size.Y, emissive)
	for y := 0; y < size.Y; y++ {
		flipY := size.Y - 1 - y
		for x := 0; x < size.X; x++ {
			c := src.RGBAAt(x, flipY)
			rgba := int(c.R)<<24 | int(c.G)<<16 | int(c.B)<<8 | int(c.A)
			texture.Set(x, y, rgba)
		}
	}
	t.resources[srcId] = texture
	return texture
}

// Get retrieves a list of textures corresponding to the given slice of IDs. Missing textures are replaced with nil.
func (t *Textures) Get(ids []string) []*textures.Texture {
	l := len(ids)
	if l == 0 {
		return nil
	} else if l == 1 {
		if len(ids[0]) == 0 {
			return nil
		}
	}
	var out []*textures.Texture
	for index, id := range ids {
		x, ok := t.resources[id]
		if !ok {
			fmt.Printf("CAN'T FIND TEXTURE %d: '%s'\n", index, id)
			out = append(out, nil)
		} else {
			out = append(out, x)
		}
	}
	return out
}

// GetNames returns a slice of strings containing the names of all textures in the Textures instance.
func (t *Textures) GetNames() []string {
	var out []string
	for id := range t.resources {
		out = append(out, id)
	}
	return out
}
