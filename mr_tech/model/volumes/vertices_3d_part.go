package volumes

import (
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Vertices3DPart adapts one skeletal segment (Vertices3DEntry) as a standalone IVertices backed by its own Volume and Solver.
// While attached to its owner it tracks keyframe/tag poses kinematically; on detach or ragdoll it switches directly to 6-DoF rigid physics.
type Vertices3DPart struct {
	parent      *Vertices3D
	entry       *Vertices3DEntry
	partIdx     int
	solver      *collisions.Solver
	ent         *rigid_body.Entity
	groupID     uint64
	physicsFree bool
}

// NewVertices3DPart creates an IVertices wrapper for segment partIdx of parent.
func NewVertices3DPart(parent *Vertices3D, entry *Vertices3DEntry, partIdx int, collisionGroup uint64) *Vertices3DPart {
	return &Vertices3DPart{
		parent:  parent,
		entry:   entry,
		partIdx: partIdx,
		solver:  entry.GetSolver(),
		ent:     entry.GetSolver().GetEntity(),
		groupID: collisionGroup,
	}
}

// Setup binds the segment's Volume Solver to thing and places it in kinematic attached mode.
func (v *Vertices3DPart) Setup(thing IThing) error {
	if err := v.entry.Setup(thing); err != nil {
		return err
	}
	v.solver = v.entry.GetSolver()
	v.ent = v.solver.GetEntity()
	v.ent.SetKeepUpright(false)
	v.ent.SetKinematic(true)
	v.solver.SetMaxStep(0.0)
	v.solver.SetCollisionGroup(v.groupID)
	return nil
}

// IsPhysicsFree reports whether this segment has transitioned from kinematic animation to autonomous 6-DoF physics.
func (v *Vertices3DPart) IsPhysicsFree() bool {
	return v.physicsFree
}

// ReleasePhysics transitions this segment in-place to autonomous 6-DoF rigid-body simulation.
func (v *Vertices3DPart) ReleasePhysics(collisionGroup uint64, vx, vy, vz, wx, wy, wz float64) {
	cx, cy, cz := v.ent.GetCenter()
	qx, qy, qz, qw := v.ent.GetQRotation().Get()

	v.entry.FreezeCurrentFrame()
	v.entry.SyncSolverToCurrentFrame()

	v.physicsFree = true
	v.ent.SetKinematic(false)
	v.ent.SetKeepUpright(false)
	v.solver.SetMaxStep(0.0)
	v.solver.SetCanSleep(true)
	v.solver.SetCollisionGroup(collisionGroup)
	v.ent.MoveCenterTo(cx, cy, cz)
	v.ent.SetRotationQuaternion(qx, qy, qz, qw)
	v.ent.SetV(vx, vy, vz)
	v.ent.SetAngularVelocity(wx, wy, wz)
	v.entry.lastRotVolA = nil
	v.entry.CompileDetached()
}

// Compile updates the owner animation pose when attached, or rotates the frozen rigid mesh when detached/ragdoll.
func (v *Vertices3DPart) Compile(tick uint64) bool {
	if v.physicsFree {
		v.entry.CompileDetached()
		return false
	}
	v.parent.Compile(tick)
	return false
}

// GetAABB returns this segment's current world-space bounding box.
func (v *Vertices3DPart) GetAABB() *aabb.AABB {
	b := v.ent.GetAABB()
	cx, cy, cz := v.ent.GetCenter()
	b.Rebuild(cx+v.entry.minX, cy+v.entry.minY, cz+v.entry.minZ, cx+v.entry.maxX, cy+v.entry.maxY, cz+v.entry.maxZ)
	return b
}

// GetSolver returns this segment's dedicated collision solver.
func (v *Vertices3DPart) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps queries face overlaps against the segment's current centered collision volume.
func (v *Vertices3DPart) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.entry.QueryOverlaps(object, callback)
}

// GetUIFaces returns this segment's render faces and interpolation weight.
func (v *Vertices3DPart) GetUIFaces() (*Faces, *Faces, float64, float64) {
	if v.physicsFree {
		return v.entry.facesA, v.entry.facesA, 0.0, RenderModeModel3D
	}
	return v.entry.facesA, v.entry.facesB, v.entry.lerpT, RenderModeModel3D
}

// SetAction delegates action selection on this entry.
func (v *Vertices3DPart) SetAction(idx int) {
	v.entry.SetAction(idx)
}

// ApplyImpact either wakes the free rigid body or forwards localized hit stress to the owning model for joint breakage.
func (v *Vertices3DPart) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	if v.physicsFree {
		v.ent.WakeUp()
		return
	}
	v.parent.ApplyPartImpact(v.partIdx, fx, fy, fz, px, py, pz)
}

// SetRagdollMode is handled at the owning Vertices3D level.
func (v *Vertices3DPart) SetRagdollMode() {}
