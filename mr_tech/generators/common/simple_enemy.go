package common

import (
	"math"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

type EnemyState int

const (
	EnemyStateUnknown EnemyState = iota
	EnemyStateIdle
	EnemyStateChase
	EnemyStateAttack
	EnemyStatePain
	EnemyStateDead
)

// framesToSeconds converts an animation frame count to seconds using the
// current animation/update FPS. Missing durations remain zero; no duration is guessed.
func framesToSeconds(frames int) float64 {
	fps := physics.GetFps()
	if frames <= 0 || fps <= 0 {
		return 0
	}
	t := float64(frames) / float64(fps)
	//t *= 5
	return t
}

// Enemy represents an entity with various states, animations, health, and timers for interaction and combat behavior.
type Enemy struct {
	state                   EnemyState
	activeAnimation         config.EnemyState
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
func NewEnemy(thingCfg *config.Thing, animations *config.EnemyAnimations) *Enemy {
	const (
		shotIntervalMin = 2
		shotIntervalMax = 4
	)
	e := &Enemy{
		state:           EnemyStateUnknown,
		activeAnimation: config.EnemyStateUnknown,
		health:          100,
		shotIntervalMin: float64(rand.Intn(shotIntervalMax-shotIntervalMin+1) + shotIntervalMin),
		actions:         make(map[string]int, len(animations.Content)),
		lastAnimation:   make(map[config.EnemyState]int),
		animations:      animations,
	}
	thingCfg.OnBind = e.OnBind
	thingCfg.OnThinking = e.OnThinking
	thingCfg.OnCollision = e.OnCollision
	thingCfg.OnImpact = e.OnImpact
	return e
}

// OnBind associates the Enemy instance with its configuration, enabling interaction and behavior initialization.
func (e *Enemy) OnBind(self config.IThingConfig) {
	e.self = self
}

// OnCollision defines the optional behavior when this Enemy collides with another object.
func (e *Enemy) OnCollision(other config.IThingConfig) {
	if e.state == EnemyStateDead {
		return
	}
	otherEntity := other.GetEntity()
	selfEntity := e.self.GetEntity()
	ovx, ovy, ovz := otherEntity.GetVelocity()
	svx, svy, svz := selfEntity.GetVelocity()
	otherMass := otherEntity.GetMass()
	selfMass := selfEntity.GetMass()
	dvx := ovx - svx
	dvy := ovy - svy
	dvz := ovz - svz
	relativeSpeedSq := dvx*dvx + dvy*dvy + dvz*dvz
	totalMass := otherMass + selfMass
	if otherMass <= 0 || selfMass <= 0 || totalMass <= 0 {
		return
	}
	reducedMass := (otherMass * selfMass) / totalMass
	impactEnergy := 0.5 * reducedMass * relativeSpeedSq
	//fmt.Println("impact energy:", impactEnergy)
	const painfulEnergy = 10.0
	if impactEnergy >= painfulEnergy {
		e.handleHit()
	}
}

// OnImpact handles the behavior of the enemy when impacted by another object, adjusting health and state accordingly.
func (e *Enemy) OnImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	if e.state == EnemyStateDead {
		return
	}
	e.health -= force * 0.5
	e.handleHit()
}

// OnThinking handles the behavior logic for the Enemy, updating its state and actions based on its environment and player position.
func (e *Enemy) OnThinking(pCenterX, pCenterY, pCenterZ float64) {
	if e.state == EnemyStateDead {
		return
	}

	getPlayerDistance := func(selfX, selfY, selfZ float64) (float64, float64, float64, float64, float64) {
		dx := pCenterX - selfX
		dy := pCenterY - selfY
		dz := pCenterZ - selfZ
		dist2dSq := dx*dx + dy*dy
		dist3dSq := dist2dSq + dz*dz
		return dx, dy, dz, dist2dSq, dist3dSq
	}

	if e.shootCooldown > 0 {
		fps := physics.GetFps()
		dt := 1.0 / float64(fps)
		e.shootCooldown -= dt
		if e.shootCooldown < 0 {
			e.shootCooldown = 0
		}
	}

	switch e.state {
	case EnemyStateUnknown:
		e.state = EnemyStateIdle
		e.setAnimationState(config.EnemyStateIdle)
		return

	case EnemyStateIdle:
		selfX, selfY, selfZ := e.self.GetEntity().GetCenter()
		_, _, _, _, dist3dSq := getPlayerDistance(selfX, selfY, selfZ)
		wakeDistance := e.self.GetWakeUpDistance()
		if dist3dSq < (wakeDistance * wakeDistance) {
			e.state = EnemyStateChase
			e.setAnimationState(config.EnemyStateRun)
		}
		return

	case EnemyStateChase:
		entity := e.self.GetEntity()
		selfX, selfY, selfZ := entity.GetCenter()
		dx, dy, _, dist2dSq, dist3dSq := getPlayerDistance(selfX, selfY, selfZ)
		wakeDistance := e.self.GetWakeUpDistance()
		if dist3dSq > (wakeDistance * wakeDistance) {
			e.state = EnemyStateIdle
			e.setAnimationState(config.EnemyStateIdle)
			return
		}
		if dist2dSq > 0.000001 {
			e.self.SetAngle(math.Atan2(dy, dx))
			nx := float64(0)
			ny := float64(0)
			targetSpeed := float64(0)
			stopDistance := entity.GetWidth() * 3
			if playerDist2d := math.Sqrt(dist2dSq); playerDist2d > stopDistance {
				nx = dx / playerDist2d
				ny = dy / playerDist2d
				targetSpeed = e.self.GetSpeed() * e.self.GetImpulse()
			}
			e.self.MoveTowards(nx, ny, targetSpeed, e.self.GetAcceleration())
		}
		if entity.IsOnGround() {
			e.handleJump(e.self, pCenterZ, selfZ, math.Sqrt(dist2dSq), dx, dy)
		}
		if e.hasAnimationState(config.EnemyStateShoot) {
			if e.shootCooldown <= 0 {
				e.state = EnemyStateAttack
				durationFrames := e.setAnimationState(config.EnemyStateShoot)
				e.shootTimer = framesToSeconds(durationFrames) + 0.1
			}
		}
		return

	case EnemyStateAttack:
		dt := 1.0 / float64(physics.GetFps())
		e.shootTimer -= dt
		if e.shootTimer <= 0 {
			selfX, selfY, selfZ := e.self.GetEntity().GetCenter()
			dx, dy, dz, dist2dSq, _ := getPlayerDistance(selfX, selfY, selfZ)
			e.shootTimer = 0
			angle := math.Atan2(dy, dx)
			playerDist2d := math.Sqrt(dist2dSq)
			e.doShoot(e.self, selfX, selfY, selfZ, dz, angle, playerDist2d)
			e.shootCooldown = e.shotIntervalMin
			e.state = EnemyStateChase
			e.setAnimationState(config.EnemyStateRun)
		}
		return

	case EnemyStatePain:
		dt := 1.0 / float64(physics.GetFps())
		e.painTimer -= dt
		if e.painTimer <= 0 {
			e.painTimer = 0
			e.state = EnemyStateChase
			e.setAnimationState(config.EnemyStateRun)
		}
		return

	case EnemyStateDead:
		return
	default:
	}
}

// hasAnimationState checks if a specific animation state is defined and valid for the enemy's animation configuration.
func (e *Enemy) hasAnimationState(state config.EnemyState) bool {
	if e.animations == nil || e.animations.Content[state] == nil || len(e.animations.Content[state].Indices) == 0 {
		return false
	}
	return true
}

// setState updates the enemy's current state and triggers the corresponding animation unless the state is unchanged or dead.
func (e *Enemy) setAnimationState(state config.EnemyState) int {
	if e.activeAnimation == state {
		return e.animationDurationFrames
	}
	if e.activeAnimation == config.EnemyStateDead {
		return 0
	}
	if !e.hasAnimationState(state) {
		//fmt.Println("No animations for state:", e.self.GetId(), state)
		return 0
	}
	durationFrames, _, ok := e.playAnimation(state, e.animations.Content[state])
	if !ok {
		return 0
	}
	e.activeAnimation = state
	e.animationDurationFrames = durationFrames
	//fmt.Println("State changed to:", e.self.GetId(), state, action)
	return durationFrames
}

// playAnimation selects an animation and returns its duration in frames.
func (e *Enemy) playAnimation(category config.EnemyState, eAnim *config.EnemyAnimation) (int, int, bool) {
	chooseAnimation := func(state config.EnemyState, animations *config.EnemyAnimation) (int, int, bool) {
		if len(animations.Indices) == 0 {
			//fmt.Println("No animations for state:", e.self.GetId(), state)
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

	action, durationFrames, ok := chooseAnimation(category, eAnim)
	if !ok {
		return 0, 0, false
	}
	e.self.SetAction(action)
	return durationFrames, action, true
}

// handleLaunch handles the logic for launching a throwable object, including positioning and aiming calculations.
func (e *Enemy) handleLaunch(self config.IThingConfig, selfX, selfY, selfZ float64, dz, angle, playerDist2d float64) {
	const (
		throwableIndex = 2
		throwableSpeed = 100.0
	)

	//durationFrames := e.setState(config.EnemyStateShoot)
	//e.shootTimer = framesToSeconds(durationFrames)
	//e.shootCooldown = e.shotIntervalMin

	entity := self.GetEntity()

	weaponForward := entity.GetWidth()
	spawnX := selfX + math.Cos(angle)*weaponForward
	spawnY := selfY + math.Sin(angle)*weaponForward
	spawnZ := selfZ + entity.GetDepth()*0.5

	bulletPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}

	aimPitch := math.Atan2(dz, playerDist2d)

	self.LaunchObject(throwableIndex, e.OnCollision, e.OnImpact, bulletPos, angle, aimPitch, throwableSpeed)
}

// doShoot sets the enemy to a shooting state, computes projectile trajectory, and performs a hitscan-based weapon attack.
func (e *Enemy) doShoot(self config.IThingConfig, selfX, selfY, selfZ, dz, angle, playerDist2d float64) {
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

// handleHealth adjusts the Enemy's state and timers when its health drops to or below zero.
func (e *Enemy) handleHit() bool {
	if e.health > 0 {
		e.state = EnemyStatePain
		durationFrames := e.setAnimationState(config.EnemyStatePain)
		e.painTimer = framesToSeconds(durationFrames)
		e.shootTimer = 0
		return true
	}
	e.painTimer = 0
	e.shootTimer = 0
	e.state = EnemyStateDead
	e.setAnimationState(config.EnemyStateDead)
	return false
}
