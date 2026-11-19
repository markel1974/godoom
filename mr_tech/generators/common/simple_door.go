package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// Door represents a configurable door entity that can bind to thing behavior and respond to collisions or impacts.
type Door struct {
	self config.IThingConfig
}

// NewDoor creates a Door and registers its bind, collision, and impact handlers on cfgThing.
func NewDoor(cfgThing *config.Thing) *Door {
	p := &Door{}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

// OnBind stores the bound thing configuration for later door behavior callbacks.
func (e *Door) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles a collision with another thing and receives the collision impact energy.
func (e *Door) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles impact notifications for the door from another thing.
func (e *Door) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
