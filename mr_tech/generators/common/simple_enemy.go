package common

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// framesToSeconds converts an animation frame count to seconds using the
// current animation/update FPS. Missing durations remain zero; no duration is guessed.
func framesToSeconds(frames int) float64 {
	fps := physics.GetFps()
	if frames <= 0 || fps <= 0 {
		return 0
	}
	t := float64(frames) / float64(fps)
	t *= 5
	return t
}

// Enemy represents an entity with various states, animations, health, and timers for interaction and combat behavior.
type Enemy struct {
	state                   config.EnemyState
	health                  float64
	shootCooldown           float64
	shotIntervalMin         float64
	painTimer               float64
	shootTimer              float64
	actions                 map[string]int
	animations              *config.EnemyAnimations
	lastAnimation           map[config.EnemyState]int
	self                    config.IThingConfig
	animationDurationFrames int
}

// NewEnemy initializes and returns a new Enemy instance with default state, health, and animations.
func NewEnemy(thingCfg *config.Thing, actions *config.EnemyAnimations) *Enemy {
	const (
		shotIntervalMin = 2
		shotIntervalMax = 4
	)
	e := &Enemy{
		state:           config.EnemyStateUnknown,
		health:          100,
		shotIntervalMin: float64(rand.Intn(shotIntervalMax-shotIntervalMin+1) + shotIntervalMin),
		actions:         make(map[string]int, len(actions.Content)),
		lastAnimation:   make(map[config.EnemyState]int),
		animations:      actions,
	}
	thingCfg.OnBind = e.OnBind
	thingCfg.OnThinking = e.OnThinking
	thingCfg.OnCollision = e.OnCollision
	thingCfg.OnImpact = e.OnImpact
	return e
}

func (e *Enemy) OnBind(self config.IThingConfig) {
	e.self = self
}

/*
if self.GetId() == "monster_soldier" {
fmt.Println("handleLaunch called for monster_soldier")
}

*/

// OnCollision defines the optional behavior when this Enemy collides with another object.
func (e *Enemy) OnCollision(other config.IThingConfig) {
	// Comportamento di collisione opzionale.
}

// OnImpact handles the behavior of the enemy when impacted by another object, adjusting health and state accordingly.
func (e *Enemy) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	if e.state == config.EnemyStateDead {
		return
	}
	e.health -= force * 0.5
	if e.health <= 0 {
		e.handleDeath()
		return
	}
	if e.state == config.EnemyStateIdle ||
		e.state == config.EnemyStateUnknown {
		e.shootCooldown = e.shotIntervalMin
	}
	e.handlePain()
}

// OnThinking handles the behavior logic for the Enemy, updating its state and actions based on its environment and player position.
func (e *Enemy) OnThinking(pCenterX, pCenterY, pCenterZ float64) {
	const (
		shootPlayerDistance = 300.0
		shootCooldown       = 0.5
	)
	if e.state == config.EnemyStateDead {
		return
	}
	fps := physics.GetFps()
	if fps <= 0 {
		return
	}
	dt := 1.0 / float64(fps)

	entity := e.self.GetEntity()
	targetZ := pCenterZ + entity.GetDepth()/2
	selfX, selfY, selfZ := entity.GetCenter()

	dx := pCenterX - selfX
	dy := pCenterY - selfY
	dz := targetZ - selfZ

	dist2dSq := dx*dx + dy*dy
	dist3dSq := dist2dSq + dz*dz

	if e.shootCooldown > 0 {
		e.shootCooldown -= dt
		if e.shootCooldown < 0 {
			e.shootCooldown = 0
		}
	}

	switch e.state {
	case config.EnemyStateUnknown:
		e.setState(config.EnemyStateIdle)
		return
	case config.EnemyStateIdle:
		wakeDistance := e.self.GetWakeUpDistance()
		if dist3dSq < wakeDistance*wakeDistance {
			e.shootCooldown = e.shotIntervalMin
			e.setState(config.EnemyStateChase)
		}
		return
	case config.EnemyStateChase:
		wakeDistance := e.self.GetWakeUpDistance()
		if dist3dSq > wakeDistance*wakeDistance {
			e.shootCooldown = e.shotIntervalMin
			e.setState(config.EnemyStateIdle)
			return
		}
	case config.EnemyStateShoot: //, config.EnemyStateMelee:
		if e.shootTimer -= dt; e.shootTimer <= 0 {
			e.shootTimer = 0
			e.setState(config.EnemyStateChase)
		}
		return
	case config.EnemyStatePain:
		if e.painTimer -= dt; e.painTimer <= 0 {
			e.painTimer = 0
			e.setState(config.EnemyStateChase)
		}
		return
	default:
	}

	if dist2dSq <= 0.000001 {
		return
	}
	playerDist2d := math.Sqrt(dist2dSq)
	playerDist3d := math.Sqrt(dist3dSq)
	angle := math.Atan2(dy, dx)
	e.self.SetAngle(angle)
	nx := dx / playerDist2d
	ny := dy / playerDist2d
	stopDistance := entity.GetWidth()*0.5 + 5.0

	//TODO
	// Attacco corpo a corpo se il modello dispone dell'animazione.
	//if playerDist2d <= stopDistance && e.shootCooldown <= 0 && len(e.animations.Content[config.EnemyStateMelee].Indices) > 0 {
	//	durationFrames := e.setState(config.EnemyStateMelee)
	//	e.shootTimer = framesToSeconds(durationFrames)
	//	e.shootCooldown = shootCooldown
	//	return
	//}

	// Movimento conservativo: nessuna modifica dinamica all'impulso.
	targetImpulse := e.self.GetImpulse()
	if playerDist2d < stopDistance {
		targetImpulse = 0
	}

	e.self.MoveTowards(nx, ny, e.self.GetSpeed()*targetImpulse, e.self.GetAcceleration())

	if entity.IsOnGround() {
		e.handleJump(e.self, pCenterZ, selfZ, playerDist2d, nx, ny)
	}

	if e.shootCooldown <= 0 && playerDist3d < shootPlayerDistance {
		e.handleShoot(e.self, selfX, selfY, selfZ, dz, angle, playerDist2d)
	}
}

// handleLaunch handles the logic for launching a throwable object, including positioning and aiming calculations.
func (e *Enemy) handleLaunch(self config.IThingConfig, selfX, selfY, selfZ float64, dz, angle, playerDist2d float64) {
	const (
		throwableIndex = 2
		throwableSpeed = 100.0
	)

	if e.state == config.EnemyStateDead {
		return
	}

	durationFrames := e.setState(config.EnemyStateShoot)
	e.shootTimer = framesToSeconds(durationFrames)
	e.shootCooldown = e.shotIntervalMin

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
func (e *Enemy) handleShoot(self config.IThingConfig, selfX, selfY, selfZ, dz, angle, playerDist2d float64) {
	durationFrames := e.setState(config.EnemyStateShoot)
	e.shootTimer = framesToSeconds(durationFrames)
	e.shootCooldown = e.shotIntervalMin

	entity := self.GetEntity()
	weaponForward := entity.GetWidth()
	spawnX := selfX + math.Cos(angle)*weaponForward
	spawnY := selfY + math.Sin(angle)*weaponForward
	spawnZ := selfZ + entity.GetDepth()*0.5
	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	// dz deve essere: quota bersaglio - quota spawn
	aimPitch := math.Atan2(dz, playerDist2d)
	// Stessa convenzione del player: pitch positivo = verso il basso
	pitchRad := -aimPitch
	dirX := math.Cos(angle) * math.Cos(pitchRad)
	dirY := math.Sin(angle) * math.Cos(pitchRad)
	dirZ := math.Sin(pitchRad)

	const weaponForce = 50000.0 * 500.0 // TODO: CONFIG
	self.FireHitscan("gun", spawnPos, weaponForce, dirX, dirY, dirZ)
}

// handleDeath transitions the Enemy to a dead state, resets timers, and ensures no action is taken if already dead.
func (e *Enemy) handleDeath() {
	if e.state == config.EnemyStateDead {
		return
	}
	e.painTimer = 0
	e.shootTimer = 0
	e.setState(config.EnemyStateDead)
}

// handlePain handles the enemy's reaction to pain by resetting the pain timer and transitioning to the pain state if not dead.
func (e *Enemy) handlePain() {
	if e.state == config.EnemyStateDead {
		return
	}
	durationFrames := e.setState(config.EnemyStatePain)
	e.painTimer = framesToSeconds(durationFrames)
}

// handleJump checks and handles the jump behavior of the enemy based on ground status, player position, and movement constraints.
func (e *Enemy) handleJump(self config.IThingConfig, playerZ, thingZ, playerDist2d, nx, ny float64) {
	entity := self.GetEntity()
	if !entity.IsOnGround() {
		return
	}
	//vx, vy, _ := entity.GetVelocity()
	//speedSq := (vx * vx) + (vy * vy)
	// Euristica A: bloccato contro un muro/ostacolo
	//isBlocked := speedSq < 0.1
	floorDz := float64(int(playerZ) - int(thingZ))
	playerIsHigher := floorDz > (self.GetMaxStep() * 4)
	if playerIsHigher {
		//if isBlocked || playerIsHigher {
		//e.setState(self, EnemyStateJump)
		//Forza verticale: Moltiplichiamo per la massa affinché demoni di peso diverso saltino in modo coerente.
		//leapForceXY := entity.GetMass() * 8.0
		//self.Jump(nx*leapForceXY, ny*leapForceXY, 1.0)
		self.Jump(0, 0, 8)
	}
}

// setState updates the enemy's current state and triggers the corresponding animation unless the state is unchanged or dead.
func (e *Enemy) setState(state config.EnemyState) int {
	if e.state == state {
		return e.animationDurationFrames
	}
	if e.state == config.EnemyStateDead || state >= config.EnemyStateLast {
		return 0
	}
	if e.animations == nil || e.animations.Content[state] == nil ||
		len(e.animations.Content[state].Indices) == 0 {
		fmt.Println("No animations for state:", e.self.GetId(), state)
		return 0
	}
	durationFrames, ok := e.playAnimation(state, e.animations.Content[state])
	if !ok {
		return 0
	}
	e.state = state
	e.animationDurationFrames = durationFrames
	fmt.Println("State changed to:", e.self.GetId(), state)
	return durationFrames
}

// chooseAnimation selects an animation index from a list of options based on the enemy's current state, ensuring variation.
// It avoids repeating the last animation for the given state, if possible, and stores the selected animation.
func (e *Enemy) chooseAnimation(state config.EnemyState, animations *config.EnemyAnimation) (int, int, bool) {
	if animations == nil || len(animations.Indices) == 0 {
		return 0, 0, false
	}
	if len(animations.Indices) == 1 {
		duration := 0
		if len(animations.Duration) > 0 {
			duration = animations.Duration[0]
		}
		e.lastAnimation[state] = animations.Indices[0]
		return animations.Indices[0], duration, true
	}
	last, hasLast := e.lastAnimation[state]
	// Conta le animazioni candidate, escludendo l'ultima scelta.
	candidateCount := 0
	for _, idx := range animations.Indices {
		if hasLast && idx == last {
			continue
		}
		candidateCount++
	}
	// Se sono tutte uguali all'ultima scelta, usa la prima posizione.
	if candidateCount == 0 {
		e.lastAnimation[state] = animations.Indices[0]
		duration := 0
		if len(animations.Duration) > 0 {
			duration = animations.Duration[0]
		}
		return animations.Indices[0], duration, true
	}
	target := rand.Intn(candidateCount)
	for pos, idx := range animations.Indices {
		if hasLast && idx == last {
			continue
		}
		if target == 0 {
			duration := 0
			if pos < len(animations.Duration) {
				duration = animations.Duration[pos]
			}
			e.lastAnimation[state] = idx
			return idx, duration, true
		}
		target--
	}
	return 0, 0, false
}

// playAnimation selects an animation and returns its duration in frames.
func (e *Enemy) playAnimation(category config.EnemyState, animations *config.EnemyAnimation) (int, bool) {
	action, durationFrames, ok := e.chooseAnimation(category, animations)
	if !ok {
		return 0, false
	}
	e.self.SetAction(action)
	return durationFrames, true
}
