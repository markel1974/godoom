package collisions

// Coordinator orchestrates per-tick collision detection, PGS velocity/position iterations,
// connected-island wake propagation, and island sleeping over active *Solver instances.
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
	wakeRunningFn    func(neighbor *Solver)
}

// NewCoordinator creates a Coordinator preallocated for initialCap solvers and solverIterations PGS sweeps.
func NewCoordinator(solverIterations, initialCap int) *Coordinator {
	if solverIterations <= 0 {
		solverIterations = 10
	}
	if initialCap <= 0 {
		initialCap = 1024
	}
	c := &Coordinator{
		solverIterations: solverIterations,
		staged:           make([]*Solver, initialCap),
		running:          make([]*Solver, initialCap),
		removables:       make([]*Solver, initialCap),
		wakeQueue:        make([]*Solver, initialCap),
		islandParent:     make([]int, initialCap),
		islandSleep:      make([]bool, initialCap),
	}
	c.wakeRunningFn = c.wakeRunningIsland
	return c
}

// EnsureCapacity grows internal working buffers to hold at least minCap solvers without per-frame allocations.
func (c *Coordinator) EnsureCapacity(minCap int) {
	if minCap <= cap(c.running) {
		return
	}
	newCap := minCap * 2
	growSolvers := func(src []*Solver) []*Solver {
		dst := make([]*Solver, newCap)
		copy(dst, src)
		return dst
	}
	c.staged = growSolvers(c.staged)
	c.running = growSolvers(c.running)
	c.removables = growSolvers(c.removables)
	c.wakeQueue = growSolvers(c.wakeQueue)

	newParents := make([]int, newCap)
	copy(newParents, c.islandParent)
	c.islandParent = newParents

	newSleep := make([]bool, newCap)
	copy(newSleep, c.islandSleep)
	c.islandSleep = newSleep
}

// Reset clears the staged solver list for the next frame.
func (c *Coordinator) Reset() {
	c.stagedLen = 0
}

// Stage adds s to the current frame's staged solvers.
func (c *Coordinator) Stage(s *Solver) {
	if c.stagedLen >= len(c.staged) {
		c.EnsureCapacity(c.stagedLen + 1)
	}
	c.staged[c.stagedLen] = s
	c.stagedLen++
}

// StagedLen returns the number of currently staged solvers.
func (c *Coordinator) StagedLen() int {
	return c.stagedLen
}

// Running returns the slice of active solvers simulated in the most recent Step.
func (c *Coordinator) Running() []*Solver {
	return c.running[:c.runningLen]
}

// Removables returns solvers marked inactive during the most recent Step.
func (c *Coordinator) Removables() []*Solver {
	return c.removables[:c.removablesLen]
}

// ClearRemovables resets the pending removable solver list after the caller handles removals.
func (c *Coordinator) ClearRemovables() {
	c.removablesLen = 0
}

func (c *Coordinator) wakeRunningIsland(neighbor *Solver) {
	c.wakeIsland(neighbor, true)
}

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
		for i := 0; i < slotsLen; i++ {
			rCage := cage.GetSlot(i).GetRCage()
			if rCage == nil {
				continue
			}
			rSolver := rCage.GetSolver()
			if rSolver == nil || !rSolver.IsActive() || !rSolver.IsSleeping() {
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

func (c *Coordinator) findIsland(x int) int {
	for c.islandParent[x] != x {
		c.islandParent[x] = c.islandParent[c.islandParent[x]]
		x = c.islandParent[x]
	}
	return x
}

// Step runs one complete physics simulation tick across all staged solvers.
// queryContacts gathers static and dynamic contacts for cage and may invoke wake for newly contacted sleeping neighbors;
// clampStatic resolves a cage against static world geometry during step-ups and positional relaxation.
func (c *Coordinator) Step(queryContacts func(cage *Cage, wake func(*Solver)), clampStatic func(cage *Cage)) {
	c.session++
	c.runningLen = 0

	// Cascade wake-up across pre-sleep resting slot graphs before StagePrepare clears slots.
	for x := 0; x < c.stagedLen; x++ {
		s := c.staged[x]
		if s.IsActive() && !s.IsSleeping() {
			c.wakeIsland(s, false)
		}
	}

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
		if s.IsSleeping() {
			s.SetSolverIndex(-1)
			continue
		}
		s.StagePrepare(c.session)
		s.SetSolverIndex(c.runningLen)
		c.running[c.runningLen] = s
		c.runningLen++
	}

	if c.runningLen == 0 {
		return
	}

	for cursor := 0; cursor < c.runningLen; cursor++ {
		queryContacts(c.running[cursor].cage, c.wakeRunningFn)
	}

	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		s.entity.SetOnGround(s.cage.HasFloor())
		s.entity.SnapshotPreSolve()
	}

	// Pre-apply warm-started impulses on persistent contacts before iterative PGS sweeps.
	for x := 0; x < c.runningLen; x++ {
		c.running[x].StageWarmStart()
	}

	for si := 0; si < c.solverIterations; si++ {
		if si&1 == 0 {
			for x := 0; x < c.runningLen; x++ {
				c.running[x].StageResolve()
			}
		} else {
			for x := c.runningLen - 1; x >= 0; x-- {
				c.running[x].StageResolve()
			}
		}
	}

	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		s.StageApply()
		if stepZ := s.cage.ConsumePendingStepZ(); stepZ > 0 {
			ApplySafeStepUp(s.cage, stepZ, clampStatic)
		}
	}

	// Positional relaxation sweep coupled with per-iteration static world clamping so pushed bodies
	// cannot be squeezed through solid walls, floors, or ceilings across solver sweeps.
	for si := 0; si < c.solverIterations; si++ {
		if si&1 == 0 {
			for x := 0; x < c.runningLen; x++ {
				c.running[x].StageResolvePosition()
			}
		} else {
			for x := c.runningLen - 1; x >= 0; x-- {
				c.running[x].StageResolvePosition()
			}
		}
		for x := 0; x < c.runningLen; x++ {
			cage := c.running[x].cage
			if cage.NeedsStaticClamp() {
				clampStatic(cage)
			}
		}
	}
	for x := 0; x < c.runningLen; x++ {
		c.running[x].cage.CommitStaticClamp()
	}

	c.updateIslandSleep()
}

func (c *Coordinator) updateIslandSleep() {
	for x := 0; x < c.runningLen; x++ {
		c.islandParent[x] = x
		c.islandSleep[x] = c.running[x].UpdateSleepState()
	}
	for x := 0; x < c.runningLen; x++ {
		cage := c.running[x].cage
		slotsLen := cage.GetSlotsLen()
		for i := 0; i < slotsLen; i++ {
			rCage := cage.GetSlot(i).GetRCage()
			if rCage == nil {
				continue
			}
			rIdx := rCage.GetSolverIndex()
			if rIdx < 0 || rIdx == x {
				continue
			}
			rx := c.findIsland(x)
			ry := c.findIsland(rIdx)
			if rx != ry {
				c.islandParent[ry] = rx
			}
		}
	}
	for x := 0; x < c.runningLen; x++ {
		root := c.findIsland(x)
		if !c.islandSleep[x] {
			c.islandSleep[root] = false
		}
	}
	for x := 0; x < c.runningLen; x++ {
		s := c.running[x]
		if c.islandSleep[c.findIsland(x)] {
			s.EnterSleep()
		}
		s.SetSolverIndex(-1)
	}
}
