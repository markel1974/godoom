package things

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// IActivatable represents an object that can be activated by another thing.
type IActivatable interface {
	Activate(by ithing.IThing) bool
}

// doorState represents the current phase of a door's movement and waiting behavior.
type doorState uint8

// Defines the possible states of a door during its open and close cycle.
const (
	doorClosed doorState = iota
	doorOpening
	doorOpen
	doorClosing
)

// Door represents an interactive barrier that can open and close, optionally requiring a key and activation trigger.
type Door struct {
	*Base
	closedPos   geometry.XYZ
	openOffset  geometry.XYZ
	speed       float64
	waitSeconds float64
	requiredKey string
	activation  config.DoorActivation
	state       doorState
	progress    float64
	waitTimer   float64
}

// NewDoor creates and initializes a door thing from configuration, applying defaults and preparing its volume.
func NewDoor(things IThingHandler, cfg *config.Thing, volume *volumes.Volume) *Door {
	spec := config.DoorSpec{
		Speed:       100.0,
		WaitSeconds: 3.0,
		Activation:  config.DoorActivateTouch,
	}
	if cfg.Door != nil {
		spec = *cfg.Door
	}
	if spec.Speed <= 0 {
		spec.Speed = 100.0
	}

	door := &Door{
		Base:        NewBase(things, cfg),
		closedPos:   cfg.Position,
		openOffset:  spec.OpenOffset,
		speed:       spec.Speed,
		waitSeconds: spec.WaitSeconds,
		requiredKey: strings.ToLower(strings.TrimSpace(spec.RequiredKey)),
		activation:  spec.Activation,
		state:       doorClosed,
		progress:    0.0,
	}
	if err := door.Prepare(door, volume); err != nil {
		panic(err)
	}

	ent := door.GetEntity()
	ent.SetKinematic(true)
	door.GetSolver().SetIgnoreStatic(true)
	door.GetSolver().SetCanSleep(true)
	door.GetSolver().SetElasticCollision(door.onDoorCollision)
	cfg.OnBind(door)

	if spec.StartOpen {
		// Symmetrize start-open doors so progress=0 is their initial open position and activation closes them.
		door.closedPos = geometry.XYZ{X: cfg.Position.X + spec.OpenOffset.X, Y: cfg.Position.Y + spec.OpenOffset.Y, Z: cfg.Position.Z + spec.OpenOffset.Z}
		door.openOffset = geometry.XYZ{X: -spec.OpenOffset.X, Y: -spec.OpenOffset.Y, Z: -spec.OpenOffset.Z}
	}
	ent.MoveCenterTo(door.closedPos.X, door.closedPos.Y, door.closedPos.Z)
	door.Compile(0)
	return door
}

// IsEqual reports whether other is the same Door instance.
func (d *Door) IsEqual(other ithing.IThing) bool {
	if v, ok := other.(*Door); ok && d == v {
		return true
	}
	return false
}

// Activate attempts to open the door for the given thing, requiring any configured key.
// Activate returns true when activation succeeds or refreshes an open door's wait timer.
func (d *Door) Activate(by ithing.IThing) bool {
	if d.requiredKey != "" {
		holder, ok := by.(IKeyHolder)
		if !ok || !holder.HasKey(d.requiredKey) {
			fmt.Printf("Door %s locked: requires key %q\n", d.id, d.requiredKey)
			return false
		}
	}

	switch d.state {
	case doorClosed, doorClosing:
		d.state = doorOpening
		d.WakeUp()
		return true
	case doorOpen:
		if d.waitSeconds >= 0 {
			d.waitTimer = d.waitSeconds
		}
		return true
	default:
		return false
	}
}

// RequiredKey returns the key identifier required to activate the door, or an empty string if unlocked.
func (d *Door) RequiredKey() string {
	return d.requiredKey
}

// IsOpen reports whether the door is open or currently opening.
func (d *Door) IsOpen() bool {
	return d.state == doorOpen || d.state == doorOpening
}

// Jump reports whether the door accepted a jump request; doors cannot jump, so it always returns false.
func (d *Door) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan ignores hitscan impacts on the door.
func (d *Door) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject ignores projectile launch requests for doors.
func (d *Door) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage queues the given thing event for asynchronous door processing.
func (d *Door) PostMessage(ec *ithing.ThingEvent) {
	d.inbox <- ec
}

// StartLoop starts a goroutine that processes door stage events until the door is done.
func (d *Door) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-d.inbox:
				switch evt.GetKind() {
				case ithing.StageThinking:
					d.StageThinking(evt.GetCoords())
				case ithing.StageResolve:
					d.StageResolve()
				case ithing.StageApply:
					d.StageApply()
				}
				evt.Done()
			case <-d.done:
				return
			}
		}
	}()
}

// HasThinking reports whether the door participates in the thinking update stage.
func (d *Door) HasThinking() bool {
	return true
}

// StageThinking advances the door state for the current tick, updating velocity and auto-close timing.
func (d *Door) StageThinking(playerX float64, playerY float64, playerZ float64) {
	ent := d.GetEntity()
	dt := ent.GetDt()
	if dt <= 0 {
		return
	}

	switch d.state {
	case doorClosed:
		ent.SetV(0, 0, 0)
	case doorOpening:
		d.advanceDoor(ent, dt, 1.0, doorOpen)
	case doorOpen:
		ent.SetV(0, 0, 0)
		if d.waitSeconds >= 0 {
			d.waitTimer -= dt
			if d.waitTimer <= 0 {
				d.state = doorClosing
				d.WakeUp()
			}
		}
	case doorClosing:
		d.advanceDoor(ent, dt, -1.0, doorClosed)
	}
}

// advanceDoor advances the door toward open or closed progress, updates velocity, and switches state when complete.
func (d *Door) advanceDoor(ent *rigid_body.Entity, dt float64, dir float64, nextState doorState) {
	dist := math.Sqrt(d.openOffset.X*d.openOffset.X + d.openOffset.Y*d.openOffset.Y + d.openOffset.Z*d.openOffset.Z)
	if dist <= 1e-6 {
		if dir > 0 {
			d.progress = 1.0
			d.waitTimer = d.waitSeconds
		} else {
			d.progress = 0.0
		}
		d.state = nextState
		ent.SetV(0, 0, 0)
		return
	}

	d.progress += dir * (d.speed * dt) / dist
	done := false
	if dir > 0 && d.progress >= 1.0 {
		d.progress = 1.0
		d.waitTimer = d.waitSeconds
		done = true
	} else if dir < 0 && d.progress <= 0.0 {
		d.progress = 0.0
		done = true
	}

	tx := d.closedPos.X + d.openOffset.X*d.progress
	ty := d.closedPos.Y + d.openOffset.Y*d.progress
	tz := d.closedPos.Z + d.openOffset.Z*d.progress
	cx, cy, cz := ent.GetCenter()
	ent.SetV((tx-cx)/dt, (ty-cy)/dt, (tz-cz)/dt)
	d.WakeUp()

	if done {
		d.state = nextState
	}
}

// onDoorCollision activates the door for key holders on touch or while closing, then applies elastic collision handling.
func (d *Door) onDoorCollision(other collisions.IObject, energy float64) {
	if otherThing, ok := other.(ithing.IThing); ok {
		if _, isHolder := other.(IKeyHolder); isHolder {
			state := d.state
			activation := d.activation

			if state == doorClosing || activation == config.DoorActivateTouch {
				d.Activate(otherThing)
			}
		}
	}
	d.onElasticCollision(other, energy)
}
