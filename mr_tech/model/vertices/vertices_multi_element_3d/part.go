package vertices_multi_element_3d

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_element_3d"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Part adapts one skeletal segment (Vertices3DEntry) as a standalone IVertices backed by its own Volume and Solver.
// While attached to its owner it tracks keyframe/tag poses kinematically; on detach or ragdoll it switches directly to 6-DoF rigid physics.
type Part struct {
	parent      *MultiElement3D
	entry       *vertices_element_3d.Element3D
	partIdx     int
	solver      *collisions.Solver
	ent         *rigid_body.Entity
	groupID     uint64
	physicsFree bool
}

// NewPart creates an IVertices wrapper for segment partIdx of parent.
func NewPart(parent *MultiElement3D, entry *vertices_element_3d.Element3D, partIdx int, collisionGroup uint64) *Part {
	return &Part{
		parent:  parent,
		entry:   entry,
		partIdx: partIdx,
		solver:  entry.GetSolver(),
		ent:     entry.GetSolver().GetEntity(),
		groupID: collisionGroup,
	}
}

// Setup binds the segment's Volume Solver to thing and places it in kinematic attached mode.
func (v *Part) Setup(thing ithing.IThing) error {
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
func (v *Part) IsPhysicsFree() bool {
	return v.physicsFree
}

// ReleasePhysics transitions this segment in-place to autonomous 6-DoF rigid-body simulation.
func (v *Part) ReleasePhysics(collisionGroup uint64, vx, vy, vz, wx, wy, wz float64) {
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
	if v.parent != nil && v.parent.ent != nil && v.solver != nil && v.solver.GetCage() != nil {
		ox, oy, oz := v.parent.ent.GetCenter()
		v.solver.GetCage().SetSpawnOrigin(ox, oy, oz)
	}
	v.entry.UnsetLastRolVolA()
	v.entry.CompileDetached()
	v.UpdateAABB()
}

// Compile updates the owner animation pose when attached, or rotates the frozen rigid mesh when detached/ragdoll.
func (v *Part) Compile(tick uint64) bool {
	if v.physicsFree {
		v.entry.CompileDetached()
		v.UpdateAABB()
		return false
	}
	v.parent.Compile(tick)
	return false
}

// UpdateAABB updates the part's world-space bounding box from its current entity center and entry bounds.
func (v *Part) UpdateAABB() {
	cx, cy, cz := v.ent.GetCenter()
	entryMinX, entryMinY, entryMinZ, entryMaxX, entryMaxY, entryMaxZ := v.entry.GetMinMax()
	v.ent.GetAABB().Rebuild(cx+entryMinX, cy+entryMinY, cz+entryMinZ, cx+entryMaxX, cy+entryMaxY, cz+entryMaxZ)
}

// GetAABB returns this segment's current world-space bounding box.
func (v *Part) GetAABB() *aabb.AABB {
	if v.ent == nil {
		return nil
	}
	return v.ent.GetAABB()
}

// Translate shifts the part AABB by the specified displacement.
func (v *Part) Translate(dx, dy, dz float64) {
}

// GetSolver returns this segment's dedicated collision solver.
func (v *Part) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps queries face overlaps against the segment's current centered collision volume.
func (v *Part) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.entry.QueryOverlaps(object, callback)
}

// GetUIFaces returns this segment's render faces and interpolation weight.
func (v *Part) GetUIFaces() (*volumes.Faces, *volumes.Faces, float64, float64) {
	if v.physicsFree {
		faces := v.entry.GetFacesA()
		return faces, faces, 0.0, vertices.RenderModeModel3D
	}
	return v.entry.GetFacesA(), v.entry.GetFacesB(), v.entry.GetLerpT(), vertices.RenderModeModel3D
}

// GetRigidFaces delegates to the underlying Element3D entry for non-morphing sub-parts.
func (v *Part) GetRigidFaces() *volumes.Faces {
	if v.physicsFree {
		return v.entry.GetRigidFaces()
	}
	if v.entry.GetLerpT() == 0 {
		return v.entry.GetRigidFaces()
	}
	return nil
}

// SetAction delegates action selection on this entry.
func (v *Part) SetAction(idx int) {
	v.entry.SetAction(idx)
}

// HandleImpactEvent forwards an emerging impact event on this attached segment to its owning model's joint chain.
func (v *Part) HandleImpactEvent(force, dirX, dirY, dirZ float64) {
	if v.physicsFree {
		return
	}
	v.parent.HandlePartImpact(v.partIdx, force, dirX, dirY, dirZ)
}

// SetRagdollMode is handled at the owning MultiElement3D level.
func (v *Part) SetRagdollMode() {}

// Deform applies an impulse-based deformation to this part within radius of impact.
// It returns true if the underlying entry was deformed.
func (v *Part) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	return v.entry.Deform(impact, impulse, radius)
}

// DeformDent applies a dent deformation at impact along normal with the given depth and radius.
func (v *Part) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	return v.entry.DeformDent(impact, normal, depth, radius)
}

// IntersectRay tests a ray against this part up to maxDist.
// IntersectRay returns the hit distance, hit point, and whether an intersection was found.
func (v *Part) IntersectRay(oX, oY, oZ, dirX, dirY, dirZ, maxDist float64) (float64, geometry.XYZ, bool) {
	return v.entry.IntersectRay(oX, oY, oZ, dirX, dirY, dirZ, maxDist)
}
