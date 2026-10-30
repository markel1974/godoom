package engine

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	textures2 "github.com/markel1974/godoom/mr_tech/model/textures"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Engine represents the core structure for managing game components and interactions.
type Engine struct {
	things *model.Things
	player *model.ThingPlayer
	//ui          []model.IThing
	volumes     *model.Volumes
	lights      *lights.Lights
	calibration *model.Calibration
}

// NewEngine creates and initializes a new Engine instance with the specified width, height, and maximum queue size.
func NewEngine() *Engine {
	return &Engine{
		things:  nil,
		volumes: nil,
		player:  nil,
		lights:  nil,
	}
}

// GetPlayer returns the current player instance managed by the engine.
func (e *Engine) GetPlayer() *model.ThingPlayer {
	return e.player
}

// GetTextures retrieves the ITextures instance, providing access to texture names and indexed textures.
func (e *Engine) GetTextures() textures2.ITextures {
	return e.things.GetTextures()
}

// GetThings returns the Things instance managed by the Engine.
func (e *Engine) GetThings() *model.Things {
	return e.things
}

// GetUI retrieves the list of UI-related IThing instances currently managed by the Engine.
//func (e *Engine) GetUI() []model.IThing {
//	return e.ui
//}

// GetLights retrieves the list of light sources currently managed by the engine.
func (e *Engine) GetLights() *lights.Lights {
	return e.lights
}

// QueryFrustum checks which objects intersect the provided frustum and invokes the callback for each intersecting object.
func (e *Engine) QueryFrustum(frustum *aabb.Frustum, callback func(object aabb.IObject) bool) {
	e.volumes.QueryFrustum(frustum, callback)
}

// QueryMultiFrustum checks which objects intersect both the front and rear frustums and invokes the callback for each intersecting object.
func (e *Engine) QueryMultiFrustum(front, rear *aabb.Frustum, callback func(object aabb.IObject) bool) {
	e.volumes.QueryMultiFrustum(front, rear, callback)
}

// Setup initializes the Engine using the provided configuration, creating volumes, player, things, things, and the portal.
func (e *Engine) Setup(cfg *config.Root) error {
	compiler := model.NewCompiler()
	if err := compiler.Compile(cfg); err != nil {
		return err
	}
	e.player = compiler.GetPlayer()
	//e.ui = compiler.GetUI()
	e.things = compiler.GetThings()
	e.lights = compiler.GetLights()
	e.calibration = compiler.GetCalibration()
	e.volumes = compiler.GetVolumes()
	return nil
}

// Compute updates the game state by synchronizing the player, processing AI, applying physics, and updating the view matrix.
func (e *Engine) Compute(player *model.ThingPlayer, vi *model.ViewMatrix) {
	// AI & External Forces: Wake up things BEFORE physics calculation
	pX, pY, pZ := player.GetEntity().GetCenter()
	// Dynamic Solver
	e.things.Compute(pX, pY, pZ)
	// Post-Sync ViewMatrix
	vi.Update(player)
	// Update Textures
	textures2.Tick()
}

// GetCalibration retrieves calibration parameters used for rendering, derived from the volumes' spatial configuration.
func (e *Engine) GetCalibration() *model.Calibration {
	return e.calibration
}

// GetVolumes returns the Volumes instance currently managed by the Engine.
func (e *Engine) GetVolumes() *model.Volumes {
	return e.volumes
}
