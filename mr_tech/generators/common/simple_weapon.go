package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// Weapon represents a bindable weapon behavior for a configurable simulation entity.
type Weapon struct {
	self config.IThingConfig
}

// NewWeapon creates a Weapon and binds its event handlers to the provided thing configuration.
func NewWeapon(cfgThing *config.Thing) *Weapon {
	p := &Weapon{}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

// OnBind stores the bound thing configuration for later weapon behavior.
func (e *Weapon) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles a weapon collision with another thing using the given impact energy.
func (e *Weapon) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles impact notifications for the weapon with another entity and impact details.
func (e *Weapon) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
