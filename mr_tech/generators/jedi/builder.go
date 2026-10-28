package jedi

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/jedi/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// aspectRatio defines the proportional relationship between width and height, commonly used in graphics and media contexts.
const aspectRatio = 1.1

// scaleFactor is a constant used to define a uniform scaling multiplier in various calculations.
const scaleFactor = 11.0

// scaleX represents the horizontal scaling factor derived from the base value defined by scaleFactor.
const scaleX = scaleFactor

// scaleY defines the vertical scaling factor for transformations, derived from the base value of scaleFactor.
const scaleY = scaleFactor

// scaleZ represents the scaling factor applied along the Z-axis, derived from the predefined scaleFactor constant.
const scaleZ = scaleFactor

// scaleSectorH represents a height scaling factor derived from the base scale factor.
const scaleSectorH = scaleFactor
const scaleSectorW = 8.0 / scaleSectorH

const scaleTextureW = 1.0 // scaleTextureW represents the horizontal scaling factor for a texture in the rendering process.
const scaleTextureH = 1.0 // scaleTextureH defines the horizontal scaling factor for texture rendering, set to a default value of 1.0.

// scaleLight represents the scaling factor applied for light intensity adjustments in specific calculations.
const scaleLight = 0.11

// scaleLightFalloff defines the falloff factor for light scaling, influencing the attenuation of light intensity over distance.
const scaleLightFalloff = 40

// playerHeight defines the height of the player model in the game, scaled proportionally to the sector height.
const playerHeight = 6.0 * scaleSectorH

// playerRadius defines the radius of the player entity, used for collision detection and spatial calculations.
const playerRadius = 2.5

// playerSpeed defines the movement speed of the player, measured in units per second.
const playerSpeed = 1800

// playerMass defines the default mass of the player in the game, measured in consistent in-game units.
const playerMass = 40

const GForce = 9.8 * 50

const JumpForce = GForce * 8

type EngineMode int

const (
	EngineModeJedi EngineMode = iota
	EngineModeOutlaws
)

// Builder provides a flexible mechanism for constructing complex objects step-by-step.
type Builder struct {
}

// NewBuilder initializes and returns a pointer to a new Builder instance.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build constructs a game configuration based on the provided mode, directory, and level number. Returns a config.Root or error.
func (b *Builder) Build(fs common.IResources, eMode EngineMode, dir string, levelNumber int) (*config.Root, error) {
	var archive lumps.IArchive
	if eMode == EngineModeOutlaws {
		archive = lumps.NewArchiveLab(fs)
	} else {
		archive = lumps.NewArchiveGob(fs)
	}

	levelNumber -= 1
	if levelNumber <= 0 {
		levelNumber = 0
	}
	if err := archive.Parse(dir); err != nil {
		return nil, err
	}
	defer archive.Close()

	if err := archive.SetLevel(levelNumber); err != nil {
		return nil, err
	}

	world := NewWorld(archive, eMode)
	cSectors, gVertices, lights, err := world.Build()
	if err != nil {
		return nil, err
	}

	things := NewThings(archive, eMode)
	cPlayer, cThings, err := things.Build()
	if err != nil {
		return nil, err
	}

	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = aspectRatio
	tex := archive.GetTextures()
	crosshair := common.NewSimpleCrosshair()
	tex.AddDirect(crosshair.GetTexture())

	cr := config.NewConfigRoot(cal, cSectors, cPlayer, cThings, geometry.XYZ{X: scaleX, Y: scaleY, Z: scaleZ}, tex)
	cr.Vertices = gVertices
	cr.Lights = lights
	cr.AddUIElement(crosshair.Create())

	return cr, nil
}

// CreateCoords generates a geometry.XYZ object from the provided x, y, and z coordinates with modified Y and Z values.
func CreateCoords(x, y, z float64) geometry.XYZ {
	//return geometry.XYZ{X: x, Y: -z, Z: -y}
	return geometry.XYZ{X: x, Y: z, Z: -y}
}
