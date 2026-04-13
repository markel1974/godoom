package quake

import (
	"image"
	_ "image/png"
	"io"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// Textures manages a collection of named 2D textures, providing methods for adding, retrieving, and registering textures.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures initializes and returns a pointer to a new Textures instance with an empty resource map.
func NewTextures() *Textures {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	return t
}

// Get retrieves a list of *textures.Texture corresponding to the provided ids. Returns nil if any id is not found.
func (w *Textures) Get(ids []string) []*textures.Texture {
	var out []*textures.Texture
	for _, id := range ids {
		x, ok := w.resources[id]
		if !ok {
			return nil
		}
		out = append(out, x)
	}
	return out
}

// GetNames returns a slice of all texture names stored in the Textures collection.
func (w *Textures) GetNames() []string {
	var out []string
	for id := range w.resources {
		out = append(out, id)
	}
	return out
}

// Add associates a texture with the given name and stores it in the resources map.
func (w *Textures) Add(name string, tex *textures.Texture) {
	w.resources[name] = tex
}

// RegisterFile registers a texture from an io.Reader and associates it with the specified name. Returns an error if loading fails.
func (w *Textures) RegisterFile(name string, rs io.Reader) error {
	if _, ok := w.resources[name]; ok {
		return nil
	}
	idx := int32(len(w.resources))
	tex, err := w.loadFromFile(name, rs, idx)
	if err != nil {
		return err
	}
	w.resources[name] = tex
	return nil
}

// RegisterPixels registers a texture using raw pixel data, a palette, dimensions, and a unique name. If the name already exists, it skips registration. Returns an error if the data cannot be processed.
func (w *Textures) RegisterPixels(name string, width, height int, indices []byte, palette []byte) error {
	if _, ok := w.resources[name]; ok {
		return nil
	}
	idx := int32(len(w.resources))
	tex, err := w.loadFromPixels(name, width, height, indices, palette, idx)
	if err != nil {
		return err
	}
	w.resources[name] = tex
	return nil
}

// loadFromPixels creates a texture from raw pixel data, applying optional alpha transparency for specific cases.
func (w *Textures) loadFromPixels(name string, width, height int, indices []byte, palette []byte, idx int32) (*textures.Texture, error) {
	tex := textures.NewTexture(name, uint32(idx), width, height)
	// In Quake, le texture il cui nome inizia per '{' (come le grate) usano l'indice 255 per la trasparenza
	hasAlpha := false
	if len(name) > 0 && name[0] == '{' {
		hasAlpha = true
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			colorIdx := indices[y*width+x]
			if hasAlpha && colorIdx == 255 {
				tex.Set(x, y, 0x00000000)
				continue
			}
			palOffset := int(colorIdx) * 3
			r := int(palette[palOffset])
			g := int(palette[palOffset+1])
			b := int(palette[palOffset+2])
			a := 255
			color := (r << 24) | (g << 16) | (b << 8) | a
			tex.Set(x, y, color)
		}
	}
	return tex, nil
}

// loadFromFile loads a texture from an image reader, decodes it, and populates texture data with pixel colors.
func (w *Textures) loadFromFile(name string, reader io.Reader, idx int32) (*textures.Texture, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	tex := textures.NewTexture(name, uint32(idx), width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			r8 := int(r >> 8)
			g8 := int(g >> 8)
			b8 := int(b >> 8)
			a8 := int(a >> 8)
			color := (r8 << 24) | (g8 << 16) | (b8 << 8) | a8
			tex.Set(x, y, color)
		}
	}
	return tex, nil
}

/*
// GetAnimation trasforma un nome texture in un oggetto Animation pronto per il compilatore.
func (w *Textures) GetAnimation(name string) *configAnimation {
	tex := w.Get([]string{name})
	if tex == nil {
		return nil
	}
	// Ritorna un'animazione a singolo frame (loop di 1)
	return config.NewConfigAnimation([]string{name}, config.AnimationKindLoop, 1.0, 1.0)
}
*/
