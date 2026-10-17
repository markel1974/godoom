package common

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Player represents an entity in the system capable of interacting with other entities and responding to collisions.
type Player struct {
	self config.IThingConfig
}

// NewPlayer creates and returns a pointer to a new instance of Player.
func NewPlayer(config *config.Player) *Player {
	player := &Player{}
	config.OnBind = player.OnBind
	config.OnCollision = player.OnCollision
	config.OnImpact = player.OnImpact
	return player
}

// OnBind associates the Player instance with a given configuration, enabling interaction logic and behavior assignment.
func (e *Player) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision handles collision events between the player and another object, identified by their configurations.
func (e *Player) OnCollision(other config.IThingConfig) {
	//otherId := "UNKNOWN"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Player.OnCollision:", self.GetId(), otherId)
}

func (e *Player) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	fmt.Println("Player IMPACT!!!!")
}
