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
	anims     map[int]*ArtTileAnim
}

// NewTextures creates and returns a new instance of Textures with initialized resource and cache maps.
func NewTextures() *Textures {
	return &Textures{
		resources: make(map[string]*textures.Texture),
		depot:     make(map[int]*image.RGBA),
		anims:     make(map[int]*ArtTileAnim),
	}
}

// AddAnims copies a map of tile animation parameters into the textures cache.
func (t *Textures) AddAnims(anims map[int]*ArtTileAnim) {
	for k, v := range anims {
		t.anims[k] = v
	}
}

// AddAnim stores an animation descriptor for a specific picture number.
func (t *Textures) AddAnim(picNum int, anim *ArtTileAnim) {
	t.anims[picNum] = anim
}

// AddDepot adds an RGBA image to the depot map indexed by the specified texture ID.
func (t *Textures) AddDepot(tId int, img *image.RGBA) {
	t.depot[tId] = img
}

// AddDirect adds a texture to the resources map using the specified name if the provided texture is not nil.
func (t *Textures) AddDirect(name string, tex *textures.Texture) {
	if tex != nil {
		t.resources[name] = tex
	}
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

// GetNames retrieves all the keys of the resources map in the Textures instance sorted alphabetically for determinism.
func (t *Textures) GetNames() []string {
	names := make([]string, 0, len(t.resources))
	for k := range t.resources {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
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

// GetTileFrames returns the list of texture frame names for an animated tile, or a single frame if static.
func (t *Textures) GetTileFrames(picNum int) []string {
	anim, ok := t.anims[picNum]
	if !ok || anim.AnimType == 0 || anim.NumFrames <= 0 {
		return []string{t.GetTile(picNum)}
	}

	pics := make([]int, 0, anim.NumFrames*2+1)
	switch anim.AnimType {
	case 1: // Oscillating (ping-pong)
		for i := 0; i <= anim.NumFrames; i++ {
			pics = append(pics, picNum+i)
		}
		for i := anim.NumFrames - 1; i >= 1; i-- {
			pics = append(pics, picNum+i)
		}
	case 2: // Forward loop
		for i := 0; i <= anim.NumFrames; i++ {
			pics = append(pics, picNum+i)
		}
	case 3: // Backward loop
		for i := 0; i <= anim.NumFrames; i++ {
			pics = append(pics, picNum-i)
		}
	default:
		return []string{t.GetTile(picNum)}
	}

	frames := make([]string, 0, len(pics))
	for _, p := range pics {
		frames = append(frames, t.GetTile(p))
	}
	return frames
}

// GetTileSize retrieves the width and height in pixels for the specified tile.
func (t *Textures) GetTileSize(picNum int) (int, int, bool) {
	if img, ok := t.depot[picNum]; ok && img != nil {
		b := img.Bounds()
		return b.Dx(), b.Dy(), true
	}
	return 0, 0, false
}

// GetSkyTile retrieves the name of a composite 360-degree cylindrical sky texture, or creates one if it does not exist.
func (t *Textures) GetSkyTile(picNum int) string {
	var panels []int
	switch {
	case picNum >= 89 && picNum <= 93:
		panels = []int{89, 91, 92, 90, 91, 90, 92, 93} //Los Angeles multi-panel panorama (8 panels of 128px = 1024px total)
	case picNum >= 80 && picNum <= 83:
		panels = []int{80, 81, 80, 80, 82, 83, 80, 82} // Moon sky (8 panels)
	case picNum >= 84 && picNum <= 88:
		panels = []int{84, 85, 86, 87, 84, 84, 88, 84} // Big Orbit / space station sky (8 panels)
	default:
		return t.GetTile(picNum)
	}

	name := fmt.Sprintf("duke_sky_%d", panels[0])
	if _, exists := t.resources[name]; exists {
		return name
	}

	const canvasSize = 512
	const horizonY = canvasSize * 0.1

	yH := horizonY

	skyImg := image.NewRGBA(image.Rect(0, 0, canvasSize, canvasSize))

	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	for y := 0; y < canvasSize; y++ {
		for x := 0; x < canvasSize; x++ {
			skyImg.SetRGBA(x, y, black)
		}
	}

	// In OpenGL texture space:
	// y = 0 is bottom (nadir, v = 0.0)
	// y = 512 is horizon (d.y = 0, v = 0.5)
	// y = 1023 is top (zenith, v = 1.0)
	// In Duke3D tiles, row 0 is top of hills / stars, and row h-1 is bottom of buildings.
	// We position the base of the skyline at horizonY (v ~ 0.47) so the horizon is grounded.
	curX := 0
	for _, p := range panels {
		img, ok := t.depot[p]
		if !ok || img == nil {
			curX += 32
			continue
		}
		w := img.Bounds().Dx()
		h := img.Bounds().Dy()
		for x := 0; x < w && curX+x < canvasSize; x++ {
			for y := 0; y < h; y++ {
				targetY := int(yH) + (h - 1 - y)
				if targetY >= 0 && targetY < canvasSize {
					skyImg.SetRGBA(curX+x, targetY, img.RGBAAt(x, y))
				}
			}
			topCol := img.RGBAAt(x, 0)
			for targetY := int(yH) + h; targetY < canvasSize; targetY++ {
				skyImg.SetRGBA(curX+x, targetY, topCol)
			}
		}
		curX += w
	}

	texture := t.createTextureFromImage(name, uint32(panels[0]), skyImg)
	t.resources[name] = texture
	return name
}

// createTextureFromImage creates a new texture from the given RGBA image and associates it with the specified picture number.
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
