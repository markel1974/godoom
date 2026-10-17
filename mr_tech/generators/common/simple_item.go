package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// Item represents an interactive object or entity in the system that can respond to collisions or other events.
type Item struct {
	self config.IThingConfig
}

// NewItem creates and initializes a new Enemy instance with default properties and returns a pointer to it.
func NewItem(cfgThing *config.Thing) *Item {
	p := &Item{}
	cfgThing.OnBind = p.OnBind
	cfgThing.OnCollision = p.OnCollision
	cfgThing.OnImpact = p.OnImpact
	return p
}

func (e *Item) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles the collision event between the current item and another object.
func (e *Item) OnCollision(other config.IThingConfig) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Item.OnCollision:", self.GetId(), otherId)
}

func (e *Item) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	//fmt.Println("Item IMPACT!!!!")
}
