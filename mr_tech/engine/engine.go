package engine

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/portal"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Engine represents a core game simulation system, managing entities, volumes, player, and rendering configurations.
type Engine struct {
	portal     *portal.Portal
	maxQueue   int
	viewFactor float64
	things     *model.Things
	entities   *model.Entities
	player     *model.ThingPlayer
	volumes    *model.Volumes
	lights     *model.Lights
}

// NewEngine creates and initializes a new Engine instance with the specified width, height, and maximum queue size.
func NewEngine(maxQueue int, viewFactor float64) *Engine {
	return &Engine{
		portal:     nil,
		maxQueue:   maxQueue,
		viewFactor: viewFactor,
		things:     nil,
		entities:   nil,
		volumes:    nil,
		player:     nil,
		lights:     nil,
	}
}

// GetPlayer returns the current player instance managed by the engine.
func (e *Engine) GetPlayer() *model.ThingPlayer {
	return e.player
}

// GetTextures retrieves the ITextures instance, providing access to texture names and indexed textures.
func (e *Engine) GetTextures() textures.ITextures {
	return e.things.GetTextures()
}

// GetThings retrieves a slice of IThing instances managed by the Engine's Things component.
func (e *Engine) GetThings() *model.Things {
	return e.things
}

// GetLights retrieves the list of light sources currently managed by the engine.
func (e *Engine) GetLights() *model.Lights {
	return e.lights
}

// PortalVolumeAt returns the volume at the specified index from the portal within the engine.
func (e *Engine) PortalVolumeAt(idx int) *model.Volume {
	return e.portal.VolumeAt(idx)
}

// QueryFrustum checks which objects intersect the provided frustum and invokes the callback for each intersecting object.
func (e *Engine) QueryFrustum(frustum *physics.Frustum, callback func(object physics.IAABB) bool) {
	e.volumes.QueryFrustum(frustum, callback)
}

// QueryMultiFrustum checks which objects intersect both the front and rear frustums and invokes the callback for each intersecting object.
func (e *Engine) QueryMultiFrustum(front, rear *physics.Frustum, callback func(object physics.IAABB) bool) {
	e.volumes.QueryMultiFrustum(front, rear, callback)
}

// Len returns the number of volumes currently managed by the Engine.
func (e *Engine) Len() int {
	return e.portal.Len()
}

// Setup initializes the Engine using the provided configuration, creating volumes, player, things, entities, and the portal.
func (e *Engine) Setup(cfg *config.ConfigRoot) error {
	compiler := model.NewCompiler()
	if err := compiler.Compile(cfg); err != nil {
		return err
	}
	e.volumes = compiler.GetVolumes()
	e.player = compiler.GetPlayer()
	e.things = compiler.GetThings()
	e.entities = compiler.GetEntities()
	e.lights = compiler.GetLights()

	e.portal = portal.NewPortal(e.maxQueue, e.viewFactor)
	if err := e.portal.Setup(e.volumes.GetVolumes()); err != nil {
		return err
	}
	return nil
}

// Compute updates the game state by synchronizing the player, processing AI, applying physics, and updating the view matrix.
func (e *Engine) Compute(player *model.ThingPlayer, vi *model.ViewMatrix) {
	// 1. Pre-Sync ViewMatrix
	vi.Update(player)

	// 2. AI & External Forces: Wake up entities BEFORE physics calculation
	pX, pY, pZ := player.GetPosition()

	e.things.Compute(pX, pY, pZ)

	// 3. Static ThingPlayer Motion
	player.Update(vi)

	// 4. Dynamic Solver
	entities := e.entities.Compute()

	// 5. Sync Up (Physics -> Model) - Things
	for _, ent := range entities {
		ent.PhysicsApply()
	}

	// 6. Post-Sync ViewMatrix
	vi.Update(player)

	// 7. Update Textures
	textures.Tick()
}

// Traverse processes the given ViewMatrix through the portal system, returning a list of compiled volumes and their count.
func (e *Engine) Traverse(fbw, fbh int32, vi *model.ViewMatrix) ([]*model.CompiledVolume, int) {
	cs, count := e.portal.Traverse(fbw, fbh, vi)
	return cs, count
}

// Build generates and retrieves the list of compiled volumes, their count, active game entities, and lights in the engine.
func (e *Engine) Build() ([]*model.CompiledVolume, int) {
	cs, count := e.portal.Build()
	return cs, count
}

// Fire spawns a bullet in the specified sector at the given coordinates and angle.
func (e *Engine) Fire(volume *model.Volume, pos geometry.XYZ, angle float64, pitch float64) {
	e.things.CreateBullet(volume, pos, angle, pitch)
}

// GetCalibration retrieves calibration parameters used for rendering, derived from the volumes' spatial configuration.
func (e *Engine) GetCalibration() *model.Calibration {
	return e.volumes.GetCalibration()
}
