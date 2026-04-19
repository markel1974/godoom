package model

import "sync"

/*
type EventType uint8
const (
	EventTick EventType = iota
	EventDamage
	EventCollide
	EventDestroy
)
*/

// ComputeStage represents the various stages of a computation process, typically modeled as distinct uint8 values.
type ComputeStage uint8

// StageThinking represents the compute stage for AI and Aggro-related calculations.
// StagePhysics represents the compute stage for Sweep and Collision-related calculations.
const (
	StageThinking ComputeStage = iota //  0 (AI, Aggro)
	StagePhysics                      //  1 (Sweep, Collision)
	StageCollision
)

// ThingEvent represents an event associated with a Thing, containing position data, a compute stage, and synchronization.
type ThingEvent struct {
	playerX float64
	playerY float64
	playerZ float64
	stage   ComputeStage
	wg      *sync.WaitGroup
}

// NewThingEvent initializes and returns a new instance of ThingEvent with an embedded sync.WaitGroup.
func NewThingEvent() *ThingEvent {
	return &ThingEvent{
		wg: &sync.WaitGroup{},
	}
}

// SetStage updates the ComputeStage of the ThingEvent to the specified stage.
func (e *ThingEvent) SetStage(stage ComputeStage) {
	e.stage = stage
}

func (e *ThingEvent) SetCoords(x, y, z float64) {
	e.playerX, e.playerY, e.playerZ = x, y, z
}

// GetCoords returns the X, Y, and Z player coordinates of the ThingEvent instance.
func (e *ThingEvent) GetCoords() (float64, float64, float64) {
	return e.playerX, e.playerY, e.playerZ
}

// Done signals that the current operation associated with the ThingEvent is complete by marking the WaitGroup as done.
func (e *ThingEvent) Done() {
	e.wg.Done()
}

// GetKind returns the ComputeStage value representing the stage or stage associated with the ThingEvent.
func (e *ThingEvent) GetKind() ComputeStage {
	return e.stage
}
