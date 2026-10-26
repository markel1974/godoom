package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	geometry "github.com/markel1974/godoom/mr_tech/geometry"
)

type ICompiler interface {
	Compile() ([]*Volume, []*Light, error)
}

// Compiler represents a core game engine component for managing world, game objects, player interactions, and things.
type Compiler struct {
	gScale      geometry.XYZ
	volumes     *Volumes
	player      *ThingPlayer
	lights      *Lights
	things      *Things
	crosshair   IThing
	calibration *Calibration
}

// NewCompiler initializes and returns a new instance of Compiler with default-nil-initialized fields.
func NewCompiler() *Compiler {
	return &Compiler{
		volumes: nil,
		player:  nil,
		things:  nil,
		lights:  nil,
	}
}

// Compile initializes and processes game data from the provided configuration, returning an error if compilation fails.
func (r *Compiler) Compile(cfg *config.Root) error {
	r.gScale = cfg.ScaleFactor
	if r.gScale.X == 0 {
		r.gScale.X = 1
	}
	if r.gScale.Y == 0 {
		r.gScale.Y = 1
	}
	if r.gScale.Z == 0 {
		r.gScale.Z = 1
	}

	cfg.Scale(r.gScale)

	materials := NewMaterials(cfg.GetTextures())
	if err := materials.Setup(); err != nil {
		return err
	}

	pipeline := []ICompiler{
		NewCompiler2d(cfg, materials),
		NewCompiler3d(cfg, materials),
		NewCompilerLights(cfg.Lights),
	}

	var allVolumes []*Volume
	var allLights []*Light

	for _, compiler := range pipeline {
		volumes, lights, err := compiler.Compile()
		if err != nil {
			return err
		}
		if len(volumes) > 0 {
			allVolumes = append(allVolumes, volumes...)
		}
		if len(lights) > 0 {
			allLights = append(allLights, lights...)
		}
	}

	r.volumes = NewVolumes(allVolumes)
	if err := r.volumes.Setup(); err != nil {
		return err
	}

	r.lights = NewLights(allLights)
	if err := r.lights.Setup(); err != nil {
		return err
	}

	r.things = NewThings(r.gScale, 10, cfg.Things, r.volumes, materials)
	if err := r.things.Setup(); err != nil {
		return err
	}

	r.player = NewThingPlayer(r.volumes, r.things, cfg.Player, false)
	if err := r.player.Setup(); err != nil {
		return err
	}

	if cfg.Crosshair != nil {
		vol, _ := r.volumes.QueryPoint(cfg.Crosshair.Position.X, cfg.Crosshair.Position.Y, cfg.Crosshair.Position.Z)
		r.crosshair = r.things.createThing(cfg.Crosshair, vol)
		r.things.AddStaticThing(r.crosshair)
	}
	r.calibration = NewCalibration(cfg.Calibration, r.volumes)
	fmt.Printf("Scan complete world: volumes %d lights: %d\n", r.volumes.Len(), r.lights.Len())
	return nil
}

// GetThings returns the Things instance managed by the Compiler.
func (r *Compiler) GetThings() *Things {
	return r.things
}

// GetCrosshair returns the compiled crosshair thing.
func (r *Compiler) GetCrosshair() IThing {
	return r.crosshair
}

// GetVolumes retrieves the Volumes instance associated with the current Compiler object.
func (r *Compiler) GetVolumes() *Volumes {
	return r.volumes
}

// GetPlayer returns the player object associated with the compiler instance.
func (r *Compiler) GetPlayer() *ThingPlayer {
	return r.player
}

// GetLights retrieves the list of Light objects managed by the Compiler.
func (r *Compiler) GetLights() *Lights {
	return r.lights
}

// GetCalibration returns the Calibration instance associated with the Compiler.
func (r *Compiler) GetCalibration() *Calibration {
	return r.calibration
}
