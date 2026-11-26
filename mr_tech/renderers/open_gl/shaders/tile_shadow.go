package shaders

import "github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"

// Shadow tile indices reserve atlas slots for room, flashlight, dynamic light, and sun shadow maps.
const (
	// ShadowTileRoom is the atlas slot index for the orthographic room shadow tile.
	ShadowTileRoom = 0
	// ShadowTileFlash is the atlas slot index for the perspective flashlight shadow tile.
	ShadowTileFlash = 1
	// ShadowTileLightBase is the starting atlas slot index for dynamic spotlights.
	ShadowTileLightBase = 2
	// ShadowTileSunBase is the starting atlas slot index for the 4 outdoor sun CSM cascades.
	ShadowTileSunBase = 10
	// MaxShadowTiles is the total number of tiles managed inside the master shadow atlas.
	MaxShadowTiles = 14
)

// TileShadow stores cached rendering state for a shadow tile.
type TileShadow struct {
	active    bool
	renderDC  *draws.DrawCommandsRender
	matrix    [16]float32
	sceneHash uint64
	reuse     bool
}

// Assign updates the shadow tile state with render commands, transform matrix, scene hash, and reuse flag.
func (d *TileShadow) Assign(active bool, renderDC *draws.DrawCommandsRender, matrix [16]float32, sceneHash uint64, reuse bool) {
	d.active = active
	d.renderDC = renderDC
	d.matrix = matrix
	d.sceneHash = sceneHash
	d.reuse = reuse
}

// SetRenderDC sets the render draw commands for the shadow tile data.
func (d *TileShadow) SetRenderDC(renderDC *draws.DrawCommandsRender) {
	d.renderDC = renderDC
}

// SetActive sets whether the shadow tile data is active.
func (d *TileShadow) SetActive(active bool) {
	d.active = active
}

// SetReuse updates whether the shadow tile data can be reused without rendering.
func (d *TileShadow) SetReuse(reuse bool) {
	d.reuse = reuse
}

// IsActive reports whether the shadow tile data is marked active.
func (d *TileShadow) IsActive() bool {
	return d.active
}

// CanReuse reports whether this shadow tile data may reuse a cached atlas tile.
func (d *TileShadow) CanReuse() bool {
	return d.reuse
}

// GetRenderDC returns the render draw commands associated with the shadow tile data.
func (d *TileShadow) GetRenderDC() *draws.DrawCommandsRender {
	return d.renderDC
}
