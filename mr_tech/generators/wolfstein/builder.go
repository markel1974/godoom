package wolfstein

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
)

// Builder represents a configurable utility type that constructs complex data structures or configurations.
type Builder struct {
}

// NewBuilder initializes and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build constructs a Root configuration by parsing original map data through the Parser, based on the specified level.
func (b *Builder) Build(level int) (*config.Root, error) {
	tex, tErr := NewTextures()
	if tErr != nil {
		return nil, tErr
	}
	w, h, data := GetOriginalMapData()
	wp := NewParser(8, 15, true)
	cr, err := wp.Parse(tex, w, h, data)
	if err != nil {
		return nil, err
	}
	crosshair := common.NewSimpleCrosshair()
	tex.AddDirect(crosshair.GetTexture())
	cr.Crosshair = crosshair.Create()
	return cr, nil
}
