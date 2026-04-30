package jedi

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// cleanId transforms the input string to uppercase and removes any leading or trailing whitespace.
func cleanId(id string) string {
	return strings.TrimSpace(strings.ToUpper(id))
}

// Textures manages a collection of Texture resources, providing functionality to add, retrieve, and query textures.
type Textures struct {
	resources map[string]*textures.Texture
	cache     map[string][]string
}

// NewTextures initializes and returns a new instance of the Textures struct with an empty resources map.
func NewTextures() *Textures {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
		cache:     make(map[string][]string),
	}
	return t
}

// AddTexture parses a texture file, generates frames as RGBA images, stores them, and returns their names as a slice of strings.
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

// add creates and adds a new texture to the resources map using the provided source ID and RGBA image.
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

// Get retrieves textures corresponding to the provided list of IDs. Missing IDs return a nil entry in the output slice.
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

// GetNames retrieves a list of all texture IDs stored in the Textures instance.
func (t *Textures) GetNames() []string {
	var out []string
	for id := range t.resources {
		out = append(out, id)
	}
	return out
}
