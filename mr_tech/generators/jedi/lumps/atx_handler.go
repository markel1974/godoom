package lumps

import (
	"bytes"
	"fmt"
)

// ATXHandler is responsible for handling animated textures (ATX) by managing their retrieval, parsing, and rendering.
type ATXHandler struct {
	al         IRetriever
	atxMap     map[string]*ATX
	frameToAtx map[string]string
	tex        *Textures
	palettes   *Palettes
}

// NewATXS initializes and returns a new instance of ATXHandler with the provided IRetriever, Textures, and Palettes instances.
func NewATXS(al IRetriever, tex *Textures, palettes *Palettes) *ATXHandler {
	return &ATXHandler{
		al:         al,
		tex:        tex,
		atxMap:     make(map[string]*ATX),
		frameToAtx: make(map[string]string),
		palettes:   palettes,
	}
}

// AddScript retrieves, parses, and stores an ATX script, updating internal maps for animation handling when looping.
func (ah *ATXHandler) AddScript(name string) error {
	cleanName := CleanName(name)
	data, pErr := ah.al.GetPayload(name)
	if pErr != nil {
		return pErr
	}
	atx := NewATX(ah.al, name)
	if err := atx.Parse(bytes.NewReader(data)); err != nil {
		return err
	}
	ah.atxMap[cleanName] = atx
	if atx.IsLooping() {
		for _, tex := range atx.Textures {
			cleanTex := CleanName(tex)
			ah.frameToAtx[cleanTex] = cleanName
		}
	}
	return nil
}

// CompileScriptTexture generates texture frames for the specified ATX script using the given color palette and returns them.
func (ah *ATXHandler) CompileScriptTexture(cleanAtx string, palIndex int) ([]string, error) {
	atx, ok := ah.atxMap[cleanAtx]
	if !ok {
		data, pErr := ah.al.GetPayload(cleanAtx)
		if pErr != nil {
			return nil, pErr
		}
		atx = NewATX(ah.al, cleanAtx)
		if err := atx.Parse(bytes.NewReader(data)); err != nil {
			return nil, err
		}
		ah.atxMap[cleanAtx] = atx
	}
	if len(atx.Textures) == 0 {
		return nil, fmt.Errorf("atx %s has no textures", cleanAtx)
	}
	colorPal, cErr := ah.palettes.Get(palIndex)
	if cErr != nil {
		return nil, cErr
	}
	images, err := atx.CreateImages(colorPal)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("no valid frames loaded for atx %s", cleanAtx)
	}
	out := ah.tex.AddTexture(cleanAtx, images)
	for _, frameName := range atx.Textures {
		ah.tex.SetCache(CleanName(frameName), out)
	}
	return out, nil
}

// CyclicFrame retrieves cyclic animation frames for a texture, returning the frames and a success flag.
func (ah *ATXHandler) CyclicFrame(cleanName string, palIndex int) ([]string, bool) {
	atxName, ok := ah.frameToAtx[cleanName]
	if !ok {
		return nil, false
	}
	frames, err := ah.CompileScriptTexture(atxName, palIndex)
	if err != nil {
		return nil, false
	}
	if len(frames) == 0 {
		return nil, false
	}
	return frames, true
}
