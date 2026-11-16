package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

type CollisionT func(other IObject, energy float64)

type HookT func()

func noopHook() {}

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
	beforePrepare        HookT
	afterPrepare         HookT
	afterApply           HookT
}

// NewSolver creates a Solver for the given entity and object with movement and collision settings.
func NewSolver(cage *Cage, entity *rigid_body.Entity) *Solver {
	if cage == nil {
		panic("cage is nil in collisions.NewSolver")
	}
	if entity == nil {
		panic("entity is nil in collisions.NewSolver")
	}
	s := &Solver{
		cage:                 cage,
		entity:               entity,
		maxStep:              0,
		onElasticCollision:   nil,
		onInelasticCollision: nil,
		canSleep:             true,
		location:             nil,
		isActive:             true,
		beforePrepare:        noopHook,
		afterPrepare:         noopHook,
		afterApply:           noopHook,
	}
	cage.SetSolver(s)
	return s
}

// SetElasticCollision sets the callback invoked when an elastic collision is handled.
func (t *Solver) SetElasticCollision(c CollisionT) {
	t.onElasticCollision = c
}

// SetInelasticCollision sets the callback invoked for inelastic collisions.
func (t *Solver) SetInelasticCollision(c CollisionT) {
	t.onInelasticCollision = c
}

// SetPrepareHooks configures callbacks invoked immediately before and after StagePrepare.
func (t *Solver) SetPrepareHooks(before, after HookT) {
	if before == nil {
		before = noopHook
	}
	if after == nil {
		after = noopHook
	}
	t.beforePrepare = before
	t.afterPrepare = after
}

// SetApplyHook configures a callback invoked at the end of StageApply.
func (t *Solver) SetApplyHook(after HookT) {
	if after == nil {
		after = noopHook
	}
	t.afterApply = after
}

// GetCage returns the collision cage managed by this solver.
func (t *Solver) GetCage() *Cage {
	return t.cage
}

// GetEntity returns the rigid-body entity managed by this solver.
func (t *Solver) GetEntity() *rigid_body.Entity {
	return t.entity
}

// GetLocation returns the current static container location recorded for this solver.
func (t *Solver) GetLocation() IObject {
	return t.location
}

// SetLocation sets the location object for the solver.
func (t *Solver) SetLocation(location IObject) {
	t.location = location
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

// SetCanSleep sets whether the solver can sleep.
func (t *Solver) SetCanSleep(canSleep bool) {
	t.canSleep = canSleep
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
	t.beforePrepare()
	ent := t.entity
	ent.ClearActiveContact()
	ent.ClearPreSolve()
	ent.Update()
	t.cage.Rebuild(t.maxStep, session)
	t.afterPrepare()
}

// ApplySafeStepUp verifies vertical headroom for a step-up before keeping it so beveled walls under
// ceilings or overhangs do not launch the entity into ceiling clamps that kick it backward.
func ApplySafeStepUp(cage *Cage, stepZ float64, clampStatic func(*Cage)) {
	ent := cage.GetEntity()
	preX, preY, preZ := ent.GetCenter()
	preVX, preVY, preVZ := ent.GetVelocity()
	origCX, origCY, origCZ, origHasFloor := cage.GetStaticClampState()

	cage.RestoreStaticClampState(preX, preY, preZ, origHasFloor)
	ent.AddTo(0, 0, stepZ)
	clampStatic(cage)
	postX, postY, postZ := ent.GetCenter()

	dzAfter := postZ - preZ
	dXY := math.Hypot(postX-preX, postY-preY)
	if dzAfter < stepZ-0.25 || dXY > 2.0 {
		ent.AddTo(preX-postX, preY-postY, preZ-postZ)
		ent.SetV(preVX, preVY, preVZ)
		cage.RestoreStaticClampState(origCX, origCY, origCZ, origHasFloor)
		return
	}
	ent.SetV(preVX, preVY, preVZ)
	cage.RestoreStaticClampState(origCX, origCY, origCZ, origHasFloor)
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

	var lastElasticId, lastInelasticId uint64
	hasElasticId, hasInelasticId := false, false
	stepZ := 0.0
	for i := 0; i < t.cage.GetSlotsLen(); i++ {
		slot := t.cage.GetSlot(i)
		switch slot.GetImpactMode() {
		case ImpactStep:
			tz := t.cage.GetTZ()
			rz := t.cage.GetRadZ()
			_, _, pz := t.cage.GetPushOffset()
			if cand := slot.GetMaxZ() - (tz - rz + pz + stepZ); cand > 0 {
				stepZ += cand
			}
		case ImpactElastic:
			rCage := slot.GetRCage()
			rEnt := rCage.GetEntity()
			rId := rEnt.GetId()
			if !hasElasticId || lastElasticId != rId {
				bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ, bestEnergy := t.bestElasticImpact(i, rId, rEnt)
				t.entity.SetActiveContact(bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ)
				if t.onElasticCollision != nil {
					t.onElasticCollision(rCage.GetObject(), bestEnergy)
				}
				t.entity.ClearActiveContact()
				lastElasticId = rId
				hasElasticId = true
			}
		case ImpactInelastic:
			rParent := slot.GetRemoteFace().GetParent()
			if rParent != nil {
				rId := rParent.GetEntity().GetId()
				if !hasInelasticId || lastInelasticId != rId {
					bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ, bestEnergy := t.bestInelasticImpact(i, rId)
					t.entity.SetActiveContact(bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ)
					if t.onInelasticCollision != nil {
						t.onInelasticCollision(rParent, bestEnergy)
					}
					lastInelasticId = rId
					hasInelasticId = true
				}
			}
		}
	}
	if t.maxStep > 0 && stepZ > t.maxStep {
		stepZ = t.maxStep
	}
	t.cage.SetPendingStepZ(stepZ)
	t.afterApply()
}

// StageResolvePosition resolves remaining penetration for elastic contacts and static world geometry.
// It applies constrained positional corrections and clamps normal velocity when only one body can move.
func (t *Solver) StageResolvePosition() {
	slotsLen := t.cage.GetSlotsLen()
	if slotsLen == 0 {
		return
	}
	targetBaseZ := t.cage.GetTZ() - t.cage.GetRadZ()
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

// bestElasticImpact finds the elastic contact with the highest normal kinetic impact for the remote entity.
// It scans consecutive matching slots starting at start and returns the best normal, point, and energy.
func (t *Solver) bestElasticImpact(start int, rId uint64, rEnt *rigid_body.Entity) (bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ, bestEnergy float64) {
	first := t.cage.GetSlot(start)
	bestNX, bestNY, bestNZ = first.GetNormal()
	bestPX, bestPY, bestPZ = first.GetPoint()
	bestEnergy = t.entity.ComputeNormalKineticImpactAt(rEnt, bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ)
	for j := start + 1; j < t.cage.GetSlotsLen(); j++ {
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
	return bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ, bestEnergy
}

// bestInelasticImpact returns the inelastic contact with the greatest static normal kinetic impact.
// It scans consecutive slots from start for the same remote entity and stops at the first nonmatching slot.
func (t *Solver) bestInelasticImpact(start int, rId uint64) (bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ, bestEnergy float64) {
	first := t.cage.GetSlot(start)
	bestNX, bestNY, bestNZ = first.GetNormal()
	bestPX, bestPY, bestPZ = first.GetPoint()
	bestEnergy = t.entity.ComputeStaticNormalKineticImpactAt(bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ)
	for j := start + 1; j < t.cage.GetSlotsLen(); j++ {
		s2 := t.cage.GetSlot(j)
		if s2.GetImpactMode() != ImpactInelastic {
			break
		}
		p2 := s2.GetRemoteFace().GetParent()
		if p2 == nil || p2.GetEntity().GetId() != rId {
			break
		}
		nx2, ny2, nz2 := s2.GetNormal()
		px2, py2, pz2 := s2.GetPoint()
		if candE := t.entity.ComputeStaticNormalKineticImpactAt(nx2, ny2, nz2, px2, py2, pz2); candE > bestEnergy {
			bestEnergy = candE
			bestNX, bestNY, bestNZ = nx2, ny2, nz2
			bestPX, bestPY, bestPZ = px2, py2, pz2
		}
	}
	return bestNX, bestNY, bestNZ, bestPX, bestPY, bestPZ, bestEnergy
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
			if rCage.GetSolverIndex() >= 0 && rCage.HasEntryFor(slot.GetBucket().Opposite(), t.entity.GetId()) {
				return nil
			}
		}
		return rParentEnt
	}
	rFace := slot.GetRemoteFace()
	rParent := rFace.GetParent()
	return rParent.GetEntity()
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
