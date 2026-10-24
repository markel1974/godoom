package lumps

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// Textures manages a collection of Texture objects and provides mapping and retrieval functionalities.
type Textures struct {
	resources map[string]*textures.Texture
	depot     map[int]*image.RGBA
}

// NewTextures creates and returns a new instance of Textures with initialized resource and cache maps.
func NewTextures() *Textures {
	return &Textures{
		resources: make(map[string]*textures.Texture),
		depot:     make(map[int]*image.RGBA),
	}
}

// AddDepot adds an RGBA image to the depot map indexed by the specified texture ID.
func (t *Textures) AddDepot(tId int, img *image.RGBA) {
	t.depot[tId] = img
}

// GetNames retrieves all the keys of the resources map in the Textures instance sorted alphabetically for determinism.
func (t *Textures) GetNames() []string {
	names := make([]string, 0, len(t.resources))
	for k := range t.resources {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Get retrieves a slice of textures based on the provided list of texture names. Only textures found in resources are returned.
func (t *Textures) Get(names []string) []*textures.Texture {
	out := make([]*textures.Texture, 0, len(names))
	for _, name := range names {
		if tex, ok := t.resources[name]; ok {
			out = append(out, tex)
		}
	}
	return out
}

// AddDirect adds a texture to the resources map using the specified name if the provided texture is not nil.
func (t *Textures) AddDirect(name string, tex *textures.Texture) {
	if tex != nil {
		t.resources[name] = tex
	}
}

// GetTile retrieves the name of a cached tile or creates a new one if it does not exist.
func (t *Textures) GetTile(picNum int) string {
	name := fmt.Sprintf("duke_tile_%d", picNum)
	if _, exists := t.resources[name]; exists {
		return name
	}
	var texture *textures.Texture
	if img, ok := t.depot[picNum]; ok {
		texture = t.createTextureFromImage(name, uint32(picNum), img)
	} else {
		texture = t.createTextureArbitrary(name, uint32(picNum))
	}
	t.resources[name] = texture
	return name
}

// addTile creates a new texture from the given RGBA image and associates it with the specified picture number.
func (t *Textures) createTextureFromImage(name string, picNum uint32, img *image.RGBA) *textures.Texture {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	texture := textures.NewTexture(name, picNum, w, h, false)
	for y := 0; y < h; y++ {
		//flipY := h - 1 - y
		flipY := y
		for x := 0; x < w; x++ {
			c := img.RGBAAt(x, flipY)
			rgba := int(c.R)<<24 | int(c.G)<<16 | int(c.B)<<8 | int(c.A)
			texture.Set(x, y, rgba)
		}
	}
	return texture
}

// createArbitraryTexture generates a 64x64 textured image using a grid pattern with colors derived from the provided picNum.
func (t *Textures) createTextureArbitrary(name string, picNum uint32) *textures.Texture {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	r := uint8((picNum*37+50)%200 + 40)
	g := uint8((picNum*73+70)%200 + 40)
	b := uint8((picNum*109+90)%200 + 40)
	baseColor := color.RGBA{R: r, G: g, B: b, A: 255}
	borderColor := color.RGBA{R: r / 2, G: g / 2, B: b / 2, A: 255}
	crossColor := color.RGBA{R: min(r+30, 255), G: min(g+30, 255), B: min(b+30, 255), A: 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := baseColor
			if x == 0 || y == 0 || x == size-1 || y == size-1 {
				c = borderColor
			} else if (x == y || x == size-1-y) && (x < 16 || x > size-17) {
				c = crossColor
			} else if (x/8+y/8)%2 == 0 {
				c = color.RGBA{R: r - 15, G: g - 15, B: b - 15, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return t.createTextureFromImage(name, picNum, img)
}
