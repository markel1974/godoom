package wad

import (
	"fmt"
	"image"
	"strings"

	"github.com/markel1974/godoom/engine/textures"
)

var _animationsBase = [][]string{
	//flats
	{"NUKAGE1", "NUKAGE2", "NUKAGE3"},
	{"FWATER1", "FWATER2", "FWATER3", "FWATER4"},
	{"SWATER1", "SWATER2", "SWATER3", "SWATER4"},
	{"LAVA1", "LAVA2", "LAVA3", "LAVA4"},
	{"BLOOD1", "BLOOD2", "BLOOD3"},
	{"FIRELAVA", "FIRELAV2", "FIRELAV3"},
	{"FIREWALA", "FIREWALB", "FIREWALL"},
	//textures
	{"BLODGR1", "BLODGR2", "BLODGR3", "BLODGR4"},
	{"SLADRIP1", "SLADRIP2", "SLADRIP3"},
	{"BLODRIP1", "BLODRIP2", "BLODRIP3", "BLODRIP4"},
	{"FIREMAG1", "FIREMAG2", "FIREMAG3"},
	{"FIREBLU1", "FIREBLU2"},
	{"ROCKRED1", "ROCKRED2", "ROCKRED3"},
	{"GSTFONT1", "GSTFONT2", "GSTFONT3"},
}

var _animations map[string][]string

func init() {
	_animations = make(map[string][]string)
	for _, v := range _animationsBase {
		for _, a := range v {
			_animations[a] = v
		}
	}
}

func TextureCreateAnimation(id string) []string {
	id = cleanId(id)
	if animation, ok := _animations[id]; ok {
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

func FlatCreateAnimation(id string) []string {
	id = cleanId(id)
	if animation, ok := _animations[id]; ok {
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

func TextureCreateId(id string) string {
	const textureId = "__TEXTURE__"
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return textureId + id
}

func FlatCreateId(id string) string {
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

// Get retrieves the texture associated with the given ID from the Textures' resources map. Returns nil if not found.
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
