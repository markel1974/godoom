package collisions

import (
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

type CollisionT func(other IObject, energy float64)

// Solver manages physics simulation state for an object, including movement, collisions, and sleeping behavior.
type Solver struct {
	maxStep              float64
	cage                 *Cage
	entity               *rigid_body.Entity
	location             IObject
	isActive             bool
	canSleep             bool
	onElasticCollision   CollisionT
	onInelasticCollision CollisionT
}

// NewSolver creates a Solver for the given entity and object with movement and collision settings.
func NewSolver(cage *Cage, entity *rigid_body.Entity, maxStep float64, canSleep bool, onElasticCollision, onInelasticCollision CollisionT) *Solver {
	return &Solver{
		cage:                 cage, //NewCageSequence(object),
		entity:               entity,
		maxStep:              maxStep,
		onElasticCollision:   onElasticCollision,
		onInelasticCollision: onInelasticCollision,
		canSleep:             canSleep,
		location:             nil,
		isActive:             true,
	}
}

// IsActive reports whether the solver is currently active.
func (t *Solver) IsActive() bool {
	return t.isActive
}

// SetActive updates whether the solver is active.
func (t *Solver) SetActive(active bool) {
	t.isActive = active
}

// GetMaxStep returns the maximum step value used by the solver.
func (t *Solver) GetMaxStep() float64 {
	return t.maxStep
}

// SetMaxStep sets the maximum step value used by the solver.
func (t *Solver) SetMaxStep(maxStep float64) {
	t.maxStep = maxStep
}

// CanSleep reports whether the solver is allowed to enter the sleep state.
func (t *Solver) CanSleep() bool {
	return t.canSleep
}

// IsSleeping reports whether the solver's entity is currently sleeping.
func (t *Solver) IsSleeping() bool {
	return t.entity.IsSleeping()
}

// WakeUp wakes the solver's entity from its sleeping state.
func (t *Solver) WakeUp() {
	t.entity.WakeUp()
}

// EnterSleep puts the solver to sleep and clears pending displacement.
func (t *Solver) EnterSleep() {
	t.entity.SetSleeping(true)
	t.cage.ZeroDisplacement()
}

// UpdateSleepState updates the entity sleep state from sleep eligibility and cage push offset.
func (t *Solver) UpdateSleepState() bool {
	px, py, pz := t.cage.GetPushOffset()
	pushSq := px*px + py*py + pz*pz
	return t.entity.UpdateSleepState(t.CanSleep(), pushSq)
}

// SetSolverIndex records this solver's active index for the current solver tick.
func (t *Solver) SetSolverIndex(idx int) {
	t.cage.SetSolverIndex(idx)
}

// GetSolverIndex returns this solver's active index in the current solver tick, or -1 when inactive.
func (t *Solver) GetSolverIndex() int {
	return t.cage.GetSolverIndex()
}

// StagePrepare resets transient contact state, updates the entity, and rebuilds its cage for the session.
func (t *Solver) StagePrepare(session uint64) {
	ent := t.entity
	ent.ClearActiveContact()
	ent.ClearPreSolve()
	ent.Update()
	//t.angle = ent.GetYaw()
	//t.Compile(textures.GlobalTick())
	t.cage.Rebuild(t.maxStep, session)
}

// resolveSlotPartner returns the remote entity for a contact slot, or nil when the slot should not be solved.
func (t *Solver) resolveSlotPartner(slot *Entry) *rigid_body.Entity {
	if slot.GetImpactMode() == ImpactStep {
		return nil
	}
	if rCage := slot.GetRCage(); rCage != nil {
		rParentEnt := rCage.GetEntity()
		// Each active dynamic pair is solved once per solver pass by the canonical owner.
		if t.entity.GetId() > rParentEnt.GetId() {
			//rThing, ok := rCage.GetObject().(volumes.IThing)
			if t.IsActive() && rCage.HasEntryFor(slot.GetBucket().Opposite(), t.entity.GetId()) {
				return nil
			}
		}
		return rParentEnt
	}
	rFace := slot.GetRemoteFace()
	rParent := rFace.GetParent()
	return rParent.GetEntity()
}

// StageWarmStart reapplies cached contact impulses for valid collision slots to improve solver convergence.
func (t *Solver) StageWarmStart() {
	slotsLen := t.cage.GetSlotsLen()
	if slotsLen == 0 {
		return
	}

	//entity := t.GetContainerVolume().GetEntity()
	for i := 0; i < slotsLen; i++ {
		slot := t.cage.GetSlot(i)
		rParentEnt := t.resolveSlotPartner(slot)
		if rParentEnt == nil {
			continue
		}
		accNormal, accTx, accTy, accTz, _, _ := slot.GetSolverState()
		if *accNormal <= 0.0 {
			continue
		}
		nX, nY, nZ := slot.GetNormal()
		pX, pY, pZ := slot.GetPoint()
		t.entity.WarmStartImpact(rParentEnt, nX, nY, nZ, pX, pY, pZ, accNormal, accTx, accTy, accTz)
	}
}

// StageResolve resolves accumulated impact and friction impulses for all valid contact slots.
func (t *Solver) StageResolve() {
	slotsLen := t.cage.GetSlotsLen()
	if slotsLen == 0 {
		return
	}

	for i := 0; i < slotsLen; i++ {
		slot := t.cage.GetSlot(i)
		rParentEnt := t.resolveSlotPartner(slot)
		if rParentEnt == nil {
			continue
		}
		// Risoluzione Impulsi e Attrito (Fase Cinetica) con Proprietà Esplicite del Materiale di Contatto.
		nX, nY, nZ := slot.GetNormal()
		pX, pY, pZ := slot.GetPoint()
		accNormal, accTx, accTy, accTz, restTarget, hasRestTarget := slot.GetSolverState()
		t.entity.ResolveImpactAccumulated(rParentEnt, nX, nY, nZ, pX, pY, pZ, accNormal, accTx, accTy, accTz, restTarget, hasRestTarget)
	}
}

// StageApply applies the cage displacement, records the current location, and marks the cage as applied.
// StageApply processes impact slots to emit collision callbacks and compute any pending step-up adjustment.
func (t *Solver) StageApply() {
	if location := t.cage.GetVolume(); location != nil {
		t.location = location
	}
	// Spostiamo l'oggetto usando il vettore pre-urto. Questo porta AABB esattamente nel punto (tX, tY, tZ) dove la gabbia ha misurato la compenetrazione.
	dx, dy, dz := t.cage.GetDisplacement()
	t.entity.AddTo(dx, dy, dz)
	t.cage.SetApplied()

	var lastCollisionId uint64
	hasCollisionId := false
	stepZ := 0.0
	for i := 0; i < t.cage.GetSlotsLen(); i++ {
		slot := t.cage.GetSlot(i)
		switch slot.GetImpactMode() {
		case ImpactStep:
			_, _, tz := t.cage.GetT()
			_, _, rz := t.cage.GetRad()
			_, _, pz := t.cage.GetPushOffset()
			if cand := slot.GetMaxZ() - (tz - rz + pz + stepZ); cand > 0 {
				stepZ += cand
			}
		case ImpactElastic:
			rCage := slot.GetRCage()
			rEnt := rCage.GetEntity()
			rId := rEnt.GetId()
			if !hasCollisionId || lastCollisionId != rId {
				bestNX, bestNY, bestNZ := slot.GetNormal()
				bestPX, bestPY, bestPZ := slot.GetPoint()
				bestEnergy := t.entity.ComputeNormalKineticImpactAt(rEnt, bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ)
				for j := i + 1; j < t.cage.GetSlotsLen(); j++ {
					s2 := t.cage.GetSlot(j)
					if s2.GetImpactMode() != ImpactElastic || s2.GetRCage().GetEntity().GetId() != rId {
						break
					}
					nx2, ny2, nz2 := s2.GetNormal()
					px2, py2, pz2 := s2.GetPoint()
					if candE := t.entity.ComputeNormalKineticImpactAt(rEnt, nx2, ny2, nz2, px2, py2, pz2); candE > bestEnergy {
						bestEnergy = candE
						bestNX, bestNY, bestNZ = nx2, ny2, nz2
						bestPX, bestPY, bestPZ = px2, py2, pz2
					}
				}
				t.entity.SetActiveContact(bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ)
				t.onElasticCollision(rCage.GetObject(), bestEnergy)
				t.entity.ClearActiveContact()
				lastCollisionId = rId
				hasCollisionId = true
			}
		case ImpactInelastic:
			rParent := slot.GetRemoteFace().GetParent()
			if rParent != nil {
				nX, nY, nZ := slot.GetNormal()
				pX, pY, pZ := slot.GetPoint()
				t.entity.SetActiveContact(nX, nY, nZ, pX, pY, pZ)
				//TODO BESTENERY!
				t.onInelasticCollision(rParent, 0)
				t.entity.ClearActiveContact()
			}
		}
	}
	if t.maxStep > 0 && stepZ > t.maxStep {
		stepZ = t.maxStep
	}
	t.cage.SetPendingStepZ(stepZ)
}

// StageResolvePosition resolves remaining penetration for elastic contacts and static world geometry.
// It applies constrained positional corrections and clamps normal velocity when only one body can move.
func (t *Solver) StageResolvePosition() {
	slotsLen := t.cage.GetSlotsLen()
	if slotsLen == 0 {
		return
	}
	_, _, targetZ := t.cage.GetT()
	_, _, targetRadZ := t.cage.GetRad()
	targetBaseZ := targetZ - targetRadZ

	for i := 0; i < slotsLen; i++ {
		slot := t.cage.GetSlot(i)
		if slot.GetImpactMode() != ImpactElastic {
			continue
		}
		nX, nY, nZ := slot.GetNormal()
		rCage := slot.GetRCage()
		sPushX, sPushY, sPushZ := t.cage.GetPushOffset()
		rPushX, rPushY, rPushZ := rCage.GetPushOffset()
		alreadyResolved := (sPushX-rPushX)*nX + (sPushY-rPushY)*nY + (sPushZ-rPushZ)*nZ
		pen := clampUpwardPenetration(slot.GetPenetration(), alreadyResolved, nZ, slot.GetMaxZ()+rPushZ, targetBaseZ+sPushZ, sPushZ)
		if pen > 0 {
			otherMovable := slot.Penetrable(0) && !rCage.IsBlockedAlong(-nX, -nY, -nZ)
			selfMovable := t.cage.IsBucketPenetrable(slot.GetBucket().Opposite(), rCage.GetEntity().GetId()) && !t.cage.IsBlockedAlong(nX, nY, nZ)
			t.entity.ResolveElasticPenetrationConstrained(rCage.GetEntity(), rCage.HasFloor(), nX, nY, nZ, pen, selfMovable, otherMovable)
			if !otherMovable && selfMovable {
				t.entity.ClampNormalVelocity(nX, nY, nZ)
			} else if !selfMovable && otherMovable {
				rCage.GetEntity().ClampNormalVelocity(-nX, -nY, -nZ)
			}
		}
	}

	// Resolve immutable static walls, floors, and ceilings after elastic pushes to prevent sinking into world geometry.
	for i := 0; i < slotsLen; i++ {
		slot := t.cage.GetSlot(i)
		if slot.GetImpactMode() != ImpactInelastic {
			continue
		}
		nX, nY, nZ := slot.GetNormal()
		sPushX, sPushY, sPushZ := t.cage.GetPushOffset()
		alreadyResolved := sPushX*nX + sPushY*nY + sPushZ*nZ
		pen := clampUpwardPenetration(slot.GetPenetration(), alreadyResolved, nZ, slot.GetMaxZ(), targetBaseZ+sPushZ, sPushZ)
		if pen > 0 {
			t.entity.ResolveInelasticPenetration(nX, nY, nZ, pen)
		}
	}
}

// clampUpwardPenetration subtracts resolved penetration and limits upward correction to remaining vertical space.
// It returns zero when no upward space remains, otherwise the adjusted penetration.
func clampUpwardPenetration(pen, alreadyResolved, nZ, maxZ, currentBaseZ, sPushZ float64) float64 {
	pen -= alreadyResolved
	if pen > 0 && sPushZ > 0.0 && nZ > 0.70 {
		remZ := maxZ - currentBaseZ
		if remZ <= 0.0 {
			return 0.0
		}
		if maxPen := remZ / nZ; pen > maxPen {
			return maxPen
		}
	}
	return pen
}
