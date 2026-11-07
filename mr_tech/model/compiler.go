package model

import (
	"fmt"
	"time"

	"github.com/markel1974/godoom/mr_tech/config"
	geometry "github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/compilers"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	materials2 "github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

type ICompiler interface {
	Compile() ([]*volumes.Volume, []*lights.Light, error)
}

// Compiler represents a core game engine component for managing world, game objects, player interactions, and things.
type Compiler struct {
	gScale      geometry.XYZ
	volumes     *Volumes
	player      *things.Player
	lights      *Lights
	things      *Things
	ui          []volumes.IThing
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
	now := time.Now()
	fmt.Println("starting compiling world")
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

	materials := materials2.NewMaterials(cfg.GetTextures())
	if err := materials.Setup(); err != nil {
		return err
	}

	pipeline := []ICompiler{
		compilers.NewCompiler2d(cfg, materials),
		compilers.NewCompiler3d(cfg, materials),
		compilers.NewCompilerLights(cfg.Lights),
	}

	var allVolumes []*volumes.Volume
	var allLights []*lights.Light

	for _, compiler := range pipeline {
		vs, ls, err := compiler.Compile()
		if err != nil {
			return err
		}
		if len(vs) > 0 {
			allVolumes = append(allVolumes, vs...)
		}
		if len(ls) > 0 {
			allLights = append(allLights, ls...)
		}
	}
	fmt.Printf("pipeline compiled in %s \n", time.Since(now))

	r.volumes = NewVolumes(allVolumes)
	if err := r.volumes.Setup(); err != nil {
		return err
	}
	fmt.Printf("volumes initialized in %s \n", time.Since(now))

	r.lights = NewLights(allLights)
	if err := r.lights.Setup(); err != nil {
		return err
	}
	fmt.Printf("lights initialized in %s \n", time.Since(now))

	r.things = NewThings(cfg.Calibration.MaxOverlaps, cfg.Calibration.SolverIterations, cfg.Things, r.volumes, materials)
	if err := r.things.Setup(); err != nil {
		return err
	}
	fmt.Printf("things initialized in %s \n", time.Since(now))

	r.player = things.NewPlayer(r.volumes, r.things, cfg.Player, false)
	if err := r.player.Setup(); err != nil {
		return err
	}
	r.things.SetPlayer(r.player)

	for _, i := range cfg.UI {
		e := things.NewUI(r.things, i)
		r.things.AddStaticThing(e)
		r.ui = append(r.ui, e)
	}

	r.calibration = NewCalibration(cfg.Calibration, r.volumes)
	if err := r.calibration.Setup(); err != nil {
		return err
	}

	fmt.Printf("world compiled in %s (volumes: %d lights: %d)\n", time.Since(now), r.volumes.Len(), r.lights.Len())
	return nil
}

// GetThings returns the Things instance managed by the Compiler.
func (r *Compiler) GetThings() *Things {
	return r.things
}

// GetUI returns the list of UI-related IThing instances managed by the Compiler.
func (r *Compiler) GetUI() []volumes.IThing {
	return r.ui
}

// GetVolumes retrieves the Volumes instance associated with the current Compiler object.
func (r *Compiler) GetVolumes() *Volumes {
	return r.volumes
}

// GetPlayer returns the player object associated with the compiler instance.
func (r *Compiler) GetPlayer() *things.Player {
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
