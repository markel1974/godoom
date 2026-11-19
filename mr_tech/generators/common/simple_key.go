package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// Key represents behavior for a collectible key entity bound to a configurable thing.
type Key struct {
	self config.IThingConfig
}

// NewKey creates a Key and registers its bind, collision, and impact callbacks on cfgThing.
func NewKey(cfgThing *config.Thing) *Key {
	p := &Key{}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

// OnBind stores the bound thing configuration for later key behavior.
func (e *Key) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles collision notifications for the key with another thing and the collision energy.
func (e *Key) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles impact events involving the key, including force, distance, and direction details.
func (e *Key) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
