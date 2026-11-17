package articulated_dynamics

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// DetachFunc handles spawning an autonomous rigid-body solver when a joint detaches.
// origX/Y/Z is the detached part's pivot offset relative to the parent entity center.
type DetachFunc func(partIdx int, origX, origY, origZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64)

// Joints manages hierarchical skeletal tag transforms, ragdoll transition state, and joint detachment specs.
type Joints struct {
	links            *Links
	data             []*Joint
	dismemberedCount int
	lastHitDirX      float64
	lastHitDirY      float64
	lastHitDirZ      float64
	baseOffsetX      float64
	baseOffsetY      float64
	baseOffsetZ      float64

	rotationsA []*TagAxis
	rotationsB []*TagAxis

	lastHitPart  int
	lastHitForce float64

	isRagdoll bool

	originsA []geometry.XYZ
	originsB []geometry.XYZ
	numParts int

	detachQuat *aabb.Quaternion
	onDetach   DetachFunc
}

// NewJoints creates a Joints instance with default state.
func NewJoints() *Joints {
	pj := &Joints{
		lastHitPart: -1,
		detachQuat:  aabb.NewQuaternionIdentity(),
	}
	return pj
}

// SetOnDetach registers a callback invoked when a detachable joint breaks into an autonomous rigid body.
func (v *Joints) SetOnDetach(fn DetachFunc) {
	v.onDetach = fn
}

// Setup initializes joint state, rotations, origins, and per-part motion constraints from configJoints and links.
func (v *Joints) Setup(configJoints []*Joint, links *Links) error {
	v.numParts = len(configJoints)
	v.links = links
	v.rotationsA = make([]*TagAxis, v.numParts)
	v.rotationsB = make([]*TagAxis, v.numParts)
	v.data = make([]*Joint, v.numParts)

	copy(v.data, configJoints)

	for i := 0; i < v.numParts; i++ {
		v.rotationsA[i] = NewTagAxisIdentity()
		v.rotationsB[i] = NewTagAxisIdentity()
	}

	v.originsA = make([]geometry.XYZ, v.numParts)
	v.originsB = make([]geometry.XYZ, v.numParts)

	return nil
}

// IsRagdoll reports whether the joints are currently in ragdoll mode.
func (v *Joints) IsRagdoll() bool {
	return v.isRagdoll
}

// GetJoint returns the joint specification at index i.
func (v *Joints) GetJoint(i int) *Joint {
	return v.data[i]
}

// LastHitImpulse returns the normalized last hit direction and force magnitude for seeding ragdoll collapse momentum.
func (v *Joints) LastHitImpulse() (dirX, dirY, dirZ, fMag float64, hitPart int) {
	return v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ, v.lastHitForce, v.lastHitPart
}

// SetBaseOffsetZ records the static Z centering offset aligning model bounds with Entity center.
func (v *Joints) SetBaseOffsetZ(z float64) {
	v.baseOffsetZ = z
}

// SetBaseOffset records the static 3D centering offset aligning model bounds with Entity center.
func (v *Joints) SetBaseOffset(x, y, z float64) {
	v.baseOffsetX = x
	v.baseOffsetY = y
	v.baseOffsetZ = z
}

// GetOriginsA returns the X, Y, and Z components of the A-side origin for the joint at index i.
func (v *Joints) GetOriginsA(i int) (float64, float64, float64) {
	return v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z
}

// GetOriginsB returns the X, Y, and Z coordinates of origin B for the joint at index i.
func (v *Joints) GetOriginsB(i int) (float64, float64, float64) {
	return v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z
}

// GetRotationA returns the primary rotation axis for the joint at index i.
func (v *Joints) GetRotationA(i int) *TagAxis {
	return v.rotationsA[i]
}

// GetRotationB returns the secondary rotation axis for the joint at index i.
func (v *Joints) GetRotationB(i int) *TagAxis {
	return v.rotationsB[i]
}

// CompileNormal derives hierarchical joint origins and rotations from parent transforms and optional tag axes.
func (v *Joints) CompileNormal(i int, pIdx int, tagAxisA, tagAxisB *TagAxis) {
	if tagAxisA != nil {
		pRotA := v.rotationsA[pIdx]
		rotOrigAX, rotOrigAY, rotOrigAZ := pRotA.RotatePointByTag(tagAxisA)
		v.originsA[i].X = v.originsA[pIdx].X + rotOrigAX
		v.originsA[i].Y = v.originsA[pIdx].Y + rotOrigAY
		v.originsA[i].Z = v.originsA[pIdx].Z + rotOrigAZ
		pRotA.MultiplyMatrix(tagAxisA, v.rotationsA[i])
	} else {
		fmt.Printf("[warning] Vertices3D: tag not found in frame A of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", pIdx, i)
		v.originsA[i] = v.originsA[pIdx]
		v.rotationsA[i].Copy(v.rotationsA[pIdx])
	}

	if tagAxisB != nil {
		pRotB := v.rotationsB[pIdx]
		rotOrigBX, rotOrigBY, rotOrigBZ := pRotB.RotatePointByTag(tagAxisB)
		v.originsB[i].X = v.originsB[pIdx].X + rotOrigBX
		v.originsB[i].Y = v.originsB[pIdx].Y + rotOrigBY
		v.originsB[i].Z = v.originsB[pIdx].Z + rotOrigBZ
		pRotB.MultiplyMatrix(tagAxisB, v.rotationsB[i])
	} else {
		fmt.Printf("[warning] Vertices3D: tag not found in frame B of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", pIdx, i)
		v.originsB[i] = v.originsB[pIdx]
		v.rotationsB[i].Copy(v.rotationsB[pIdx])
	}
}

// DetachPart marks a detachable joint as detached and invokes onDetach with launch momentum.
func (v *Joints) DetachPart(idx int, dirX, dirY, dirZ, fMag, pvx, pvy, pvz float64) {
	if idx <= 0 || idx >= len(v.data) || v.dismemberedCount >= 3 {
		return
	}
	j := v.data[idx]
	if j.detached || !j.canDetach {
		return
	}
	v.dismemberedCount++
	sideSign := 1.0
	if idx&1 == 1 {
		sideSign = -1.0
	}
	vx, vy, vz, wx, wy, wz := j.Detach(dirX, dirY, dirZ, fMag, sideSign)
	vx += pvx
	vy += pvy
	vz += pvz
	v.rotationsA[idx].ToQuaternion(v.detachQuat)
	qx, qy, qz, qw := v.detachQuat.Get()
	orX, orY, orZ := v.originsA[idx].X, v.originsA[idx].Y, v.originsA[idx].Z
	if v.onDetach != nil {
		v.onDetach(idx, orX, orY, orZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz)
	}
}

// RecordLastHit stores the normalized hit vector, magnitude, and matched part index for ragdoll collapse handoff.
func (v *Joints) RecordLastHit(dirX, dirY, dirZ, fMag float64, hitPart int) {
	v.lastHitForce = fMag
	v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ = dirX, dirY, dirZ
	if hitPart >= 0 {
		v.lastHitPart = hitPart
	}
}

// CanDetachMore reports whether the dismembered limb cap has not yet been reached.
func (v *Joints) CanDetachMore() bool {
	return v.dismemberedCount < 3
}

// IsPartDetached reports whether the part at i or any valid ancestor is marked detached.
func (v *Joints) IsPartDetached(i int) bool {
	curr := i
	for steps := 0; steps < len(v.data) && curr > 0 && curr < len(v.data); steps++ {
		if v.data[curr].detached {
			return true
		}
		curr = v.links.GetParent(curr)
		if curr < 0 {
			break
		}
	}
	return false
}

// ActivateRagdoll switches the articulated system into multi-body ragdoll mode and optionally detaches a fatally damaged extremity.
func (v *Joints) ActivateRagdoll(pvx, pvy, pvz float64) {
	v.isRagdoll = true

	dx, dy, dz := v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ
	if dx*dx+dy*dy+dz*dz < 1e-6 {
		dx = 1.0
	}

	if v.dismemberedCount < 3 && v.lastHitPart > 0 && v.lastHitPart < len(v.data) {
		hj := v.data[v.lastHitPart]
		if hj.IsCriticallyDamaged() {
			v.DetachPart(v.lastHitPart, dx, dy, dz, v.lastHitForce, pvx, pvy, pvz)
		}
	}
}

// Restore resets the joint origins and rotations at i using q.
func (v *Joints) Restore(i int, q *aabb.Quaternion) {
	v.rotationsA[i].SetFromQuaternion(q)
	v.rotationsB[i].SetFromQuaternion(q)
	ox, oy, oz := v.rotationsA[i].RotatePoint(v.baseOffsetX, v.baseOffsetY, v.baseOffsetZ)
	v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z = ox, oy, oz
	v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z = ox, oy, oz
}

// Reset copies parent origins and rotations to joint i.
func (v *Joints) Reset(i int, pIdx int) {
	v.originsA[i] = v.originsA[pIdx]
	v.originsB[i] = v.originsB[pIdx]
	*v.rotationsA[i] = *v.rotationsA[pIdx]
	*v.rotationsB[i] = *v.rotationsB[pIdx]
}

// IsDetached reports whether the joint at index i is detached.
func (v *Joints) IsDetached(i int) bool {
	return v.data[i].detached
}
