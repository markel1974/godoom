package wolfstein

import (
	"embed"
	"image"
	_ "image/png"
	"io"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// assets represents an embedded file system containing application resources such as container or assets.
//
//go:embed images
var assets embed.FS

// Textures is a collection of texture resources identified by unique string keys.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures loads textures from files in the specified directory and returns a Textures instance or an error.
func NewTextures() (*Textures, error) {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	files, err := assets.ReadDir("images")
	if err != nil {
		return nil, err
	}
	for idx, f := range files {
		if !f.IsDir() {
			target := "images" + "/" + f.Name()
			tex, err := t.load(target, int32(idx))
			if err == nil || err == io.EOF {
				t.resources[f.Name()] = tex
			} else {
				return nil, err
			}
		}
	}
	return t, nil
}

func (w *Textures) load(filename string, idx int32) (*textures.Texture, error) {
	file, err := assets.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	emissive := false
	if len(filename) > 0 && filename[0] == '*' || filename[0] == '+' {
		emissive = true
	}

	tex := textures.NewTexture(filename, uint32(idx), width, height, emissive)
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

// Get retrieves textures matching the provided `ids` from the Textures resource map. Returns nil if an id is not found.
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

// GetNames returns a list of all texture names (keys) stored in the Textures' resources map.
func (w *Textures) GetNames() []string {
	var out []string
	for id := range w.resources {
		out = append(out, id)
	}
	return out
}
