package wad

import (
	"fmt"
	"image"
	"strings"

	"github.com/markel1974/godoom/engine/textures"
)

// CreateTextureId generates a valid texture ID by prepending a fixed prefix to a cleaned, non-empty input string.
func CreateTextureId(id string) string {
	const textureId = "__TEXTURE__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return textureId + id
}

// CreateFlatId generates a unique flat identifier by normalizing and appending a prefix to the input ID. Returns an empty string for invalid inputs.
func CreateFlatId(id string) string {
	const flatId = "__FLAT__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return flatId + id
}

// Textures is a collection that maps string IDs to Texture objects, allowing storage and retrieval of texture resources.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures creates and returns a new instance of Textures with an initialized resource map.
func NewTextures() (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	return t, nil
}

// Add creates and adds a new texture to the Textures resource map using the provided ID and RGBA image data.
// It converts the image's pixel data into texture-specific format and returns the newly created texture.
func (t *Textures) Add(srcId string, src *image.RGBA) *textures.Texture {
	size := src.Bounds().Size()
	id := len(t.resources)

	texture := textures.NewTexture(srcId, uint32(id), size.X, size.Y)

	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			// RGBAAt legge direttamente i byte bypassando le interfacce color.Color
			c := src.RGBAAt(x, y)

			if c.A == 0 {
				// Pixel realmente trasparente
				texture.Set(x, y, -1)
			} else {
				// Codifica standard 24-bit per il tuo Software Renderer
				rgb := int(c.R)*65536 + int(c.G)*256 + int(c.B)

				// HACK VITALE: Se il pixel è nero puro (0,0,0), forziamo a 1.
				// Se lasciamo 0, il software renderer lo interpreta come "buco invisibile".
				if rgb == 0 {
					rgb = 1
				}

				texture.Set(x, y, rgb)
			}
		}
	}

	t.resources[srcId] = texture
	return texture
}

// Get retrieves the texture associated with the given ID from the Textures' resources map. Returns nil if not found.
func (t *Textures) Get(id string) *textures.Texture {
	if len(id) == 0 {
		return nil
	}
	// TODO VERIFICA PER TEKWALL4 completamente ALFA
	if strings.Contains(id, "TEKWALL4") {
		id = "__TEXTURE__CEMENT6"
	}
	x, ok := t.resources[id]
	if !ok {
		fmt.Println("CAN'T FIND TEXTURE", id)
		return nil
	}
	return x
}

// GetNames returns a slice of all texture names (IDs) stored in the Textures collection.
func (t *Textures) GetNames() []string {
	var out []string
	for id := range t.resources {
		out = append(out, id)
	}
	return out
}

// cleanId normalizes the input string by trimming spaces and converting it to uppercase.
func cleanId(id string) string {
	return strings.TrimSpace(strings.ToUpper(id))
}
