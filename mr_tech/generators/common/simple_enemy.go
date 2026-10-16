package common

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// EnemyState represents the various states an enemy entity can be in during its lifecycle.
type EnemyState uint8

// EnemyStateUnknown represents an undefined or unknown state of an enemy.
// EnemyStateIdle represents the state of an enemy when it is not performing any action.
// EnemyStateChase represents the state of an enemy when it is chasing a target.
// EnemyStateShoot represents the state of an enemy when it is shooting at a target.
// EnemyStateMelee represents the state of an enemy when it is engaging in melee combat.
// EnemyStatePain represents the state of an enemy when it is experiencing pain or being damaged.
// EnemyStateDead represents the state of an enemy when it is defeated or dead.
// EnemyStateSwim represents the state of an enemy when it is swimming or in water.
// EnemyStateJump represents the state of an enemy when it is jumping.
// EnemyStateOther represents any state of an enemy that does not fit into predefined categories.
const (
	EnemyStateUnknown EnemyState = iota
	EnemyStateIdle
	EnemyStateChase
	EnemyStateShoot
	EnemyStateMelee
	EnemyStatePain
	EnemyStateDead
	EnemyStateSwim
	EnemyStateJump
	EnemyStateOther
)

// EnemyAnimations represents a collection of animation sequences categorized by enemy behaviors or states.
type EnemyAnimations struct {
	Idle  []int
	Chase []int
	Shoot []int
	Melee []int
	Pain  []int
	Dead  []int
	Swim  []int
	Jump  []int
	Other []int
}

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
	state         EnemyState
	health        float64
	throwCooldown float64
	throwMin      float64
	painTimer     float64
	shootTimer    float64
	actions       map[string]int
	animations    EnemyAnimations
	lastAnimation map[EnemyState]int
}

// NewEnemy creates and initializes a new Enemy instance with default properties and animations based on provided actions.
func NewEnemy(actions []string) *Enemy {
	const (
		throwMin = 2
		throwMax = 6
	)

	e := &Enemy{
		state:         EnemyStateUnknown,
		health:        100,
		throwMin:      float64(rand.Intn(throwMax-throwMin+1) + throwMin),
		actions:       make(map[string]int, len(actions)),
		lastAnimation: make(map[EnemyState]int),
	}

	for idx, action := range actions {
		name := normalizeAction(action)
		if name == "" || strings.HasPrefix(name, "(") {
			continue
		}
		e.actions[name] = idx
	}

	e.animations = EnemyAnimations{
		Idle: e.collectAnimations(
			[]string{"idle", "stand"},
			[]string{"idle", "stand"},
		),
		Chase: e.collectAnimations(
			[]string{"walk", "run", "back"},
			[]string{"walk", "run", "back"},
		),
		Shoot: e.collectAnimations(
			[]string{"shoot", "fire", "gun"},
			[]string{"shoot", "fire", "gun"},
		),
		Melee: e.collectAnimations(
			[]string{"melee", "attack", "attak", "swing", "smash", "bite", "claw"},
			[]string{"melee", "attack", "attak", "swing", "smash", "bite", "claw"},
		),
		Pain: e.collectAnimations(
			[]string{"pain"},
			[]string{"pain"},
		),
		Dead: e.collectAnimations(
			[]string{"death", "dead", "bdeath", "die"},
			[]string{"death", "dead", "bdeath", "die"},
		),
		Swim: e.collectAnimations(
			[]string{"swim"},
			[]string{"swim"},
		),
		Jump: e.collectAnimations(
			[]string{"jump", "jumpb", "land", "landb"},
			[]string{"jump", "land"},
		),
		Other: e.collectAnimations(
			[]string{"pull", "turn", "duck", "block"},
			nil,
		),
	}

	e.logUnassignedAnimations()

	return e
}

// collectAnimations filters and returns the indices of animations matching exact names or prefixes specified.
func (e *Enemy) collectAnimations(exactNames []string, prefixes []string) []int {
	exact := make(map[string]struct{}, len(exactNames))
	for _, name := range exactNames {
		exact[baseAction(normalizeAction(name))] = struct{}{}
	}

	normalizedPrefixes := make([]string, 0, len(prefixes))

	for _, prefix := range prefixes {
		normalizedPrefixes = append(
			normalizedPrefixes,
			baseAction(normalizeAction(prefix)),
		)
	}
	seen := make(map[int]struct{})
	result := make([]int, 0)
	for name, idx := range e.actions {
		base := baseAction(name)
		_, matched := exact[base]
		if !matched {
			for _, prefix := range normalizedPrefixes {
				if prefix != "" &&
					strings.HasPrefix(base, prefix) &&
					len(base) > len(prefix) {
					matched = true
					break
				}
			}
		}
		if !matched {
			continue
		}
		if _, exists := seen[idx]; exists {
			continue
		}
		seen[idx] = struct{}{}
		result = append(result, idx)
	}
	sort.Ints(result)
	return result
}

// logUnassignedAnimations identifies and logs actions that are not assigned to any animation group in the Enemy entity.
func (e *Enemy) logUnassignedAnimations() {
	assigned := make(map[int]struct{})

	groups := [][]int{
		e.animations.Idle,
		e.animations.Chase,
		e.animations.Shoot,
		e.animations.Melee,
		e.animations.Pain,
		e.animations.Dead,
		e.animations.Swim,
		e.animations.Jump,
		e.animations.Other,
	}

	for _, group := range groups {
		for _, idx := range group {
			assigned[idx] = struct{}{}
		}
	}

	type unassignedAction struct {
		index int
		name  string
	}

	var unassigned []unassignedAction

	for name, idx := range e.actions {
		if _, ok := assigned[idx]; ok {
			continue
		}

		unassigned = append(unassigned, unassignedAction{index: idx, name: name})
	}

	sort.Slice(unassigned, func(i, j int) bool {
		if unassigned[i].index != unassigned[j].index {
			return unassigned[i].index < unassigned[j].index
		}
		return unassigned[i].name < unassigned[j].name
	})

	for _, action := range unassigned {
		fmt.Printf("[Enemy] unrecognized action: index=%d name=%q\n", action.index, action.name)
	}
}

// chooseAnimation selects an animation index from a list of options based on the enemy's current state, ensuring variation.
// It avoids repeating the last animation for the given state, if possible, and stores the selected animation.
func (e *Enemy) chooseAnimation(state EnemyState, animations []int) (int, bool) {
	if len(animations) == 0 {
		return 0, false
	}

	if len(animations) == 1 {
		e.lastAnimation[state] = animations[0]
		return animations[0], true
	}

	last, hasLast := e.lastAnimation[state]
	candidates := animations

	if hasLast {
		candidates = make([]int, 0, len(animations))

		for _, idx := range animations {
			if idx != last {
				candidates = append(candidates, idx)
			}
		}
	}

	if len(candidates) == 0 {
		return 0, false
	}

	idx := candidates[rand.Intn(len(candidates))]
	e.lastAnimation[state] = idx

	return idx, true
}

// playAnimation selects an appropriate animation for the enemy and sets it as the current action in the given config.
func (e *Enemy) playAnimation(self config.IThingConfig, category EnemyState, animations []int) bool {
	action, ok := e.chooseAnimation(category, animations)
	if !ok {
		return false
	}

	self.SetAction(action)
	return true
}

// setState updates the enemy's current state and triggers the corresponding animation unless the state is unchanged or dead.
func (e *Enemy) setState(self config.IThingConfig, state EnemyState) {
	if e.state == state || e.state == EnemyStateDead {
		return
	}

	var animations []int

	switch state {
	case EnemyStateIdle:
		animations = e.animations.Idle
	case EnemyStateChase:
		animations = e.animations.Chase
	case EnemyStateShoot:
		animations = e.animations.Shoot
	case EnemyStateMelee:
		animations = e.animations.Melee
	case EnemyStatePain:
		animations = e.animations.Pain
	case EnemyStateDead:
		animations = e.animations.Dead
	case EnemyStateSwim:
		animations = e.animations.Swim
	case EnemyStateJump:
		animations = e.animations.Jump
	case EnemyStateOther:
		animations = e.animations.Other
	default:
		return
	}

	e.state = state
	e.playAnimation(self, state, animations)
}

// OnCollision defines the optional behavior when this Enemy collides with another object.
func (e *Enemy) OnCollision(self config.IThingConfig, other config.IThingConfig) {
	// Comportamento di collisione opzionale.
}

// OnImpact handles the behavior of the enemy when impacted by another object, adjusting health and state accordingly.
func (e *Enemy) OnImpact(self config.IThingConfig, other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	if e.state == EnemyStateDead {
		return
	}
	e.health -= force * 0.5
	if e.health <= 0 {
		e.handleDeath(self)
		return
	}
	if e.state == EnemyStateIdle ||
		e.state == EnemyStateUnknown {
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

	if e.state == EnemyStateDead {
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

	if e.state == EnemyStateUnknown {
		e.setState(self, EnemyStateIdle)
	}
	if e.state == EnemyStatePain {
		e.painTimer -= dt

		if e.painTimer <= 0 {
			e.painTimer = 0
			e.setState(self, EnemyStateChase)
		}

		return
	}

	if e.state == EnemyStateShoot ||
		e.state == EnemyStateMelee {
		e.shootTimer -= dt
		if e.shootTimer <= 0 {
			e.shootTimer = 0
			e.setState(self, EnemyStateChase)
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

	if e.state == EnemyStateIdle {
		wakeDistance := self.GetWakeUpDistance()

		if dist3dSq < wakeDistance*wakeDistance {
			e.throwCooldown = e.throwMin
			e.setState(self, EnemyStateChase)
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
	if playerDist2d <= stopDistance &&
		e.throwCooldown <= 0 &&
		len(e.animations.Melee) > 0 {
		e.setState(self, EnemyStateMelee)
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

	if e.throwCooldown <= 0 && playerDist3d < throwPlayerDistance && len(e.animations.Shoot) > 0 {
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

	if e.state == EnemyStateDead {
		return
	}

	e.setState(self, EnemyStateShoot)
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
	if e.state == EnemyStateDead {
		return
	}
	e.setState(self, EnemyStateShoot)
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
	if e.state == EnemyStateDead {
		return
	}
	e.painTimer = 0
	e.shootTimer = 0
	e.setState(self, EnemyStateDead)
}

// handlePain handles the enemy's reaction to pain by resetting the pain timer and transitioning to the pain state if not dead.
func (e *Enemy) handlePain(self config.IThingConfig) {
	if e.state == EnemyStateDead {
		return
	}
	e.painTimer = 0.5
	e.setState(self, EnemyStatePain)
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
