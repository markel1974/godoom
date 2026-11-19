package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Key represents behavior for a collectible key entity bound to a configurable thing.
type Key struct {
	*config.Thing
	self config.IThingConfig
}

// NewKey creates a Key and registers its bind, collision, and impact callbacks on cfgThing.
func NewKey(classname string, pos geometry.XYZ, angle float64, mass float64, keyId string) *Key {
	p := &Key{
		Thing: config.NewConfigKey(classname, pos, angle, mass, keyId),
	}
	p.Thing.OnBind = p.OnBind
	p.Thing.OnCollision = p.OnCollision
	p.Thing.OnImpact = p.OnImpact
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
