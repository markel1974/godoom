package script

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Player represents an in-game entity with position and orientation in 2D space.
type Player struct {
	posX  float64
	posY  float64
	angle float64
}

// NewPlayer initializes and returns a new Player instance with the specified position and angle.
func NewPlayer(posX, posY, angle float64) *Player {
	return &Player{
		posX:  posX,
		posY:  posY,
		angle: angle,
	}
}

// Build creates and configures a new Player instance with predefined physical, visual, and motion attributes.
func (p *Player) Build() *config.Player {
	playerPos := geometry.XYZ{X: p.posX, Y: p.posY, Z: 0}
	player := config.NewConfigPlayer(playerPos, p.angle, PlayerMass, PlayerSpeed, PlayerRadius, PlayerHeight)
	_ = common.NewPlayer(player)
	player.GForce = GForce
	player.JumpForce = JumpForce
	player.Impulse = 0.05 //mod

	player.Flash.ZFar = 8192
	player.Flash.Intensity = 0.02
	player.Flash.Falloff = 2000
	player.Flash.OffsetX = 0.2
	player.Flash.OffsetY = 0.1
	player.Bobbing.SwayScale = 1.0 //mod
	player.Bobbing.SwayOffsetX = 50
	player.Bobbing.SwayOffsetY = -0.9
	player.Bobbing.MaxAmplitudeX = 0.5
	player.Bobbing.MaxAmplitudeY = 0.6
	player.Bobbing.StrideLength = 0.07 //mod //0.0015 // FREQUENCY: 1000 * 0.0007 = 0.7 rad/frame.
	//breathing
	player.Bobbing.IdleAmpX = 0.2
	player.Bobbing.IdleAmpY = 0.2
	player.Bobbing.IdleDrift = 1.0

	player.Bobbing.SpeedLerp = 0.30 // Instant reactivity to speed
	player.Bobbing.AmpLerp = 0.20
	player.Bobbing.ImpactMax = 1000.0
	player.Bobbing.ImpactScale = 0.02   // LANDING: 1000 * 0.02 = 20 units of vertical shake
	player.Bobbing.SpringTension = 0.20 // Stiffer spring (faster return)
	player.Bobbing.SpringDamping = 0.80
	player.Bobbing.TiltAmp = 0.05

	return player
}
