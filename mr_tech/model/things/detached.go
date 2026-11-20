package things

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_multi_element_3d"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// Detached represents one rigid-body segment in Things.tree: initially attached to a living owner
// (tracking keyframe/tag poses kinematically) and transitioning in-place to free/ragdoll 6-DoF with an optional JointConstraint.
type Detached struct {
	*Base
	partVerts      *vertices_multi_element_3d.Part
	owner          ithing.IThing
	ragdollJoint   *collisions.JointConstraint
	breakThreshold float64
	damage         float64
}

func noopDetachedCollision(other config.IThingConfig, energy float64) {}

func noopDetachedImpact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
}

// NewAttachedPart creates a tree-resident segment Thing bound to owner until detached or switched to ragdoll.
func NewAttachedPart(things IThingHandler, owner ithing.IThing, location *volumes.Volume, id string, verts *vertices_multi_element_3d.Part) *Detached {
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
	d.SetMaxStep(0.0)
	d.solver.SetElasticCollision(d.onPartElasticCollision)
	d.solver.SetInelasticCollision(d.onPartInelasticCollision)
	return d
}

// GetOwner returns the owning parent IThing while this segment belongs to it, or nil once detached.
func (t *Detached) GetOwner() ithing.IThing {
	return t.owner
}

// IsPhysicsFree reports whether this segment simulates as an autonomous 6-DoF rigid body.
func (t *Detached) IsPhysicsFree() bool {
	return t.partVerts.IsPhysicsFree()
}

// IsSleeping returns false while attached/kinematic so contact queries against live limbs never try to wake an unstaged solver.
func (t *Detached) IsSleeping() bool {
	if !t.IsPhysicsFree() {
		return false
	}
	return t.Base.IsSleeping()
}

// BindJoint associates a breakable ragdoll joint constraint with this limb so impact events can sever it.
func (t *Detached) BindJoint(jc *collisions.JointConstraint, breakThreshold float64) {
	t.ragdollJoint = jc
	t.breakThreshold = breakThreshold
}

func (t *Detached) applyRagdollStress(force float64) {
	if t.ragdollJoint == nil || !t.ragdollJoint.IsActive() || t.breakThreshold <= 0 {
		return
	}
	stress := math.Min(force/500000.0, 1.6)
	if stress < 0.45 {
		stress = 0.45
	}
	t.damage += stress * 0.90
	if t.damage >= t.breakThreshold {
		t.ragdollJoint.Break()
	}
}

func (t *Detached) onPartElasticCollision(other aabb.IObject, energy float64) {
	if !t.IsPhysicsFree() && t.owner != nil {
		if b, ok := t.owner.(interface {
			onElasticCollision(other aabb.IObject, energy float64)
		}); ok {
			b.onElasticCollision(other, energy)
		}
		if energy >= 200.0 {
			sx, sy, sz := t.GetEntity().GetCenter()
			dx, dy, dz := 1.0, 0.0, 0.0
			if oThing, ok := other.(ithing.IThing); ok {
				ox, oy, oz := oThing.GetEntity().GetCenter()
				dx, dy, dz = sx-ox, sy-oy, sz-oz
			}
			t.partVerts.HandleImpactEvent(energy*1000.0, dx, dy, dz)
			if t.IsPhysicsFree() {
				t.owner = nil
			}
		}
		return
	}
	if energy >= 400.0 {
		t.applyRagdollStress(energy * 1000.0)
	}
}

func (t *Detached) onPartInelasticCollision(_ aabb.IObject, energy float64) {
	if t.IsPhysicsFree() && energy >= 400.0 {
		t.applyRagdollStress(energy * 1000.0)
	}
}

// Impact receives emerging hitscan impact events on this segment and routes them into joint stress/detachment
// (forwarding damage to the living owner while attached) or into breakable ragdoll joints once free.
func (t *Detached) Impact(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
	if !t.IsPhysicsFree() {
		prevOwner := t.owner
		t.partVerts.HandleImpactEvent(force, dirX, dirY, dirZ)
		if t.IsPhysicsFree() {
			t.owner = nil
		}
		if prevOwner != nil {
			prevOwner.Impact(other, id, force, closestDist, dirX, dirY, dirZ)
		}
		return
	}
	t.GetEntity().WakeUp()
	t.applyRagdollStress(force)
}

// IsEqual reports whether other is the same Detached instance.
func (t *Detached) IsEqual(other ithing.IThing) bool {
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
func (t *Detached) PostMessage(ec *ithing.ThingEvent) {}

// StartLoop is a no-op because detached limbs do not run AI thinking.
func (t *Detached) StartLoop() {}

// StageThinking is a no-op for detached limbs.
func (t *Detached) StageThinking(playerX float64, playerY float64, playerZ float64) {}
