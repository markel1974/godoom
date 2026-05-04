package wad

import (
	"fmt"
	"image"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const FlatId = "__FLAT__"
const TextureId = "__TEXTURE__"
const SpriteId = "__SPRITE__"

// TextureCreateId generates a unique texture identifier by appending a fixed prefix to a sanitized version of the input id.
func TextureCreateId(id string) string {
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return TextureId + id
}

// FlatCreateId generates a flat identifier by appending a predefined prefix to a cleaned version of the given id.
func FlatCreateId(id string) string {
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return FlatId + id
}

// SpriteCreateId generates a unique sprite identifier by appending a prefix to a cleaned version of the input string.
func SpriteCreateId(id string) string {
	id = cleanId(id)
	if len(id) == 0 || id == "-" {
		return ""
	}
	return SpriteId + id
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
	emissive := false
	if len(srcId) > 0 && srcId[0] == '*' || srcId[0] == '+' {
		emissive = true
	}
	texture := textures.NewTexture(srcId, uint32(id), size.X, size.Y, emissive)
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			c := src.RGBAAt(x, y)
			rgba := int(c.R)<<24 | int(c.G)<<16 | int(c.B)<<8 | int(c.A)
			texture.Set(x, y, rgba)

			/*
				if c.A == 0 {
					texture.Set(x, y, 0)
				} else {
					//rgb := int(c.R)*65536 + int(c.G)*256 + int(c.B)
					rgba := int(c.R)<<24 | int(c.G)<<16 | int(c.B)<<8 | 0xff
					texture.Set(x, y, rgba)
				}

			*/
		}
	}
	t.resources[srcId] = texture
	//_ = common.SaveImage(srcId, src)
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

// BuildSprite estrae dal WAD le sequenze fornite e le compatta in un MultiSprite lineare.
func (t *Textures) BuildSprite(prefix string) *config.MultiSprite {
	var EnemyStateSequences = [][]byte{
		{'A', 'B', 'C', 'D'},      // Action 0: Walk / Chase
		{'E', 'F', 'G'},           // Action 1: Attack (Melee/Missile)
		{'H'},                     // Action 2: Pain
		{'I', 'J', 'K', 'L', 'M'}, // Action 3: Death
		{'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V'}, // Action 4: Extreme Death (Gibs)
	}

	const frames = 8

	target := SpriteCreateId(prefix[:4])

	type frameData [frames]string
	slots := make(map[byte]*frameData)

	for res := range t.resources {
		if !strings.HasPrefix(res, target) {
			continue
		}
		spriteName := strings.TrimPrefix(res, SpriteId)
		if len(spriteName) <= 5 {
			continue
		}
		frameChar := spriteName[4]
		angleChar := spriteName[5]
		if slots[frameChar] == nil {
			slots[frameChar] = &frameData{}
		}
		if angleChar == '0' {
			for i := 0; i < frames; i++ {
				slots[frameChar][i] = spriteName
			}
		} else {
			angleIdx := int(angleChar - '1')
			if angleIdx >= 0 && angleIdx < frames {
				slots[frameChar][angleIdx] = spriteName
			}
		}
		// Gestione della specchiatura (es. A2A8)
		if len(spriteName) == frames {
			frameChar2 := spriteName[6]
			angleChar2 := spriteName[7]
			if slots[frameChar2] == nil {
				slots[frameChar2] = &frameData{}
			}
			angleIdx2 := int(angleChar2 - '1')
			if angleIdx2 >= 0 && angleIdx2 < frames {
				slots[frameChar2][angleIdx2] = spriteName
			}
		}
	}

	ms := config.NewMultiSprite()

	for _, sequence := range EnemyStateSequences {
		for angle := 0; angle < frames; angle++ {
			var temporalFrames []string
			for _, frameLetter := range sequence {
				anglesData := slots[frameLetter]
				if anglesData == nil {
					continue
				}
				texId := anglesData[angle]
				if texId == "" {
					texId = anglesData[0]
				}
				temporalFrames = append(temporalFrames, SpriteCreateId(texId))
			}
			if len(temporalFrames) == 0 {
				ms.Add(nil)
				continue
			}
			mat := config.NewConfigMaterial(temporalFrames, config.MaterialKindLoop, ScaleWThings, ScaleHThings, 0, 0)
			ms.Add(mat)
		}
	}

	return ms
}

/*
func (t *Textures) GetSprite(id string) []string {
	var names []string
	src := SpriteCreateId(id)
	for k := range t.resources {
		if strings.HasPrefix(k, src) {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	return names
}
*/
