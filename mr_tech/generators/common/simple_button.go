package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Button represents an interactive entity that can bind to its configuration and react to collisions or impacts.
type Button struct {
	*config.Thing
	self config.IThingConfig
}

// NewButton creates a Button and registers its bind, collision, and impact handlers.
func NewButton(classname string, pos geometry.XYZ, angle float64, mass, speed float64) *Button {
	return NewButtonFromConfig(config.NewConfigButton(classname, pos, angle, mass, speed))
}

// NewButtonFromConfig wraps an existing button Thing configuration.
func NewButtonFromConfig(cfgThing *config.Thing) *Button {
	p := &Button{Thing: cfgThing}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

// OnBind stores the bound thing configuration for later callbacks.
func (e *Button) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles collision notifications for the button.
func (e *Button) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles an impact event against another thing, with force, closest distance, and direction data.
func (e *Button) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
