package config

import "fmt"

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
	EnemyStateWalk
	EnemyStateProwl
	EnemyStateChase
	EnemyStateShoot
	EnemyStateReload
	EnemyStateMelee
	EnemyStatePain
	EnemyStateDead
	EnemyStatePull
	EnemyStateSwim
	EnemyStateJump
	EnemyStateLand
	EnemyStateSwing
	EnemyStateFly
	EnemyStateKneel
	EnemyStateBlock
	EnemyStateTurn
	EnemyStateBack
	EnemyStateOther
	EnemyStateLast
)

func (s EnemyState) String() string {
	switch s {
	case EnemyStateUnknown:
		return "Unknown"
	case EnemyStateIdle:
		return "Idle"
	case EnemyStateWalk:
		return "Walk"
	case EnemyStateProwl:
		return "Prowl"
	case EnemyStateChase:
		return "Chase"
	case EnemyStateShoot:
		return "Shoot"
	case EnemyStateReload:
		return "Reload"
	case EnemyStateMelee:
		return "Melee"
	case EnemyStatePain:
		return "Pain"
	case EnemyStateDead:
		return "Dead"
	case EnemyStatePull:
		return "Pull"
	case EnemyStateSwim:
		return "Swim"
	case EnemyStateJump:
		return "Jump"
	case EnemyStateLand:
		return "Land"
	case EnemyStateSwing:
		return "Swing"
	case EnemyStateFly:
		return "Fly"
	case EnemyStateKneel:
		return "Kneel"
	case EnemyStateBlock:
		return "Block"
	case EnemyStateTurn:
		return "Turn"
	case EnemyStateBack:
		return "Back"
	case EnemyStateOther:
		return "Other"
	case EnemyStateLast:
		return "Last"
	default:
		return fmt.Sprintf("EnemyState(%d)", uint8(s))
	}
}

// EnemyAnimation defines the structure for managing enemy animation sequences with frame indices and durations.
type EnemyAnimation struct {
	Indices  []int
	Duration []int
}

// NewEnemyAnimation creates a new instance of EnemyAnimation with initialized empty slices for Indices and Duration.
func NewEnemyAnimation() *EnemyAnimation {
	return &EnemyAnimation{}
}

// EnemyAnimations holds a set of animations associated with different EnemyState values.
type EnemyAnimations struct {
	Content [EnemyStateLast]*EnemyAnimation
}

// NewEnemyAnimations initializes and returns a new EnemyAnimations instance with default EnemyAnimation objects for each state.
func NewEnemyAnimations() *EnemyAnimations {
	e := &EnemyAnimations{}
	for idx := range e.Content {
		e.Content[idx] = NewEnemyAnimation()
	}
	return e
}

// Add appends the given indices and duration to the corresponding enemy state's animation data.
func (e *EnemyAnimations) Add(idx EnemyState, indices int, duration int) {
	e.Content[idx].Indices = append(e.Content[idx].Indices, indices)
	e.Content[idx].Duration = append(e.Content[idx].Duration, duration)
}
