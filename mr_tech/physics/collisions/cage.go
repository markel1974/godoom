package collisions

import (
	"math"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// ImpactMode identifies how a collision impact should be resolved and prioritized.
type ImpactMode int

const (
	// EdgeABActive marks triangle edge v0->v1 as an active convex or boundary collision edge.
	EdgeABActive uint8 = 1 << 0
	// EdgeBCActive marks triangle edge v1->v2 as an active convex or boundary collision edge.
	EdgeBCActive uint8 = 1 << 1
	// EdgeCAActive marks triangle edge v2->v0 as an active convex or boundary collision edge.
	EdgeCAActive uint8 = 1 << 2
	// EdgeAllActive marks all three triangle edges as active collision edges.
	EdgeAllActive uint8 = EdgeABActive | EdgeBCActive | EdgeCAActive
)

type IFace interface {
	GetAABB() *aabb.AABB

	GetNormal() (float64, float64, float64)

	GetNormalAbs() (float64, float64, float64)

	Translate(int, float64, float64, float64) (float64, float64, float64)

	GetParent() *Solver

	Get0() (float64, float64, float64)

	Get1() (float64, float64, float64)

	Get2() (float64, float64, float64)

	IsSolid() bool

	IsSingleSided() bool

	ActiveEdgeMask() uint8
}

// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
const (
	// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
	ImpactInelastic = ImpactMode(1)
	// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
	ImpactElastic = ImpactMode(2)
	// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
	ImpactStep = ImpactMode(3)
)

// Priority returns the collision ordering rank for the impact mode, with unknown modes ranked lowest.
func (iMode ImpactMode) Priority() int {
	switch iMode {
	case ImpactInelastic:
		return 3
	case ImpactElastic:
		return 2
	case ImpactStep:
		return 1
	default:
		return 0
	}
}

// satFilterEpsilon defines the tolerance threshold used in SAT to filter phantom plane projections in collision detection.
const satFilterEpsilon = 0.05

// planeEpsilon is a tolerance value used for floating-point precision when comparing coplanar planes or distances.
const planeEpsilon = 0.01

// toiTieEpsilon represents a small positive threshold used to handle numerical imprecision in time of impact calculations.
const toiTieEpsilon = 1e-4

// TotalSlots represents the total available slots, computed as BucketSize * FacesPerBucket.
const totalSlots = BucketSize * FacesPerBucket

// floorGroundTolerance keeps floor contact grounded within a small vertical hover band.
const floorGroundTolerance = 0.25

// floorNormalThreshold matches the rigid-body floor normal threshold for valid ground support.
const floorNormalThreshold = 0.7

const LocalSlotMax = 2

// Cage represents a data structure for managing collision information for a physical entity.[cite: 6]
type Cage struct {
	seen                []uint64
	session             uint64
	id                  uint64
	object              aabb.IObject
	entity              *rigid_body.Entity
	buckets             [BucketSize]*Bucket
	ellipsoid           *rigid_body.Entity
	ellipsoidLocal      [LocalSlotMax]*rigid_body.Entity
	cX, cY, cZ          float64
	dX, dY, dZ          float64
	tX, tY, tZ          float64
	eRadX, eRadY, eRadZ float64
	volumeObj           aabb.IObject
	distance            float64

	cradle           *Cradle
	maxStep          float64
	faces            []*Faces
	facesIdx         int
	manifold         *aabb.OBBManifold
	staticContact    *StaticFaceContact
	hasFloor         bool
	applied          bool
	appliedX         float64
	appliedY         float64
	appliedZ         float64
	clampX           float64
	clampY           float64
	clampZ           float64
	prevRadX         float64
	prevRadY         float64
	prevRadZ         float64
	prevQX           float64
	prevQY           float64
	prevQZ           float64
	prevQW           float64
	prevKeepUpright  bool
	forceStaticClamp bool

	passDX          float64
	passDY          float64
	passDZ          float64
	stepDX          float64
	stepDY          float64
	stepDZ          float64
	hasSpawnOrigin  bool
	spawnX          float64
	spawnY          float64
	spawnZ          float64
	pendingStepZ    float64
	collisionGroup  uint64
	solverIdx       int
	solver          *Solver
	slots           *Slots
	acquireWave     uint64
	acquiring       bool
	staticSlotStart int
	staticSlotEnd   int
	pending         *PendingDynamicContacts
}

var _counter atomic.Uint64

// NewCageSequence creates a new Cage for object using the next sequence ID.
func NewCageSequence() *Cage {
	id := _counter.Add(1) - 1
	return NewCage(id)
}

// NewCage creates and returns a new instance of Cage initialized with the specified IThing object.[cite: 6]
func NewCage(id uint64) *Cage {
	c := &Cage{
		id:        id,
		seen:      make([]uint64, 64),
		solver:    nil,
		ellipsoid: rigid_body.NewEntity(0, 0, 0, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0),
		volumeObj: nil,
		slots:     NewSlots(),

		faces:         make([]*Faces, 8),
		facesIdx:      0,
		hasFloor:      false,
		manifold:      aabb.NewOBBManifold(),
		solverIdx:     -1,
		staticContact: NewStaticFaceContact(),
		cradle:        NewCradleCache(),
		pending:       NewPendingDynamicContacts(16),
		//pendingDyn:    make([]*PendingDynamicContact, 16),
	}
	for i := 0; i < len(c.faces); i++ {
		c.faces[i] = NewFaces()
	}
	for i := range BucketSize {
		c.buckets[i] = NewBucket(i)
	}
	for i := range len(c.ellipsoidLocal) {
		c.ellipsoidLocal[i] = rigid_body.NewEntity(0, 0, 0, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0)
	}
	return c
}

// Setup assigns the object used by the cage and returns nil.
func (s *Cage) Setup(solver *Solver) error {
	s.solver = solver
	s.object = solver.GetObject()
	s.entity = s.solver.GetEntity()
	return nil
}

// GetObject returns the object associated with the cage.[cite: 6]
func (s *Cage) GetObject() aabb.IObject {
	return s.object
}

// GetSlots returns the cage's active collision slot collection.
func (s *Cage) GetSlots() *Slots {
	return s.slots
}

// Rebuild recalculates the collision cage dimensions, position, and broad-phase swept volume using the given maxStep value.[cite: 6]
func (s *Cage) Rebuild(maxStep float64, session uint64) {
	s.session = session
	lEnt := s.entity
	// Calculate Position
	s.stepDX, s.stepDY, s.stepDZ = lEnt.GetFrameDisplacement()
	s.dX, s.dY, s.dZ = s.stepDX, s.stepDY, s.stepDZ

	entCX, entCY, entCZ := lEnt.GetCenter()
	s.cX, s.cY, s.cZ = entCX, entCY, entCZ
	s.eRadX, s.eRadY, s.eRadZ = lEnt.HalfExtents()

	keepUpright := lEnt.KeepUpright()
	qx, qy, qz, qw := lEnt.GetQRotation().Get()
	if keepUpright != s.prevKeepUpright ||
		s.eRadX != s.prevRadX || s.eRadY != s.prevRadY || s.eRadZ != s.prevRadZ ||
		qx != s.prevQX || qy != s.prevQY || qz != s.prevQZ || qw != s.prevQW {
		s.forceStaticClamp = true
		s.prevKeepUpright = keepUpright
		s.prevRadX, s.prevRadY, s.prevRadZ = s.eRadX, s.eRadY, s.eRadZ
		s.prevQX, s.prevQY, s.prevQZ, s.prevQW = qx, qy, qz, qw
	}

	s.tX, s.tY, s.tZ = s.cX+s.stepDX, s.cY+s.stepDY, s.cZ+s.stepDZ
	clampBaseX, clampBaseY, clampBaseZ := entCX, entCY, entCZ
	if s.hasSpawnOrigin {
		s.cX, s.cY, s.cZ = s.spawnX, s.spawnY, s.spawnZ
		s.dX = s.tX - s.cX
		s.dY = s.tY - s.cY
		s.dZ = s.tZ - s.cZ
		clampBaseX, clampBaseY, clampBaseZ = s.spawnX, s.spawnY, s.spawnZ
		s.forceStaticClamp = true
		s.hasSpawnOrigin = false
	}
	s.passDX, s.passDY, s.passDZ = s.dX, s.dY, s.dZ

	sx, sy, sz := s.sweptHalfExtents(lEnt)

	// Calculate absolute extremes (Broad-Phase Linear + Angular Swept Volume)
	minX := s.cX - sx + math.Min(0, s.dX)
	maxX := s.cX + sx + math.Max(0, s.dX)
	minY := s.cY - sy + math.Min(0, s.dY)
	maxY := s.cY + sy + math.Max(0, s.dY)
	minZ := s.cZ - sz + math.Min(0, s.dZ)
	maxZ := s.cZ + sz + math.Max(0, s.dZ)

	// Canonical mapping for Rect/AABB
	s.ellipsoid.Rebuild(minX, minY, minZ, maxX-minX, maxY-minY, maxZ-minZ)

	for i := range len(s.buckets) {
		s.buckets[i].Rebuild()
	}

	s.volumeObj = nil
	s.distance = math.MaxFloat64

	s.slots.Reset()
	s.cradle.Reset()
	s.hasFloor = false
	s.applied = false
	s.acquiring = false
	s.acquireWave = 0
	s.staticSlotStart = 0
	s.staticSlotEnd = 0
	s.pending.Reset()
	//s.pendingDynLen = 0
	s.clampX, s.clampY, s.clampZ = clampBaseX, clampBaseY, clampBaseZ

	s.staticContact.Reset()

	//clear(s.seen)
	s.maxStep = maxStep
	s.pendingStepZ = 0
}

// BeginAcquire switches this cage into read-only parallel acquisition mode for wave.
func (s *Cage) BeginAcquire(wave uint64) {
	s.acquireWave = wave
	s.acquiring = true
	s.staticSlotStart = 0
	s.staticSlotEnd = 0
	s.pending.Reset()
	//s.pendingDynLen = 0
}

// FinalizeAcquireSlots moves any static slots acquired before dynamic merges to the end of s.slots
// so PGS warm-start and resolve sweeps anchor multi-body pushes into static world surfaces cleanly.
func (s *Cage) FinalizeAcquireSlots() {
	if s.staticSlotEnd > s.staticSlotStart && s.staticSlotEnd < s.slots.Len() {
		s.slots.RotateRangeToEnd(s.staticSlotStart, s.staticSlotEnd)
	}
	s.staticSlotStart = 0
	s.staticSlotEnd = 0
}

// EndAcquire leaves parallel acquisition mode for deterministic merge and solve.
func (s *Cage) EndAcquire() {
	s.acquiring = false
}

// IsAcquiring reports whether this cage is in parallel read-only acquisition mode.
func (s *Cage) IsAcquiring() bool {
	return s.acquiring
}

// CanAcquireDynamic reports whether this cage owns evaluating the dynamic pair (s, rCage)
// during parallel read-only acquisition without writing to rCage.
func (s *Cage) CanAcquireDynamic(rCage *Cage) bool {
	if s == rCage || s.HasSeen(rCage) {
		return false
	}
	if s.acquiring && !rCage.entity.IsSleeping() && rCage.acquireWave == s.acquireWave {
		if s.id > rCage.id || (s.id == rCage.id && s.entity.GetId() > rCage.entity.GetId()) {
			return false
		}
	}
	s.Seen(rCage)
	return true
}

// SortPendingDynamic deterministically sorts buffered dynamic contacts by partner ID.
func (s *Cage) SortPendingDynamic() {
	s.pending.Sort()
}

// FlushPendingDynamic merges buffered read-only dynamic contacts symmetrically into buckets and slots.
func (s *Cage) FlushPendingDynamic() {
	n := s.pending.Len()
	for i := range n {
		pd := s.pending.At(i)
		rCage := pd.partner
		rSleeping := rCage.entity.IsSleeping()
		if !rSleeping {
			rCage.Seen(s)
			if pd.wasSleeping {
				// Partner transitioned sleeping -> awake in mergeWaveWakes and ran StagePrepare:
				// re-evaluate narrow-phase manifold using its updated post-prepare displacement.
				s.AddFace(pd.rFace, pd.recipFace)
				var refreshed PendingDynamicContact
				ok := s.buildPendingDynamic(rCage, &refreshed)
				s.DiscardPendingFaces()
				if ok {
					s.applyPendingDynamic(&refreshed)
				}
				continue
			}
		}
		s.applyPendingDynamic(pd)
	}
	s.pending.Reset()
}

// AddFace adds a pair of faces (rFace and lFace) to the Cage, resizing the internal slice if needed.[cite: 6]
func (s *Cage) AddFace(rFace IFace, lFace IFace) {
	if s.facesIdx >= len(s.faces) {
		n := make([]*Faces, len(s.faces)*2)
		copy(n, s.faces)
		for i := len(s.faces); i < len(n); i++ {
			n[i] = NewFaces()
		}
		s.faces = n
	}
	f := s.faces[s.facesIdx]
	f.rFace.SetIFace(rFace)
	f.lFace.SetIFace(lFace)
	s.facesIdx++
}

// Commit processes pending face overlaps in this Cage against either another dynamic Cage or static world geometry.
func (s *Cage) Commit(rCage *Cage) {
	if rCage != nil {
		if s.acquiring {
			s.acquireDynamicManifold(rCage)
		} else {
			s.commitDynamicManifold(rCage)
		}
		s.facesIdx = 0
		return
	}
	if s.acquiring {
		s.staticSlotStart = s.slots.Len()
		s.staticContact.Commit(s, s.ellipsoid.GetAABB())
		s.staticSlotEnd = s.slots.Len()
	} else {
		s.staticContact.Commit(s, s.ellipsoid.GetAABB())
	}
	s.facesIdx = 0
}

// StaticFaceClamp clamps static contact data against the given AABB and face, reporting whether it succeeded.
func (s *Cage) StaticFaceClamp(lAABB *aabb.AABB, rFace IFace) bool {
	if !rFace.IsSolid() {
		return false
	}
	evalFace := s.staticContact.evalFace
	evalFace.SetIFace(rFace)
	return s.staticFaceClamp(lAABB, evalFace)
}

// staticFaceClamp evaluates a solid face against the local AABB and applies a static clamp when penetration requires it.
// It updates floor and wall contact state and reports whether a clamp was applied.
func (s *Cage) staticFaceClamp(lAABB *aabb.AABB, rFace *Face) bool {
	if !rFace.IsSolid() {
		return false
	}
	fc := s.staticContact
	if !s.evalStaticFace(lAABB, rFace, fc) {
		return false
	}
	if fc.pen <= 0.0 {
		if fc.bucket == BucketFloor && fc.dist <= floorGroundTolerance {
			s.hasFloor = true
			s.entity.SetOnGround(true)
		}
		return false
	}
	if fc.bucket == BucketFloor {
		s.hasFloor = true
		s.entity.SetOnGround(true)
	}
	if fc.bucket.IsWall() {
		if s.clampWallShouldSkip(fc) {
			return false
		}
		if fc.cornerHit {
			if fc.HasCoplanarFlatWall() {
				return false
			}
			if fc.AddClampCorner(rFace.IFace) {
				return false
			}
		} else {
			fc.RecordFlatWall()
		}
	}
	return s.clampApplyContact(fc)
}

// StaticFaceFlushClamp reports whether the cage's static contact can flush-clamp against the given AABB.
func (s *Cage) StaticFaceFlushClamp(lAABB *aabb.AABB) bool {
	moved := false
	fc := s.staticContact
	for i := 0; i < fc.clampCornersLen; i++ {
		if !s.evalStaticFace(lAABB, fc.clampCorners[i], fc) || fc.pen <= 0.0 {
			continue
		}
		if fc.bucket.IsWall() {
			if s.clampWallShouldSkip(fc) || fc.HasCoplanarFlatWall() {
				continue
			}
		}
		if s.clampApplyContact(fc) {
			moved = true
		}
	}
	fc.clampCornersLen = 0
	return moved
}

// BeginStaticFaceCache initializes the per-frame cached static face list for the given broadphase AABB.
func (s *Cage) BeginStaticFaceCache(lAABB *aabb.AABB) {
	s.staticContact.BeginStaticFaceCache(lAABB)
}

// AddCachedStaticFace records a solid static face in this cage's per-frame broadphase cache.
func (s *Cage) AddCachedStaticFace(rFace IFace) {
	s.staticContact.AddCachedFace(rFace)
}

// CommitStaticFaceCache marks the cage's per-frame cached static faces as complete for reuse.
func (s *Cage) CommitStaticFaceCache() {
	s.staticContact.CommitStaticFaceCache()
}

// BeginPaddedStaticFaceQuery expands lAABB temporarily to cover nearby post-push clamping and starts the static face cache.
func (s *Cage) BeginPaddedStaticFaceQuery(lAABB *aabb.AABB) (minX, minY, minZ, maxX, maxY, maxZ float64) {
	minX, minY, minZ, maxX, maxY, maxZ = lAABB.GetMinMax()
	padX := math.Max(s.eRadX, 4.0) + math.Abs(s.dX)
	padY := math.Max(s.eRadY, 4.0) + math.Abs(s.dY)
	padZ := math.Max(s.maxStep, math.Max(s.eRadZ, 4.0)) + math.Abs(s.dZ)
	lAABB.Rebuild(minX-padX, minY-padY, minZ-padZ, maxX+padX, maxY+padY, maxZ+padZ)
	s.BeginStaticFaceCache(lAABB)
	return minX, minY, minZ, maxX, maxY, maxZ
}

// EndPaddedStaticFaceQuery restores the tight clamp AABB bounds and commits the cached static faces for reuse.
func (s *Cage) EndPaddedStaticFaceQuery(lAABB *aabb.AABB, minX, minY, minZ, maxX, maxY, maxZ float64) {
	lAABB.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
	s.CommitStaticFaceCache()
}

// HasStaticFaceCache reports whether cached static faces cover lAABB without needing a fresh BVH query.
func (s *Cage) HasStaticFaceCache(lAABB *aabb.AABB) bool {
	return s.staticContact.HasStaticFaceCache(lAABB)
}

// CachePendingStaticFaces copies pending static faces queued via AddFace into the per-frame broadphase cache.
func (s *Cage) CachePendingStaticFaces(lAABB *aabb.AABB) {
	fc := s.staticContact
	fc.BeginStaticFaceCache(lAABB)
	for i := 0; i < s.facesIdx; i++ {
		fc.AddCachedFace(s.faces[i].rFace.IFace)
	}
	fc.CommitStaticFaceCache()
}

// ClampCachedStaticFaces resolves static face clamping against all faces stored in the per-frame broadphase cache.
func (s *Cage) ClampCachedStaticFaces(lAABB *aabb.AABB) bool {
	moved := false
	fc := s.staticContact
	for i := 0; i < fc.cachedFacesLen; i++ {
		if s.staticFaceClamp(lAABB, fc.cachedFaces[i]) {
			moved = true
		}
	}
	return moved
}

// clampWallShouldSkip reports whether a wall contact should be ignored as stepable or ground-adjacent.
func (s *Cage) clampWallShouldSkip(fc *StaticFaceContact) bool {
	if fc.wallTopLip {
		return true
	}
	curBaseZ := s.tZ - s.eRadZ
	preBaseZ := s.GetBaseZ()
	if fc.maxZ <= curBaseZ+satFilterEpsilon && preBaseZ+s.maxStep >= fc.maxZ-satFilterEpsilon {
		return true
	}
	return preBaseZ >= fc.maxZ-satFilterEpsilon && s.dZ < -satFilterEpsilon && fc.maxZ-curBaseZ <= floorGroundTolerance
}

// clampApplyContact clamps velocity, resolves penetration, updates cage position, and reports whether movement occurred.
func (s *Cage) clampApplyContact(fc *StaticFaceContact) bool {
	fc.AddClampNormal(fc.nX, fc.nY, fc.nZ)

	ent := s.entity
	ent.ClampNormalVelocity(fc.nX, fc.nY, fc.nZ)
	p, ok := ent.Cinematic.ResolveInelasticPenetration(fc.nZ, fc.pen)
	if !ok || p <= 0.0 {
		return false
	}
	ent.AddTo(fc.nX*p, fc.nY*p, fc.nZ*p)
	s.tX += fc.nX * p
	s.tY += fc.nY * p
	s.tZ += fc.nZ * p
	s.dX = s.tX - s.cX
	s.dY = s.tY - s.cY
	s.dZ = s.tZ - s.cZ
	return true
}

// addStaticTriangleManifold emits either the single closest contact for KeepUpright entities or a clipped OBB
// multi-point manifold against static triangle rFace, falling back to a stabilized contact when clipping misses.
func (s *Cage) addStaticTriangleManifold(rFace *Face, fc *StaticFaceContact, iMode ImpactMode) {
	rIFace := rFace.IFace
	if s.entity.KeepUpright() {
		px, py, pz := fc.contactX, fc.contactY, fc.contactZ
		if !fc.hasContact {
			px, py, pz = s.projectStabilizedStaticContact(fc)
		}
		entry := s.addToBucketEntry(fc.bucket, nil, rIFace, fc.dist, fc.pen, fc.nX, fc.nY, fc.nZ, px, py, pz, fc.maxZ, iMode)
		if entry != nil && fc.bucket.IsWall() {
			entry.SetCornerOrigin(fc.cornerHit, fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
		}
		return
	}
	v0x, v0y, v0z := rFace.Get0()
	v1x, v1y, v1z := rFace.Get1()
	v2x, v2y, v2z := rFace.Get2()
	entCX, entCY, entCZ := s.entity.GetCenter()
	offX, offY, offZ := s.tX-entCX, s.tY-entCY, s.tZ-entCZ
	distStart := (s.cX-v0x)*fc.nX + (s.cY-v0y)*fc.nY + (s.cZ-v0z)*fc.nZ
	distTarget := (s.tX-v0x)*fc.nX + (s.tY-v0y)*fc.nY + (s.tZ-v0z)*fc.nZ
	extraPen := 0.0
	shiftToTX, shiftToTY, shiftToTZ := 0.0, 0.0, 0.0
	if distTarget < 0.0 {
		tau := 0.0
		if denom := distStart - distTarget; denom > 1e-12 && distStart > 0.0 {
			tau = math.Max(0.0, math.Min(1.0, distStart/denom))
		}
		hitX := s.cX + tau*s.dX
		hitY := s.cY + tau*s.dY
		hitZ := s.cZ + tau*s.dZ
		offX, offY, offZ = hitX-entCX, hitY-entCY, hitZ-entCZ
		shiftToTX, shiftToTY, shiftToTZ = s.tX-hitX, s.tY-hitY, s.tZ-hitZ
		extraPen = -distTarget
	}
	m := s.manifold
	if m.ComputeStaticTriangle(s.entity.BoundingBox, offX, offY, offZ, fc.nX, fc.nY, fc.nZ, v0x, v0y, v0z, v1x, v1y, v1z, v2x, v2y, v2z, fc.pen) {
		added := 0
		for pi := 0; pi < m.GetCount(); pi++ {
			ptX, ptY, ptZ, ptPen := m.GetPoint(pi)
			if extraPen > 0.0 {
				ptPen = math.Min(fc.pen, ptPen+extraPen)
				if pi == 0 && ptPen < fc.pen {
					ptPen = fc.pen
				}
			}
			if ptPen <= 0 {
				continue
			}
			ptX = math.Max(s.tX-s.eRadX, math.Min(s.tX+s.eRadX, ptX+shiftToTX))
			ptY = math.Max(s.tY-s.eRadY, math.Min(s.tY+s.eRadY, ptY+shiftToTY))
			ptZ = math.Max(s.tZ-s.eRadZ, math.Min(s.tZ+s.eRadZ, ptZ+shiftToTZ))
			entry := s.addToBucketEntry(fc.bucket, nil, rIFace, -ptPen, ptPen, fc.nX, fc.nY, fc.nZ, ptX, ptY, ptZ, fc.maxZ, iMode)
			if entry != nil && fc.bucket.IsWall() {
				entry.SetCornerOrigin(fc.cornerHit, fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
			}
			added++
		}
		if added > 0 {
			return
		}
	}
	if !fc.cornerHit {
		if activeMask := rFace.ActiveEdgeMask(); activeMask != EdgeAllActive && s.eRadX > 0 && s.eRadY > 0 && s.eRadZ > 0 {
			invRX, invRY, invRZ := 1.0/s.eRadX, 1.0/s.eRadY, 1.0/s.eRadZ
			_, _, _, feat := ClosestPointOnTriangleFeature((v0x-s.tX)*invRX, (v0y-s.tY)*invRY, (v0z-s.tZ)*invRZ, (v1x-s.tX)*invRX, (v1y-s.tY)*invRY, (v1z-s.tZ)*invRZ, (v2x-s.tX)*invRX, (v2y-s.tY)*invRY, (v2z-s.tZ)*invRZ)
			if IsInactiveTriangleFeature(feat, activeMask) {
				return
			}
		}
	}
	px, py, pz := s.projectStabilizedStaticContact(fc)
	entry := s.addToBucketEntry(fc.bucket, nil, rIFace, fc.dist, fc.pen, fc.nX, fc.nY, fc.nZ, px, py, pz, fc.maxZ, iMode)
	if entry != nil && fc.bucket.IsWall() {
		entry.SetCornerOrigin(fc.cornerHit, fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
	}
}

// projectStabilizedStaticContact projects the stabilized single-point support onto the static face plane,
// clamped to the triangle's world AABB bounds.
func (s *Cage) projectStabilizedStaticContact(fc *StaticFaceContact) (float64, float64, float64) {
	supX, supY, supZ := s.entity.GetStabilizedContactPoint(-fc.nX, -fc.nY, -fc.nZ)
	entCX, entCY, entCZ := s.entity.GetCenter()
	supX += s.tX - entCX
	supY += s.tY - entCY
	supZ += s.tZ - entCZ
	px, py, pz := fc.Project(supX, supY, supZ)
	if !s.entity.KeepUpright() {
		px = math.Max(s.tX-s.eRadX, math.Min(s.tX+s.eRadX, px))
		py = math.Max(s.tY-s.eRadY, math.Min(s.tY+s.eRadY, py))
		pz = math.Max(s.tZ-s.eRadZ, math.Min(s.tZ+s.eRadZ, pz))
	}
	return px, py, pz
}

// addToBucket adds a collision cage to the specified bucket and updates the slot list if a new cage is created.[cite: 6]
func (s *Cage) addToBucket(bucket BucketType, rCage *Cage, rFace IFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) bool {
	return s.addToBucketEntry(bucket, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode) != nil
}

// addToBucketEntry adds a collision entry to the selected bucket and tracks it in the cage slots.
// It returns the created entry, or nil if no entry was added.
func (s *Cage) addToBucketEntry(bucket BucketType, rCage *Cage, rFace IFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) *Entry {
	s.cradle.SetDirty()
	entry := s.buckets[bucket].Add(bucket, s, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode)
	if entry != nil {
		s.slots.Add(entry)
		return entry
	}
	return nil
}

// DiscardPendingFaces clears queued faces without committing them when a speculative query does not hit.
func (s *Cage) DiscardPendingFaces() {
	s.facesIdx = 0
}

// firstSolidDynamicFaces returns the first dynamic face pair queued on the cage where either face is solid.
func (s *Cage) firstSolidDynamicFaces() (IFace, IFace, bool) {
	for i := 0; i < s.facesIdx; i++ {
		f := s.faces[i]
		if f.rFace.IsSolid() || f.lFace.IsSolid() {
			return f.rFace.IFace, f.lFace.IFace, true
		}
	}
	return nil, nil, false
}

// evalDynamicManifold populates s.manifold for a dynamic pair and returns the partner's target geometry.
func (s *Cage) evalDynamicManifold(rCage *Cage) (lUpright, rUpright bool, rTx, rTy, rTz, rDX, rDY, rDZ, rRadX, rRadY, rRadZ float64) {
	lEnt := s.entity
	rEnt := rCage.GetEntity()
	lUpright = lEnt.KeepUpright()
	rUpright = rEnt.KeepUpright()
	s.eRadX, s.eRadY, s.eRadZ = lEnt.HalfExtents()

	rEntDX, rEntDY, rEntDZ := rEnt.GetFrameDisplacement()
	rCX, rCY, rCZ := rEnt.GetCenter()
	rDX, rDY, rDZ = rEntDX, rEntDY, rEntDZ
	if rCage.session == s.session {
		rCX, rCY, rCZ = rCage.cX, rCage.cY, rCage.cZ
		rDX, rDY, rDZ = rCage.dX, rCage.dY, rCage.dZ
	}
	rTx, rTy, rTz = rCX+rDX, rCY+rDY, rCZ+rDZ
	rRadX, rRadY, rRadZ = rEnt.HalfExtents()

	m := s.manifold
	switch {
	case lUpright && rUpright:
		m.ComputeCylinderSweptAtOffset(
			s.cX, s.cY, s.cZ, s.dX, s.dY, s.dZ, s.eRadX, s.eRadZ,
			rCX, rCY, rCZ, rDX, rDY, rDZ, rRadX, rRadZ,
		)
	case lUpright:
		m.ComputeCylinderOBBSweptAtOffset(
			s.cX, s.cY, s.cZ, s.dX, s.dY, s.dZ, s.eRadX, s.eRadZ,
			rEnt.BoundingBox, rEntDX, rEntDY, rEntDZ,
		)
	case rUpright:
		m.ComputeOBBCylinderSweptAtOffset(
			lEnt.BoundingBox, s.dX, s.dY, s.dZ,
			rCX, rCY, rCZ, rDX, rDY, rDZ, rRadX, rRadZ,
		)
	default:
		m.ComputeSweptAtOffset(lEnt.BoundingBox, rEnt.BoundingBox, s.dX, s.dY, s.dZ, rEntDX, rEntDY, rEntDZ)
	}
	return lUpright, rUpright, rTx, rTy, rTz, rDX, rDY, rDZ, rRadX, rRadY, rRadZ
}

// WillHitDynamic reports whether the queued solid faces against rCage produce a real narrow-phase hit.
func (s *Cage) WillHitDynamic(rCage *Cage) bool {
	if _, _, ok := s.firstSolidDynamicFaces(); !ok {
		return false
	}
	s.evalDynamicManifold(rCage)
	m := s.manifold
	return m.GetHit() && m.GetPenetration() > 0 && m.GetCount() > 0
}

// ShouldWakeDynamic reports whether the latest WillHitDynamic manifold warrants waking sleeping rCage.
// Calm resting contacts against a grounded sleeping floor partner avoid triggering pile-wide wake cascades.
func (s *Cage) ShouldWakeDynamic(rCage *Cage) bool {
	lEnt := s.entity
	rEnt := rCage.GetEntity()
	if lEnt.KeepUpright() || !rEnt.IsOnGround() {
		return true
	}
	m := s.manifold
	_, _, nz := m.GetNormal()
	if nz < -0.30 && lEnt.HasSupportMotion(0.0, true) {
		return true
	}
	if lEnt.HasWakeMotion(0.0, true) {
		return true
	}
	penThresh := 0.05
	if nz > 0.30 {
		penThresh = lEnt.FloorWakePenThreshold()
	}
	maxPtPen := 0.0
	cnt := m.GetCount()
	for pi := range cnt {
		if _, _, _, ptPen := m.GetPoint(pi); ptPen > maxPtPen {
			maxPtPen = ptPen
		}
	}
	return maxPtPen > penThresh && nz > -0.70
}

// acquireDynamicManifold evaluates a dynamic collision pair into s.pendingDyn without mutating rCage.
func (s *Cage) acquireDynamicManifold(rCage *Cage) {
	out := s.pending.Current()
	if s.buildPendingDynamic(rCage, out) {
		s.pending.Next()
	}
}

// commitDynamicManifold resolves an overlapping or swept-colliding dynamic pair using upright cylinder
// geometry when KeepUpright is active and 15-axis OBB SAT + conservative advancement otherwise.
func (s *Cage) commitDynamicManifold(rCage *Cage) {
	var pd PendingDynamicContact
	if s.buildPendingDynamic(rCage, &pd) {
		s.applyPendingDynamic(&pd)
	}
}

// buildPendingDynamic populates out with read-only narrow-phase dynamic pair results without mutating rCage.
func (s *Cage) buildPendingDynamic(rCage *Cage, out *PendingDynamicContact) bool {
	rFace, recipFace, ok := s.firstSolidDynamicFaces()
	if !ok {
		return false
	}

	lEnt := s.entity
	rEnt := rCage.GetEntity()
	lUpright, rUpright, rTx, rTy, rTz, _, _, rDZ, rRadX, rRadY, rRadZ := s.evalDynamicManifold(rCage)

	rSleep := rEnt.IsSleeping()
	m := s.manifold
	mNormalX, mNormalY, mNormalZ := m.GetNormal()
	b := Normal2BucketType(mNormalX, mNormalY, mNormalZ)
	if !m.GetHit() {
		xyOverlap := s.overlapsTargetXY(rTx-rRadX, rTx+rRadX, rTy-rRadY, rTy+rRadY)
		if lUpright && rUpright {
			sumR := s.eRadX + rRadX
			dx, dy := s.tX-rTx, s.tY-rTy
			xyOverlap = dx*dx+dy*dy < sumR*sumR
		}
		gap := m.GetGap()
		var selfFloor, partnerFloor bool
		if gap <= floorGroundTolerance && xyOverlap {
			if b == BucketFloor && mNormalZ >= floorNormalThreshold {
				selfFloor = lUpright || lEnt.HasDynamicRestFloorGap(s.dZ, gap, rDZ)
			} else if b == BucketCeiling && mNormalZ <= -floorNormalThreshold {
				partnerFloor = rUpright || lEnt.HasDynamicRestFloorGap(rDZ, gap, s.dZ)
			}
		}
		if !selfFloor && !partnerFloor {
			return false
		}
		out.partner = rCage
		out.rFace = rFace
		out.recipFace = recipFace
		out.selfFloor = selfFloor
		out.partnerFloor = partnerFloor
		out.wasSleeping = rSleep
		out.hit = false
		out.pointCount = -1
		return true
	}

	penetration := m.GetPenetration()
	count := m.GetCount()
	dist := -penetration
	selfFloor := b == BucketFloor && mNormalZ >= floorNormalThreshold && dist <= floorGroundTolerance
	partnerFloor := b == BucketCeiling && mNormalZ <= -floorNormalThreshold && dist <= floorGroundTolerance

	if penetration <= 0 || count == 0 {
		if !selfFloor && !partnerFloor {
			return false
		}
		out.partner = rCage
		out.rFace = rFace
		out.recipFace = recipFace
		out.selfFloor = selfFloor
		out.partnerFloor = partnerFloor
		out.wasSleeping = rSleep
		out.hit = false
		out.pointCount = 0
		return true
	}

	out.partner = rCage
	out.rFace = rFace
	out.recipFace = recipFace
	out.normalX = mNormalX
	out.normalY = mNormalY
	out.normalZ = mNormalZ
	out.penetration = penetration
	out.rMaxZ = rTz + rRadZ
	out.selfFloor = selfFloor
	out.partnerFloor = partnerFloor
	out.wasSleeping = rSleep
	out.hit = true
	out.pointCount = count
	for pi := range count {
		out.points[pi][0], out.points[pi][1], out.points[pi][2], out.points[pi][3] = m.GetPoint(pi)
	}
	return true
}

// applyPendingDynamic commits a previously captured dynamic contact pair into both cages' buckets and slots.
func (s *Cage) applyPendingDynamic(pd *PendingDynamicContact) {
	rCage := pd.partner
	lEnt := s.entity
	rEnt := rCage.GetEntity()
	lSleep := lEnt.IsSleeping()
	rSleep := rEnt.IsSleeping()

	restGap := !pd.hit && pd.pointCount < 0
	if pd.selfFloor && (restGap || !lSleep) {
		s.hasFloor = true
	}
	if pd.partnerFloor && (restGap || !rSleep) {
		rCage.hasFloor = true
	}
	if !pd.hit {
		return
	}

	b := Normal2BucketType(pd.normalX, pd.normalY, pd.normalZ)
	rMaxZ := pd.rMaxZ
	lMaxZ := s.tZ + s.eRadZ
	iMode := ImpactElastic
	if b.IsWall() && (lEnt.KeepUpright() || s.maxStep > 0) {
		baseZ := s.GetBaseZ()
		minBaseZ := math.Min(baseZ, baseZ+s.dZ)
		if rMaxZ <= minBaseZ+satFilterEpsilon {
			return
		}
		stepZ := baseZ + s.maxStep
		if s.maxStep > 0 && rMaxZ <= stepZ {
			iMode = ImpactStep
		}
	}

	oppB := b.Opposite()
	for pi := 0; pi < pd.pointCount; pi++ {
		pt := &pd.points[pi]
		ptPen := pt[3]
		if ptPen <= 0 {
			continue
		}
		ptX, ptY, ptZ := pt[0], pt[1], pt[2]
		ptDist := -ptPen
		if !lSleep {
			s.addToBucket(b, rCage, pd.rFace, ptDist, ptPen, pd.normalX, pd.normalY, pd.normalZ, ptX, ptY, ptZ, rMaxZ, iMode)
		}
		if iMode == ImpactElastic && !rSleep {
			rCage.addToBucket(oppB, s, pd.recipFace, ptDist, ptPen, -pd.normalX, -pd.normalY, -pd.normalZ, ptX, ptY, ptZ, lMaxZ, ImpactElastic)
			if oppB == BucketFloor {
				rCage.hasFloor = true
			}
		}
	}

	switch b {
	case BucketFloor:
		if !lSleep {
			s.hasFloor = true
		}
	case BucketCeiling:
		if !rSleep {
			rCage.hasFloor = true
		}
	}
}

// TranslateWorldToLocal transforms the world-space AABB of the ellipsoid to local-space relative to the given delta values.[cite: 6]
func (s *Cage) TranslateWorldToLocal(slot int, deltaX, deltaY, deltaZ float64) *rigid_body.Entity {
	w := s.ellipsoidLocal[slot]
	from := s.ellipsoid.GetAABB()
	lMinX := from.GetMinX() - deltaX
	lMaxX := from.GetMaxX() - deltaX
	lMinY := from.GetMinY() - deltaY
	lMaxY := from.GetMaxY() - deltaY
	lMinZ := from.GetMinZ() - deltaZ
	lMaxZ := from.GetMaxZ() - deltaZ
	w.Rebuild(lMinX, lMinY, lMinZ, lMaxX-lMinX, lMaxY-lMinY, lMaxZ-lMinZ)
	return w
}

// TranslateCage translates the collision cage based on the given slot and reference cage, returning the local AABB and offsets.[cite: 6]
func (s *Cage) TranslateCage(slot int, rCage *Cage) (aabb.IObject, float64, float64, float64) {
	lEnt := s.entity
	lOx, lOy, lOz := lEnt.GetCenter()
	lOx += s.dX
	lOy += s.dY
	lOz += s.dZ
	rEnt := rCage.GetEntity()
	rOx, rOy, rOz := rEnt.GetCenter()
	rDX, rDY, rDZ := rEnt.GetFrameDisplacement()
	rOx += rDX
	rOy += rDY
	rOz += rDZ

	from := s.ellipsoid.GetAABB()
	wMinX, wMaxX := from.GetMinX()-rOx, from.GetMaxX()-rOx
	wMinY, wMaxY := from.GetMinY()-rOy, from.GetMaxY()-rOy
	wMinZ, wMaxZ := from.GetMinZ()-rOz, from.GetMaxZ()-rOz

	wCx, wCy, wCz := (wMinX+wMaxX)*0.5, (wMinY+wMaxY)*0.5, (wMinZ+wMaxZ)*0.5
	wHx, wHy, wHz := (wMaxX-wMinX)*0.5, (wMaxY-wMinY)*0.5, (wMaxZ-wMinZ)*0.5

	a00, a01, a02, a10, a11, a12, a20, a21, a22 := rEnt.GetAxes()
	locCx := wCx*a00 + wCy*a01 + wCz*a02
	locCy := wCx*a10 + wCy*a11 + wCz*a12
	locCz := wCx*a20 + wCy*a21 + wCz*a22

	locHx := math.Abs(a00)*wHx + math.Abs(a01)*wHy + math.Abs(a02)*wHz
	locHy := math.Abs(a10)*wHx + math.Abs(a11)*wHy + math.Abs(a12)*wHz
	locHz := math.Abs(a20)*wHx + math.Abs(a21)*wHy + math.Abs(a22)*wHz

	w := s.ellipsoidLocal[slot]
	w.Rebuild(locCx-locHx, locCy-locHy, locCz-locHz, locHx*2.0, locHy*2.0, locHz*2.0)
	return w, rOx - lOx, rOy - lOy, rOz - lOz
}

// Translate updates the local ellipsoid of the given slot based on the provided AABB source and displacement values.[cite: 6]
func (s *Cage) Translate(slot int, src aabb.IObject, deltaX, deltaY, deltaZ float64) *rigid_body.Entity {
	w := s.ellipsoidLocal[slot]
	to := src.GetAABB()
	rLocCx := (to.GetMinX() + to.GetMaxX()) * 0.5
	rLocCy := (to.GetMinY() + to.GetMaxY()) * 0.5
	rLocCz := (to.GetMinZ() + to.GetMaxZ()) * 0.5
	rLocHx := (to.GetMaxX() - to.GetMinX()) * 0.5
	rLocHy := (to.GetMaxY() - to.GetMinY()) * 0.5
	rLocHz := (to.GetMaxZ() - to.GetMinZ()) * 0.5

	wCx, wCy, wCz := rLocCx+deltaX, rLocCy+deltaY, rLocCz+deltaZ
	wHx, wHy, wHz := rLocHx, rLocHy, rLocHz

	lEnt := s.entity
	if f, ok := src.(IFace); ok {
		nX, nY, nZ := f.GetNormal()
		if p := f.GetParent(); p != nil {

			if rEnt := p.GetEntity(); rEnt != nil {
				rA00, rA01, rA02, rA10, rA11, rA12, rA20, rA21, rA22 := rEnt.GetAxes()
				wCx = rA00*rLocCx + rA10*rLocCy + rA20*rLocCz + deltaX
				wCy = rA01*rLocCx + rA11*rLocCy + rA21*rLocCz + deltaY
				wCz = rA02*rLocCx + rA12*rLocCy + rA22*rLocCz + deltaZ
				wHx = math.Abs(rA00)*rLocHx + math.Abs(rA10)*rLocHy + math.Abs(rA20)*rLocHz
				wHy = math.Abs(rA01)*rLocHx + math.Abs(rA11)*rLocHy + math.Abs(rA21)*rLocHz
				wHz = math.Abs(rA02)*rLocHx + math.Abs(rA12)*rLocHy + math.Abs(rA22)*rLocHz
				fnX, fnY, fnZ := nX, nY, nZ
				nX = rA00*fnX + rA10*fnY + rA20*fnZ
				nY = rA01*fnX + rA11*fnY + rA21*fnZ
				nZ = rA02*fnX + rA12*fnY + rA22*fnZ
				rDX, rDY, rDZ := rEnt.GetFrameDisplacement()
				wHx += math.Abs(rDX - s.dX)
				wHy += math.Abs(rDY - s.dY)
				wHz += math.Abs(rDZ - s.dZ)
			}
		}
		sx, sy, sz := s.sweptHalfExtents(lEnt)
		wHx += math.Abs(nX) * sx * 2.0
		wHy += math.Abs(nY) * sy * 2.0
		wHz += math.Abs(nZ) * sz * 2.0
	}

	lA00, lA01, lA02, lA10, lA11, lA12, lA20, lA21, lA22 := lEnt.GetAxes()
	lLocCx := wCx*lA00 + wCy*lA01 + wCz*lA02
	lLocCy := wCx*lA10 + wCy*lA11 + wCz*lA12
	lLocCz := wCx*lA20 + wCy*lA21 + wCz*lA22

	lLocHx := math.Abs(lA00)*wHx + math.Abs(lA01)*wHy + math.Abs(lA02)*wHz
	lLocHy := math.Abs(lA10)*wHx + math.Abs(lA11)*wHy + math.Abs(lA12)*wHz
	lLocHz := math.Abs(lA20)*wHx + math.Abs(lA21)*wHy + math.Abs(lA22)*wHz

	w.Rebuild(lLocCx-lLocHx, lLocCy-lLocHy, lLocCz-lLocHz, lLocHx*2.0, lLocHy*2.0, lLocHz*2.0)
	return w
}

// HasSeen checks if the given Cage has already been marked as seen in the current Cage.[cite: 6]
func (s *Cage) HasSeen(rCage *Cage) bool {
	if rCage.id >= uint64(len(s.seen)) {
		return s.session == 0
	}
	return s.seen[rCage.id] == s.session
}

// Seen marks a given Cage as observed by the current Cage within its internal tracking.[cite: 6]
func (s *Cage) Seen(rCage *Cage) {
	if rCage.id >= uint64(len(s.seen)) {
		s.growSeen(rCage.id)
	}
	s.seen[rCage.id] = s.session
}

// GetBaseZ computes the base Z value of the collision cage by subtracting the ellipsoid radius in Z direction from its center Z coordinate.[cite: 6]
func (s *Cage) GetBaseZ() float64 { return s.cZ - s.eRadZ }

// HasBucket reports whether the collision cage contains a bucket of the specified type.[cite: 6]
func (s *Cage) HasBucket(t BucketType) bool {
	return s.slots.HasBucket(t)
}

// GetSlotsLen returns the current number of active slots in the Cage.[cite: 6]
func (s *Cage) GetSlotsLen() int { return s.slots.Len() }

// GetSlot returns the Entry at the specified index in the slots slice.[cite: 6]
func (s *Cage) GetSlot(i int) *Entry { return s.slots.At(i) }

// GetVolumeObject returns the cage volume object, or nil if no volume object is set.
func (s *Cage) GetVolumeObject() aabb.IObject { return s.volumeObj }

// GetRad retrieves the ellipsoid radii along the X, Y, and Z axes.[cite: 6]
func (s *Cage) GetRad() (float64, float64, float64) { return s.eRadX, s.eRadY, s.eRadZ }

func (s *Cage) GetRadZ() float64 { return s.eRadZ }

// GetC returns the center coordinates (cX, cY, cZ) of the Cage.[cite: 6]
func (s *Cage) GetC() (float64, float64, float64) { return s.cX, s.cY, s.cZ }

// GetDisplacement returns the displacement of the collision cage along the x, y, and z axes.[cite: 6]
func (s *Cage) GetDisplacement() (float64, float64, float64) { return s.stepDX, s.stepDY, s.stepDZ }

// GetT retrieves the current transformed coordinates (tX, tY, tZ) of the Cage.[cite: 6]
func (s *Cage) GetT() (float64, float64, float64) { return s.tX, s.tY, s.tZ }

// GetNetDisplacement returns the net center shift across the entire tick from start-of-step (cX, cY, cZ).
func (s *Cage) GetNetDisplacement() (float64, float64, float64) {
	curX, curY, curZ := s.entity.GetCenter()
	return curX - s.cX, curY - s.cY, curZ - s.cZ
}

func (s *Cage) GetTZ() float64 { return s.tZ }

// BucketCount returns the number of entries in the specified bucket.[cite: 6]
func (s *Cage) BucketCount(t BucketType) int { return s.buckets[t].Count() }

// HasEntryFor reports whether the specified bucket holds an active dynamic entry for entityId.
func (s *Cage) HasEntryFor(t BucketType, entityId uint64) bool {
	return s.buckets[t].HasEntryFor(entityId)
}

// IsBucketPenetrable reports whether pushing along bucket b's blocking direction is free of inelastic/pinned obstacles.
func (s *Cage) IsBucketPenetrable(b BucketType, from uint64) bool {
	return s.buckets[b].Penetrable(from, 0)
}

// NeedsStaticClamp reports whether the entity center moved or KeepUpright toggled since the last validated static clamp.
func (s *Cage) NeedsStaticClamp() bool {
	if s.solver != nil && s.solver.IgnoresStatic() {
		return false
	}
	if s.forceStaticClamp || s.entity.KeepUpright() != s.prevKeepUpright {
		return true
	}
	curX, curY, curZ := s.entity.GetCenter()
	dx := curX - s.clampX
	dy := curY - s.clampY
	dz := curZ - s.clampZ
	const clampToleranceSq = 1e-6
	return dx*dx+dy*dy+dz*dz > clampToleranceSq
}

// CommitStaticClamp records the current entity center and KeepUpright state as the latest statically clamped baseline.
func (s *Cage) CommitStaticClamp() {
	s.clampX, s.clampY, s.clampZ = s.entity.GetCenter()
	s.prevKeepUpright = s.entity.KeepUpright()
	s.forceStaticClamp = false
}

// IsBlockedAlong reports whether pushing this cage along unit direction (dirX, dirY, dirZ)
// pushes into any active or clamped static world face.
func (s *Cage) IsBlockedAlong(dirX, dirY, dirZ float64) bool {
	const wallBlockDotThreshold = -0.85
	if s.staticContact.IsBlockedAlong(dirX, dirY, dirZ, wallBlockDotThreshold) {
		return true
	}
	return s.slots.HasImpactInelastic(dirX, dirY, dirZ, wallBlockDotThreshold)
}

// sweptHalfExtents returns the broad-phase half-extents for s without inflating yaw-invariant KeepUpright entities.
func (s *Cage) sweptHalfExtents(ent *rigid_body.Entity) (float64, float64, float64) {
	if ent.KeepUpright() {
		return s.eRadX, s.eRadY, s.eRadZ
	}
	sx, sy, sz := ent.GetSweptHalfExtents()
	if sx < s.eRadX {
		sx = s.eRadX
	}
	if sy < s.eRadY {
		sy = s.eRadY
	}
	if sz < s.eRadZ {
		sz = s.eRadZ
	}
	return sx, sy, sz
}

// BeginStaticClamp prepares a temporary swept broad-phase cage from the last valid clamped center to the current entity center,
// returning saved frame center, displacement, and target values for restoration in EndStaticClamp.
func (s *Cage) BeginStaticClamp() (origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ float64, lAABB *aabb.AABB) {
	origCX, origCY, origCZ = s.cX, s.cY, s.cZ
	origDX, origDY, origDZ = s.dX, s.dY, s.dZ
	origTX, origTY, origTZ = s.tX, s.tY, s.tZ
	ent := s.entity
	s.eRadX, s.eRadY, s.eRadZ = ent.HalfExtents()
	curX, curY, curZ := ent.GetCenter()
	s.cX, s.cY, s.cZ = s.clampX, s.clampY, s.clampZ
	s.tX, s.tY, s.tZ = curX, curY, curZ
	s.dX, s.dY, s.dZ = s.tX-s.cX, s.tY-s.cY, s.tZ-s.cZ
	s.passDX, s.passDY, s.passDZ = s.dX, s.dY, s.dZ
	sx, sy, sz := s.sweptHalfExtents(ent)
	minX := s.cX - sx + math.Min(0.0, s.dX)
	maxX := s.cX + sx + math.Max(0.0, s.dX)
	minY := s.cY - sy + math.Min(0.0, s.dY)
	maxY := s.cY + sy + math.Max(0.0, s.dY)
	minZ := s.cZ - sz + math.Min(0.0, s.dZ)
	maxZ := s.cZ + sz + math.Max(0.0, s.dZ)
	s.ellipsoid.Rebuild(minX, minY, minZ, maxX-minX, maxY-minY, maxZ-minZ)
	s.staticContact.ResetFlatWalls()
	s.staticContact.ResetClampCorners()
	return origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ, s.ellipsoid.GetAABB()
}

// EndStaticClamp restores the pre-clamp cage center, displacement, and target fields after post-push static clamping.
func (s *Cage) EndStaticClamp(origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ float64) {
	s.cX, s.cY, s.cZ = origCX, origCY, origCZ
	s.dX, s.dY, s.dZ = origDX, origDY, origDZ
	s.passDX, s.passDY, s.passDZ = origDX, origDY, origDZ
	s.tX, s.tY, s.tZ = origTX, origTY, origTZ
}

// overlapsTargetXY reports whether the target XY footprint at tX, tY overlaps [minX, maxX] x [minY, maxY].
func (s *Cage) overlapsTargetXY(minX, maxX, minY, maxY float64) bool {
	return s.tX+s.eRadX > minX && maxX > s.tX-s.eRadX && s.tY+s.eRadY > minY && maxY > s.tY-s.eRadY
}

// effectiveRadius returns the half-extent along unit normal (nX, nY, nZ),
// using ellipsoid support for KeepUpright entities and exact OBB axis projection for all rigid OBBs.
func effectiveRadius(ent *rigid_body.Entity, eRadX, eRadY, eRadZ, nX, nY, nZ float64) float64 {
	if !ent.KeepUpright() {
		return ent.ProjectExtent(nX, nY, nZ)
	}
	return math.Sqrt((nX*eRadX)*(nX*eRadX) + (nY*eRadY)*(nY*eRadY) + (nZ*eRadZ)*(nZ*eRadZ))
}

// GetAABB returns the axis-aligned bounding box (AABB) representing the broad-phase collision geometry of the cage.[cite: 6]
func (s *Cage) GetAABB() *aabb.AABB { return s.ellipsoid.GetAABB() }

// HasFloor reports whether the collision cage has floor or multi-contact cradle support.[cite: 6]
func (s *Cage) HasFloor() bool {
	if s.hasFloor {
		return true
	}
	if s.entity.KeepUpright() || s.slots.Len() < 2 {
		return false
	}
	return s.cradle.HasSupport(s.slots)
}

// SetSolverIndex sets the temporary per-tick solver index of this cage (-1 when inactive).
func (s *Cage) SetSolverIndex(idx int) {
	s.solverIdx = idx
}

// GetSolverIndex returns the temporary per-tick solver index of this cage (-1 when inactive).
func (s *Cage) GetSolverIndex() int {
	return s.solverIdx
}

// SetCollisionGroup assigns a non-zero skeleton/island collision filter group ID.
func (s *Cage) SetCollisionGroup(group uint64) {
	s.collisionGroup = group
}

// GetCollisionGroup returns this cage's collision filter group ID (0 means default, collides with all).
func (s *Cage) GetCollisionGroup() uint64 {
	return s.collisionGroup
}

// GetSolver returns the owning Solver bound to this cage, or nil when unbound.
func (s *Cage) GetSolver() *Solver {
	return s.solver
}

// ZeroDisplacement resets any pending displacement delta and frame push baselines on this cage when entering sleep.
func (s *Cage) ZeroDisplacement() {
	s.dX, s.dY, s.dZ = 0, 0, 0
	s.stepDX, s.stepDY, s.stepDZ = 0, 0, 0
	s.passDX, s.passDY, s.passDZ = 0, 0, 0
	s.applied = false
	if s.entity != nil {
		curX, curY, curZ := s.entity.GetCenter()
		s.cX, s.cY, s.cZ = curX, curY, curZ
		s.tX, s.tY, s.tZ = curX, curY, curZ
		s.appliedX, s.appliedY, s.appliedZ = curX, curY, curZ
	}
}

// SetSpawnOrigin records a safe pre-spawn or pre-rotation origin for the next Rebuild sweep and static clamp.
func (s *Cage) SetSpawnOrigin(x, y, z float64) {
	s.spawnX, s.spawnY, s.spawnZ = x, y, z
	s.hasSpawnOrigin = true
	s.forceStaticClamp = true
}

// HasSpawnOrigin reports whether a pending spawn origin is set on this cage.
func (s *Cage) HasSpawnOrigin() bool {
	return s.hasSpawnOrigin
}

// GetEntity returns the physics entity associated with the collision cage.[cite: 6]
func (s *Cage) GetEntity() *rigid_body.Entity {
	return s.entity
}

// SetApplied marks the collision cage as having applied its frame displacement and records entity center for tracking push-out offset.[cite: 6]
func (s *Cage) SetApplied() {
	s.applied = true
	lEnt := s.entity
	s.appliedX, s.appliedY, s.appliedZ = lEnt.GetCenter()
}

// SetPendingStepZ stores a pending vertical step-up candidate computed in StageApply.
func (s *Cage) SetPendingStepZ(stepZ float64) {
	s.pendingStepZ = stepZ
}

// ConsumePendingStepZ returns and clears any pending vertical step-up candidate.
func (s *Cage) ConsumePendingStepZ() float64 {
	v := s.pendingStepZ
	s.pendingStepZ = 0
	return v
}

// ApplySafeStepUp verifies vertical headroom for a step-up before keeping it so beveled walls under
// ceilings or overhangs do not launch the entity into ceiling clamps that kick it backward.
func (s *Cage) ApplySafeStepUp(stepZ float64, clampStatic func(*Cage)) {
	ent := s.GetEntity()
	preX, preY, preZ := ent.GetCenter()
	preVX, preVY, preVZ := ent.GetVelocity()
	origCX, origCY, origCZ, origHasFloor := s.GetStaticClampState()

	s.RestoreStaticClampState(preX, preY, preZ, origHasFloor)
	ent.AddTo(0, 0, stepZ)
	clampStatic(s)
	postX, postY, postZ := ent.GetCenter()

	dzAfter := postZ - preZ
	dXY := math.Hypot(postX-preX, postY-preY)
	if dzAfter < stepZ-0.25 || dXY > 2.0 {
		ent.AddTo(preX-postX, preY-postY, preZ-postZ)
		ent.SetV(preVX, preVY, preVZ)
		s.RestoreStaticClampState(origCX, origCY, origCZ, origHasFloor)
		return
	}
	ent.SetV(preVX, preVY, preVZ)
	s.RestoreStaticClampState(origCX, origCY, origCZ, origHasFloor)
}

// GetStaticClampState returns the current static clamp baseline coordinates and floor flag.
func (s *Cage) GetStaticClampState() (float64, float64, float64, bool) {
	return s.clampX, s.clampY, s.clampZ, s.hasFloor
}

// RestoreStaticClampState restores the static clamp baseline coordinates and floor flag.
func (s *Cage) RestoreStaticClampState(cx, cy, cz float64, hasFloor bool) {
	s.clampX, s.clampY, s.clampZ = cx, cy, cz
	s.hasFloor = hasFloor
}

// GetPushOffset returns the accumulated positional push-out offset of the entity in the current frame.[cite: 6]
func (s *Cage) GetPushOffset() (float64, float64, float64) {
	lEnt := s.entity
	curX, curY, curZ := lEnt.GetCenter()
	if s.applied {
		return curX - s.appliedX, curY - s.appliedY, curZ - s.appliedZ
	}
	return curX - s.cX, curY - s.cY, curZ - s.cZ
}

// evalStaticFace evaluates a static world triangle against the cage sweep (bucket, plane normal,
// signed separation distance, anti-phantom-plane filtered penetration, and world face bounds) into out
// without heap allocation or large value copies in the hot path.
func (s *Cage) evalStaticFace(lAABB *aabb.AABB, rFace *Face, out *StaticFaceContact) bool {
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return false
	}

	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	p0x, p0y, p0z := rFace.Get0()
	distStart := (s.cX-p0x)*nX + (s.cY-p0y)*nY + (s.cZ-p0z)*nZ
	rayEff := effectiveRadius(s.entity, s.eRadX, s.eRadY, s.eRadZ, nX, nY, nZ)

	flipped := false
	var bucket BucketType
	if solidWE || solidNS {
		if distStart < -satFilterEpsilon {
			if rFace.IsSingleSided() {
				if distStart < -rayEff {
					return false
				}
				if !s.entity.KeepUpright() {
					impDotOrig := s.passDX*nX + s.passDY*nY + s.passDZ*nZ
					distT := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
					if distStart <= -0.5*rayEff || impDotOrig >= -satFilterEpsilon || !rFace.IntersectsFiniteTriangle3D(s.eRadX, s.eRadY, s.eRadZ, s.cX, s.cY, s.cZ, s.dX, s.dY, s.dZ, s.tX, s.tY, s.tZ, distStart, distT, false) {
						return false
					}
				}
			} else {
				impDotOrig := s.passDX*nX + s.passDY*nY + s.passDZ*nZ
				if distStart <= -0.5*rayEff || impDotOrig > satFilterEpsilon {
					nX, nY, nZ = -nX, -nY, -nZ
					distStart = -distStart
					flipped = true
				} else if !s.entity.KeepUpright() {
					distT := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
					if impDotOrig >= -satFilterEpsilon || !rFace.IntersectsFiniteTriangle3D(s.eRadX, s.eRadY, s.eRadZ, s.cX, s.cY, s.cZ, s.dX, s.dY, s.dZ, s.tX, s.tY, s.tZ, distStart, distT, false) {
						return false
					}
				}
			}
		}
		if solidWE {
			if nX < 0 {
				bucket = BucketWallWest
			} else {
				bucket = BucketWallEast
			}
		} else {
			if nY < 0 {
				bucket = BucketWallNorth
			} else {
				bucket = BucketWallSouth
			}
		}
	} else {
		planeZ := p0z
		if nAbsZ > 1e-5 {
			planeZ = p0z - (nX*(s.cX-p0x)+nY*(s.cY-p0y))/nZ
		}
		if s.cZ >= planeZ {
			bucket = BucketFloor
			if nZ < 0 {
				nX, nY, nZ = -nX, -nY, -nZ
				distStart = -distStart
			}
		} else {
			bucket = BucketCeiling
			if nZ > 0 {
				nX, nY, nZ = -nX, -nY, -nZ
				distStart = -distStart
			}
		}
	}

	distTarget := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
	dist := distTarget - rayEff
	penetration := rayEff - distTarget

	rFaceAABB := rFace.GetAABB()
	rMinX := rFaceAABB.GetMinX()
	rMaxX := rFaceAABB.GetMaxX()
	rMinY := rFaceAABB.GetMinY()
	rMaxY := rFaceAABB.GetMaxY()
	rMinZ := rFaceAABB.GetMinZ()
	rMaxZ := rFaceAABB.GetMaxZ()

	oX := math.Max(0.0, math.Min(lAABB.GetMaxX()-rMinX, rMaxX-lAABB.GetMinX()))
	oY := math.Max(0.0, math.Min(lAABB.GetMaxY()-rMinY, rMaxY-lAABB.GetMinY()))
	if oX <= 0 || oY <= 0 {
		return false
	}

	out.bucket = bucket
	out.dist = dist
	out.nX, out.nY, out.nZ = nX, nY, nZ
	out.anchorX, out.anchorY, out.anchorZ = p0x, p0y, p0z
	out.origNX, out.origNY, out.origNZ = nX, nY, nZ
	out.origAX, out.origAY, out.origAZ = p0x, p0y, p0z
	out.minX, out.maxX = rMinX, rMaxX
	out.minY, out.maxY = rMinY, rMaxY
	out.minZ, out.maxZ = rMinZ, rMaxZ
	out.cornerHit = false
	out.hasContact = false
	out.wallTopLip = false

	if penetration <= 0 {
		if bucket == BucketFloor && dist <= floorGroundTolerance {
			tBaseZ := s.tZ - rayEff
			if nZ < floorNormalThreshold ||
				!s.overlapsTargetXY(rMinX, rMaxX, rMinY, rMaxY) ||
				rMaxZ < tBaseZ-floorGroundTolerance-satFilterEpsilon ||
				rMinZ > s.tZ+satFilterEpsilon {
				return false
			}
		}
		out.pen = 0
		return true
	}

	oZ := math.Max(0.0, math.Min(lAABB.GetMaxZ()-rMinZ, rMaxZ-lAABB.GetMinZ()))
	if oZ <= 0 {
		return false
	}

	if s.entity.KeepUpright() {
		valid, adjusted, cornerHit, cxN, cyN, czN, cPen, qxW, qyW, qzW, hasPt := rFace.EvalKeepUprightFace(s.entity, bucket, solidWE, solidNS, s.eRadX, s.eRadY, s.eRadZ, s.cX, s.cY, s.cZ, s.dX, s.dY, s.dZ, s.tX, s.tY, s.tZ, nX, nY, nZ, distStart, distTarget, penetration)
		if !valid || ((solidWE || solidNS) && !cornerHit && !flipped && distStart < -satFilterEpsilon) {
			return false
		}
		if hasPt {
			out.contactX, out.contactY, out.contactZ = qxW, qyW, qzW
			out.hasContact = true
		}
		if adjusted {
			nX, nY, nZ = cxN, cyN, czN
			nAbsX, nAbsY, nAbsZ = math.Abs(nX), math.Abs(nY), math.Abs(nZ)
			solidWE = nAbsX > nAbsY && nAbsX > nAbsZ
			solidNS = !solidWE && nAbsY > nAbsZ
			bucket = Normal2BucketType(nX, nY, nZ)
			rayEff = effectiveRadius(s.entity, s.eRadX, s.eRadY, s.eRadZ, nX, nY, nZ)
			penetration = cPen
			dist = -cPen
			distTarget = rayEff - cPen
			distStart = distTarget - (s.dX*nX + s.dY*nY + s.dZ*nZ)
			out.bucket = bucket
			out.dist = dist
			out.nX, out.nY, out.nZ = nX, nY, nZ
			out.anchorX, out.anchorY, out.anchorZ = qxW, qyW, qzW
			out.cornerHit = cornerHit
		}
	}

	// Calcolo TOI (Time Of Impact) tra il moto dell'ellissoide e la faccia statica[cite: 6]
	rCx := (rMinX + rMaxX) * 0.5
	rCy := (rMinY + rMaxY) * 0.5
	rCz := (rMinZ + rMaxZ) * 0.5
	rRadX := (rMaxX - rMinX) * 0.5
	rRadY := (rMaxY - rMinY) * 0.5
	rRadZ := (rMaxZ - rMinZ) * 0.5

	tEnter, tExit, _, _, _, ok := ComputeTOI(s.cX-rCx, s.cY-rCy, s.cZ-rCz, s.eRadX+rRadX, s.eRadY+rRadY, s.eRadZ+rRadZ, s.dX, s.dY, s.dZ)
	if !ok {
		return false
	}

	impDot := s.dX*nX + s.dY*nY + s.dZ*nZ
	tPlaneEnter, tPlaneExit, okPlane := SweptAxisInterval(distStart, rayEff, impDot)
	if !okPlane {
		return false
	}

	overlapStart := math.Max(tEnter, tPlaneEnter)
	overlapEnd := math.Min(math.Min(1.0, tExit), tPlaneExit)
	if overlapStart > overlapEnd+toiTieEpsilon || overlapEnd < -toiTieEpsilon || overlapStart > 1.0+toiTieEpsilon {
		return false
	}

	// Separazione tra la compenetrazione già presente a t = 0 (pen0) e l'affondamento prodotto dal moto nel frame (deltaPen).
	// Un piano infinito è fantasma se la compenetrazione proiettata eccede l'overlap fisico effettivo dell'AABB
	// lungo l'asse principale della normale del piano.[cite: 6]
	pen0 := math.Max(0.0, rayEff-distStart)
	deltaPen := math.Max(0.0, penetration-pen0)

	const multiAxisThreshold = 0.05
	normOX := oX
	if nAbsX > 1e-5 && nAbsX < 1.0 {
		normOX = oX / nAbsX
	}
	normOY := oY
	if nAbsY > 1e-5 && nAbsY < 1.0 {
		normOY = oY / nAbsY
	}
	normOZ := oZ
	if nAbsZ > 1e-5 && nAbsZ < 1.0 {
		normOZ = oZ / nAbsZ
	}
	var axisOverlap float64
	if solidWE {
		axisOverlap = normOX
	} else if solidNS {
		axisOverlap = normOY
	} else {
		axisOverlap = normOZ
	}
	if distTarget >= 0.0 {
		activeAxes := 0
		if nAbsX > multiAxisThreshold {
			activeAxes++
		}
		if nAbsY > multiAxisThreshold {
			activeAxes++
		}
		if nAbsZ > multiAxisThreshold {
			activeAxes++
		}
		if activeAxes > 1 {
			if nAbsX > multiAxisThreshold && normOX < axisOverlap {
				axisOverlap = normOX
			}
			if nAbsY > multiAxisThreshold && normOY < axisOverlap {
				axisOverlap = normOY
			}
			if nAbsZ > multiAxisThreshold && normOZ < axisOverlap {
				axisOverlap = normOZ
			}
		}
	}
	// Cap steep upward spike-tip pushout by vertical clearance only when an upward spike tip pokes beneath the body top.
	if (solidWE || solidNS) && nZ >= 0.6 && (s.tZ-s.eRadZ) > rMinZ && rMaxZ < s.tZ+s.eRadZ {
		if oz := oZ / nZ; oz < axisOverlap {
			axisOverlap = oz
		}
	}
	minOverlap := axisOverlap + deltaPen

	if !s.entity.KeepUpright() {
		if !rFace.IntersectsFiniteTriangleOBB(s.entity, s.cX, s.cY, s.cZ, s.dX, s.dY, s.dZ, s.tX, s.tY, s.tZ, distStart, distTarget) {
			return false
		}
		if distTarget >= -satFilterEpsilon {
			if clipPen, clipMinZ, hasClip := out.prism.Penetration(s, rFace, s.tX, s.tY, s.tZ, nX, nY, nZ); hasClip {
				if bucket.IsWall() && rMaxZ-clipMinZ <= floorGroundTolerance {
					effPreMinZ := clipMinZ - s.dZ
					if _, preMinZ, preHas := out.prism.Penetration(s, rFace, s.cX, s.cY, s.cZ, nX, nY, nZ); preHas {
						effPreMinZ = preMinZ
					}
					if (rMaxZ <= clipMinZ+satFilterEpsilon && effPreMinZ+s.maxStep >= rMaxZ-satFilterEpsilon) ||
						(effPreMinZ >= rMaxZ-satFilterEpsilon && s.dZ < -satFilterEpsilon) {
						out.wallTopLip = true
					}
				}
				if clipPen < penetration {
					penetration = clipPen
				}
				if penetration <= 0.0 {
					out.pen = 0.0
					return true
				}
			}
			if penetration > minOverlap {
				penetration = minOverlap
			}
		}
	}

	out.pen = penetration
	return true
}

// growSeen expands the seen-session tracking slice to accommodate idx while preserving existing marks.
func (s *Cage) growSeen(idx uint64) {
	newCap := uint64(len(s.seen)) * 2
	if newCap <= idx {
		newCap = idx + 64
	}
	n := make([]uint64, newCap)
	copy(n, s.seen)
	s.seen = n
}
