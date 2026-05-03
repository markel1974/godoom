package common

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// Player represents an entity in the system capable of interacting with other entities and responding to collisions.
type Player struct {
}

// NewPlayer creates and returns a pointer to a new instance of Player.
func NewPlayer() *Player {
	return &Player{}
}

// OnCollision handles collision events between the player and another object, identified by their configurations.
func (e *Player) OnCollision(self config.IThingConfig, other config.IThingConfig) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Player.OnCollision:", self.GetId(), otherId)
}
