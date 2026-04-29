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
	active         bool
	throwCooldown  float64
	throwMin       float64
	wakeUpDistance float64
}

// NewThingEnemy creates and initializes a new ThingEnemy instance.
func NewThingEnemy(things *Things, cfg *config.Thing, anim *textures.Material, volume *Volume) *ThingEnemy {
	pos := cfg.Position
	if cfg.Speed <= 0 {
		cfg.Speed = 6
	}
	if cfg.Acceleration <= 0 {
		cfg.Acceleration = 3
	}
	const throwMin, throwMax = 5, 10
	e := &ThingEnemy{
		ThingBase:      NewThingBase(things, cfg, pos, anim, volume),
		wakeUpDistance: cfg.WakeUpDistance,
		active:         false,
		throwMin:       float64(rand.Intn(throwMax-throwMin+1) + throwMin),
		throwCooldown:  0.0,
	}
	return e
}

func (t *ThingEnemy) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

func (t *ThingEnemy) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-t.inbox:
				switch evt.GetKind() {
				case StageThinking:
					t.StageThinking(evt.GetCoords())
				case StageCompute:
					t.StageCompute()
				case StageResolve:
					t.StageResolve(evt.GetSolverJitter())
				case StageApply:
					t.StageApply()
				}
				evt.Done()
			case <-t.done:
				return
			}
		}
	}()
}

// StageThinking updates the Thing's direction, position, and attack logic based on the player's coordinates.
func (t *ThingEnemy) StageThinking(playerX float64, playerY float64, playerZ float64) {
	// Il target Z deve essere circa a metà altezza del giocatore (es. petto) per mirare bene
	targetZ := playerZ + (t.entity.GetDepth() / 2)
	dx := playerX - t.pos.X
	dy := playerY - t.pos.Y
	dz := targetZ - t.pos.Z
	// 1. Attivazione (Aggro): Utilizza la distanza sferica 3D
	playerDist3d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if !t.active {
		if playerDist3d < t.wakeUpDistance { // Raggio di risveglio
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
	playerDist2d := math.Sqrt(dx*dx + dy*dy)
	if playerDist2d <= 0.001 {
		return
	}
	// Aggiorniamo l'angolo del nemico affinché lo sprite o il modello si giri verso il bersaglio
	t.SetAngle(math.Atan2(dy, dx))
	invDist := 1.0 / playerDist2d
	nx := dx * invDist
	ny := dy * invDist
	impulse := 1.0
	// Distanza di stop (es. somma dei raggi per non compenetrare)
	stopDistance := t.GetRadius() + 5.0
	if playerDist2d < stopDistance {
		impulse = 0.0
	} else if playerDist3d < 20.0 && t.throwCooldown <= 0 {
		impulse = 0.0
	}
	thingZ := t.pos.Z
	t.MoveTowards(nx, ny, t.speed*impulse, t.acceleration)
	t.doJump(playerZ, thingZ, playerDist2d, nx, ny)
	t.doThrow(playerDist3d, playerDist2d, dz)
}

// doJump applies a vertical and forward force to the entity if it is blocked or the target floor is higher than its current one.
func (t *ThingEnemy) doJump(playerZ, thingZ, playerDist2d, nx, ny float64) {
	if !t.entity.IsOnGround() {
		return
	}
	vx, vy, _ := t.entity.GetVelocity()
	speedSq := (vx * vx) + (vy * vy)
	// Euristica A: bloccato contro un muro/ostacolo
	isBlocked := speedSq < 0.1
	// Euristica B: il giocatore è più in alto
	floorDz := float64(int(playerZ) - int(thingZ))
	playerIsHigher := floorDz > t.maxStep && playerDist2d < 20.0
	if isBlocked || playerIsHigher {
		//1. Forza verticale: Moltiplichiamo per la massa affinché demoni di peso diverso
		//saltino in modo coerente. Regola l'800.0 in base alla tua gravità.
		mass := t.entity.GetMass()
		jumpForceZ := mass * 800.0
		t.entity.AddForce(0.0, 0.0, jumpForceZ)
		// Svincolo immediato dall'attrito
		t.entity.SetOnGround(false)
		//Forza di slancio orizzontale (Leap) nella direzione del vettore nx, ny
		//Questo aiuta il demone a scavalcare l'ostacolo invece di saltare solo sul posto.
		leapForceXY := mass * 200.0
		t.entity.AddForce(nx*leapForceXY, ny*leapForceXY, 0.0)
	}
}

// doFire enables the entity to fire a projectile if within range and off cooldown, calculating its spawn position and trajectory.
func (t *ThingEnemy) doThrow(dist3d, dist2d, dz float64) {
	if t.throwCooldown <= 0 && dist3d < 20.0 {
		weaponForward := t.entity.GetWidth()
		spawnX := t.pos.X + (math.Cos(t.angle) * weaponForward)
		spawnY := t.pos.Y + (math.Sin(t.angle) * weaponForward)
		spawnZ := t.pos.Z + (t.entity.GetDepth() * 0.5)
		bulletPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
		aimPitch := math.Atan2(dz, dist2d)
		t.LaunchObject(bulletPos, t.angle, aimPitch)
		// Resetta il timer
		t.throwCooldown = t.throwMin
	}
}
