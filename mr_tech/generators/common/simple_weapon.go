package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// Weapon represents an interactive object or entity in the system that can respond to collisions or other events.
type Weapon struct {
	self config.IThingConfig
}

// NewWeapon creates and initializes a new Weapon instance with default properties and returns a pointer to it.
func NewWeapon(cfgThing *config.Thing) *Weapon {
	p := &Weapon{}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

func (e *Weapon) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles the collision event between the current item and another object.
func (e *Weapon) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

func (e *Weapon) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
