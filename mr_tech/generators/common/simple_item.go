package common

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Item represents an interactive object or entity in the system that can respond to collisions or other events.
type Item struct {
}

// NewItem creates and initializes a new Enemy instance with default properties and returns a pointer to it.
func NewItem() *Item {
	return &Item{}
}

// OnCollision handles the collision event between the current item and another object.
func (e *Item) OnCollision(self config.IThingConfig, other config.IThingConfig) {
	fmt.Println("Item.OnCollision:", self.GetId(), other.GetId())
}
