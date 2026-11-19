package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Teleport handles binding and event callbacks for a teleport entity.
type Teleport struct {
	*config.Thing
	self config.IThingConfig
}

// NewTeleport creates a Teleport and assigns its bind, collision, and impact handlers.
func NewTeleport(id string, pos geometry.XYZ, radius, height float64, spec *config.TeleportSpec) *Teleport {
	return NewTeleportFromConfig(config.NewConfigTeleport(id, pos, radius, height, spec))
}

// NewTeleportFromConfig wraps an existing teleport Thing configuration.
func NewTeleportFromConfig(cfgThing *config.Thing) *Teleport {
	p := &Teleport{Thing: cfgThing}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

// OnBind stores the bound thing configuration for later teleport behavior.
func (e *Teleport) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles collision notifications for the teleport entity.
func (e *Teleport) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles impact notifications for the teleport entity.
func (e *Teleport) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
