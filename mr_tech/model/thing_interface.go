package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// ThingInterface is a specialized wrapper around ThingBase, representing a HUD entity that follows the camera.
type ThingInterface struct {
	*ThingBase
}

// NewThingInterface creates a new ThingInterface instance.
func NewThingInterface(things *Things, cfg *config.Thing) *ThingInterface {
	inter := &ThingInterface{}
	inter.ThingBase = NewThingBase(inter, things, cfg, nil)
	cfg.OnBind(inter)
	return inter
}

// IsEqual reports whether other is the same ThingInterface instance.
func (t *ThingInterface) IsEqual(other IThing) bool {
	if v, ok := other.(*ThingInterface); ok && t == v {
		return true
	}
	return false
}

// Jump attempts to make the enemy leap with the given horizontal movement and vertical scaling, reporting success.
func (t *ThingInterface) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *ThingInterface) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *ThingInterface) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage sends an ThingEvent instance to the ThingItem's inbox channel for processing.
func (t *ThingInterface) PostMessage(_ *ThingEvent) {
}

// StartLoop begins a goroutine that processes incoming events.
func (t *ThingInterface) StartLoop() {
}

// HasThinking determines if the ThingInterface instance has a thinking function assigned. Returns true if it is set.
func (t *ThingInterface) HasThinking() bool {
	return false
}

func (t *ThingInterface) StageThinking(playerX float64, playerY float64, playerZ float64) {
}

// CompileVertices rebuilds the interface vertex data for the given tick and reports no further updates.
func (t *ThingInterface) VerticesCompile(tick uint64) bool {
	t.ThingBase.VerticesCompile(tick)
	return false
}

// GetVertices overrides the default GetVertices to use the HUD render mode.
// The vertex shader will bypass the view matrix completely and place it in screen space.
func (t *ThingInterface) VerticesGet() (*Faces, *Faces, float64, float64) {
	//TODO posizione assoluta e' un pezzo dell'interfaccia
	t.VerticesGetEntity().MoveTo(0, 0, 0)
	f1, f2, lp, _ := t.ThingBase.VerticesGet()
	// Usiamo 1.1 (Spherical Billboard) in modo che lo shader originale lo processi correttamente
	return f1, f2, lp, RenderModeInterface
}
