package things

import (
	"sync/atomic"
	"time"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

const defaultTeleportCooldown = 500 * time.Millisecond

// Teleport is a trigger entity that relocates colliding actors to a destination position and orientation.
type Teleport struct {
	*Base
	destination   geometry.XYZ
	targetAngle   float64
	preserveAngle bool
	resetVelocity bool
	playerOnly    bool
	cooldown      time.Duration
	lastTrigger   atomic.Int64
}

// NewTeleport creates and initializes a Teleport Thing with collision-based relocation behavior.
func NewTeleport(things IThingHandler, cfg *config.Thing, volume *volumes.Volume) *Teleport {
	tp := &Teleport{
		Base:     NewBase(things, cfg),
		cooldown: defaultTeleportCooldown,
	}
	if cfg.Teleport != nil {
		tp.destination = cfg.Teleport.Destination
		tp.targetAngle = cfg.Teleport.TargetAngle
		tp.preserveAngle = cfg.Teleport.PreserveAngle
		tp.resetVelocity = cfg.Teleport.ResetVelocity
		tp.playerOnly = cfg.Teleport.PlayerOnly
		if cfg.Teleport.Cooldown < 0 {
			tp.cooldown = 0
		} else if cfg.Teleport.Cooldown > 0 {
			tp.cooldown = time.Duration(cfg.Teleport.Cooldown * float64(time.Second))
		}
	}
	if err := tp.Prepare(tp, volume); err != nil {
		panic(err)
	}
	tp.GetSolver().SetElasticCollision(tp.onTeleportCollision)
	if cfg.OnBind != nil {
		cfg.OnBind(tp)
	}
	return tp
}

// IsEqual reports whether other is the same Teleport instance.
func (tp *Teleport) IsEqual(other ithing.IThing) bool {
	if v, ok := other.(*Teleport); ok && tp == v {
		return true
	}
	return false
}

// Destination returns the teleport target coordinates.
func (tp *Teleport) Destination() geometry.XYZ {
	return tp.destination
}

// Jump is unsupported for static teleport triggers.
func (tp *Teleport) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan is unsupported for static teleport triggers.
func (tp *Teleport) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject is unsupported for static teleport triggers.
func (tp *Teleport) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage sends a ThingEvent to the Teleport's inbox channel.
func (tp *Teleport) PostMessage(ec *ithing.ThingEvent) {
	tp.inbox <- ec
}

// StartLoop starts the goroutine processing physics stages for the Teleport.
func (tp *Teleport) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-tp.inbox:
				switch evt.GetKind() {
				case ithing.StageThinking:
					tp.StageThinking(evt.GetCoords())
				case ithing.StageResolve:
					tp.StageResolve()
				case ithing.StageApply:
					tp.StageApply()
				}
				evt.Done()
			case <-tp.done:
				return
			}
		}
	}()
}

// HasThinking reports false because teleport triggers are passive until touched.
func (tp *Teleport) HasThinking() bool {
	return false
}

// StageThinking is a no-op for passive teleport triggers.
func (tp *Teleport) StageThinking(playerX float64, playerY float64, playerZ float64) {
}

func (tp *Teleport) onTeleportCollision(other collisions.IObject, energy float64) {
	actor, ok := other.(ithing.IThing)
	if !ok {
		return
	}
	if tp.playerOnly && actor.GetKind() != config.ThingPlayerDef {
		return
	}
	if tp.cooldown > 0 {
		now := time.Now().UnixNano()
		prev := tp.lastTrigger.Load()
		if now-prev < tp.cooldown.Nanoseconds() {
			return
		}
		if !tp.lastTrigger.CompareAndSwap(prev, now) {
			return
		}
	}

	ent := actor.GetEntity()
	ent.MoveTo(tp.destination.X, tp.destination.Y, tp.destination.Z)
	if tp.resetVelocity {
		ent.Stop()
	}
	if !tp.preserveAngle {
		actor.SetAngle(tp.targetAngle)
	}
	if destVol := tp.things.QueryVolume(tp.destination.X, tp.destination.Y, tp.destination.Z); destVol != nil {
		actor.GetSolver().SetLocation(destVol)
	}
}
