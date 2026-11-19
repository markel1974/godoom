package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Item represents a bindable game object that handles collision and impact callbacks.
type Item struct {
	*config.Thing
	self config.IThingConfig
}

// NewItem creates an Item and registers its bind, collision, and impact handlers.
func NewItem(classname string, pos geometry.XYZ, angle float64, mass, speed float64) *Item {
	return NewItemFromConfig(config.NewConfigItem(classname, pos, angle, mass, speed))
}

// NewItemFromConfig wraps an existing item Thing configuration.
func NewItemFromConfig(cfgThing *config.Thing) *Item {
	p := &Item{Thing: cfgThing}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

// OnBind stores the bound thing configuration for later item behavior callbacks.
func (e *Item) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles a collision with another thing and receives the impact energy.
func (e *Item) OnCollision(other config.IThingConfig, impactEnergy float64) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles impact notifications for the item.
func (e *Item) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
