package things

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// Detached represents one rigid-body segment in Things.tree: either initially attached to a living owner
// (tracking keyframe/tag poses kinematically) or free/ragdoll 6-DoF with an optional breakable JointConstraint.
type Detached struct {
	*Base
	partVerts          *vertices.Vertices3DPart
	owner              vertices.IThing
	pendingImpactOwner vertices.IThing
	ragdollJoint       *collisions.JointConstraint
	breakThreshold     float64
	damage             float64
}

func noopDetachedCollision(other config.IThingConfig, energy float64) {}

func noopDetachedImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
}

// NewAttachedPart creates a tree-resident segment Thing bound to owner until detached or switched to ragdoll.
func NewAttachedPart(things IThingHandler, owner vertices.IThing, location *volumes.Volume, id string, verts *vertices.Vertices3DPart) *Detached {
	ent := verts.GetSolver().GetEntity()
	cfg := config.NewConfigThing(id, geometry.XYZ{}, 0, config.ThingItemDef, ent.GetMass(), 1, 1, 0)
	cfg.OnCollision = noopDetachedCollision
	cfg.OnImpact = noopDetachedImpact

	d := &Detached{
		Base:      NewBaseWithVertices(things, cfg, verts),
		partVerts: verts,
		owner:     owner,
	}
	if err := d.Prepare(d, location); err != nil {
		panic(err)
	}
	// Preserve kinematic attached state initialized by Vertices3DPart.Setup after Base.Prepare resets angle/sleep.
	d.SetMaxStep(0.0)
	d.solver.SetElasticCollision(d.onPartElasticCollision)
	return d
}

// NewDetached initializes an autonomous 6-DoF rigid-body limb from vol at pos with the given orientation, velocities,
// and optional self-collision exclusion group (0 when freely dismembered).
func NewDetached(things IThingHandler, vol *volumes.Volume, location *volumes.Volume, id string, pos geometry.XYZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64, collisionGroup uint64) *Detached {
	ent := vol.GetSolver().GetEntity()
	cfg := config.NewConfigThing(id, pos, 0, config.ThingItemDef, ent.GetMass(), 1, 1, 0)
	cfg.OnCollision = noopDetachedCollision
	cfg.OnImpact = noopDetachedImpact

	d := &Detached{
		Base: NewBaseWithVertices(things, cfg, vertices.NewVerticesVolume(vol)),
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

// GetOwner returns the owning parent IThing while this segment belongs to it, or nil once detached.
func (t *Detached) GetOwner() vertices.IThing {
	return t.owner
}

// IsPhysicsFree reports whether this segment simulates as an autonomous 6-DoF rigid body.
func (t *Detached) IsPhysicsFree() bool {
	if t.partVerts != nil {
		return t.partVerts.IsPhysicsFree()
	}
	return true
}

// IsSleeping returns false while attached/kinematic so contact queries against live limbs never try to wake an unstaged solver.
func (t *Detached) IsSleeping() bool {
	if !t.IsPhysicsFree() {
		return false
	}
	return t.Base.IsSleeping()
}

// BindJoint associates a breakable ragdoll joint constraint with this limb so heavy hits can sever it.
func (t *Detached) BindJoint(jc *collisions.JointConstraint, breakThreshold float64) {
	t.ragdollJoint = jc
	t.breakThreshold = breakThreshold
}

func (t *Detached) onPartElasticCollision(other collisions.IObject, energy float64) {
	if !t.IsPhysicsFree() && t.owner != nil {
		if b, ok := t.owner.(interface {
			onElasticCollision(other collisions.IObject, energy float64)
		}); ok {
			b.onElasticCollision(other, energy)
		}
	}
}

// ApplyImpact routes hits either to the owning living model (for knockback/dismemberment) or to this free/ragdoll limb.
func (t *Detached) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	if !t.IsPhysicsFree() {
		prevOwner := t.owner
		t.Base.ApplyImpact(fx, fy, fz, px, py, pz)
		if t.IsPhysicsFree() {
			t.pendingImpactOwner = prevOwner
			t.owner = nil
		}
		return
	}
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

// Impact delivers hit damage callbacks to the owning living model until this segment becomes detached.
func (t *Detached) Impact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	if t.pendingImpactOwner != nil {
		owner := t.pendingImpactOwner
		t.pendingImpactOwner = nil
		owner.Impact(other, id, force, closestDist, dirX, dirY, dirZ)
		return
	}
	if !t.IsPhysicsFree() && t.owner != nil {
		t.owner.Impact(other, id, force, closestDist, dirX, dirY, dirZ)
	}
}

// IsEqual reports whether other is the same Detached instance.
func (t *Detached) IsEqual(other vertices.IThing) bool {
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
