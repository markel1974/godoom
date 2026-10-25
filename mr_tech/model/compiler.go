package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	geometry "github.com/markel1974/godoom/mr_tech/geometry"
)

type ICompiler interface {
	Compile() ([]*Volume, error)
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
	r.lights = NewLights()

	compilers := []ICompiler{
		NewCompiler2d(cfg, materials, r.lights),
		NewCompiler3d(cfg, materials, r.lights),
	}

	var allVolumes []*Volume

	for _, compiler := range compilers {
		av, err := compiler.Compile()
		if err != nil {
			return err
		}
		allVolumes = append(allVolumes, av...)
	}

	r.volumes = NewVolumes(allVolumes)
	r.volumes.Setup()

	r.lights.AddLights(r.compileLights(cfg.Lights))
	r.things = NewThings(r.gScale, 10, cfg.Things, r.volumes, materials)
	r.player = NewThingPlayer(r.things, cfg.Player, r.volumes, false)
	if r.player == nil {
		return fmt.Errorf("player not found")
	}
	r.things.SetPlayer(r.player)

	if cfg.Crosshair != nil {
		vol, _ := r.volumes.QueryPoint(cfg.Crosshair.Position.X, cfg.Crosshair.Position.Y, cfg.Crosshair.Position.Z)
		r.crosshair = r.things.createThing(cfg.Crosshair, vol)
		r.things.AddStaticThing(r.crosshair)
	}
	r.calibration = NewCalibration(cfg.Calibration, r.volumes)
	fmt.Printf("Scan complete world: %d\n", r.volumes.Len())
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

// compileLights processes a list of configuration light positions and returns a slice of initialized Light objects.
func (r *Compiler) compileLights(cLights []*config.Light) []*Light {
	var out []*Light
	for _, cl := range cLights {
		if cl == nil {
			continue
		}
		light := NewLight()
		light.Setup(cl, cl.Pos)
		out = append(out, light)
	}
	return out
}
