package collisions

import "runtime"

const parallelAcquireMinBodies = 8

type WakeRunningFn func(neighbor *Solver)

type QueryContactsFn func(cage *Cage, wake WakeRunningFn)

type ClampStaticFn func(cage *Cage)

// Coordinator manages solver lifecycles, wake queues, islands, and joint constraints during simulation.
type Coordinator struct {
	solverIterations  int
	session           uint64
	staged            []*Solver
	stagedLen         int
	running           []*Solver
	runningLen        int
	removables        []*Solver
	removablesLen     int
	wakeQueue         []*Solver
	islandParent      []int
	islandSleep       []bool
	jointSupported    []bool
	joints            []*JointConstraint
	jointsLen         int
	wakeRunningFn     WakeRunningFn
	queryContactsFn   QueryContactsFn
	clampStaticFn     ClampStaticFn
	commonCap         int
	parallelAcquire   bool
	parallelMinBodies int
	workerWakes       [][]*Solver
	workerWakeFns     []WakeRunningFn
	wakeBatch         []*Solver
	mergeOrder        []int
	clampQueue        []int
	waveStart         int
	prepareRunner     func(workerID, start, end int)
	acquireRunner     func(workerID, start, end int)
	clampRunner       func(workerID, start, end int)
}

// NewCoordinator creates a Coordinator with default iterations and capacity when nonpositive values are provided.
func NewCoordinator(solverIterations, initialCap int, queryContactsFn QueryContactsFn, clampStaticFn ClampStaticFn) *Coordinator {
	if solverIterations <= 0 {
		solverIterations = 10
	}
	commonCap := initialCap
	if commonCap <= 0 {
		commonCap = 1024
	}
	c := &Coordinator{
		commonCap:         commonCap,
		solverIterations:  solverIterations,
		staged:            make([]*Solver, commonCap),
		running:           make([]*Solver, commonCap),
		removables:        make([]*Solver, commonCap),
		wakeQueue:         make([]*Solver, commonCap),
		islandParent:      make([]int, commonCap),
		islandSleep:       make([]bool, commonCap),
		jointSupported:    make([]bool, commonCap),
		joints:            make([]*JointConstraint, 0, commonCap),
		queryContactsFn:   queryContactsFn,
		clampStaticFn:     clampStaticFn,
		parallelAcquire:   true,
		parallelMinBodies: parallelAcquireMinBodies,
		wakeBatch:         make([]*Solver, 0, commonCap),
		mergeOrder:        make([]int, 0, commonCap),
		clampQueue:        make([]int, 0, commonCap),
	}
	c.wakeRunningFn = c.wakeRunningIsland
	c.prepareRunner = func(_ int, start, end int) {
		session := c.session
		for idx := start; idx < end; idx++ {
			c.running[idx].StagePreparePhysics(session)
		}
	}
	c.acquireRunner = func(workerID, start, end int) {
		wakeFn := c.workerWakeFns[workerID]
		base := c.waveStart
		for idx := base + start; idx < base+end; idx++ {
			cage := c.running[idx].cage
			c.queryContactsFn(cage, wakeFn)
			cage.SortPendingDynamic()
		}
	}
	c.clampRunner = func(_ int, start, end int) {
		for i := start; i < end; i++ {
			cage := c.running[c.clampQueue[i]].cage
			c.clampStaticFn(cage)
			cage.CommitStaticClamp()
		}
	}
	c.ensureWorkers(sharedWorkerPool.WorkerCount())
	return c
}

// SetParallelAcquire enables or disables multi-goroutine wavefront contact acquisition.
func (c *Coordinator) SetParallelAcquire(enabled bool) {
	c.parallelAcquire = enabled
}

// ensureWorkers initializes reusable per-worker wake queues and callbacks without per-step allocations.
func (c *Coordinator) ensureWorkers(workers int) {
	if workers <= len(c.workerWakeFns) {
		return
	}
	oldLen := len(c.workerWakeFns)
	wakes := make([][]*Solver, workers)
	copy(wakes, c.workerWakes)
	fns := make([]WakeRunningFn, workers)
	copy(fns, c.workerWakeFns)
	for w := oldLen; w < workers; w++ {
		workerID := w
		wakes[workerID] = make([]*Solver, 0, 16)
		fns[workerID] = func(neighbor *Solver) {
			c.workerWakes[workerID] = append(c.workerWakes[workerID], neighbor)
		}
	}
	c.workerWakes = wakes
	c.workerWakeFns = fns
}

// AddJoint registers a joint constraint with the coordinator, ignoring nil constraints.
func (c *Coordinator) AddJoint(j *JointConstraint) {
	if j == nil {
		return
	}
	if c.jointsLen < len(c.joints) {
		c.joints[c.jointsLen] = j
	} else {
		c.joints = append(c.joints, j)
	}
	c.jointsLen++
}

// EnsureCapacity grows coordinator buffers to hold at least minCap solvers while preserving existing entries.
func (c *Coordinator) EnsureCapacity(minCap int) {
	if minCap <= c.commonCap {
		return
	}
	c.commonCap = minCap * 2
	growSolvers := func(src []*Solver) []*Solver {
		dst := make([]*Solver, c.commonCap)
		copy(dst, src)
		return dst
	}
	c.staged = growSolvers(c.staged)
	c.running = growSolvers(c.running)
	c.removables = growSolvers(c.removables)
	c.wakeQueue = growSolvers(c.wakeQueue)

	newParents := make([]int, c.commonCap)
	copy(newParents, c.islandParent)
	c.islandParent = newParents

	newSleep := make([]bool, c.commonCap)
	copy(newSleep, c.islandSleep)
	c.islandSleep = newSleep

	newJointSup := make([]bool, c.commonCap)
	copy(newJointSup, c.jointSupported)
	c.jointSupported = newJointSup

	if cap(c.clampQueue) < c.commonCap {
		c.clampQueue = make([]int, 0, c.commonCap)
	}
}

// Reset clears the staged solver list.
func (c *Coordinator) Reset() {
	c.stagedLen = 0
}

// Stage adds s to the staged solver list, growing storage when needed.
func (c *Coordinator) Stage(s *Solver) {
	if c.stagedLen >= len(c.staged) {
		c.EnsureCapacity(c.stagedLen + 1)
	}
	c.staged[c.stagedLen] = s
	c.stagedLen++
}

// StagedLen returns the number of solvers currently staged.
func (c *Coordinator) StagedLen() int {
	return c.stagedLen
}

func (c *Coordinator) RunningLen() int {
	return c.runningLen
}

func (c *Coordinator) RunningAt(idx int) *Solver {
	return c.running[idx]
}

// RemovablesLen returns the current solvers marked for removal.
func (c *Coordinator) RemovablesLen() int {
	return c.removablesLen
}

// RemovablesAt returns the current solvers marked for removal.
func (c *Coordinator) RemovablesAt(idx int) *Solver {
	return c.removables[idx]
}

// ClearRemovables clears the tracked removable solvers list.
func (c *Coordinator) ClearRemovables() {
	c.removablesLen = 0
}

// wakeRunningIsland wakes the island containing neighbor and keeps it in the running set.
func (c *Coordinator) wakeRunningIsland(neighbor *Solver) {
	c.wakeIsland(neighbor, true)
}

// wakeIsland wakes root and sleeping active solvers connected through contacts or joints.
// If addToRunning is true, it prepares woken solvers and appends them to the running set.
func (c *Coordinator) wakeIsland(root *Solver, addToRunning bool) {
	qLen := 0
	if root.IsSleeping() {
		root.WakeUp()
	}
	if qLen < len(c.wakeQueue) {
		c.wakeQueue[qLen] = root
		qLen++
	}
	for head := 0; head < qLen; head++ {
		curr := c.wakeQueue[head]
		cage := curr.cage
		slotsLen := cage.GetSlotsLen()
		dx, dy, dz := cage.GetNetDisplacement()
		dispSq := dx*dx + dy*dy + dz*dz
		currMoveWake := curr.entity.KeepUpright() || curr.entity.HasWakeMotion(dispSq, false)
		currSupportWake := curr.entity.HasSupportMotion(dispSq, false)
		for i := range slotsLen {
			slot := cage.GetSlot(i)
			rCage := slot.GetRCage()
			if rCage == nil {
				continue
			}
			rSolver := rCage.GetSolver()
			if rSolver == nil || !rSolver.IsActive() || !rSolver.IsSleeping() {
				continue
			}
			_, _, snz := slot.GetNormal()
			if !currMoveWake && (snz >= -0.30 || !currSupportWake) && rSolver.entity.IsOnGround() {
				continue
			}
			rSolver.WakeUp()
			if qLen < len(c.wakeQueue) {
				c.wakeQueue[qLen] = rSolver
				qLen++
			}
		}
		for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
			jc := c.joints[jIdx]
			if !jc.IsActive() {
				continue
			}
			var rSolver *Solver
			if jc.solverA == curr {
				rSolver = jc.solverB
			} else if jc.solverB == curr {
				rSolver = jc.solverA
			}
			if rSolver == nil || !rSolver.IsSleeping() {
				continue
			}
			rSolver.WakeUp()
			if qLen < len(c.wakeQueue) {
				c.wakeQueue[qLen] = rSolver
				qLen++
			}
		}
	}
	if addToRunning {
		for i := 0; i < qLen; i++ {
			w := c.wakeQueue[i]
			w.StagePrepare(c.session)
			if c.runningLen < len(c.running) {
				w.SetSolverIndex(c.runningLen)
				c.running[c.runningLen] = w
				c.runningLen++
			}
		}
	}
}

// findIsland returns the root island index for x, compressing parent links along the path.
func (c *Coordinator) findIsland(x int) int {
	for c.islandParent[x] != x {
		c.islandParent[x] = c.islandParent[c.islandParent[x]]
		x = c.islandParent[x]
	}
	return x
}

// Step advances the coordinator one solver tick, updating contacts, constraints, static clamps, and sleep state.
func (c *Coordinator) Step() {
	c.session++
	c.runningLen = 0

	// Cascade wake-up across pre-sleep resting slot graphs before StagePrepare clears slots.
	for x := 0; x < c.stagedLen; x++ {
		s := c.staged[x]
		if s.IsActive() && !s.IsSleeping() {
			c.wakeIsland(s, false)
		}
	}

	writeStaged := 0
	hasAfterPrepare := false
	for x := 0; x < c.stagedLen; x++ {
		s := c.staged[x]
		if !s.IsActive() {
			s.SetSolverIndex(-1)
			if c.removablesLen >= len(c.removables) {
				c.EnsureCapacity(c.removablesLen + 1)
			}
			c.removables[c.removablesLen] = s
			c.removablesLen++
			continue
		}
		c.staged[writeStaged] = s
		writeStaged++
		if s.IsSleeping() {
			s.SetSolverIndex(-1)
			continue
		}
		s.StagePrepareBefore()
		if s.hasAfterPrepare {
			hasAfterPrepare = true
		}
		s.SetSolverIndex(c.runningLen)
		c.running[c.runningLen] = s
		c.runningLen++
	}
	for k := writeStaged; k < c.stagedLen; k++ {
		c.staged[k] = nil
	}
	c.stagedLen = writeStaged

	if c.runningLen == 0 {
		return
	}

	if c.parallelAcquire && c.runningLen >= max(1024, c.parallelMinBodies) && runtime.GOMAXPROCS(0) > 1 {
		minPerWorker := max(256, c.runningLen/4)
		chunk := max(64, minPerWorker/2)
		sharedWorkerPool.Dispatch(c.runningLen, minPerWorker, chunk, c.prepareRunner)
	} else {
		c.prepareRunner(0, 0, c.runningLen)
	}
	if hasAfterPrepare {
		for x := 0; x < c.runningLen; x++ {
			c.running[x].StagePrepareAfter()
		}
	}

	c.acquireContactsParallel()

	// Compact broken or inactive joints once per step before solving.
	writeJ := 0
	for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
		if c.joints[jIdx].IsActive() {
			c.joints[writeJ] = c.joints[jIdx]
			writeJ++
		}
	}
	for k := writeJ; k < c.jointsLen; k++ {
		c.joints[k] = nil
	}
	c.jointsLen = writeJ

	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		hasFloor := s.cage.HasFloor()
		s.entity.SetOnGround(hasFloor)
		c.jointSupported[x] = hasFloor
		s.entity.SnapshotPreSolve()
	}

	if c.jointsLen > 0 {
		for range 6 {
			changed := false
			for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
				jc := c.joints[jIdx]
				ia := jc.solverA.GetSolverIndex()
				ib := jc.solverB.GetSolverIndex()
				if ia < 0 || ib < 0 {
					continue
				}
				if c.jointSupported[ia] && !c.jointSupported[ib] {
					c.jointSupported[ib] = true
					changed = true
				} else if c.jointSupported[ib] && !c.jointSupported[ia] {
					c.jointSupported[ia] = true
					changed = true
				}
			}
			if !changed {
				break
			}
		}
	}

	// Pre-apply warm-started impulses on persistent contacts and joints before iterative PGS sweeps.
	for x := 0; x < c.runningLen; x++ {
		c.running[x].StageWarmStart()
	}
	for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
		c.joints[jIdx].WarmStart()
	}

	for si := 0; si < c.solverIterations; si++ {
		if si&1 == 0 {
			for x := 0; x < c.runningLen; x++ {
				c.running[x].StageResolve()
			}
			for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
				c.joints[jIdx].ResolveVelocity()
			}
		} else {
			for x := c.runningLen - 1; x >= 0; x-- {
				c.running[x].StageResolve()
			}
			for jIdx := c.jointsLen - 1; jIdx >= 0; jIdx-- {
				c.joints[jIdx].ResolveVelocity()
			}
		}
	}

	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		s.StageApply()
		if stepZ := s.ConsumePendingStepZ(); stepZ > 0 {
			s.ApplySafeStepUp(stepZ, c.clampStaticFn)
		}
	}

	// Positional relaxation sweep coupled with parallel static world clamping so pushed bodies
	// cannot be squeezed through solid walls, floors, or ceilings across solver sweeps.
	for si := 0; si < c.solverIterations; si++ {
		if si&1 == 0 {
			for x := 0; x < c.runningLen; x++ {
				c.running[x].StageResolvePosition()
			}
			for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
				c.joints[jIdx].ResolvePosition()
			}
		} else {
			for x := c.runningLen - 1; x >= 0; x-- {
				c.running[x].StageResolvePosition()
			}
			for jIdx := c.jointsLen - 1; jIdx >= 0; jIdx-- {
				c.joints[jIdx].ResolvePosition()
			}
		}
		c.clampQueue = c.clampQueue[:0]
		for x := 0; x < c.runningLen; x++ {
			if c.running[x].NeedsStaticClamp() {
				c.clampQueue = append(c.clampQueue, x)
			}
		}
		if clampLen := len(c.clampQueue); clampLen > 0 {
			if c.parallelAcquire && clampLen >= max(1024, c.parallelMinBodies) && runtime.GOMAXPROCS(0) > 1 {
				minPerWorker := max(256, clampLen/4)
				chunk := max(64, minPerWorker/2)
				sharedWorkerPool.Dispatch(clampLen, minPerWorker, chunk, c.clampRunner)
			} else {
				c.clampRunner(0, 0, clampLen)
			}
		}
	}
	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		s.cage.CommitStaticClamp()
		s.entity.SetOnGround(s.cage.HasFloor())
		if c.jointsLen > 0 && c.jointSupported[x] {
			s.entity.DampCalmRagdoll()
		} else if !s.entity.KeepUpright() && s.entity.IsOnGround() {
			nx, ny, nz := s.cage.GetNetDisplacement()
			s.entity.DampCalmSettling(nx*nx + ny*ny + nz*nz)
		}
	}

	c.updateIslandSleep()
}

// updateIslandSleep updates joint-coupled solver islands and local support dependencies,
// allowing calm grounded sub-clusters to enter sleep even while distant pile elements settle.
func (c *Coordinator) updateIslandSleep() {
	for x := 0; x < c.runningLen; x++ {
		c.islandParent[x] = x
		c.islandSleep[x] = c.running[x].UpdateSleepStateWithSupport(c.jointSupported[x])
	}
	for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
		jc := c.joints[jIdx]
		ia := jc.solverA.GetSolverIndex()
		ib := jc.solverB.GetSolverIndex()
		if ia < 0 || ib < 0 || ia == ib {
			continue
		}
		ra := c.findIsland(ia)
		rb := c.findIsland(ib)
		if ra != rb {
			c.islandParent[rb] = ra
		}
	}
	for x := 0; x < c.runningLen; x++ {
		root := c.findIsland(x)
		if !c.islandSleep[x] {
			c.islandSleep[root] = false
		}
	}
	for x := 0; x < c.runningLen; x++ {
		c.islandSleep[x] = c.islandSleep[c.findIsland(x)]
	}
	for x := 0; x < c.runningLen; x++ {
		if c.islandSleep[x] {
			continue
		}
		sx := c.running[x]
		dx, dy, dz := sx.cage.GetNetDisplacement()
		hasMotion := sx.entity.KeepUpright() || sx.entity.HasWakeMotion(dx*dx+dy*dy+dz*dz, false)
		cage := sx.cage
		slotsLen := cage.GetSlotsLen()
		for i := range slotsLen {
			slot := cage.GetSlot(i)
			rCage := slot.GetRCage()
			if rCage == nil {
				continue
			}
			rIdx := rCage.GetSolver().GetSolverIndex()
			if rIdx < 0 || rIdx == x || !c.islandSleep[rIdx] {
				continue
			}
			_, _, snz := slot.GetNormal()
			penThresh := 0.05
			if snz > 0.30 {
				penThresh = sx.entity.FloorWakePenThreshold()
			}
			if snz < -0.30 || hasMotion || (slot.GetPenetration() > penThresh && (snz < 0.70 || !c.running[rIdx].entity.IsOnGround())) {
				c.islandSleep[rIdx] = false
			}
		}
	}
	for pass := 0; pass < c.runningLen; pass++ {
		changed := false
		for x := 0; x < c.runningLen; x++ {
			if !c.islandSleep[x] {
				continue
			}
			cage := c.running[x].cage
			slotsLen := cage.GetSlotsLen()
			for i := range slotsLen {
				slot := cage.GetSlot(i)
				rCage := slot.GetRCage()
				if rCage == nil {
					continue
				}
				rIdx := rCage.GetSolver().GetSolverIndex()
				if rIdx < 0 || rIdx == x || c.islandSleep[rIdx] {
					continue
				}
				_, _, snz := slot.GetNormal()
				if snz > 0.30 {
					c.islandSleep[x] = false
					changed = true
					break
				}
			}
		}
		if c.jointsLen > 0 {
			for jIdx := 0; jIdx < c.jointsLen; jIdx++ {
				jc := c.joints[jIdx]
				ia := jc.solverA.GetSolverIndex()
				ib := jc.solverB.GetSolverIndex()
				if ia < 0 || ib < 0 {
					continue
				}
				if !c.islandSleep[ia] && c.islandSleep[ib] {
					c.islandSleep[ib] = false
					changed = true
				} else if !c.islandSleep[ib] && c.islandSleep[ia] {
					c.islandSleep[ia] = false
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		if c.islandSleep[x] {
			s.EnterSleep()
		}
		s.SetSolverIndex(-1)
	}
}

// acquireContactsParallel executes read-only broadphase/manifold acquisition in parallel waves across active cages,
// then deterministically merges wake requests and buffered dynamic contacts by partner ID before solving.
func (c *Coordinator) acquireContactsParallel() {
	workerCap := sharedWorkerPool.WorkerCount()
	c.ensureWorkers(workerCap)
	wave := uint64(1)
	waveStart := 0
	for waveStart < c.runningLen {
		waveEnd := c.runningLen
		for x := waveStart; x < waveEnd; x++ {
			c.running[x].cage.BeginAcquire(wave)
		}

		for w := 0; w < len(c.workerWakes); w++ {
			c.workerWakes[w] = c.workerWakes[w][:0]
		}

		waveCount := waveEnd - waveStart
		c.waveStart = waveStart
		if c.parallelAcquire && waveCount >= c.parallelMinBodies && runtime.GOMAXPROCS(0) > 1 {
			minPerWorker := max(c.parallelMinBodies, min(96, waveCount/4))
			chunk := max(8, waveCount/16)
			if waveCount < 64 {
				chunk = max(1, waveCount/8)
			}
			sharedWorkerPool.Dispatch(waveCount, minPerWorker, chunk, c.acquireRunner)
		} else {
			c.acquireRunner(0, 0, waveCount)
		}

		c.mergeWaveWakes()
		c.mergeWaveDynamicContacts(waveStart, waveEnd)

		waveStart = waveEnd
		wave++
	}
	for x := 0; x < c.runningLen; x++ {
		c.running[x].cage.FinalizeAcquireSlots()
	}
}

// mergeWaveWakes drains per-worker wake queues in deterministic Cage ID order and prepares woken islands.
func (c *Coordinator) mergeWaveWakes() {
	c.wakeBatch = c.wakeBatch[:0]
	for w := 0; w < len(c.workerWakes); w++ {
		ww := c.workerWakes[w]
		if len(ww) > 0 {
			c.wakeBatch = append(c.wakeBatch, ww...)
			c.workerWakes[w] = ww[:0]
		}
	}
	if len(c.wakeBatch) == 0 {
		return
	}
	if len(c.wakeBatch) > 1 {
		sortSolversByCageID(c.wakeBatch)
	}
	for _, target := range c.wakeBatch {
		if target.IsSleeping() && target.IsActive() {
			c.wakeIsland(target, true)
		}
	}
}

// mergeWaveDynamicContacts commits acquired dynamic manifolds into partner cages in ascending Cage ID order.
func (c *Coordinator) mergeWaveDynamicContacts(waveStart, waveEnd int) {
	waveLen := waveEnd - waveStart
	if cap(c.mergeOrder) < waveLen {
		c.mergeOrder = make([]int, waveLen)
	} else {
		c.mergeOrder = c.mergeOrder[:waveLen]
	}
	for i := range waveLen {
		c.mergeOrder[i] = waveStart + i
	}
	if waveLen > 1 {
		sortIndicesByCageID(c.mergeOrder, c.running)
	}
	for _, idx := range c.mergeOrder {
		cage := c.running[idx].cage
		cage.EndAcquire()
		cage.FlushPendingDynamic()
	}
}

func cageOrderLess(a, b *Cage) bool {
	if a.id != b.id {
		return a.id < b.id
	}
	return a.entity.GetId() < b.entity.GetId()
}

func sortSolversByCageID(s []*Solver) {
	n := len(s)
	for gap := n / 2; gap > 0; gap /= 2 {
		for i := gap; i < n; i++ {
			cur := s[i]
			j := i
			for j >= gap && cageOrderLess(cur.cage, s[j-gap].cage) {
				s[j] = s[j-gap]
				j -= gap
			}
			s[j] = cur
		}
	}
}

func sortIndicesByCageID(indices []int, running []*Solver) {
	n := len(indices)
	for gap := n / 2; gap > 0; gap /= 2 {
		for i := gap; i < n; i++ {
			cur := indices[i]
			curCage := running[cur].cage
			j := i
			for j >= gap && cageOrderLess(curCage, running[indices[j-gap]].cage) {
				indices[j] = indices[j-gap]
				j -= gap
			}
			indices[j] = cur
		}
	}
}
