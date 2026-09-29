package lumps

import (
	"image"
	"image/color"
	_ "image/png"
	"io"

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

// Get retrieves a list of textures corresponding to the given slice of IDs from the Textures' resources map.
// Returns nil if any ID is not found or if resources are uninitialized.
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

// RegisterFile registers a texture from an io.Reader source using the given name and returns an error if loading fails.
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

// RegisterPixelsPalette registers a texture using indexed color data and a color palette.
// name specifies the texture name, width and height define dimensions, and indices contain indexed pixel data.
// palette is a 256-sized RGBA color array; isTransparent determines if transparency should be used.
// transIndex indicates the transparent color index; invertY inverts the vertical axis. Returns an error on failure.
func (w *Textures) RegisterPixelsPalette(name string, width, height int, indices []byte, palette [256]color.RGBA, isTransparent bool, transIndex byte, invertY bool) error {
	if _, ok := w.resources[name]; ok {
		return nil
	}
	idx := int32(len(w.resources))
	tex, err := w.loadFromPixelsPalette(name, width, height, indices, palette, idx, isTransparent, transIndex, invertY)
	if err != nil {
		return err
	}
	w.resources[name] = tex
	return nil
}

// RegisterPixelsRGBA registers a new RGBA texture from a pixel byte array with specified dimensions and Y-axis inversion.
func (w *Textures) RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	if _, ok := w.resources[name]; ok {
		return nil
	}
	idx := int32(len(w.resources))
	tex, err := w.loadFromPixelsRGBA(name, width, height, pixels, idx, invertY)
	if err != nil {
		return err
	}
	w.resources[name] = tex
	return nil
}

// loadFromPixelsPalette creates a new texture from indexed pixel data and a color palette, managing transparency and Y inversion.
func (w *Textures) loadFromPixelsPalette(name string, width, height int, indices []byte, palette [256]color.RGBA, idx int32, isTransparent bool, transIndex byte, invertY bool) (*textures.Texture, error) {
	emissive := false
	if len(name) > 0 && name[0] == '*' || name[0] == '+' {
		emissive = true
	}
	tex := textures.NewTexture(name, uint32(idx), width, height, emissive)
	// Gestione unificata dell'Alpha (HL BSP + Override esplicito)
	hasAlpha := isTransparent || (len(name) > 0 && name[0] == '{')
	transparentColor := transIndex

	if len(name) > 0 && name[0] == '{' {
		transparentColor = 255
	}

	for y := 0; y < height; y++ {
		// L'inversione Y avviene solo se il formato lo richiede esplicitamente
		targetY := y
		if invertY {
			targetY = height - 1 - y
		}
		for x := 0; x < width; x++ {
			colorIdx := indices[y*width+x]
			if hasAlpha && colorIdx == transparentColor {
				tex.Set(x, targetY, 0x00000000)
				continue
			}
			r := uint32(palette[colorIdx].R)
			g := uint32(palette[colorIdx].G)
			b := uint32(palette[colorIdx].B)
			a := uint32(255)
			cl := (r << 24) | (g << 16) | (b << 8) | a
			tex.Set(x, targetY, int(cl))
		}
	}
	return tex, nil
}

// loadFromPixelsRGBA loads a texture from raw RGBA pixel data with optional Y-axis inversion for OpenGL compatibility.
func (w *Textures) loadFromPixelsRGBA(name string, width, height int, pixels []byte, idx int32, invertY bool) (*textures.Texture, error) {
	emissive := false
	if len(name) > 0 && (name[0] == '*' || name[0] == '+') {
		emissive = true
	}
	tex := textures.NewTexture(name, uint32(idx), width, height, emissive)
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
				tex.Set(x, targetY, int(cl))
			} else {
				// Fallback color (magenta debugging) se mancano dati nel buffer
				tex.Set(x, targetY, 0xFF00FF00)
			}
		}
	}
	return tex, nil
}

// loadFromFile loads a texture from an io.Reader, decodes the image, and initializes it with OpenGL-compatible memory layout.
func (w *Textures) loadFromFile(name string, reader io.Reader, idx int32) (*textures.Texture, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	emissive := false
	if len(name) > 0 && name[0] == '*' || name[0] == '+' {
		emissive = true
	}
	tex := textures.NewTexture(name, uint32(idx), width, height, emissive)
	for y := 0; y < height; y++ {
		flippedY := height - 1 - y
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			r8 := int(r >> 8)
			g8 := int(g >> 8)
			b8 := int(b >> 8)
			a8 := int(a >> 8)
			cl := (r8 << 24) | (g8 << 16) | (b8 << 8) | a8
			tex.Set(x, flippedY, cl)
		}
	}
	return tex, nil
}

// AddDirect adds a texture to the collection using the specified name without performing additional checks or processing.
func (w *Textures) AddDirect(name string, tex *textures.Texture) {
	w.Add(name, tex)
}
