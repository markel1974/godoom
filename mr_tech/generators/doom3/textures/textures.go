package textures

import (
	_ "image/png"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// Textures manages a collection of Texture objects, allowing storage, retrieval, and registration of textures by name.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures initializes and returns a new instance of the Textures structure with an empty resource map.
func NewTextures() *Textures {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	return t
}

// GetFrames retrieves a list of textures corresponding to the given slice of IDs from the Textures' resources map.
// Returns nil if any ID is not found or if resources are uninitialized.
func (w *Textures) GetFrames(ids []string) []*textures.Texture {
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

// GetNames retrieves a list of all texture names stored in the Textures resource map.
func (w *Textures) GetNames() []string {
	var out []string
	for id := range w.resources {
		out = append(out, id)
	}
	return out
}

// Add adds a texture to the Textures resource map using the specified name as the key.
func (w *Textures) Add(name string, tex *textures.Texture) {
	w.resources[name] = tex
}

// RegisterPixelsRGBA registers a new RGBA texture from a pixel byte array with specified dimensions and Y-axis inversion.
func (w *Textures) RegisterPixelsRGBA(kind int, name string, width, height int, pixels []byte, invertY bool) error {
	if _, ok := w.resources[name]; ok {
		return nil
	}
	idx := int32(len(w.resources))
	tex, err := w.loadFromPixelsRGBA(kind, name, width, height, pixels, idx, invertY)
	if err != nil {
		return err
	}
	w.resources[name] = tex
	return nil
}

// loadFromPixelsRGBA loads a texture from raw RGBA pixel data with optional Y-axis inversion for OpenGL compatibility.
func (w *Textures) loadFromPixelsRGBA(kind int, name string, width, height int, pixels []byte, idx int32, invertY bool) (*textures.Texture, error) {
	tex := textures.NewTexture(name, uint32(idx), width, height, false)

	var setter func(int, int, int)
	switch kind {
	case 0:
		setter = tex.Set
	case 1:
		if !tex.HasNormal() {
			tex.CreateNormal()
		}
		setter = tex.SetNormal
	case 2:
		if !tex.HasEmissive() {
			tex.CreateEmissive()
		}
		setter = tex.SetEmissive
	}
	for y := 0; y < height; y++ {
		// Gestione inversione asse Y (Top-Left to Bottom-Left per OpenGL)
		targetY := y
		if invertY {
			targetY = height - 1 - y
		}
		for x := 0; x < width; x++ {
			// Offset lineare con stride fisso a 4 (RGBA)
			offset := (y*width + x) * 4
			// Sanity check per evitare panic su buffer troncati
			if offset+3 < len(pixels) {
				r := uint32(pixels[offset])
				g := uint32(pixels[offset+1])
				b := uint32(pixels[offset+2])
				a := uint32(pixels[offset+3])
				cl := (r << 24) | (g << 16) | (b << 8) | a
				setter(x, targetY, int(cl))
			} else {
				// Fallback color (magenta debugging) se mancano dati nel buffer
				setter(x, targetY, 0xFF00FF00)
			}
		}
	}
	return tex, nil
}

// AddDirect adds a texture to the collection using the specified name without performing additional checks or processing.
func (w *Textures) AddDirect(name string, tex *textures.Texture) {
	w.Add(name, tex)
}
