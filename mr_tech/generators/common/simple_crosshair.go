package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// SimpleCrosshair represents a graphical element used as a visual targeting reference, typically displayed at the center of the screen.
type SimpleCrosshair struct {
	tex  *textures.Texture
	name string
}

// NewSimpleCrosshair creates a new SimpleCrosshair instance with a predefined texture and dimensions.
func NewSimpleCrosshair() *SimpleCrosshair {
	name := "__CROSSHAIR__"
	w, h := 16, 16
	tex := textures.NewTexture(name, 999999, w, h, true)

	for i := 0; i < w; i++ {
		for j := 0; j < h; j++ {
			if (i == w/2 || i == w/2-1) && (j >= h/2-4 && j <= h/2+3) {
				tex.Set(i, j, 0xFFFFFFFF) // Centro
			} else if (j == h/2 || j == h/2-1) && (i >= w/2-4 && i <= w/2+3) {
				tex.Set(i, j, 0xFFFFFFFF) // Centro
			} else {
				tex.Set(i, j, 0x00000000) // Trasparente
			}
		}
	}
	return &SimpleCrosshair{
		name: name,
		tex:  tex,
	}
}

// GetTexture returns the name and texture associated with the Crosshair instance.
func (c *SimpleCrosshair) GetTexture() (string, *textures.Texture) {
	return c.name, c.tex
}

// Create initializes and returns a new `Thing` configured as a "crosshair" with specific materials, behaviors, and attributes.
func (c *SimpleCrosshair) Create() *config.Thing {
	mat := config.NewConfigMaterial([]string{c.name}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
	mat.BlendMode = config.BlendModeAdditive
	mat.ScaleW = 1.0
	mat.ScaleH = 1.0
	sprite := config.NewConfigSprite(mat)
	thing := config.NewConfigThing(
		"crosshair",
		geometry.XYZ{X: 0, Y: 0, Z: 0},
		0,
		config.ThingInterfaceDef,
		1, 0, 0, 0,
	)
	thing.OnImpact = func(self config.IThingConfig, other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {

	}
	thing.OnCollision = func(self config.IThingConfig, other config.IThingConfig) {

	}
	thing.OnThinking = func(self config.IThingConfig, playerX, playerY, playerZ float64) {

	}
	thing.Sprite = sprite
	return thing
}
