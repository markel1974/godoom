package common

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Enemy represents an in-game enemy with activation, attack cooldown, and wake-up behavior based on distance.
type Enemy struct {
	active         bool
	throwCooldown  float64
	throwMin       float64
	wakeUpDistance float64
}

// NewEnemy creates and initializes a new Enemy instance with the specified wake-up distance.
func NewEnemy(wakeUpDistance float64) *Enemy {
	const throwMin, throwMax = 5, 10
	return &Enemy{
		active:         false,
		throwMin:       float64(rand.Intn(throwMax-throwMin+1) + throwMin),
		throwCooldown:  0.0,
		wakeUpDistance: wakeUpDistance,
	}
}

// OnCollision is triggered when the enemy collides with another object, handling interaction logic between entities.
func (e *Enemy) OnCollision(self config.IThingConfig, other config.IThingConfig) {
	fmt.Println("Enemy.OnCollision:", self.GetId(), other.GetId())
}

// OnThinking handles the logic for enemy behavior, including activation, movement, aiming, and attack decision-making.
func (e *Enemy) OnThinking(self config.IThingConfig, playerX, playerY, playerZ float64) {
	// Il target Z deve essere circa a metà altezza del giocatore (es. petto) per mirare bene
	targetZ := playerZ + (self.GetDepth() / 2)
	selfX, selfY, selfZ := self.GetPosition()
	dx := playerX - selfX
	dy := playerY - selfY
	dz := targetZ - selfZ
	// 1. Attivazione (Aggro): Utilizza la distanza sferica 3D
	playerDist3d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if !e.active {
		if playerDist3d < e.wakeUpDistance { // Raggio di risveglio
			e.active = true
			e.throwCooldown = e.throwMin
		}
		return
	}
	// Aggiornamento timer armi (assumendo dt fisso a 1/60)
	if e.throwCooldown > 0 {
		e.throwCooldown -= 1.0 / 60.0
	}
	// Inseguimento Terrestre
	playerDist2d := math.Sqrt(dx*dx + dy*dy)
	if playerDist2d <= 0.001 {
		return
	}
	// Aggiorniamo l'angolo del nemico affinché lo material o il modello si giri verso il bersaglio
	angle := math.Atan2(dy, dx)
	self.SetAngle(angle)
	invDist := 1.0 / playerDist2d
	nx := dx * invDist
	ny := dy * invDist
	impulse := 1.0
	// Distanza di stop (es. somma dei raggi per non compenetrare)
	stopDistance := self.GetRadius() + 5.0
	if playerDist2d < stopDistance {
		impulse = 0.0
	} else if playerDist3d < 20.0 && e.throwCooldown <= 0 {
		impulse = 0.0
	}
	thingZ := selfZ
	acceleration := self.GetAcceleration()
	speed := self.GetSpeed()
	self.MoveTowards(nx, ny, speed*impulse, acceleration)
	e.tryJump(self, playerZ, thingZ, playerDist2d, nx, ny)
	const throwableIndex = 2
	const throwableSpeed = 100
	if e.throwCooldown <= 0 && playerDist3d < 20.0 {
		weaponForward := self.GetWidth()
		spawnX := selfX + (math.Cos(angle) * weaponForward)
		spawnY := selfY + (math.Sin(angle) * weaponForward)
		spawnZ := selfZ + (self.GetDepth() * 0.5)
		bulletPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
		aimPitch := math.Atan2(dz, playerDist2d)
		self.LaunchObject(throwableIndex, e.OnCollision, bulletPos, angle, aimPitch, throwableSpeed)
		// Resetta il timer
		e.throwCooldown = e.throwMin
	}
}

// tryJump allows the enemy to perform a jump if blocked or the player is at a higher elevation.
func (e *Enemy) tryJump(self config.IThingConfig, playerZ, thingZ, playerDist2d, nx, ny float64) {
	if !self.IsOnGround() {
		return
	}
	vx, vy, _ := self.GetVelocity()
	speedSq := (vx * vx) + (vy * vy)
	// Euristica A: bloccato contro un muro/ostacolo
	isBlocked := speedSq < 0.1
	// Euristica B: il giocatore è più in alto
	floorDz := float64(int(playerZ) - int(thingZ))
	playerIsHigher := floorDz > self.GetMaxStep() && playerDist2d < 20.0
	if isBlocked || playerIsHigher {
		//1. Forza verticale: Moltiplichiamo per la massa affinché demoni di peso diverso
		//saltino in modo coerente. Regola l'800.0 in base alla tua gravità.
		mass := self.GetMass()
		jumpForceZ := mass * 800.0
		self.AddForce(0.0, 0.0, jumpForceZ)
		// Svincolo immediato dall'attrito
		self.SetOnGround(false)
		//Forza di slancio orizzontale (Leap) nella direzione del vettore nx, ny
		//Questo aiuta il demone a scavalcare l'ostacolo invece di saltare solo sul posto.
		leapForceXY := mass * 200.0
		self.AddForce(nx*leapForceXY, ny*leapForceXY, 0.0)
	}
}
