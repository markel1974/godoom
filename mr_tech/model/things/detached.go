package things

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

func noopDetachedCollision(other config.IThingConfig, energy float64) {}

func noopDetachedImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
}

// Detached represents an autonomous or ragdoll-linked 3D body part simulated as a 6-DoF rigid body.
type Detached struct {
	*Base
	ragdollJoint   *collisions.JointConstraint
	breakThreshold float64
	damage         float64
}

// NewDetached initializes a 6-DoF rigid-body limb from vol at pos with the given orientation, velocities,
// and optional self-collision exclusion group (0 when freely dismembered).
func NewDetached(things IThingHandler, vol *volumes.Volume, location *volumes.Volume, id string, pos geometry.XYZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64, collisionGroup uint64) *Detached {
	ent := vol.GetSolver().GetEntity()
	cfg := config.NewConfigThing(id, pos, 0, config.ThingItemDef, ent.GetMass(), 1, 1, 0)
	cfg.OnCollision = noopDetachedCollision
	cfg.OnImpact = noopDetachedImpact

	d := &Detached{
		Base: NewBaseWithVertices(things, cfg, volumes.NewVerticesVolume(vol)),
	}
	if err := d.Prepare(d, location); err != nil {
		panic(err)
	}
	d.SetMaxStep(0.0)
	d.GetSolver().SetCollisionGroup(collisionGroup)
	dEnt := d.GetEntity()
	dEnt.SetKeepUpright(false)
	dEnt.MoveCenterTo(pos.X, pos.Y, pos.Z)
	dEnt.SetRotationQuaternion(qx, qy, qz, qw)
	dEnt.SetV(vx, vy, vz)
	dEnt.SetAngularVelocity(wx, wy, wz)
	d.Compile(0)
	return d
}

// BindJoint associates a breakable ragdoll joint constraint with this limb so heavy hits can sever it.
func (t *Detached) BindJoint(jc *collisions.JointConstraint, breakThreshold float64) {
	t.ragdollJoint = jc
	t.breakThreshold = breakThreshold
}

// ApplyImpact wakes the limb and breaks its attached ragdoll joint if accumulated hit stress exceeds threshold.
func (t *Detached) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	t.Base.ApplyImpact(fx, fy, fz, px, py, pz)
	if t.ragdollJoint != nil && t.ragdollJoint.IsActive() && t.breakThreshold > 0 {
		fMag := math.Sqrt(fx*fx + fy*fy + fz*fz)
		stress := math.Min(fMag/500000.0, 1.6)
		if stress < 0.45 {
			stress = 0.45
		}
		t.damage += stress * 0.90
		if t.damage >= t.breakThreshold {
			t.ragdollJoint.Break()
		}
	}
}

// IsEqual reports whether other is the same Detached instance.
func (t *Detached) IsEqual(other volumes.IThing) bool {
	if v, ok := other.(*Detached); ok && t == v {
		return true
	}
	return false
}

// HasThinking reports false because detached limbs do not run AI thinking.
func (t *Detached) HasThinking() bool {
	return false
}

// Jump returns false because detached limbs cannot jump.
func (t *Detached) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan is a no-op for detached limbs.
func (t *Detached) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject is a no-op for detached limbs.
func (t *Detached) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage is a no-op because detached limbs do not spawn a thinker goroutine.
func (t *Detached) PostMessage(ec *volumes.ThingEvent) {}

// StartLoop is a no-op because detached limbs do not run AI thinking.
func (t *Detached) StartLoop() {}

// StageThinking is a no-op for detached limbs.
func (t *Detached) StageThinking(playerX float64, playerY float64, playerZ float64) {}
