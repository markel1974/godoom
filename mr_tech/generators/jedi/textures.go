package jedi

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// cleanId transforms the input string by trimming leading/trailing spaces and converting it to uppercase.
func cleanId(id string) string {
	return strings.TrimSpace(strings.ToUpper(id))
}

// Textures maintain a collection of textures and their associated animations.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures initializes and returns a new instance of Textures with preloaded animations and an empty resources map.
func NewTextures() *Textures {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	return t
}

func (t *Textures) AddTexture(d *GobHandler, bm *BM, texName string, palette [256]color.RGBA) []string {
	fileName := texName
	bmData, err := d.GetPayload(fileName)
	if err != nil {
		fmt.Printf("payload %s not found: %v\n", fileName, err)
		return nil
	}
	images, err := bm.Parse(bytes.NewReader(bmData), palette)
	if err != nil {
		fmt.Printf("decode error %s: %v\n", fileName, err)
		return nil
	}
	out := make([]string, len(images))
	for counter, img := range images {
		name := fmt.Sprintf("%s__FRAME__%d", texName, counter)
		t.add(name, img)
		out = append(out, name)
	}
	return out
}

// Add adds a new texture to the Textures collection using the provided source ID and RGBA image, and returns the created Animations.
func (t *Textures) add(srcId string, src *image.RGBA) *textures.Texture {
	size := src.Bounds().Size()
	id := len(t.resources)
	texture := textures.NewTexture(srcId, uint32(id), size.X, size.Y)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			c := src.RGBAAt(x, y)
			rgba := int(c.R)<<24 | int(c.G)<<16 | int(c.B)<<8 | int(c.A)
			texture.Set(x, y, rgba)
		}
	}
	t.resources[srcId] = texture
	return texture
}

// Get retrieves a list of textures by their identifiers. Returns nil if the list is empty or contains invalid IDs.
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

// GetNames returns a slice of all texture IDs currently stored in the Textures instance.
func (t *Textures) GetNames() []string {
	var out []string
	for id := range t.resources {
		out = append(out, id)
	}
	return out
}
