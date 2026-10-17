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
	const borderSize = 2
	name := "__CROSSHAIR__"
	w, h := 16, 16
	tex := textures.NewTexture(name, 999999, w, h, true)

	isCrosshair := func(i, j int) bool {
		return ((i == w/2 || i == w/2-1) && (j >= h/2-4 && j <= h/2+3)) || ((j == h/2 || j == h/2-1) && (i >= w/2-4 && i <= w/2+3))
	}

	for i := 0; i < w; i++ {
		for j := 0; j < h; j++ {
			if isCrosshair(i, j) {
				// Bordo nero: controlla gli 8 pixel adiacenti.
				for dx := -borderSize; dx <= borderSize; dx++ {
					for dy := -borderSize; dy <= borderSize; dy++ {
						x, y := i+dx, j+dy
						if x >= 0 && x < w && y >= 0 && y < h {
							if !isCrosshair(x, y) {
								tex.Set(x, y, 0xff1111ff)
							}
						}
					}
				}
			}
		}
	}

	// Disegna il mirino bianco sopra il bordo nero.
	for i := 0; i < w; i++ {
		for j := 0; j < h; j++ {
			if isCrosshair(i, j) {
				tex.Set(i, j, 0xFFFFFFFF)
			} else {
				// I pixel non coinvolti nel bordo restano trasparenti.
				// Il bordo nero già disegnato viene preservato.
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
	thing.OnBind = func(self config.IThingConfig) {}
	thing.OnImpact = func(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {}
	thing.OnCollision = func(other config.IThingConfig) {}
	thing.OnThinking = func(playerX, playerY, playerZ float64) {}
	thing.Sprite = sprite
	return thing
}
