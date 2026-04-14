package model

import (
	"math"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingEnemy represents a physical or logical entity in the environment with attributes like position, mass, and associated data.
type ThingEnemy struct {
	*ThingBase
	active        bool
	throwCooldown float64
	throwMin      float64
}

// NewThingEnemy creates and initializes a new ThingEnemy instance.
func NewThingEnemy(things *Things, cfg *config.Thing, anim *textures.Animation, volume *Volume) *ThingEnemy {
	pos := cfg.Position
	//TODO REMOVE
	if cfg.Speed <= 0 {
		cfg.Speed = 6
	}
	if cfg.Acceleration <= 0 {
		cfg.Acceleration = 3
	}
	pos.Z = volume.GetMinZ()
	const throwMin, throwMax = 5, 10
	e := &ThingEnemy{
		ThingBase:     NewThingBase(things, cfg, pos, anim, volume),
		active:        false,
		throwMin:      float64(rand.Intn(throwMax-throwMin+1) + throwMin),
		throwCooldown: 0.0,
	}
	e.things.AddThing(e)
	return e
}

// Compute updates the Thing's direction, position, and attack logic based on the player's coordinates.
func (t *ThingEnemy) Compute(playerX float64, playerY float64, playerZ float64) {
	dx := playerX - t.pos.X
	dy := playerY - t.pos.Y
	// Il target Z deve essere circa a metà altezza del giocatore (es. petto) per mirare bene
	targetZ := playerZ + (t.height / 2)
	dz := targetZ - t.pos.Z
	// 1. Attivazione (Aggro): Utilizza la distanza sferica 3D
	dist3d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if !t.active {
		if dist3d < 25.0 { // Raggio di risveglio
			t.active = true
			t.throwCooldown = t.throwMin
		}
		return
	}
	// Aggiornamento timer armi (assumendo dt fisso a 1/60)
	if t.throwCooldown > 0 {
		t.throwCooldown -= 1.0 / 60.0
	}
	// Inseguimento Terrestre
	dist2d := math.Sqrt(dx*dx + dy*dy)
	if dist2d <= 0.001 {
		return
	}
	// Aggiorniamo l'angolo del nemico affinché lo sprite o il modello si giri verso il bersaglio
	t.angle = math.Atan2(dy, dx)
	invDist := 1.0 / dist2d
	nx := dx * invDist
	ny := dy * invDist
	impulse := 1.0
	// Distanza di stop (es. somma dei raggi per non compenetrare)
	stopDistance := t.radius + 5.0
	if dist2d < stopDistance {
		impulse = 0.0
	} else if dist3d < 20.0 && t.throwCooldown <= 0 {
		impulse = 0.0
	}
	t.MoveTowards(nx, ny, t.speed*impulse, t.acceleration)

	t.doJump(playerZ, dist2d, nx, ny)
	t.doThrow(dist3d, dist2d, dz)
}

// doJump applies a vertical and forward force to the entity if it is blocked or the target floor is higher than its current one.
func (t *ThingEnemy) doJump(thingZ, dist2d, nx, ny float64) {
	if !t.entity.IsOnGround() {
		return
	}

	vx, vy, _ := t.entity.GetVelocity()
	speedSq := (vx * vx) + (vy * vy)

	// Euristica A: bloccato contro un muro/ostacolo
	isBlocked := speedSq < 0.1
	// Euristica B: il giocatore è più in alto
	floorDz := thingZ - t.pos.Z
	playerIsHigher := floorDz > t.maxStep && dist2d < 20.0

	if isBlocked || playerIsHigher {
		//1. Forza verticale: Moltiplichiamo per la massa affinché demoni di peso diverso
		//saltino in modo coerente. Regola l'800.0 in base alla tua gravità.
		jumpForceZ := t.mass * 800.0
		t.entity.AddForce(0.0, 0.0, jumpForceZ)
		// Svincolo immediato dall'attrito
		t.entity.SetOnGround(false)
		//Forza di slancio orizzontale (Leap) nella direzione del vettore nx, ny
		//Questo aiuta il demone a scavalcare l'ostacolo invece di saltare solo sul posto.
		leapForceXY := t.mass * 200.0
		t.entity.AddForce(nx*leapForceXY, ny*leapForceXY, 0.0)
	}
}

// doFire enables the entity to fire a projectile if within range and off cooldown, calculating its spawn position and trajectory.
func (t *ThingEnemy) doThrow(dist3d, dist2d, dz float64) {
	if t.throwCooldown <= 0 && dist3d < 20.0 {
		weaponForward := t.radius * 2.0
		spawnX := t.pos.X + (math.Cos(t.angle) * weaponForward)
		spawnY := t.pos.Y + (math.Sin(t.angle) * weaponForward)
		spawnZ := t.pos.Z + (t.height * 0.5)
		bulletPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
		aimPitch := math.Atan2(dz, dist2d)
		t.LaunchObject(bulletPos, t.angle, aimPitch)
		// Resetta il timer
		t.throwCooldown = t.throwMin
	}
}
