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

// EnemyAnimations represents a collection of animation sequences categorized by enemy behaviors or states.

// normalizeAction converts the input action name to lowercase and trims surrounding whitespace.
func normalizeAction(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// baseAction normalizes an action name by removing specific prefixes iteratively until no matching prefix is found.
func baseAction(name string) string {
	for {
		switch {
		case strings.HasPrefix(name, "legs_"):
			name = strings.TrimPrefix(name, "legs_")
		case strings.HasPrefix(name, "both_"):
			name = strings.TrimPrefix(name, "both_")
		case strings.HasPrefix(name, "torso_"):
			name = strings.TrimPrefix(name, "torso_")
		default:
			return name
		}
	}
}

// Enemy represents an entity with various states, animations, health, and timers for interaction and combat behavior.
type Enemy struct {
	state         config.EnemyState
	health        float64
	throwCooldown float64
	throwMin      float64
	painTimer     float64
	shootTimer    float64
	actions       map[string]int
	animations    *config.EnemyAnimations
	lastAnimation map[config.EnemyState]int
}

func NewEnemy(actions *config.EnemyAnimations) *Enemy {
	const (
		throwMin = 2
		throwMax = 6
	)
	e := &Enemy{
		state:         config.EnemyStateUnknown,
		health:        100,
		throwMin:      float64(rand.Intn(throwMax-throwMin+1) + throwMin),
		actions:       make(map[string]int, len(actions.Content)),
		lastAnimation: make(map[config.EnemyState]int),
		animations:    actions,
	}
	return e
}

// chooseAnimation selects an animation index from a list of options based on the enemy's current state, ensuring variation.
// It avoids repeating the last animation for the given state, if possible, and stores the selected animation.
func (e *Enemy) chooseAnimation(state config.EnemyState, animations *config.EnemyAnimation) (int, bool) {
	if len(animations.Indices) == 0 {
		return 0, false
	}

	if len(animations.Indices) == 1 {
		e.lastAnimation[state] = animations.Indices[0]
		return animations.Indices[0], true
	}

	var index int

	if last, hasLast := e.lastAnimation[state]; hasLast {
		candidates := make([]int, 0, len(animations.Indices))
		for _, idx := range animations.Indices {
			if idx != last {
				candidates = append(candidates, idx)
			}
		}
		if len(candidates) == 0 {
			return 0, false
		}
		index = candidates[rand.Intn(len(candidates))]
	} else {
		index = animations.Indices[rand.Intn(len(animations.Indices))]
	}

	e.lastAnimation[state] = index

	return index, true
}

// playAnimation selects an appropriate animation for the enemy and sets it as the current action in the given config.
func (e *Enemy) playAnimation(self config.IThingConfig, category config.EnemyState, animations *config.EnemyAnimation) bool {
	action, ok := e.chooseAnimation(category, animations)
	if !ok {
		return false
	}

	self.SetAction(action)
	return true
}

// setState updates the enemy's current state and triggers the corresponding animation unless the state is unchanged or dead.
func (e *Enemy) setState(self config.IThingConfig, state config.EnemyState) {
	if e.state == state || e.state == config.EnemyStateDead {
		return
	}
	if len(e.animations.Content[state].Indices) > 0 {
		e.state = state
		e.playAnimation(self, state, e.animations.Content[state])
	} else {
		fmt.Println("No animations for state:", state)
	}
}

// OnCollision defines the optional behavior when this Enemy collides with another object.
func (e *Enemy) OnCollision(self config.IThingConfig, other config.IThingConfig) {
	// Comportamento di collisione opzionale.
}

// OnImpact handles the behavior of the enemy when impacted by another object, adjusting health and state accordingly.
func (e *Enemy) OnImpact(self config.IThingConfig, other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	if e.state == config.EnemyStateDead {
		return
	}
	e.health -= force * 0.5
	if e.health <= 0 {
		e.handleDeath(self)
		return
	}
	if e.state == config.EnemyStateIdle ||
		e.state == config.EnemyStateUnknown {
		e.throwCooldown = e.throwMin
	}
	e.handlePain(self)
}

// OnThinking handles the behavior logic for the Enemy, updating its state and actions based on its environment and player position.
func (e *Enemy) OnThinking(self config.IThingConfig, pCenterX, pCenterY, pCenterZ float64) {
	const (
		throwPlayerDistance = 300.0
		meleeDuration       = 0.7
		meleeCooldown       = 1.0
	)

	if e.state == config.EnemyStateDead {
		return
	}

	fps := physics.GetFps()
	if fps <= 0 {
		return
	}

	dt := 1.0 / float64(fps)

	if e.throwCooldown > 0 {
		e.throwCooldown -= dt
		if e.throwCooldown < 0 {
			e.throwCooldown = 0
		}
	}

	if e.state == config.EnemyStateUnknown {
		e.setState(self, config.EnemyStateIdle)
	}
	if e.state == config.EnemyStatePain {
		e.painTimer -= dt

		if e.painTimer <= 0 {
			e.painTimer = 0
			e.setState(self, config.EnemyStateChase)
		}

		return
	}

	if e.state == config.EnemyStateShoot || e.state == config.EnemyStateMelee {
		e.shootTimer -= dt
		if e.shootTimer <= 0 {
			e.shootTimer = 0
			e.setState(self, config.EnemyStateChase)
		}
		return
	}

	entity := self.GetEntity()
	targetZ := pCenterZ + entity.GetDepth()/2
	selfX, selfY, selfZ := entity.GetCenter()

	dx := pCenterX - selfX
	dy := pCenterY - selfY
	dz := targetZ - selfZ

	dist2dSq := dx*dx + dy*dy
	dist3dSq := dist2dSq + dz*dz

	if e.state == config.EnemyStateIdle {
		wakeDistance := self.GetWakeUpDistance()
		if dist3dSq < wakeDistance*wakeDistance {
			e.throwCooldown = e.throwMin
			e.setState(self, config.EnemyStateChase)
		}
		return
	}

	if dist2dSq <= 0.000001 {
		return
	}

	playerDist2d := math.Sqrt(dist2dSq)
	playerDist3d := math.Sqrt(dist3dSq)

	angle := math.Atan2(dy, dx)
	self.SetAngle(angle)

	nx := dx / playerDist2d
	ny := dy / playerDist2d

	stopDistance := entity.GetWidth()*0.5 + 5.0

	// Attacco corpo a corpo se il modello dispone dell'animazione.
	if playerDist2d <= stopDistance && e.throwCooldown <= 0 && len(e.animations.Content[config.EnemyStateMelee].Indices) > 0 {
		e.setState(self, config.EnemyStateMelee)
		e.shootTimer = meleeDuration
		e.throwCooldown = meleeCooldown
		return
	}

	// Movimento conservativo: nessuna modifica dinamica all'impulso.
	targetImpulse := self.GetImpulse()
	if playerDist2d < stopDistance {
		targetImpulse = 0
	}

	self.MoveTowards(nx, ny, self.GetSpeed()*targetImpulse, self.GetAcceleration())

	if entity.IsOnGround() {
		e.handleJump(self, pCenterZ, selfZ, playerDist2d, nx, ny)
	}

	if e.throwCooldown <= 0 && playerDist3d < throwPlayerDistance {
		e.handleShoot(self, selfX, selfY, selfZ, dz, angle, playerDist2d)
	}
}

// handleLaunch handles the logic for launching a throwable object, including positioning and aiming calculations.
func (e *Enemy) handleLaunch(self config.IThingConfig, selfX, selfY, selfZ float64, dz, angle, playerDist2d float64) {
	const (
		throwableIndex = 2
		throwableSpeed = 100.0
		shootDuration  = 0.5
	)

	if e.state == config.EnemyStateDead {
		return
	}

	e.setState(self, config.EnemyStateShoot)
	e.shootTimer = shootDuration
	e.throwCooldown = e.throwMin

	entity := self.GetEntity()

	weaponForward := entity.GetWidth()
	spawnX := selfX + math.Cos(angle)*weaponForward
	spawnY := selfY + math.Sin(angle)*weaponForward
	spawnZ := selfZ + entity.GetDepth()*0.5

	bulletPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}

	aimPitch := math.Atan2(dz, playerDist2d)

	self.LaunchObject(throwableIndex, e.OnCollision, e.OnImpact, bulletPos, angle, aimPitch, throwableSpeed)
}

// handleShoot sets the enemy to a shooting state, computes projectile trajectory, and performs a hitscan-based weapon attack.
func (e *Enemy) handleShoot(self config.IThingConfig, selfX, selfY, selfZ float64, dz, angle, playerDist2d float64) {
	const shootDuration = 0.5
	if e.state == config.EnemyStateDead {
		return
	}

	e.setState(self, config.EnemyStateShoot)
	e.shootTimer = shootDuration
	e.throwCooldown = e.throwMin
	entity := self.GetEntity()

	weaponForward := entity.GetWidth()
	spawnX := selfX + math.Cos(angle)*weaponForward
	spawnY := selfY + math.Sin(angle)*weaponForward
	spawnZ := selfZ + entity.GetDepth()*0.5

	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	aimPitch := math.Atan2(dz, playerDist2d)

	dirX := math.Cos(angle) * math.Cos(aimPitch)
	dirY := math.Sin(angle) * math.Cos(aimPitch)
	dirZ := math.Sin(aimPitch)
	const weaponForce = 50000.0 * 50.0 // TODO: CONFIG
	self.FireHitscan("gun", spawnPos, weaponForce, dirX, dirY, dirZ)
}

// handleDeath transitions the Enemy to a dead state, resets timers, and ensures no action is taken if already dead.
func (e *Enemy) handleDeath(self config.IThingConfig) {
	if e.state == config.EnemyStateDead {
		return
	}
	e.painTimer = 0
	e.shootTimer = 0
	e.setState(self, config.EnemyStateDead)
}

// handlePain handles the enemy's reaction to pain by resetting the pain timer and transitioning to the pain state if not dead.
func (e *Enemy) handlePain(self config.IThingConfig) {
	if e.state == config.EnemyStateDead {
		return
	}
	e.painTimer = 0.5
	e.setState(self, config.EnemyStatePain)
}

// handleJump checks and handles the jump behavior of the enemy based on ground status, player position, and movement constraints.
func (e *Enemy) handleJump(self config.IThingConfig, playerZ, thingZ, playerDist2d, nx, ny float64) {
	entity := self.GetEntity()
	if !entity.IsOnGround() {
		return
	}
	// Euristica A: bloccato contro un muro/ostacolo
	//isBlocked := speedSq < 0.1
	floorDz := float64(int(playerZ) - int(thingZ))
	playerIsHigher := floorDz > self.GetMaxStep()
	if playerIsHigher { //isBlocked || playerIsHigher {
		//e.setState(self, EnemyStateJump)
		//Forza verticale: Moltiplichiamo per la massa affinché demoni di peso diverso saltino in modo coerente.
		//leapForceXY := entity.GetMass() * 8.0
		//self.Jump(nx*leapForceXY, ny*leapForceXY, 1.0)
		self.Jump(0, 0, 8)
	}
}
