package articulated_dynamics

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// DetachFunc handles spawning an autonomous rigid-body solver when a joint detaches.
// origX/Y/Z is the detached part's pivot offset relative to the parent entity center.
type DetachFunc func(partIdx int, origX, origY, origZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64)

// Joints manages joint physics, ragdoll state, detachment, and transforms for linked 3D model parts.
type Joints struct {
	links             *Links
	data              []*Joint
	dismemberedCount  int
	hasActiveDynamics bool
	lastHitDirX       float64
	lastHitDirY       float64
	lastHitDirZ       float64
	baseOffsetZ       float64

	rotationsA []*TagAxis
	rotationsB []*TagAxis

	lastHitPart  int
	lastHitForce float64

	isRagdoll bool
	lastTick  uint64

	originsA []geometry.XYZ
	originsB []geometry.XYZ
	numParts int

	solver     *collisions.Solver
	ent        *rigid_body.Entity
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

// Setup initializes joint state, rotations, origins, and per-part motion constraints from elements and links.
func (v *Joints) Setup(solver *collisions.Solver, configJoints []*Joint, links *Links) error {
	v.solver = solver
	v.ent = v.solver.GetEntity()
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

// CompileNormal derives joint origins and rotations from parent transforms and optional tag axes.
// Missing tags inherit parent transforms; configured deflection is applied when present.
func (v *Joints) CompileNormal(i int, pIdx int, tagAxisA, tagAxisB *TagAxis) {
	j := v.data[i]
	if tagAxisA != nil {
		pRotA := v.rotationsA[pIdx]
		rotA := v.rotationsA[i]
		rotOrigAX, rotOrigAY, rotOrigAZ := pRotA.RotatePointByTag(tagAxisA)
		v.originsA[i].X = v.originsA[pIdx].X + rotOrigAX
		v.originsA[i].Y = v.originsA[pIdx].Y + rotOrigAY
		v.originsA[i].Z = v.originsA[pIdx].Z + rotOrigAZ
		if j.hasDeflection {
			pRotA.MultiplyMatrix(tagAxisA, j.combinedA)
			j.combinedA.MultiplyMatrix(j.axis, rotA)
		} else {
			pRotA.MultiplyMatrix(tagAxisA, rotA)
		}
	} else {
		fmt.Printf("[warning] Vertices3D: tag not found in frame A of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", pIdx, i)
		v.originsA[i] = v.originsA[pIdx]
		if j.hasDeflection {
			v.rotationsA[pIdx].MultiplyMatrix(j.axis, v.rotationsA[i])
		} else {
			v.rotationsA[i].Copy(v.rotationsA[pIdx])
		}
	}

	if tagAxisB != nil {
		pRotB := v.rotationsB[pIdx]
		rotB := v.rotationsB[i]
		rotOrigBX, rotOrigBY, rotOrigBZ := pRotB.RotatePointByTag(tagAxisB)
		v.originsB[i].X = v.originsB[pIdx].X + rotOrigBX
		v.originsB[i].Y = v.originsB[pIdx].Y + rotOrigBY
		v.originsB[i].Z = v.originsB[pIdx].Z + rotOrigBZ
		if j.hasDeflection {
			pRotB.MultiplyMatrix(tagAxisB, j.combinedB)
			j.combinedB.MultiplyMatrix(j.axis, rotB)
		} else {
			pRotB.MultiplyMatrix(tagAxisB, rotB)
		}
	} else {
		fmt.Printf("[warning] Vertices3D: tag not found in frame B of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", pIdx, i)
		v.originsB[i] = v.originsB[pIdx]
		if j.hasDeflection {
			v.rotationsB[pIdx].MultiplyMatrix(j.axis, v.rotationsB[i])
		} else {
			v.rotationsB[i].Copy(v.rotationsB[pIdx])
		}
	}
}

// detachPart marks a detachable joint as detached, freezes its subtree frames, and emits a rigid-body handoff event.
func (v *Joints) detachPart(idx int, dirX, dirY, dirZ, fMag float64) {
	if idx <= 0 || idx >= len(v.data) || v.dismemberedCount >= 3 {
		return
	}
	j := v.data[idx]
	if j.detached || !j.canDetach {
		return
	}
	v.dismemberedCount++
	v.hasActiveDynamics = true
	sideSign := 1.0
	if idx&1 == 1 {
		sideSign = -1.0
	}
	vx, vy, vz, wx, wy, wz := j.Detach(dirX, dirY, dirZ, fMag, sideSign)
	for i := 0; i < v.numParts; i++ {
		if i == idx || v.IsPartDetached(i) {
			v.data[i].elem.FreezeCurrentFrame()
		}
	}
	pvx, pvy, pvz := v.ent.GetVelocity()
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
func (v *Joints) ActivateRagdoll() {
	v.isRagdoll = true
	v.hasActiveDynamics = false

	dx, dy, dz := v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ
	if dx*dx+dy*dy+dz*dz < 1e-6 {
		dx = 1.0
	}

	// If the killing blow hit a detachable extremity with sufficient accumulated stress, sever it on death.
	if v.dismemberedCount < 3 && v.lastHitPart > 0 && v.lastHitPart < len(v.data) {
		hj := v.data[v.lastHitPart]
		if hj.canDetach && !hj.detached && hj.damage >= hj.detachThreshold*0.45 {
			v.detachPart(v.lastHitPart, dx, dy, dz, v.lastHitForce)
		}
	}
}

// StepJoints advances live hit-flinch spring-dampers to the given simulation tick.
// Once in multi-body ragdoll mode, simulation is handled directly by PGS solvers and constraints.
func (v *Joints) StepJoints(tick uint64) {
	if tick == 0 || v.isRagdoll {
		return
	}
	if v.lastTick == 0 {
		v.lastTick = tick
		return
	}
	if tick <= v.lastTick {
		return
	}
	dt := float64(tick-v.lastTick) / 120.0
	v.lastTick = tick
	if !v.hasActiveDynamics {
		return
	}
	if dt > 1.0/30.0 {
		dt = 1.0 / 30.0
	}

	anyActive := false
	for i := 1; i < len(v.data); i++ {
		j := v.data[i]
		if j.detached {
			continue
		}
		if j.Connect(dt) {
			anyActive = true
		}
	}
	v.hasActiveDynamics = anyActive
}

// ApplyImpact applies an external force at a point, updating ragdoll motion, joint impulses, and detachment damage.
func (v *Joints) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	if v.numParts == 0 {
		return
	}
	fMag := math.Sqrt(fx*fx + fy*fy + fz*fz)
	if fMag < 1e-6 {
		return
	}
	dirX, dirY, dirZ := fx/fMag, fy/fMag, fz/fMag

	v.lastHitForce = fMag
	v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ = dirX, dirY, dirZ
	if v.isRagdoll || v.numParts <= 1 {
		return
	}
	v.hasActiveDynamics = true

	// Normalize force magnitude to angular joint impulse (calibrated to weaponForce = 500,000)
	normImpulse := math.Min(fMag/500000.0, 1.6)
	if normImpulse < 0.45 {
		normImpulse = 0.45
	}

	cx, cy, cz := v.ent.GetCenter()
	relPx, relPy, relPz := px-cx, py-cy, pz-cz

	// Locate the closest non-detached articulated part to the impact ray/point
	bestPart := -1
	bestScore := math.MaxFloat64

	for i := 0; i < v.numParts; i++ {
		if i > 0 && v.IsPartDetached(i) {
			continue
		}
		j := v.data[i]
		lcX, lcY, lcZ := v.rotationsA[i].RotatePoint(j.localCenterX, j.localCenterY, j.localCenterZ)
		partCx := v.originsA[i].X + lcX
		partCy := v.originsA[i].Y + lcY
		partCz := v.originsA[i].Z + lcZ

		vx := partCx - relPx
		vy := partCy - relPy
		vz := partCz - relPz

		// Decompose vector from ray entry point to part center into longitudinal (along ray) and perpendicular components
		along := vx*dirX + vy*dirY + vz*dirZ
		perpX := vx - along*dirX
		perpY := vy - along*dirY
		perpZ := vz - along*dirZ
		perpDist := math.Sqrt(perpX*perpX + perpY*perpY + perpZ*perpZ)
		rad := math.Max(j.partRadius, 6.0)

		score := (perpDist / rad) + math.Abs(along)/(rad*3.5)
		// Favor articulated child limbs over root part 0 when the ray passes near the child limb
		if i > 0 && perpDist <= rad*1.8 {
			score *= 0.60
		}
		if score < bestScore {
			bestScore = score
			bestPart = i
		}
	}

	if bestPart < 0 {
		return
	}

	// If root part 0 was matched, also excite its primary torso child if present
	targetPart := bestPart
	if targetPart == 0 {
		for i := 1; i < len(v.links.data); i++ {
			if v.links.data[i].parent == 0 && !v.data[i].detached {
				targetPart = i
				break
			}
		}
	}

	v.lastHitPart = targetPart

	// Propagate joint torque up the kinematic chain (hit limb -> parent -> grandparent)
	curr := targetPart
	attenuation := 1.0
	for depth := 0; depth < 3 && curr > 0 && curr < len(v.data); depth++ {
		j := v.data[curr]
		if j.detached {
			break
		}
		pivX, pivY, pivZ := v.originsA[curr].X, v.originsA[curr].Y, v.originsA[curr].Z
		lcX, lcY, lcZ := v.rotationsA[curr].RotatePoint(j.localCenterX, j.localCenterY, j.localCenterZ)

		// Blend impact lever arm with limb center-of-mass lever arm so direct hits always bend the joint backward
		rx := 0.45*(relPx-pivX) + 0.55*lcX
		ry := 0.45*(relPy-pivY) + 0.55*lcY
		rz := 0.45*(relPz-pivZ) + 0.55*lcZ
		rLen := math.Sqrt(rx*rx + ry*ry + rz*rz)
		if rLen < 1e-4 {
			rx, ry, rz = 0, 0, j.partRadius
			rLen = j.partRadius
		}

		// Unit torque direction τ = (r / ||r||) × dir
		tx := (ry*dirZ - rz*dirY) / rLen
		ty := (rz*dirX - rx*dirZ) / rLen
		tz := (rx*dirY - ry*dirX) / rLen
		if tx*tx+ty*ty+tz*tz < 0.04 {
			// Fallback orthogonal bend axis when shot is purely radial along the bone
			tx, ty, tz = -dirY, dirX, 0.30
		}

		// Transform torque into the joint's local coordinate system
		ltx, lty, ltz := v.rotationsA[curr].RotatePointInverse(tx, ty, tz)
		kick := 9.5 * normImpulse * attenuation

		const maxJointVel = 14.0
		j.velX = math.Max(-maxJointVel, math.Min(maxJointVel, j.velX+ltx*kick*j.dofScaleX))
		j.velY = math.Max(-maxJointVel, math.Min(maxJointVel, j.velY+lty*kick*j.dofScaleY))
		j.velZ = math.Max(-maxJointVel, math.Min(maxJointVel, j.velZ+ltz*kick*j.dofScaleZ))

		if j.canDetach && v.dismemberedCount < 3 {
			stress := 0.75 * normImpulse * attenuation
			j.damage += stress
			if j.damage >= j.detachThreshold {
				v.detachPart(curr, dirX, dirY, dirZ, fMag)
				break
			}
		}

		curr = v.links.data[curr].parent
		attenuation *= 0.60
	}
}

// Restore resets the joint origins and rotations at i using q.
func (v *Joints) Restore(i int, q *aabb.Quaternion) {
	v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z = 0, 0, v.baseOffsetZ
	v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z = 0, 0, v.baseOffsetZ
	v.rotationsA[i].SetFromQuaternion(q)
	v.rotationsB[i].SetFromQuaternion(q)
}

// Reset copies parent origins to joint i and updates its rotations, applying deflection when configured.
func (v *Joints) Reset(i int, pIdx int) {
	v.originsA[i] = v.originsA[pIdx]
	v.originsB[i] = v.originsB[pIdx]
	j := v.data[i]
	if j.hasDeflection {
		v.rotationsA[pIdx].MultiplyMatrix(j.axis, v.rotationsA[i])
		v.rotationsB[pIdx].MultiplyMatrix(j.axis, v.rotationsB[i])
	} else {
		*v.rotationsA[i] = *v.rotationsA[pIdx]
		*v.rotationsB[i] = *v.rotationsB[pIdx]
	}
}

// IsDetached reports whether the joint at index i is detached.
func (v *Joints) IsDetached(i int) bool {
	return v.data[i].detached
}
