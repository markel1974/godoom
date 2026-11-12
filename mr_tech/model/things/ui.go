package things

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

// UI represents a user interface thing with base object behavior and no physical interaction logic.
type UI struct {
	*Base
}

// NewUI creates a UI thing from the provided handler and configuration, initializes its base, and binds it.
func NewUI(things IThingHandler, cfg *config.Thing) *UI {
	ui := &UI{}
	ui.Base = NewBase(ui, things, cfg, nil)
	ui.SetAngle(0)
	cfg.OnBind(ui)
	return ui
}

// IsEqual reports whether other is the same UI instance.
func (t *UI) IsEqual(other volumes.IThing) bool {
	if v, ok := other.(*UI); ok && t == v {
		return true
	}
	return false
}

// Jump reports whether the UI performed a jump; UI elements do not jump, so it always returns false.
func (t *UI) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan emits a hitscan effect with the given ID, origin, force, and direction vector.
func (t *UI) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject launches a throwable object from pos with the given orientation, speed, and impact handlers.
func (t *UI) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage ignores the provided thing event and performs no action.
func (t *UI) PostMessage(_ *volumes.ThingEvent) {
}

// StartLoop starts the UI processing loop.
func (t *UI) StartLoop() {
}

// HasThinking reports whether the UI performs staged thinking updates.
func (t *UI) HasThinking() bool {
	return false
}

// StageThinking updates the UI thinking stage using the player's current position.
func (t *UI) StageThinking(playerX float64, playerY float64, playerZ float64) {
}

// VerticesCompile compiles base vertices for the given tick and always reports no UI-specific changes.
func (t *UI) VerticesCompile(tick uint64) bool {
	t.Base.VerticesCompile(tick)
	return false
}

// VerticesGet returns UI face data positioned at the origin with interface render mode.
func (t *UI) VerticesGet() (*volumes.Faces, *volumes.Faces, float64, float64) {
	//TODO posizione assoluta e' un pezzo dell'interfaccia
	t.GetEntity().MoveTo(0, 0, 0)
	f1, f2, lp, _ := t.Base.VerticesGet()
	// Usiamo 1.1 (Spherical Billboard) in modo che lo shader originale lo processi correttamente
	return f1, f2, lp, volumes.RenderModeInterface
}
