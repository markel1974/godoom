package volumes

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

type IElements interface {
	FreezeCurrentFrame()
}

// Joint defines a 3D vertex joint with angular motion, damping, damage, and detachment state.
type Joint struct {
	localCenterX    float64
	localCenterY    float64
	localCenterZ    float64
	partRadius      float64
	dofScaleX       float64
	dofScaleY       float64
	dofScaleZ       float64
	maxAngleAlive   float64
	maxAngleRagdoll float64
	stiffness       float64
	damping         float64
	canDetach       bool
	detachThreshold float64

	angX           float64
	angY           float64
	angZ           float64
	velX           float64
	velY           float64
	velZ           float64
	ragdollTargetX float64
	ragdollTargetY float64
	ragdollTargetZ float64
	damage         float64
	hasDeflection  bool

	detached   bool
	detSettled bool
	detWorldX  float64
	detWorldY  float64
	detWorldZ  float64
	detVx      float64
	detVy      float64
	detVz      float64
	detWx      float64
	detWy      float64
	detWz      float64
	detFloorZ  float64

	quat      *physics.Quaternion
	detQuat   *physics.Quaternion
	axis      *physics.TagAxis
	combinedA *physics.TagAxis
	combinedB *physics.TagAxis
}

// NewJoint creates a Joint with identity orientation quaternions and axes.
func NewJoint() *Joint {
	return &Joint{
		quat:      physics.NewQuaternionIdentity(),
		detQuat:   physics.NewQuaternionIdentity(),
		axis:      physics.NewTagAxisIdentity(),
		combinedA: physics.NewTagAxisIdentity(),
		combinedB: physics.NewTagAxisIdentity(),
	}
}

// ActivateRagdoll sets clamped ragdoll targets from local motion inputs and applies angular velocity impulses.
// sign biases the Y-axis target to vary the collapse direction.
func (j *Joint) ActivateRagdoll(lx, ly, lz float64, sign float64) {
	j.ragdollTargetX = math.Max(-j.maxAngleRagdoll*0.85, math.Min(j.maxAngleRagdoll*0.85, lx*j.dofScaleX*0.95))
	j.ragdollTargetY = math.Max(-j.maxAngleRagdoll*0.85, math.Min(j.maxAngleRagdoll*0.85, (ly+0.28*sign)*j.dofScaleY*0.95))
	j.ragdollTargetZ = math.Max(-j.maxAngleRagdoll*0.70, math.Min(j.maxAngleRagdoll*0.70, lz*j.dofScaleZ*0.80))

	j.velX += j.ragdollTargetX * 5.0
	j.velY += j.ragdollTargetY * 5.0
	j.velZ += j.ragdollTargetZ * 4.0
}

// Detach marks the joint as detached and initializes its world position, floor height, launch velocity, and tumble.
func (j *Joint) Detach(halfD float64, cx, cy, cz float64, dirX, dirY, dirZ, fMag float64, orX, orY, orZ float64, sideSign float64) {
	if j.detached || !j.canDetach {
		return
	}
	j.detached = true
	j.detSettled = false
	j.detWorldX = cx + orX
	j.detWorldY = cy + orY
	j.detWorldZ = cz + orZ
	j.detFloorZ = (cz - halfD) + math.Min(j.partRadius*0.30, 5.0)
	if j.detWorldZ < j.detFloorZ {
		j.detWorldZ = j.detFloorZ
	}
	// TODO Ballistic launch velocity along impact ray + upward pop + slight lateral scatter
	launchSpeed := 125.0 + math.Min(fMag/500000.0, 2.5)*70.0
	j.detVx = dirX*launchSpeed - dirY*(25.0*sideSign)
	j.detVy = dirY*launchSpeed + dirX*(25.0*sideSign)
	j.detVz = math.Max(dirZ*launchSpeed+105.0, 75.0)
	// 3D tumbling angular velocity
	j.detWx = -dirY*8.5 + sideSign*3.5
	j.detWy = dirX*8.5 - 3.0
	j.detWz = sideSign * 7.0
}

// Settle advances detached joint motion by dt, applying gravity, damping, floor bounce, friction, and rest detection.
func (j *Joint) Settle(dt float64) {
	const limbGravity = 560.0
	j.detVz -= limbGravity * dt

	airDamp := math.Pow(0.65, dt)
	j.detVx *= airDamp
	j.detVy *= airDamp

	j.detWorldX += j.detVx * dt
	j.detWorldY += j.detVy * dt
	j.detWorldZ += j.detVz * dt

	if j.detWorldZ <= j.detFloorZ {
		j.detWorldZ = j.detFloorZ
		if j.detVz < -28.0 {
			j.detVz = -j.detVz * 0.34
			j.detWx += -j.detVy * 0.04
			j.detWy += j.detVx * 0.04
		} else {
			j.detVz = 0.0
		}
		groundFriction := math.Pow(0.015, dt)
		j.detVx *= groundFriction
		j.detVy *= groundFriction

		angFriction := math.Pow(0.025, dt)
		j.detWx *= angFriction
		j.detWy *= angFriction
		j.detWz *= angFriction

		if j.detVz == 0.0 && (j.detVx*j.detVx+j.detVy*j.detVy) < 1.0 &&
			(j.detWx*j.detWx+j.detWy*j.detWy+j.detWz*j.detWz) < 0.02 {
			j.detVx, j.detVy, j.detVz = 0, 0, 0
			j.detWx, j.detWy, j.detWz = 0, 0, 0
			j.detSettled = true
		}
	}
	j.detQuat.IntegrateAngular(j.detWx, j.detWy, j.detWz, dt)
}

// Connect advances the joint spring-damper state and cone limits, returning true while motion remains active.
func (j *Joint) Connect(isRagdoll bool, dt float64) bool {
	// Connected articulated joint (PD spring-damper + cone joint limits)
	ks := j.stiffness
	kd := j.damping
	maxAng := j.maxAngleAlive
	var targetX, targetY, targetZ float64
	active := false

	if isRagdoll {
		ks *= 0.12
		kd *= 0.55
		maxAng = j.maxAngleRagdoll
		targetX, targetY, targetZ = j.ragdollTargetX, j.ragdollTargetY, j.ragdollTargetZ
	} else if j.damage > 0 {
		j.damage = math.Max(0.0, j.damage-0.10*dt)
	}

	alphaX := -ks*(j.angX-targetX) - kd*j.velX
	alphaY := -ks*(j.angY-targetY) - kd*j.velY
	alphaZ := -ks*(j.angZ-targetZ) - kd*j.velZ

	j.velX += alphaX * dt
	j.velY += alphaY * dt
	j.velZ += alphaZ * dt

	j.angX += j.velX * dt
	j.angY += j.velY * dt
	j.angZ += j.velZ * dt

	angMagSq := j.angX*j.angX + j.angY*j.angY + j.angZ*j.angZ
	if maxAng > 0.0 && angMagSq > maxAng*maxAng {
		angMag := math.Sqrt(angMagSq)
		nx, ny, nz := j.angX/angMag, j.angY/angMag, j.angZ/angMag
		j.angX, j.angY, j.angZ = nx*maxAng, ny*maxAng, nz*maxAng
		angMagSq = maxAng * maxAng
		vDotN := j.velX*nx + j.velY*ny + j.velZ*nz
		if vDotN > 0.0 {
			j.velX -= 1.30 * vDotN * nx
			j.velY -= 1.30 * vDotN * ny
			j.velZ -= 1.30 * vDotN * nz
		}
	}

	diffX := j.angX - targetX
	diffY := j.angY - targetY
	diffZ := j.angZ - targetZ
	diffSq := diffX*diffX + diffY*diffY + diffZ*diffZ
	velMagSq := j.velX*j.velX + j.velY*j.velY + j.velZ*j.velZ

	if diffSq > 1e-6 || velMagSq > 1e-5 {
		active = true
	}

	if angMagSq > 1e-6 {
		angMag := math.Sqrt(angMagSq)
		j.quat.SetAxisAngle(j.angX/angMag, j.angY/angMag, j.angZ/angMag, angMag)
		j.axis.SetFromQuaternion(j.quat)
		j.hasDeflection = true
	} else {
		j.axis.Reset()
		j.hasDeflection = false
	}

	if diffSq <= 1e-6 && velMagSq <= 1e-5 {
		j.angX, j.angY, j.angZ = targetX, targetY, targetZ
		j.velX, j.velY, j.velZ = 0, 0, 0
	}

	return active
}

// Joints manages joint physics, ragdoll state, detachment, and transforms for linked 3D model parts.
type Joints struct {
	links             *Links
	data              []*Joint
	dismemberedCount  int
	hasActiveDynamics bool
	lastHitDirX       float64
	lastHitDirY       float64
	lastHitDirZ       float64

	fallAxisX float64
	fallAxisY float64
	fallAngle float64
	fallVel   float64

	ragdollCollapse bool

	rootRollAng      float64
	rootRollVel      float64
	rootBounceZ      float64
	rootBounceVz     float64
	rootDropZ        float64
	hasRootTransform bool

	rotationsA []*physics.TagAxis
	rotationsB []*physics.TagAxis

	lastHitPart  int
	lastHitForce float64

	rootQuat         *physics.Quaternion
	rootFallQuat     *physics.Quaternion
	rootRollQuat     *physics.Quaternion
	rootCombinedQuat *physics.Quaternion

	isRagdoll bool

	lastTick uint64

	originsA []geometry.XYZ
	originsB []geometry.XYZ
	numParts int

	elems []IElements

	ent *physics.Entity

	//elements *Vertices3DElements
}

// NewJoints creates a Joints instance with default state and identity root quaternions.
func NewJoints() *Joints {
	pj := &Joints{
		fallAxisX:        1.0,
		lastHitPart:      -1,
		rootQuat:         physics.NewQuaternionIdentity(),
		rootFallQuat:     physics.NewQuaternionIdentity(),
		rootRollQuat:     physics.NewQuaternionIdentity(),
		rootCombinedQuat: physics.NewQuaternionIdentity(),
	}
	return pj
}

// Setup initializes joint state, rotations, origins, and per-part motion constraints from elements and links.
func (v *Joints) Setup(cfg *config.Model3D, ent *physics.Entity, elements *Vertices3DElements, links *Links) error {
	v.ent = ent
	v.numParts = elements.Len()
	v.links = links
	v.elems = make([]IElements, v.numParts)
	v.rotationsA = make([]*physics.TagAxis, v.numParts)
	v.rotationsB = make([]*physics.TagAxis, v.numParts)
	v.data = make([]*Joint, v.numParts)

	for i := 0; i < v.numParts; i++ {
		v.elems[i] = elements.GetEntry(i)
		v.rotationsA[i] = physics.NewTagAxisIdentity()
		v.rotationsB[i] = physics.NewTagAxisIdentity()
		v.data[i] = NewJoint()
	}

	v.originsA = make([]geometry.XYZ, v.numParts)
	v.originsB = make([]geometry.XYZ, v.numParts)

	for i := range v.data {
		j := v.data[i]
		j.quat.SetIdentity()
		j.detQuat.SetIdentity()
		j.axis.Reset()
		j.combinedA.Reset()
		j.combinedB.Reset()
		//TODO FROM CONFIG
		j.partRadius = 8.0

		if element := elements.GetEntry(i); element != nil {
			if vol := element.volumes[0]; vol != nil {
				fc := vol.GetFaceCount()
				if fc > 0 {
					minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
					maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
					for fIdx := 0; fIdx < fc; fIdx++ {
						f := vol.GetFace(fIdx)
						minX, minY, minZ, maxX, maxY, maxZ = f.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
					}
					j.localCenterX = (minX + maxX) * 0.5
					j.localCenterY = (minY + maxY) * 0.5
					j.localCenterZ = (minZ + maxZ) * 0.5
					dx, dy, dz := maxX-minX, maxY-minY, maxZ-minZ
					r := math.Max(dx, math.Max(dy, dz)) * 0.5
					j.partRadius = math.Max(4.0, math.Min(48.0, r))
				}
			}
		}

		if i == 0 || v.links.GetParent(i) < 0 {
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 0, 0, 0
			j.maxAngleAlive = 0
			j.maxAngleRagdoll = 0
			j.stiffness = 80.0
			j.damping = 10.0
			j.canDetach = false
		} else {
			tag := v.links.GetTag(i)
			z := cfg.Joints.Get(tag)
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = z.DofScale.X, z.DofScale.Y, z.DofScale.Z
			j.maxAngleAlive = z.MaxAngleAlive
			j.maxAngleRagdoll = z.MaxAngleRagdoll
			j.stiffness = z.Stiffness
			j.damping = z.Damping
			j.detachThreshold = z.DetachThreshold
			j.canDetach = z.CanDetach
		}
	}

	return nil
}

// IsRagdoll reports whether the joints are currently in ragdoll mode.
func (v *Joints) IsRagdoll() bool {
	return v.isRagdoll
}

// SetRagdollCollapse marks the joint system to collapse when ragdoll mode is activated.
func (v *Joints) SetRagdollCollapse() {
	v.ragdollCollapse = true
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
func (v *Joints) GetRotationA(i int) *physics.TagAxis {
	return v.rotationsA[i]
}

// GetRotationB returns the secondary rotation axis for the joint at index i.
func (v *Joints) GetRotationB(i int) *physics.TagAxis {
	return v.rotationsB[i]
}

// CompileDetached updates detached joint origins relative to the center and applies its stored detached rotation.
func (v *Joints) CompileDetached(i int) {
	cx, cy, cz := v.ent.GetCenter()
	j := v.data[i]
	v.originsA[i].X = j.detWorldX - cx
	v.originsA[i].Y = j.detWorldY - cy
	v.originsA[i].Z = j.detWorldZ - cz
	v.originsB[i] = v.originsA[i]
	v.rotationsA[i].SetFromQuaternion(j.detQuat)
	v.rotationsB[i].SetFromQuaternion(j.detQuat)
}

// CompileNormale derives joint origins and rotations from parent transforms and optional tag axes.
// Missing tags inherit parent transforms; configured deflection is applied when present.
func (v *Joints) CompileNormale(i int, pIdx int, tagAxisA, tagAxisB *physics.TagAxis) {
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
		fmt.Printf("[warning] Vertices3D: tag %q not found in frame A of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links.data[i].tag, pIdx, i)
		v.originsA[i] = v.originsA[pIdx]
		if j.hasDeflection {
			v.rotationsA[pIdx].MultiplyMatrix(j.axis, v.rotationsA[i])
		} else {
			*v.rotationsA[i] = *v.rotationsA[pIdx]
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
		fmt.Printf("[warning] Vertices3D: tag %q not found in frame B of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links.data[i].tag, pIdx, i)
		v.originsB[i] = v.originsB[pIdx]
		if j.hasDeflection {
			v.rotationsB[pIdx].MultiplyMatrix(j.axis, v.rotationsB[i])
		} else {
			*v.rotationsB[i] = *v.rotationsB[pIdx]
		}
	}
}

// detachPart marks a detachable joint as detached and initializes its frozen frame and ballistic dynamics.
func (v *Joints) detachPart(idx int, cx, cy, cz, dirX, dirY, dirZ, fMag float64) {
	if idx <= 0 || idx >= len(v.data) || v.dismemberedCount >= 3 {
		return
	}
	j := v.data[idx]
	if j.detached || !j.canDetach {
		return
	}
	v.dismemberedCount++
	v.hasActiveDynamics = true
	for i := 0; i < v.numParts; i++ {
		if i == idx || v.IsPartDetached(i) {
			v.elems[i].FreezeCurrentFrame()
		}
	}
	v.rotationsA[idx].ToQuaternion(j.detQuat)
	orX, orY, orZ := v.originsA[idx].X, v.originsA[idx].Y, v.originsA[idx].Z
	sideSign := 1.0
	if idx&1 == 1 {
		sideSign = -1.0
	}
	_, _, halfD := v.ent.GetSizeCenter()
	j.Detach(halfD, cx, cy, cz, dirX, dirY, dirZ, fMag, orX, orY, orZ, sideSign)
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

// ActivateRagdoll enables ragdoll dynamics, seeds fall motion from the last hit, and may detach a damaged part.
func (v *Joints) ActivateRagdoll() {
	v.isRagdoll = true
	v.hasActiveDynamics = true

	dx, dy, dz := v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ
	if dx*dx+dy*dy+dz*dz < 1e-6 {
		dx = 1.0
	}

	horizLen := math.Hypot(dx, dy)
	if horizLen > 1e-4 {
		v.fallAxisX = -dy / horizLen
		v.fallAxisY = dx / horizLen
	} else {
		v.fallAxisX = 1.0
		v.fallAxisY = 0.0
	}

	if v.ragdollCollapse && v.fallAngle == 0 {
		v.fallAngle = 0.08
		v.fallVel = 3.8
	}
	v.rootRollVel += 2.6
	v.rootBounceVz = 48.0

	// If the killing blow hit a detachable extremity with sufficient accumulated stress, sever it on death
	if v.dismemberedCount < 3 && v.lastHitPart > 0 && v.lastHitPart < len(v.data) {
		hj := v.data[v.lastHitPart]
		if hj.canDetach && !hj.detached && hj.damage >= hj.detachThreshold*0.45 {
			cx, cy, cz := v.ent.GetCenter()
			v.detachPart(v.lastHitPart, cx, cy, cz, dx, dy, dz, v.lastHitForce)
		}
	}

	tx := dy*0.55 - dz*0.25
	ty := -dx*0.55 + dz*0.40
	tz := (dx - dy) * 0.30

	for i := 1; i < len(v.data); i++ {
		j := v.data[i]
		if j.detached {
			continue
		}
		// Compute gravitational/impact slump target in the joint's local frame
		lx, ly, lz := v.rotationsA[i].RotatePointInverse(tx, ty, tz)
		sign := 1.0
		if i&1 == 1 {
			sign = -0.90
		}
		j.ActivateRagdoll(lx, ly, lz, sign)
	}
}

// StepJoints advances joint and ragdoll dynamics to the given simulation tick.
// It ignores zero, repeated, or older ticks and clamps large time steps for stability.
func (v *Joints) StepJoints(tick uint64) {
	if tick == 0 {
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
	if !v.hasActiveDynamics && !v.isRagdoll {
		return
	}
	if dt > 1.0/30.0 {
		dt = 1.0 / 30.0
	}

	anyActive := false

	// Root ragdoll collapse & post-mortem impact jolt integration
	if v.isRagdoll {
		rootActive := false
		if v.ragdollCollapse {
			const maxFallAngle = 1.48 // ~85 degrees onto floor
			if v.fallAngle < maxFallAngle || math.Abs(v.fallVel) > 0.02 {
				rootActive = true
				grav := 18.0 * math.Sin(math.Max(v.fallAngle, 0.15))
				v.fallVel += grav * dt
				v.fallVel *= math.Pow(0.45, dt)
				v.fallAngle += v.fallVel * dt
				if v.fallAngle >= maxFallAngle {
					v.fallAngle = maxFallAngle
					if v.fallVel > 0.45 {
						v.fallVel = -v.fallVel * 0.25
					} else {
						v.fallVel = 0.0
					}
				} else if v.fallAngle < 0.0 {
					v.fallAngle = 0.0
					v.fallVel = 0.0
				}
			}
		}

		// Secondary roll/jolt oscillation when corpse is shot
		if math.Abs(v.rootRollAng) > 1e-4 || math.Abs(v.rootRollVel) > 1e-4 {
			rootActive = true
			alphaRoll := -32.0*v.rootRollAng - 5.5*v.rootRollVel
			v.rootRollVel += alphaRoll * dt
			v.rootRollAng += v.rootRollVel * dt
			if math.Abs(v.rootRollAng) > 0.55 {
				v.rootRollAng = math.Copysign(0.55, v.rootRollAng)
				v.rootRollVel *= -0.35
			}
			if math.Abs(v.rootRollAng) < 1e-3 && math.Abs(v.rootRollVel) < 1e-3 {
				v.rootRollAng = 0
				v.rootRollVel = 0
			}
		}

		// Vertical pop/bounce when corpse is hit
		if v.rootBounceZ > 0 || math.Abs(v.rootBounceVz) > 1e-3 {
			rootActive = true
			v.rootBounceVz -= 480.0 * dt
			v.rootBounceZ += v.rootBounceVz * dt
			if v.rootBounceZ <= 0 {
				v.rootBounceZ = 0
				if v.rootBounceVz < -22.0 {
					v.rootBounceVz = -v.rootBounceVz * 0.28
				} else {
					v.rootBounceVz = 0
				}
			}
		}

		collapseDrop := 0.0
		if v.ragdollCollapse {
			_, _, halfD := v.ent.GetSizeCenter()
			collapseDrop = -math.Max(halfD-6.0, 0.0) * math.Sin(v.fallAngle)
		}
		v.rootDropZ = collapseDrop + v.rootBounceZ

		if v.fallAngle > 1e-4 || math.Abs(v.rootRollAng) > 1e-4 {
			v.rootFallQuat.SetAxisAngle(v.fallAxisX, v.fallAxisY, 0, v.fallAngle)
			if math.Abs(v.rootRollAng) > 1e-4 {
				v.rootRollQuat.SetAxisAngle(-v.fallAxisY, v.fallAxisX, 0.35, v.rootRollAng)
				v.rootRollQuat.Multiplies(v.rootFallQuat, v.rootQuat)
			} else {
				v.rootQuat.CopyFrom(v.rootFallQuat)
			}
			v.hasRootTransform = true
		}

		if rootActive {
			anyActive = true
		}
	}

	// Detached limbs and connected 5+ DoF joint spring-dampers
	for i := 1; i < len(v.data); i++ {
		j := v.data[i]
		if j.detached {
			if !j.detSettled {
				anyActive = true
				j.Settle(dt)
			}
			continue
		}
		if j.Connect(v.isRagdoll, dt) {
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
	v.hasActiveDynamics = true

	// Normalize force magnitude to angular joint impulse (calibrated to weaponForce = 500,000)
	normImpulse := math.Min(fMag/500000.0, 1.6)
	if normImpulse < 0.45 {
		normImpulse = 0.45
	}

	// Always jolt root corpse when already in ragdoll state (even for single-mesh or torso hits)
	if v.isRagdoll {
		if v.ragdollCollapse && v.fallAngle > 1.15 {
			// Make lying corpse pop/shudder on impact
			v.fallVel -= 2.4 * normImpulse
		} else {
			v.fallVel += 2.8 * normImpulse
		}
		crossZ := dirX*v.fallAxisY - dirY*v.fallAxisX
		if math.Abs(crossZ) < 0.25 {
			crossZ = 0.55
		}
		v.rootRollVel += crossZ * 5.5 * normImpulse
		v.rootBounceVz = math.Max(v.rootBounceVz, 60.0*normImpulse)
	}

	if v.numParts <= 1 {
		return
	}

	cx, cy, cz := v.ent.GetCenter()
	relPx, relPy, relPz := px-cx, py-cy, pz-cz

	// Locate the closest non-detached articulated part to the impact ray/point
	bestPart := -1
	bestScore := math.MaxFloat64
	bestDetachPart := -1
	bestDetachScore := math.MaxFloat64

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
		if i > 0 && j.canDetach && !j.detached && score < bestDetachScore {
			bestDetachScore = score
			bestDetachPart = i
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
		if v.isRagdoll {
			kick *= 1.45
		}

		const maxJointVel = 14.0
		j.velX = math.Max(-maxJointVel, math.Min(maxJointVel, j.velX+ltx*kick*j.dofScaleX))
		j.velY = math.Max(-maxJointVel, math.Min(maxJointVel, j.velY+lty*kick*j.dofScaleY))
		j.velZ = math.Max(-maxJointVel, math.Min(maxJointVel, j.velZ+ltz*kick*j.dofScaleZ))

		if j.canDetach && v.dismemberedCount < 3 {
			stress := 0.75 * normImpulse * attenuation
			if v.isRagdoll {
				stress *= 1.65
			}
			j.damage += stress
			if j.damage >= j.detachThreshold {
				v.detachPart(curr, cx, cy, cz, dirX, dirY, dirZ, fMag)
				break
			}
		}

		curr = v.links.data[curr].parent
		attenuation *= 0.60
	}

	// When shooting a ragdoll corpse on the torso, also whip all connected limbs and stress the nearest detachable limb
	if v.isRagdoll {
		for i := 1; i < len(v.data); i++ {
			j := v.data[i]
			if i == targetPart || j.detached {
				continue
			}
			sign := 1.0
			if i&1 == 1 {
				sign = -1.0
			}
			j.velX += (dirY * 3.2 * normImpulse * sign) * j.dofScaleX
			j.velY += (-dirX * 3.2 * normImpulse) * j.dofScaleY
			j.velZ += (2.2 * normImpulse * sign) * j.dofScaleZ
		}
		if bestDetachPart > 0 && bestDetachPart != targetPart && bestDetachScore < 3.2 && v.dismemberedCount < 3 {
			dj := v.data[bestDetachPart]
			dj.damage += 0.85 * normImpulse
			if dj.damage >= dj.detachThreshold {
				v.detachPart(bestDetachPart, cx, cy, cz, dirX, dirY, dirZ, fMag)
			}
		}
	}
}

// Restore resets the joint origins and rotations at i using q and any active root transform.
func (v *Joints) Restore(i int, q *physics.Quaternion) {
	v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z = 0, 0, v.rootDropZ
	v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z = 0, 0, v.rootDropZ

	if v.hasRootTransform {
		v.rootQuat.Multiplies(q, v.rootCombinedQuat)
		v.rotationsA[i].SetFromQuaternion(v.rootCombinedQuat)
		v.rotationsB[i].SetFromQuaternion(v.rootCombinedQuat)
	} else {
		v.rotationsA[i].SetFromQuaternion(q)
		v.rotationsB[i].SetFromQuaternion(q)
	}
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
