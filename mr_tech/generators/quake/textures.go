package quake

import (
	"image"
	_ "image/png"
	"os"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// Textures is a collection of texture resources identified by unique string keys.
type Textures struct {
	resources map[string]*textures.Texture
}

// NewTextures loads textures from files in the specified directory and returns a Textures instance or an error.
func NewTextures() *Textures {
	t := &Textures{
		resources: make(map[string]*textures.Texture),
	}
	return t
}

func (w *Textures) load(filename string, idx int32) (*textures.Texture, error) {
	file, err := os.Open(filename)
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

	tex := textures.NewTexture(filename, uint32(idx), width, height)
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

func (w *Textures) Add(name string, tex *textures.Texture) {
	w.resources[name] = tex
}

func (w *Textures) Register(basePath string, name string) error {
	// Se la texture è già registrata, non facciamo nulla
	if _, ok := w.resources[name]; ok {
		return nil
	}

	// Costruiamo il percorso e carichiamo l'immagine tramite il tuo metodo load
	filename := basePath + name
	// Se Quake usa estensioni diverse (es. .png o .tga), puoi aggiungerle qui
	idx := int32(len(w.resources))
	tex, err := w.load(filename, idx)
	if err != nil {
		return err
	}

	// Iniezione nel pool di risorse
	w.resources[name] = tex
	return nil
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
