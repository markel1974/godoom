package lumps

import (
	"fmt"
	"image"
	"image/color"

	"github.com/markel1974/godoom/mr_tech/textures"
)

type Textures struct {
	resources map[string]*textures.Texture
	cache     map[int]string
}

func NewTextures() *Textures {
	return &Textures{
		resources: make(map[string]*textures.Texture),
		cache:     make(map[int]string),
	}
}

func (t *Textures) GetNames() []string {
	names := make([]string, 0, len(t.resources))
	for k := range t.resources {
		names = append(names, k)
	}
	return names
}

func (t *Textures) Get(names []string) []*textures.Texture {
	out := make([]*textures.Texture, 0, len(names))
	for _, name := range names {
		if tex, ok := t.resources[name]; ok {
			out = append(out, tex)
		}
	}
	return out
}

func (t *Textures) AddDirect(name string, tex *textures.Texture) {
	if tex != nil {
		t.resources[name] = tex
	}
}

// GetOrCreateTile retrieves or generates a tile texture by tile index.
func (t *Textures) GetOrCreateTile(picNum int) string {
	if name, ok := t.cache[picNum]; ok {
		return name
	}

	name := fmt.Sprintf("duke_tile_%d", picNum)
	if _, exists := t.resources[name]; exists {
		t.cache[picNum] = name
		return name
	}

	// Generate a 64x64 texture with a subtle grid and color based on picNum.
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Base palette generation from picNum
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

	texture := textures.NewTexture(name, uint32(picNum), size, size, false)
	for y := 0; y < size; y++ {
		flipY := size - 1 - y
		for x := 0; x < size; x++ {
			c := img.RGBAAt(x, flipY)
			rgba := int(c.R)<<24 | int(c.G)<<16 | int(c.B)<<8 | int(c.A)
			texture.Set(x, y, rgba)
		}
	}
	t.resources[name] = texture
	t.cache[picNum] = name
	return name
}
