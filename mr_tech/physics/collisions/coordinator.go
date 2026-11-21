package collisions

type WakeRunningFn func(neighbor *Solver)

type QueryContactsFn func(cage *Cage, wake WakeRunningFn)

type ClampStaticFn func(cage *Cage)

// Coordinator manages solver lifecycles, wake queues, islands, and joint constraints during simulation.
type Coordinator struct {
	solverIterations int
	session          uint64
	staged           []*Solver
	stagedLen        int
	running          []*Solver
	runningLen       int
	removables       []*Solver
	removablesLen    int
	wakeQueue        []*Solver
	islandParent     []int
	islandSleep      []bool
	jointSupported   []bool
	joints           []*JointConstraint
	jointsLen        int
	wakeRunningFn    WakeRunningFn
	queryContactsFn  QueryContactsFn
	clampStaticFn    ClampStaticFn
	commonCap        int
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
		commonCap:        commonCap,
		solverIterations: solverIterations,
		staged:           make([]*Solver, commonCap),
		running:          make([]*Solver, commonCap),
		removables:       make([]*Solver, commonCap),
		wakeQueue:        make([]*Solver, commonCap),
		islandParent:     make([]int, commonCap),
		islandSleep:      make([]bool, commonCap),
		jointSupported:   make([]bool, commonCap),
		joints:           make([]*JointConstraint, 0, commonCap),
		queryContactsFn:  queryContactsFn,
		clampStaticFn:    clampStaticFn,
	}
	c.wakeRunningFn = c.wakeRunningIsland
	return c
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
		for i := 0; i < slotsLen; i++ {
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
		s.StagePrepare(c.session)
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

	for cursor := 0; cursor < c.runningLen; cursor++ {
		c.queryContactsFn(c.running[cursor].cage, c.wakeRunningFn)
	}

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
		if stepZ := s.cage.ConsumePendingStepZ(); stepZ > 0 {
			s.cage.ApplySafeStepUp(stepZ, c.clampStaticFn)
		}
	}

	// Positional relaxation sweep coupled with per-iteration static world clamping so pushed bodies
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
		for x := 0; x < c.runningLen; x++ {
			cage := c.running[x].cage
			if cage.NeedsStaticClamp() {
				c.clampStaticFn(cage)
				cage.CommitStaticClamp()
			}
		}
	}
	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		s.cage.CommitStaticClamp()
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
		for i := 0; i < slotsLen; i++ {
			slot := cage.GetSlot(i)
			rCage := slot.GetRCage()
			if rCage == nil {
				continue
			}
			rIdx := rCage.GetSolverIndex()
			if rIdx < 0 || rIdx == x || !c.islandSleep[rIdx] {
				continue
			}
			_, _, snz := slot.GetNormal()
			penThresh := 0.05
			if snz > 0.30 {
				penThresh = 0.08
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
			for i := 0; i < slotsLen; i++ {
				slot := cage.GetSlot(i)
				rCage := slot.GetRCage()
				if rCage == nil {
					continue
				}
				rIdx := rCage.GetSolverIndex()
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
