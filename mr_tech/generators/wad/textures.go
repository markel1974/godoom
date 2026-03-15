package wad

import (
	"fmt"
	"image"
	"strings"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// TextureCreateId generates a unique texture identifier by appending a fixed prefix to a sanitized version of the input id.
func TextureCreateId(id string) string {
	const textureId = "__TEXTURE__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return textureId + id
}

// FlatCreateId generates a flat identifier by appending a predefined prefix to a cleaned version of the given id.
func FlatCreateId(id string) string {
	const flatId = "__FLAT__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return flatId + id
}

// SpriteCreateId generates a unique sprite identifier by appending a prefix to a cleaned version of the input string.
func SpriteCreateId(id string) string {
	const flatId = "__SPRITE__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return flatId + id
}

// cleanId transforms the input string by trimming leading/trailing spaces and converting it to uppercase.
func cleanId(id string) string {
	return strings.TrimSpace(strings.ToUpper(id))
}

// Textures maintain a collection of textures and their associated animations.
type Textures struct {
	resources  map[string]*textures.Texture
	animations map[string][]string
}

// NewTextures initializes and returns a new instance of Textures with preloaded animations and an empty resources map.
func NewTextures() (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	t.animations = make(map[string][]string)
	for _, v := range _animationsBase {
		for _, a := range v {
			t.animations[a] = v
		}
	}
	return t, nil
}

// Add adds a new texture to the Textures collection using the provided source ID and RGBA image, and returns the created Animations.
func (t *Textures) Add(srcId string, src *image.RGBA) *textures.Texture {
	size := src.Bounds().Size()
	id := len(t.resources)
	texture := textures.NewTexture(srcId, uint32(id), size.X, size.Y)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			c := src.RGBAAt(x, y)
			if c.A == 0 {
				// Truly transparent pixel
				texture.Set(x, y, -1)
			} else {
				rgb := int(c.R)*65536 + int(c.G)*256 + int(c.B)
				texture.Set(x, y, rgb)
			}
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

// TextureCreateAnimation generates a list of texture IDs based on an input identifier and existing animations.
func (t *Textures) TextureCreateAnimation(id string) []string {
	id = cleanId(id)
	if animation, ok := t.animations[id]; ok {
		var out []string
		for _, i := range animation {
			out = append(out, TextureCreateId(i))
		}
		return out
	}
	targetId := TextureCreateId(id)
	if len(targetId) == 0 {
		return nil
	}
	return []string{targetId}
}

// FlatCreateAnimation generates a list of flattened animation IDs for a given texture ID, resolving nested animations recursively.
func (t *Textures) FlatCreateAnimation(id string) []string {
	id = cleanId(id)
	if animation, ok := t.animations[id]; ok {
		var out []string
		for _, i := range animation {
			out = append(out, FlatCreateId(i))
		}
		return out
	}
	targetId := FlatCreateId(id)
	if len(targetId) == 0 {
		return nil
	}
	return []string{targetId}
}

// SpriteCreateAnimation generates a list of sprite animation IDs for a given texture ID, resolving nested animations recursively.
func (t *Textures) SpriteCreateAnimation(ids []string) []string {
	var out []string
	for _, id := range ids {
		id = cleanId(id)
		out = append(out, SpriteCreateId(id))
	}
	return out
}
