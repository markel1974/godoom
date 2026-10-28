package wolfstein

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
)

const (
	AspectRatio  = 1.333
	ScaleSectorH = 1.0
	PlayerHeight = 56.0 * ScaleSectorH
	PlayerRadius = 16.0
	PlayerSpeed  = 2500.0
	PlayerMass   = 50.0

	GForce    = 9.8 * 50
	JumpForce = GForce * 8

	ScaleFactorH = 5.2

	// floorCeilScaleW defines the default scaling factor for floor and ceiling texture animations in the configuration.
	floorCeilScaleW = 1.0

	// floorCeilScaleH defines the scale factor applied to the height of floor and ceiling textures in a sector.
	floorCeilScaleH = 1.0

	playerHeight = 10 * ScaleFactorH

	falloff = 40.0

	lightIntensity = 1.0 * 3
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
	cr.AddUIElement(crosshair.Create())
	return cr, nil
}
