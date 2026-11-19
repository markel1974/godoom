package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Platform provides behavior callbacks for a platform entity in the simulation.
type Platform struct {
	*config.Thing
	self config.IThingConfig
}

// NewPlatform creates a Platform and registers its bind, collision, and impact handlers.
func NewPlatform(classname string, pos geometry.XYZ, angle float64, mass, speed float64) *Platform {
	return NewPlatformFromConfig(config.NewConfigPlatform(classname, pos, angle, mass, speed))
}

// NewPlatformFromConfig wraps an existing platform Thing configuration.
func NewPlatformFromConfig(cfgThing *config.Thing) *Platform {
	p := &Platform{Thing: cfgThing}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

// OnBind stores the bound thing configuration for later platform callbacks.
func (e *Platform) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles a collision with another thing, receiving the other entity and impact energy.
func (e *Platform) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles an impact event for the platform with force, distance, and direction details.
func (e *Platform) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
