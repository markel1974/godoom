package common

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Enemy represents an enemy character in the game with state, actions, and behaviors controlled during game logic.
type Enemy struct {
	active        bool
	isDead        bool
	firstTick     bool
	health        float64 // Add health
	throwCooldown float64
	throwMin      float64
	actions       map[string]int
	painCooldown  float64 // Timer to prevent pain animation spam

	// Cached actions for zero allocations during run loop
	actionRun    int
	actionWalk   int
	actionIdle   int
	actionStand  int
	actionPain   int
	actionDeath  int
	actionAttack int
	actionShoot  int
}

// NewEnemy initializes and returns a new Enemy instance with default attributes and actions mapping.
func NewEnemy(actions []string) *Enemy {
	const throwMin, throwMax = 5, 10
	e := &Enemy{
		actions:       make(map[string]int),
		active:        false,
		isDead:        false,
		health:        100.0, // Set default health
		throwMin:      float64(rand.Intn(throwMax-throwMin+1) + throwMin),
		throwCooldown: 0.0,
		painCooldown:  0.0,
	}
	for idx, action := range actions {
		lowerAction := strings.ToLower(strings.TrimSpace(action))
		e.actions[lowerAction] = idx
		// Map common Quake 3 suffixes to standard action names
		if strings.Contains(lowerAction, "run") {
			e.actions["run"] = idx
		}
		if strings.Contains(lowerAction, "walk") {
			e.actions["walk"] = idx
		}
		if strings.Contains(lowerAction, "death") || strings.Contains(lowerAction, "die") || strings.Contains(lowerAction, "dead") {
			if _, exists := e.actions["death"]; !exists {
				e.actions["death"] = idx
			}
		}
		if strings.Contains(lowerAction, "pain") {
			e.actions["pain"] = idx
		}
		if strings.Contains(lowerAction, "idle") || strings.Contains(lowerAction, "stand") {
			e.actions["idle"] = idx
		}
	}

	e.actionAttack, _ = e.findAction("attack")
	e.actionShoot, _ = e.findAction("shoot")

	e.actionRun, _ = e.findAction("run")
	e.actionWalk, _ = e.findAction("walk")
	e.actionIdle, _ = e.findAction("idle")
	e.actionStand, _ = e.findAction("stand")
	e.actionPain, _ = e.findAction("pain")
	if e.actionPain == 0 {
		e.actionPain, _ = e.findAction("pain1")
	}
	e.actionDeath, _ = e.findAction("death")

	if e.actionDeath == 0 {
		e.actionDeath, _ = e.findAction("die")
	}

	e.firstTick = true
	return e
}

// OnCollision handles the interaction logic when the Enemy object collides with another object.
func (e *Enemy) OnCollision(self config.IThingConfig, other config.IThingConfig) {
	//otherId := "UNKNOW"
	//if other != nil {
	//	otherId = other.GetId()
	//}
	//fmt.Println("Enemy.OnCollision:", self.GetId(), otherId)
}

// OnImpact handles the reaction of the enemy when impacted, including damage, knockback, and state changes like aggro or death.
func (e *Enemy) OnImpact(self config.IThingConfig, other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	if e.isDead {
		return // Dead enemies don't react to new impacts
	}

	// 1. Calculate and apply damage
	// Assuming 'force' correlates to damage. Adjust the multiplier as needed.
	damage := force * 0.5
	e.health -= damage

	// 2. Apply Knockback Physics
	// Push the enemy back along the impact direction vector
	knockbackMultiplier := 50.0 // Adjust based on your physics scale
	entity := self.GetEntity()
	entity.AddForce(dirX*force*knockbackMultiplier, dirY*force*knockbackMultiplier, dirZ*force*knockbackMultiplier)

	// Instant Aggro
	if !e.active {
		e.active = true
		e.throwCooldown = e.throwMin
	}

	// Handle Death or Pain
	if e.health <= 0 {
		e.handleDeath(self)
	} else {
		e.handlePain(self)
	}
}

// handleDeath marks the enemy as dead and initiates the death animation if defined. It may also adjust collision behavior.
func (e *Enemy) handleDeath(self config.IThingConfig) {
	e.isDead = true
	fmt.Println("ENEMY DEAD!!!!")

	// Set death animation
	if e.actionDeath != 0 {
		self.SetAction(e.actionDeath)
	}

	// Optional: Disable collisions or change collision group so player walks over the corpse
	// self.SetSolid(false)
}

// handlePain manages the enemy's reaction to taking damage by triggering a pain animation and setting a cooldown timer.
func (e *Enemy) handlePain(self config.IThingConfig) {
	// Don't restart the pain animation if we are already in pain
	if e.painCooldown > 0 {
		return
	}
	// Set pain animation
	if e.actionPain != 0 {
		self.SetAction(e.actionPain)
		e.painCooldown = 0.5 // Lock in pain state for 0.5 seconds
	}
}

// findAction searches for an action by name in the Enemy's action map. It supports prefixes like "legs_" and "both_".
// Returns the action's index and a boolean indicating if it was found.
func (e *Enemy) findAction(name string) (int, bool) {
	if idx, ok := e.actions[name]; ok {
		return idx, true
	}
	if idx, ok := e.actions["legs_"+name]; ok {
		return idx, true
	}
	if idx, ok := e.actions["both_"+name]; ok {
		return idx, true
	}
	return 0, false
}

// OnThinking handles the behavior and decision-making process of the enemy based on the player's position and game context.
func (e *Enemy) OnThinking(self config.IThingConfig, playerX, playerY, playerZ float64) {
	const throwPlayerDistance = 100.0

	if e.isDead {
		return // Stop thinking if dead
	}

	if e.firstTick {
		e.firstTick = false
		if e.actionIdle != 0 {
			self.SetAction(e.actionIdle)
		} else if e.actionStand != 0 {
			self.SetAction(e.actionStand)
		} else if e.actionRun != 0 {
			self.SetAction(e.actionRun)
		} else if e.actionWalk != 0 {
			self.SetAction(e.actionWalk)
		}
	}

	// Process pain timer
	if e.painCooldown > 0 {
		e.painCooldown -= 1.0 / float64(physics.GetFps())
		if e.painCooldown <= 0 {
			// Pain finished, return to running/chasing
			if e.actionRun != 0 {
				self.SetAction(e.actionRun)
			}
		}
		return
	}

	entity := self.GetEntity()
	// Il target Z deve essere circa a metà altezza del giocatore (es. petto) per mirare bene
	targetZ := playerZ + (entity.GetDepth() / 2)
	selfX, selfY, selfZ := entity.GetBottomCenter()
	dx := playerX - selfX
	dy := playerY - selfY
	dz := targetZ - selfZ
	// Attivazione (Aggro): Utilizza la distanza sferica 3D
	playerDist3d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if !e.active {
		if playerDist3d < self.GetWakeUpDistance() { // Raggio di risveglio
			e.active = true
			e.throwCooldown = e.throwMin
			if e.actionRun != 0 {
				self.SetAction(e.actionRun)
			}
		}
		return
	}
	// Aggiornamento timer armi
	if e.throwCooldown > 0 {
		e.throwCooldown -= 1.0 / float64(physics.GetFps())
		if e.throwCooldown <= 0 {
			if e.actionRun != 0 {
				self.SetAction(e.actionRun)
			}
		}
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
	targetImpulse := self.GetImpulse()
	// Distanza di stop (es. somma dei raggi per non compenetrare)
	stopDistance := (entity.GetWidth() * 0.5) + 5.0
	if playerDist2d < stopDistance {
		targetImpulse = 0.0
	} else if playerDist3d < throwPlayerDistance && e.throwCooldown <= 0 {
		targetImpulse = 0.0
	}
	thingZ := selfZ
	acceleration := self.GetAcceleration()
	speed := self.GetSpeed()
	self.MoveTowards(nx, ny, speed*targetImpulse, acceleration)

	if entity.IsOnGround() {
		e.handleJump(self, playerZ, thingZ, playerDist2d, nx, ny)
	}

	const throwableIndex = 2
	const throwableSpeed = 100
	if e.throwCooldown <= 0 && playerDist3d < throwPlayerDistance {
		fmt.Println("ATTACKING!!!!")
		self.SetAction(e.actionShoot)
		weaponForward := entity.GetWidth()
		spawnX := selfX + (math.Cos(angle) * weaponForward)
		spawnY := selfY + (math.Sin(angle) * weaponForward)
		spawnZ := selfZ + (entity.GetDepth() * 0.5)
		bulletPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
		aimPitch := math.Atan2(dz, playerDist2d)
		self.LaunchObject(throwableIndex, e.OnCollision, e.OnImpact, bulletPos, angle, aimPitch, throwableSpeed)
		// Resetta il timer
		e.throwCooldown = e.throwMin
	}
}

// handleJump checks and handles the jump behavior of the enemy based on ground status, player position, and movement constraints.
func (e *Enemy) handleJump(self config.IThingConfig, playerZ, thingZ, playerDist2d, nx, ny float64) {
	entity := self.GetEntity()
	if !entity.IsOnGround() {
		return
	}
	vx, vy, _ := entity.GetVelocity()
	speedSq := (vx * vx) + (vy * vy)
	// Euristica A: bloccato contro un muro/ostacolo
	isBlocked := speedSq < 0.1
	// Euristica B: il giocatore è più in alto
	floorDz := float64(int(playerZ) - int(thingZ))
	playerIsHigher := floorDz > self.GetMaxStep() && playerDist2d < 20.0
	if isBlocked || playerIsHigher {
		//Forza verticale: Moltiplichiamo per la massa affinché demoni di peso diverso saltino in modo coerente.
		leapForceXY := entity.GetMass() * 200.0
		self.Jump(nx*leapForceXY, ny*leapForceXY, 1.0)
	}
}
