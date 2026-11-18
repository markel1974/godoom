package things

import (
	"fmt"
	"sync"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

// Weapon represents an interactive barrier that can open and close, optionally requiring a key and activation trigger.
type Weapon struct {
	*Base
	mu          sync.Mutex
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
func NewWeapon(things IThingHandler, cfg *config.Thing, volume *volumes.Volume) *Weapon {
	return &Weapon{}

	/*
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
			door.closedPos = geometry.XYZ{
				X: cfg.Position.X + spec.OpenOffset.X,
				Y: cfg.Position.Y + spec.OpenOffset.Y,
				Z: cfg.Position.Z + spec.OpenOffset.Z,
			}
			door.openOffset = geometry.XYZ{
				X: -spec.OpenOffset.X,
				Y: -spec.OpenOffset.Y,
				Z: -spec.OpenOffset.Z,
			}
		}
		ent.MoveCenterTo(door.closedPos.X, door.closedPos.Y, door.closedPos.Z)
		door.Compile(0)
		return door

	*/

}

// IsEqual reports whether other is the same Door instance.
func (d *Weapon) IsEqual(other vertices.IThing) bool {
	if v, ok := other.(*Weapon); ok && d == v {
		return true
	}
	return false
}

// Activate attempts to open the door for the given thing, requiring any configured key.
// Activate returns true when activation succeeds or refreshes an open door's wait timer.
func (d *Weapon) Activate(by vertices.IThing) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

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
func (d *Weapon) RequiredKey() string {
	return d.requiredKey
}

// IsOpen reports whether the door is open or currently opening.
func (d *Weapon) IsOpen() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.state == doorOpen || d.state == doorOpening
}

// Jump reports whether the door accepted a jump request; doors cannot jump, so it always returns false.
func (d *Weapon) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan ignores hitscan impacts on the door.
func (d *Weapon) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject ignores projectile launch requests for doors.
func (d *Weapon) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage queues the given thing event for asynchronous door processing.
func (d *Weapon) PostMessage(ec *vertices.ThingEvent) {
	d.inbox <- ec
}

// StartLoop starts a goroutine that processes door stage events until the door is done.
func (d *Weapon) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-d.inbox:
				switch evt.GetKind() {
				case vertices.StageThinking:
					d.StageThinking(evt.GetCoords())
				case vertices.StageResolve:
					d.StageResolve()
				case vertices.StageApply:
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
func (d *Weapon) HasThinking() bool {
	return true
}

// StageThinking advances the door state for the current tick, updating velocity and auto-close timing.
func (d *Weapon) StageThinking(playerX float64, playerY float64, playerZ float64) {
}

/*
// onDoorCollision activates the door for key holders on touch or while closing, then applies elastic collision handling.
func (d *Weapon) onDoorCollision(other collisions.IObject, energy float64) {
	if otherThing, ok := other.(vertices.IThing); ok {
		if _, isHolder := other.(IKeyHolder); isHolder {
			d.mu.Lock()
			state := d.state
			activation := d.activation
			d.mu.Unlock()

			if state == doorClosing || activation == config.DoorActivateTouch {
				d.Activate(otherThing)
			}
		}
	}
	d.onElasticCollision(other, energy)
}


*/
