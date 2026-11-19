package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// Teleport handles binding and event callbacks for a teleport entity.
type Teleport struct {
	self config.IThingConfig
}

// NewTeleport creates a Teleport and assigns its bind, collision, and impact handlers to cfgThing.
func NewTeleport(cfgThing *config.Thing) *Teleport {
	p := &Teleport{}
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
