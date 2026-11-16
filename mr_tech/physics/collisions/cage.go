package collisions

import (
	"math"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// ImpactMode identifies how a collision impact should be resolved and prioritized.
type ImpactMode int

type IObject interface {
	GetAABB() *aabb.AABB

	GetEntity() *rigid_body.Entity
}

type IFace interface {
	GetAABB() *aabb.AABB

	GetNormal() (float64, float64, float64)

	GetNormalAbs() (float64, float64, float64)

	Translate(int, float64, float64, float64) (float64, float64, float64)

	GetParent() IObject

	Get0() (float64, float64, float64)

	Get1() (float64, float64, float64)

	Get2() (float64, float64, float64)

	IsSolid() bool
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

// Cage represents a data structure for managing collision information for a physical entity.[cite: 6]
type Cage struct {
	seen                []uint64
	session             uint64
	id                  uint64
	object              IObject
	buckets             [BucketSize]*Bucket
	ellipsoid           *rigid_body.Entity
	ellipsoidLocal      [4]*rigid_body.Entity
	cX, cY, cZ          float64
	dX, dY, dZ          float64
	tX, tY, tZ          float64
	eRadX, eRadY, eRadZ float64
	volume              IObject
	distance            float64
	slots               []*Entry
	slotsLen            int
	maxStep             float64
	faces               []*Faces
	facesIdx            int
	manifold            *aabb.OBBManifold
	hasFloor            bool
	applied             bool
	appliedX            float64
	appliedY            float64
	appliedZ            float64
	clampX              float64
	clampY              float64
	clampZ              float64
	objOffX             float64
	objOffY             float64
	objOffZ             float64
	prevOffX            float64
	prevOffY            float64
	prevOffZ            float64
	prevRadX            float64
	prevRadY            float64
	prevRadZ            float64
	prevQX              float64
	prevQY              float64
	prevQZ              float64
	prevQW              float64
	forceStaticClamp    bool
	uprightHull         bool
	clampNormals        [16][3]float64
	clampNormalsLen     int
	flatWalls           [32]flatWallPlane
	flatWallsLen        int
	clampCorners        [16]IFace
	clampCornersLen     int
	pendingStepZ        float64
	solverIdx           int
	solver              *Solver
}

type flatWallPlane struct {
	nX, nY, nZ float64
	pX, pY, pZ float64
}

var _counter uint64

// NewCageSequence creates a new Cage for object using the next sequence ID.
func NewCageSequence() *Cage {
	c := NewCage(_counter)
	atomic.AddUint64(&_counter, 1)
	return c
}

// NewCage creates and returns a new instance of Cage initialized with the specified IThing object.[cite: 6]
func NewCage(id uint64) *Cage {
	c := &Cage{
		id:        id,
		seen:      make([]uint64, max(id*2, 1024)),
		solver:    nil,
		ellipsoid: rigid_body.NewEntity(0, 0, 0, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0),
		volume:    nil,
		slots:     make([]*Entry, totalSlots),
		slotsLen:  0,
		faces:     make([]*Faces, 8),
		facesIdx:  0,
		hasFloor:  false,
		manifold:  aabb.NewOBBManifold(),
		solverIdx: -1,
	}
	for i := 0; i < len(c.faces); i++ {
		c.faces[i] = NewFaces()
	}
	for i := BucketType(0); i < BucketSize; i++ {
		c.buckets[i] = NewBucket(i)
	}
	for i := 0; i < len(c.ellipsoidLocal); i++ {
		c.ellipsoidLocal[i] = rigid_body.NewEntity(0, 0, 0, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0)
	}
	return c
}

// Setup assigns the object used by the cage and returns nil.
func (s *Cage) Setup(solver *Solver, object IObject) error {
	s.solver = solver
	s.object = object
	return nil
}

// GetObject returns the object associated with the cage.[cite: 6]
func (s *Cage) GetObject() IObject {
	return s.object
}

// Rebuild recalculates the collision cage dimensions, position, and broad-phase swept volume using the given maxStep value.[cite: 6]
func (s *Cage) Rebuild(maxStep float64, session uint64) {
	s.session = session
	lEnt := s.object.GetEntity()
	// Calculate Position
	s.dX, s.dY, s.dZ = lEnt.GetFrameDisplacement()

	// Oriented extents from entity OBB (kept synchronized by Entity.Update).
	s.eRadX, s.eRadY, s.eRadZ = lEnt.GetOrientedHalfExtents()
	entCX, entCY, entCZ := lEnt.GetCenter()
	s.cX, s.cY, s.cZ = entCX, entCY, entCZ
	s.objOffX, s.objOffY, s.objOffZ = 0, 0, 0
	s.uprightHull = false

	if lEnt.KeepUpright() {
		if objAABB := s.object.GetAABB(); objAABB != nil {
			minX := objAABB.GetMinX()
			maxX := objAABB.GetMaxX()
			minY := objAABB.GetMinY()
			maxY := objAABB.GetMaxY()
			minZ := objAABB.GetMinZ()
			maxZ := objAABB.GetMaxZ()
			if maxX > minX && maxY > minY && maxZ > minZ {
				s.cX = (minX + maxX) * 0.5
				s.cY = (minY + maxY) * 0.5
				s.cZ = (minZ + maxZ) * 0.5
				s.eRadX = (maxX - minX) * 0.5
				s.eRadY = (maxY - minY) * 0.5
				s.eRadZ = (maxZ - minZ) * 0.5
				s.objOffX = s.cX - entCX
				s.objOffY = s.cY - entCY
				s.objOffZ = s.cZ - entCZ
				s.uprightHull = true
			}
		}
	}

	qx, qy, qz, qw := lEnt.GetQRotation().Get()
	if s.eRadX != s.prevRadX || s.eRadY != s.prevRadY || s.eRadZ != s.prevRadZ ||
		s.objOffX != s.prevOffX || s.objOffY != s.prevOffY || s.objOffZ != s.prevOffZ ||
		qx != s.prevQX || qy != s.prevQY || qz != s.prevQZ || qw != s.prevQW {
		s.forceStaticClamp = true
		s.prevRadX, s.prevRadY, s.prevRadZ = s.eRadX, s.eRadY, s.eRadZ
		s.prevOffX, s.prevOffY, s.prevOffZ = s.objOffX, s.objOffY, s.objOffZ
		s.prevQX, s.prevQY, s.prevQZ, s.prevQW = qx, qy, qz, qw
	}

	s.tX, s.tY, s.tZ = s.cX+s.dX, s.cY+s.dY, s.cZ+s.dZ

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

	for i := 0; i < len(s.buckets); i++ {
		s.buckets[i].Rebuild()
	}

	s.volume = nil
	s.distance = math.MaxFloat64

	s.slotsLen = 0
	s.hasFloor = false
	s.applied = false
	s.clampX, s.clampY, s.clampZ = entCX, entCY, entCZ
	s.clampNormalsLen = 0
	s.flatWallsLen = 0
	s.clampCornersLen = 0

	//clear(s.seen)
	s.maxStep = maxStep
	s.pendingStepZ = 0
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
	f.rFace = rFace
	f.lFace = lFace
	s.facesIdx++
}

type staticFaceContact struct {
	bucket                 BucketType
	dist                   float64
	pen                    float64
	nX, nY, nZ             float64
	anchorX                float64
	anchorY                float64
	anchorZ                float64
	origNX, origNY, origNZ float64
	origAX, origAY, origAZ float64
	minX, maxX             float64
	minY, maxY             float64
	minZ, maxZ             float64
	cornerHit              bool
}

// Commit processes pending face overlaps in this Cage against either another dynamic Cage or static world geometry.
func (s *Cage) Commit(rCage *Cage) {
	if rCage != nil {
		s.commitDynamicManifold(rCage)
		s.facesIdx = 0
		return
	}
	s.commitStaticFaces(s.ellipsoid.GetAABB())
	s.facesIdx = 0
}

// commitStaticFaces evaluates accumulated static world triangles and inserts single-point or clipped multi-point contacts.
func (s *Cage) commitStaticFaces(lAABB *aabb.AABB) {
	keepUpright := s.object.GetEntity().KeepUpright()
	baseZ := s.GetBaseZ()
	minBaseZ := math.Min(baseZ, baseZ+s.dZ)
	stepZ := baseZ + s.maxStep

	for x := 0; x < s.facesIdx; x++ {
		rFace := s.faces[x].rFace
		if rFace == nil || !rFace.IsSolid() {
			continue // Skybox or non-solid surface: ignore collision and volume tracking[cite: 6]
		}

		fc, ok := s.evaluateStaticFace(lAABB, rFace)
		if !ok {
			continue
		}

		// Volume Priority per le mesh statiche[cite: 6]
		if fc.dist < s.distance {
			if volume := rFace.GetParent(); volume != nil {
				s.volume = volume
				s.distance = fc.dist
			}
		}

		// Only genuine resting or validated contacts may grant floor support.
		if fc.pen <= 0 {
			if fc.bucket == BucketFloor && fc.dist <= floorGroundTolerance {
				s.hasFloor = true
			}
			continue
		}

		iMode := ImpactInelastic
		if fc.bucket.IsWall() {
			if fc.maxZ <= minBaseZ+satFilterEpsilon { // down-hill or within floor penetration slop[cite: 6]
				continue
			}
			if s.maxStep > 0 && fc.maxZ <= stepZ { // up-hill (climbable step)[cite: 6]
				iMode = ImpactStep
			}
		}

		if fc.bucket.IsWall() {
			if iMode == ImpactStep {
				if s.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) ||
					s.hasCoplanarInelasticSlot(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
					continue
				}
			} else {
				if fc.cornerHit && s.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
					continue
				}
				s.pruneCoplanarStepSlots(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
				if !fc.cornerHit {
					s.recordFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
					s.pruneCoplanarCornerHits(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
				}
			}
		}

		if iMode == ImpactInelastic && !keepUpright {
			s.addStaticTriangleManifold(rFace, &fc)
		} else {
			px, py, pz := s.projectStabilizedStaticContact(&fc)
			if entry := s.addToBucketEntry(fc.bucket, nil, rFace, fc.dist, fc.pen, fc.nX, fc.nY, fc.nZ, px, py, pz, fc.maxZ, iMode); entry != nil && fc.bucket.IsWall() {
				entry.SetCornerOrigin(fc.cornerHit, fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
			}
		}

		if fc.bucket == BucketFloor {
			s.hasFloor = true
		}
	}
}

// addStaticTriangleManifold clips the entity OBB against the static triangle rFace, falling back to a stabilized
// single contact point if clipping produces no penetrating candidates (e.g. diagonal endpoint or degenerate edge).
func (s *Cage) addStaticTriangleManifold(rFace IFace, fc *staticFaceContact) {
	v0x, v0y, v0z := rFace.Get0()
	v1x, v1y, v1z := rFace.Get1()
	v2x, v2y, v2z := rFace.Get2()
	m := s.manifold
	if m.ComputeStaticTriangle(s.object.GetEntity().BoundingBox, s.dX, s.dY, s.dZ, fc.nX, fc.nY, fc.nZ, v0x, v0y, v0z, v1x, v1y, v1z, v2x, v2y, v2z, fc.pen) {
		for pi := 0; pi < m.GetCount(); pi++ {
			ptX, ptY, ptZ, ptPen := m.GetPoint(pi)
			if ptPen <= 0 {
				continue
			}
			s.addToBucket(fc.bucket, nil, rFace, -ptPen, ptPen, fc.nX, fc.nY, fc.nZ, ptX, ptY, ptZ, fc.maxZ, ImpactInelastic)
		}
		return
	}
	px, py, pz := s.projectStabilizedStaticContact(fc)
	s.addToBucket(fc.bucket, nil, rFace, fc.dist, fc.pen, fc.nX, fc.nY, fc.nZ, px, py, pz, fc.maxZ, ImpactInelastic)
}

// projectStabilizedStaticContact projects the stabilized single-point support onto the static face plane,
// clamped to the triangle's world AABB bounds.
func (s *Cage) projectStabilizedStaticContact(fc *staticFaceContact) (float64, float64, float64) {
	supX, supY, supZ := s.object.GetEntity().GetStabilizedContactPoint(-fc.nX, -fc.nY, -fc.nZ)
	supX += s.dX
	supY += s.dY
	supZ += s.dZ
	distSup := (supX-fc.anchorX)*fc.nX + (supY-fc.anchorY)*fc.nY + (supZ-fc.anchorZ)*fc.nZ
	px := math.Max(fc.minX, math.Min(fc.maxX, supX-fc.nX*distSup))
	py := math.Max(fc.minY, math.Min(fc.maxY, supY-fc.nY*distSup))
	pz := math.Max(fc.minZ, math.Min(fc.maxZ, supZ-fc.nZ*distSup))
	return px, py, pz
}

// addToBucket adds a collision cage to the specified bucket and updates the slot list if a new cage is created.[cite: 6]
func (s *Cage) addToBucket(bucket BucketType, rCage *Cage, rFace IFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) bool {
	return s.addToBucketEntry(bucket, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode) != nil
}

func (s *Cage) addToBucketEntry(bucket BucketType, rCage *Cage, rFace IFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) *Entry {
	entry := s.buckets[bucket].Add(bucket, s, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode)
	if entry != nil {
		s.slots[s.slotsLen] = entry
		s.slotsLen++
		return entry
	}
	return nil
}

func (s *Cage) hasCoplanarFlatWall(nX, nY, nZ, ax, ay, az float64) bool {
	for i := 0; i < s.flatWallsLen; i++ {
		w := &s.flatWalls[i]
		if nX*w.nX+nY*w.nY+nZ*w.nZ > 0.999 &&
			math.Abs((ax-w.pX)*w.nX+(ay-w.pY)*w.nY+(az-w.pZ)*w.nZ) < planeEpsilon {
			return true
		}
	}
	return false
}

func (s *Cage) recordFlatWall(nX, nY, nZ, ax, ay, az float64) {
	if s.hasCoplanarFlatWall(nX, nY, nZ, ax, ay, az) || s.flatWallsLen >= len(s.flatWalls) {
		return
	}
	s.flatWalls[s.flatWallsLen] = flatWallPlane{
		nX: nX, nY: nY, nZ: nZ,
		pX: ax, pY: ay, pZ: az,
	}
	s.flatWallsLen++
}

func slotMatchesOrigPlane(slot *Entry, nX, nY, nZ, ax, ay, az float64) bool {
	return slot.rCage == nil &&
		slot.origNX*nX+slot.origNY*nY+slot.origNZ*nZ > 0.999 &&
		math.Abs((slot.origAX-ax)*nX+(slot.origAY-ay)*nY+(slot.origAZ-az)*nZ) < planeEpsilon
}

func (s *Cage) hasCoplanarInelasticSlot(nX, nY, nZ, ax, ay, az float64) bool {
	for i := 0; i < s.slotsLen; i++ {
		slot := s.slots[i]
		if slot.iMode == ImpactInelastic && slot.bucket.IsWall() && slotMatchesOrigPlane(slot, nX, nY, nZ, ax, ay, az) {
			return true
		}
	}
	return false
}

func (s *Cage) pruneCoplanarStepSlots(nX, nY, nZ, ax, ay, az float64) {
	for i := 0; i < s.slotsLen; {
		slot := s.slots[i]
		if slot.iMode == ImpactStep && slot.bucket.IsWall() && slotMatchesOrigPlane(slot, nX, nY, nZ, ax, ay, az) {
			s.buckets[slot.bucket].RemoveEntry(slot)
			s.slotsLen--
			s.slots[i] = s.slots[s.slotsLen]
			continue
		}
		i++
	}
}

func (s *Cage) pruneCoplanarCornerHits(nX, nY, nZ, ax, ay, az float64) {
	for i := 0; i < s.slotsLen; {
		slot := s.slots[i]
		if slot.isCorner && slotMatchesOrigPlane(slot, nX, nY, nZ, ax, ay, az) {
			s.buckets[slot.bucket].RemoveEntry(slot)
			s.slotsLen--
			s.slots[i] = s.slots[s.slotsLen]
			continue
		}
		i++
	}
}

// commitDynamicManifold resolves an overlapping or swept-colliding OBB pair using 15-axis SAT (faces + edge-edge)
// with angular+linear conservative advancement CCD and up to 4 clipped manifold contact points.
func (s *Cage) commitDynamicManifold(rCage *Cage) {
	var rFace, recipFace IFace
	for i := 0; i < s.facesIdx; i++ {
		f := s.faces[i]
		if f.rFace != nil && f.rFace.IsSolid() {
			rFace = f.rFace
			recipFace = f.lFace
			break
		}
	}
	if rFace == nil {
		return
	}
	if recipFace == nil {
		recipFace = rFace
	}

	lEnt := s.object.GetEntity()
	rEnt := rCage.GetEntity()
	rDX, rDY, rDZ := rEnt.GetFrameDisplacement()
	m := s.manifold
	m.ComputeSweptAtOffset(lEnt.BoundingBox, rEnt.BoundingBox, s.dX, s.dY, s.dZ, rDX, rDY, rDZ)

	rTx, rTy, rTz := rCage.GetT()
	rRadX, rRadY, rRadZ := rCage.GetRad()
	mNormalX, mNormalY, mNormalZ := m.GetNormal()
	b := Normal2BucketType(mNormalX, mNormalY, mNormalZ)
	if !m.GetHit() {
		if m.GetGap() <= floorGroundTolerance && s.overlapsTargetXY(rTx-rRadX, rTx+rRadX, rTy-rRadY, rTy+rRadY) {
			if b == BucketFloor && mNormalZ >= floorNormalThreshold {
				s.hasFloor = true
			} else if b == BucketCeiling && mNormalZ <= -floorNormalThreshold {
				rCage.hasFloor = true
			}
		}
		return
	}

	penetration := m.GetPenetration()
	count := m.GetCount()
	dist := -penetration
	if b == BucketFloor && mNormalZ >= floorNormalThreshold && dist <= floorGroundTolerance {
		s.hasFloor = true
	} else if b == BucketCeiling && mNormalZ <= -floorNormalThreshold && dist <= floorGroundTolerance {
		rCage.hasFloor = true
	}

	if penetration <= 0 || count == 0 {
		return
	}

	rMaxZ := rTz + rRadZ
	lMaxZ := s.tZ + s.eRadZ

	iMode := ImpactElastic
	if b.IsWall() {
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

	for pi := 0; pi < count; pi++ {
		ptX, ptY, ptZ, ptPen := m.GetPoint(pi)
		if ptPen <= 0 {
			continue
		}

		ptDist := -ptPen
		//mNormalX, mNormalY, mNormalZ := m.GetNormal()
		s.addToBucket(b, rCage, rFace, ptDist, ptPen, mNormalX, mNormalY, mNormalZ, ptX, ptY, ptZ, rMaxZ, iMode)
		if iMode == ImpactElastic {
			oppB := b.Opposite()
			rCage.addToBucket(oppB, s, recipFace, ptDist, ptPen, -mNormalX, -mNormalY, -mNormalZ, ptX, ptY, ptZ, lMaxZ, ImpactElastic)
			if oppB == BucketFloor {
				rCage.hasFloor = true
			}
		}
	}

	switch b {
	case BucketFloor:
		s.hasFloor = true
	case BucketCeiling:
		rCage.hasFloor = true
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
	lEnt := s.object.GetEntity()
	lOx, lOy, lOz := lEnt.GetCenter()
	lOx += s.dX
	lOy += s.dY
	lOz += s.dZ
	rEnt := rCage.object.GetEntity()
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

	lEnt := s.object.GetEntity()
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
		s.growSeen(rCage.id)
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
	for z := 0; z < s.slotsLen; z++ {
		if s.slots[z].bucket == t {
			return true
		}
	}
	return false
}

// GetSlotsLen returns the current number of active slots in the Cage.[cite: 6]
func (s *Cage) GetSlotsLen() int { return s.slotsLen }

// GetSlot returns the Entry at the specified index in the slots slice.[cite: 6]
func (s *Cage) GetSlot(i int) *Entry { return s.slots[i] }

// GetVolume returns the volume associated with the Cage.[cite: 6]
func (s *Cage) GetVolume() IObject { return s.volume }

// GetRad retrieves the ellipsoid radii along the X, Y, and Z axes.[cite: 6]
func (s *Cage) GetRad() (float64, float64, float64) { return s.eRadX, s.eRadY, s.eRadZ }

func (s *Cage) GetRadZ() float64 { return s.eRadZ }

// GetC returns the center coordinates (cX, cY, cZ) of the Cage.[cite: 6]
func (s *Cage) GetC() (float64, float64, float64) { return s.cX, s.cY, s.cZ }

// GetDisplacement returns the displacement of the collision cage along the x, y, and z axes.[cite: 6]
func (s *Cage) GetDisplacement() (float64, float64, float64) { return s.dX, s.dY, s.dZ }

// GetT retrieves the current transformed coordinates (tX, tY, tZ) of the Cage.[cite: 6]
func (s *Cage) GetT() (float64, float64, float64) { return s.tX, s.tY, s.tZ }

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

// NeedsStaticClamp reports whether the entity center moved since the last validated static world clamp.
func (s *Cage) NeedsStaticClamp() bool {
	if s.forceStaticClamp {
		return true
	}
	curX, curY, curZ := s.object.GetEntity().GetCenter()
	return curX != s.clampX || curY != s.clampY || curZ != s.clampZ
}

// CommitStaticClamp records the current entity center as the latest statically clamped baseline position.
func (s *Cage) CommitStaticClamp() {
	s.clampX, s.clampY, s.clampZ = s.object.GetEntity().GetCenter()
	s.forceStaticClamp = false
}

func (s *Cage) addClampNormal(nX, nY, nZ float64) {
	for i := 0; i < s.clampNormalsLen; i++ {
		c := &s.clampNormals[i]
		if c[0]*nX+c[1]*nY+c[2]*nZ > 0.999 {
			return
		}
	}
	if s.clampNormalsLen < len(s.clampNormals) {
		s.clampNormals[s.clampNormalsLen][0] = nX
		s.clampNormals[s.clampNormalsLen][1] = nY
		s.clampNormals[s.clampNormalsLen][2] = nZ
		s.clampNormalsLen++
	}
}

// IsBlockedAlong reports whether pushing this cage along unit direction (dirX, dirY, dirZ)
// pushes into any active or clamped static world face.
func (s *Cage) IsBlockedAlong(dirX, dirY, dirZ float64) bool {
	const wallBlockDotThreshold = -0.05
	for i := 0; i < s.clampNormalsLen; i++ {
		c := &s.clampNormals[i]
		if dirX*c[0]+dirY*c[1]+dirZ*c[2] < wallBlockDotThreshold {
			return true
		}
	}
	for i := 0; i < s.slotsLen; i++ {
		slot := s.slots[i]
		if slot.iMode == ImpactInelastic {
			if dirX*slot.nX+dirY*slot.nY+dirZ*slot.nZ < wallBlockDotThreshold {
				return true
			}
		}
	}
	return false
}

// sweptHalfExtents returns the broad-phase half-extents for s without inflating upright unrotated hulls on yaw.
func (s *Cage) sweptHalfExtents(ent *rigid_body.Entity) (float64, float64, float64) {
	if s.uprightHull {
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

// BeginStaticClamp prepares a temporary swept broadphase cage from the last valid clamped center to the current entity center,
// returning saved frame center, displacement, and target values for restoration in EndStaticClamp.
func (s *Cage) BeginStaticClamp() (origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ float64, lAABB *aabb.AABB) {
	origCX, origCY, origCZ = s.cX, s.cY, s.cZ
	origDX, origDY, origDZ = s.dX, s.dY, s.dZ
	origTX, origTY, origTZ = s.tX, s.tY, s.tZ
	ent := s.object.GetEntity()
	curX, curY, curZ := ent.GetCenter()
	s.cX, s.cY, s.cZ = s.clampX+s.objOffX, s.clampY+s.objOffY, s.clampZ+s.objOffZ
	s.tX, s.tY, s.tZ = curX+s.objOffX, curY+s.objOffY, curZ+s.objOffZ
	s.dX, s.dY, s.dZ = s.tX-s.cX, s.tY-s.cY, s.tZ-s.cZ
	sx, sy, sz := s.sweptHalfExtents(ent)
	minX := s.cX - sx + math.Min(0.0, s.dX)
	maxX := s.cX + sx + math.Max(0.0, s.dX)
	minY := s.cY - sy + math.Min(0.0, s.dY)
	maxY := s.cY + sy + math.Max(0.0, s.dY)
	minZ := s.cZ - sz + math.Min(0.0, s.dZ)
	maxZ := s.cZ + sz + math.Max(0.0, s.dZ)
	s.ellipsoid.Rebuild(minX, minY, minZ, maxX-minX, maxY-minY, maxZ-minZ)
	s.flatWallsLen = 0
	s.clampCornersLen = 0
	return origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ, s.ellipsoid.GetAABB()
}

func (s *Cage) clampStaticWallShouldSkip(fc *staticFaceContact) bool {
	curBaseZ := s.tZ - s.eRadZ
	if fc.maxZ <= curBaseZ+satFilterEpsilon {
		return true
	}
	preBaseZ := s.GetBaseZ()
	return preBaseZ >= fc.maxZ-satFilterEpsilon && s.dZ < -satFilterEpsilon && fc.maxZ-curBaseZ <= floorGroundTolerance
}

// staticClampApplyContact clamps velocity, resolves penetration, updates cage displacement, and reports whether it moved.
func (s *Cage) staticClampApplyContact(fc *staticFaceContact) bool {
	ent := s.object.GetEntity()
	s.addClampNormal(fc.nX, fc.nY, fc.nZ)
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

// ClampStaticFace tests one static face against the entity sweep [clamp -> cur] and immediately projects out
// any wall, floor, or ceiling penetration without modifying bucket warm-start buffers.
func (s *Cage) ClampStaticFace(lAABB *aabb.AABB, rFace IFace) bool {
	if !rFace.IsSolid() {
		return false
	}
	fc, ok := s.evaluateStaticFace(lAABB, rFace)
	if !ok {
		return false
	}
	if fc.pen <= 0.0 {
		if fc.bucket == BucketFloor && fc.dist <= floorGroundTolerance {
			s.hasFloor = true
			s.object.GetEntity().SetOnGround(true)
		}
		return false
	}
	if fc.bucket == BucketFloor {
		s.hasFloor = true
		s.object.GetEntity().SetOnGround(true)
	}
	if fc.bucket.IsWall() {
		if s.clampStaticWallShouldSkip(&fc) {
			return false
		}
		if fc.cornerHit {
			if s.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
				return false
			}
			if s.clampCornersLen < len(s.clampCorners) {
				s.clampCorners[s.clampCornersLen] = rFace
				s.clampCornersLen++
				return false
			}
		} else {
			s.recordFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
		}
	}
	return s.staticClampApplyContact(&fc)
}

// FlushStaticClamp resolves deferred wall corner contacts not overridden by a coplanar flat wall in the same pass.
func (s *Cage) FlushStaticClamp(lAABB *aabb.AABB) bool {
	moved := false
	for i := 0; i < s.clampCornersLen; i++ {
		fc, ok := s.evaluateStaticFace(lAABB, s.clampCorners[i])
		if !ok || fc.pen <= 0.0 {
			continue
		}
		if fc.bucket.IsWall() {
			if s.clampStaticWallShouldSkip(&fc) ||
				s.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
				continue
			}
		}
		if s.staticClampApplyContact(&fc) {
			moved = true
		}
	}
	s.clampCornersLen = 0
	return moved
}

// EndStaticClamp restores the pre-clamp cage center, displacement, and target fields after post-push static clamping.
func (s *Cage) EndStaticClamp(origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ float64) {
	s.cX, s.cY, s.cZ = origCX, origCY, origCZ
	s.dX, s.dY, s.dZ = origDX, origDY, origDZ
	s.tX, s.tY, s.tZ = origTX, origTY, origTZ
}

// overlapsTargetXY reports whether the target XY footprint at tX, tY overlaps [minX, maxX] x [minY, maxY].
func (s *Cage) overlapsTargetXY(minX, maxX, minY, maxY float64) bool {
	return s.tX+s.eRadX > minX && maxX > s.tX-s.eRadX && s.tY+s.eRadY > minY && maxY > s.tY-s.eRadY
}

// effectiveRadius returns the half-extent along unit normal (nX, nY, nZ),
// using exact OBB axis projection for free 6-DoF bodies and ellipsoid extents otherwise.
func (s *Cage) effectiveRadius(nX, nY, nZ float64) float64 {
	lEnt := s.object.GetEntity()
	if !lEnt.KeepUpright() {
		return lEnt.ProjectExtent(nX, nY, nZ)
	}
	return math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))
}

// GetAABB returns the axis-aligned bounding box (AABB) representing the broad-phase collision geometry of the cage.[cite: 6]
func (s *Cage) GetAABB() *aabb.AABB { return s.ellipsoid.GetAABB() }

// HasFloor reports whether the collision cage has a floor.[cite: 6]
func (s *Cage) HasFloor() bool {
	return s.hasFloor
}

// SetSolverIndex sets the temporary per-tick solver index of this cage (-1 when inactive).
func (s *Cage) SetSolverIndex(idx int) {
	s.solverIdx = idx
}

// GetSolverIndex returns the temporary per-tick solver index of this cage (-1 when inactive).
func (s *Cage) GetSolverIndex() int {
	return s.solverIdx
}

// GetSolver returns the owning Solver bound to this cage, or nil when unbound.
func (s *Cage) GetSolver() *Solver {
	return s.solver
}

// ZeroDisplacement resets any pending displacement delta on this cage when entering sleep.
func (s *Cage) ZeroDisplacement() {
	s.dX, s.dY, s.dZ = 0, 0, 0
}

// GetEntity returns the physics entity associated with the collision cage.[cite: 6]
func (s *Cage) GetEntity() *rigid_body.Entity {
	return s.object.GetEntity()
}

// SetApplied marks the collision cage as having applied its frame displacement and records entity center for tracking push-out offset.[cite: 6]
func (s *Cage) SetApplied() {
	s.applied = true
	lEnt := s.object.GetEntity()
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
	lEnt := s.object.GetEntity()
	curX, curY, curZ := lEnt.GetCenter()
	if s.applied {
		return curX - s.appliedX, curY - s.appliedY, curZ - s.appliedZ
	}
	return (curX + s.objOffX) - s.cX, (curY + s.objOffY) - s.cY, (curZ + s.objOffZ) - s.cZ
}

// evaluateStaticFace evaluates a static world triangle against the cage sweep (bucket, plane normal,
// signed separation distance, anti-phantom-plane filtered penetration, and world face bounds) without computing
// stabilized single-point contact projections until actually needed.
func (s *Cage) evaluateStaticFace(lAABB *aabb.AABB, rFace IFace) (staticFaceContact, bool) {
	var out staticFaceContact
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return out, false
	}

	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	p0x, p0y, p0z := rFace.Get0()
	distStart := (s.cX-p0x)*nX + (s.cY-p0y)*nY + (s.cZ-p0z)*nZ
	rayEff := s.effectiveRadius(nX, nY, nZ)

	var bucket BucketType
	if solidWE || solidNS {
		if distStart < 0 {
			impDotOrig := s.dX*nX + s.dY*nY + s.dZ*nZ
			if distStart <= -rayEff+satFilterEpsilon || (distStart <= -0.5*rayEff && impDotOrig > satFilterEpsilon) {
				nX, nY, nZ = -nX, -nY, -nZ
				distStart = -distStart
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
		return out, false
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

	if penetration <= 0 {
		if bucket == BucketFloor && dist <= floorGroundTolerance {
			tBaseZ := s.tZ - rayEff
			if nZ < floorNormalThreshold ||
				!s.overlapsTargetXY(rMinX, rMaxX, rMinY, rMaxY) ||
				rMaxZ < tBaseZ-floorGroundTolerance-satFilterEpsilon ||
				rMinZ > s.tZ+satFilterEpsilon {
				return out, false
			}
		}
		out.pen = 0
		return out, true
	}

	oZ := math.Max(0.0, math.Min(lAABB.GetMaxZ()-rMinZ, rMaxZ-lAABB.GetMinZ()))
	if oZ <= 0 {
		return out, false
	}

	slopeAdjusted := false
	if (solidWE || solidNS) && s.object.GetEntity().KeepUpright() {
		tunneled := distStart >= rayEff-satFilterEpsilon && distTarget < 0.0
		var cornerHit bool
		var cxN, cyN, cPen, qxW, qyW float64
		var valid bool
		valid, cornerHit, cxN, cyN, cPen, qxW, qyW = s.refineKeepUprightWall(rFace, nX, nY, penetration, tunneled)
		if !valid {
			return out, false
		}
		if cornerHit {
			nX, nY, nZ = cxN, cyN, 0.0
			nAbsX, nAbsY, nAbsZ = math.Abs(nX), math.Abs(nY), 0.0
			solidWE = nAbsX > nAbsY
			solidNS = !solidWE
			bucket = Normal2BucketType(nX, nY, 0.0)
			rayEff = s.effectiveRadius(nX, nY, 0.0)
			penetration = cPen
			dist = -cPen
			distTarget = rayEff - cPen
			distStart = distTarget - (s.dX*nX + s.dY*nY)
			out.bucket = bucket
			out.dist = dist
			out.nX, out.nY, out.nZ = nX, nY, 0.0
			out.anchorX, out.anchorY, out.anchorZ = qxW, qyW, s.tZ
			out.cornerHit = true
		}
	} else if s.object.GetEntity().KeepUpright() {
		tunneled := distStart >= rayEff-satFilterEpsilon && distTarget < 0.0
		valid, adjusted, cornerHit, cxN, cyN, czN, cPen, qxW, qyW, qzW := s.refineKeepUprightSlope(rFace, bucket, nX, nY, nZ, penetration, tunneled)
		if !valid {
			return out, false
		}
		if adjusted {
			nX, nY, nZ = cxN, cyN, czN
			nAbsX, nAbsY, nAbsZ = math.Abs(nX), math.Abs(nY), math.Abs(nZ)
			solidWE = nAbsX > nAbsY && nAbsX > nAbsZ
			solidNS = !solidWE && nAbsY > nAbsZ
			bucket = Normal2BucketType(nX, nY, nZ)
			rayEff = s.effectiveRadius(nX, nY, nZ)
			penetration = cPen
			dist = -cPen
			distTarget = rayEff - cPen
			distStart = distTarget - (s.dX*nX + s.dY*nY + s.dZ*nZ)
			out.bucket = bucket
			out.dist = dist
			out.nX, out.nY, out.nZ = nX, nY, nZ
			out.anchorX, out.anchorY, out.anchorZ = qxW, qyW, qzW
			out.cornerHit = cornerHit
			slopeAdjusted = true
		}
	}

	// Calcolo TOI (Time Of Impact) tra il moto dell'ellissoide e la faccia statica[cite: 6]
	rCx := (rMinX + rMaxX) * 0.5
	rCy := (rMinY + rMaxY) * 0.5
	rCz := (rMinZ + rMaxZ) * 0.5
	rRadX := (rMaxX - rMinX) * 0.5
	rRadY := (rMaxY - rMinY) * 0.5
	rRadZ := (rMaxZ - rMinZ) * 0.5

	tEnter, tExit, _, _, _, ok := computeTOI(s.cX-rCx, s.cY-rCy, s.cZ-rCz, s.eRadX+rRadX, s.eRadY+rRadY, s.eRadZ+rRadZ, s.dX, s.dY, s.dZ)
	if !ok {
		return out, false
	}

	impDot := s.dX*nX + s.dY*nY + s.dZ*nZ
	tPlaneEnter, tPlaneExit, okPlane := sweptAxisInterval(distStart, rayEff, impDot)
	if !okPlane {
		return out, false
	}

	overlapStart := math.Max(tEnter, tPlaneEnter)
	overlapEnd := math.Min(math.Min(1.0, tExit), tPlaneExit)
	if overlapStart > overlapEnd+toiTieEpsilon || overlapEnd < -toiTieEpsilon || overlapStart > 1.0+toiTieEpsilon {
		return out, false
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
	spikeTipCap := false
	if (solidWE || solidNS) && nZ >= 0.6 && (s.tZ-s.eRadZ) > rMinZ && rMaxZ < s.tZ+s.eRadZ {
		if oz := oZ / nZ; oz < axisOverlap {
			axisOverlap = oz
			spikeTipCap = true
		}
	}
	minOverlap := axisOverlap + deltaPen

	if !slopeAdjusted && penetration > minOverlap {
		if spikeTipCap && distTarget >= -satFilterEpsilon {
			penetration = minOverlap
		} else if penetration > minOverlap+satFilterEpsilon {
			return out, false
		} else {
			penetration = minOverlap
		}
	}

	out.pen = penetration
	return out, true
}

// refineKeepUprightWall clips a wall triangle to the upright body's Z slab and checks its scaled 2D footprint,
// rejecting phantom infinite-plane hits outside the slab and computing smooth corner pushouts on edge/vertex hits.
func (s *Cage) refineKeepUprightWall(rFace IFace, nX, nY, penetration float64, tunneled bool) (valid, cornerHit bool, cNX, cNY, cPen, qxW, qyW float64) {
	clipPolyZSlab := func(in *[6][3]float64, count int, out *[6][3]float64, zBound float64, keepAbove bool) int {
		outCount := 0
		for i := 0; i < count; i++ {
			curr := in[i]
			nextIdx := i + 1
			if nextIdx == count {
				nextIdx = 0
			}
			next := in[nextIdx]
			var currIn, nextIn bool
			if keepAbove {
				currIn = curr[2] >= zBound
				nextIn = next[2] >= zBound
			} else {
				currIn = curr[2] <= zBound
				nextIn = next[2] <= zBound
			}
			if currIn {
				out[outCount] = curr
				outCount++
				if !nextIn {
					t := (zBound - curr[2]) / (next[2] - curr[2])
					out[outCount][0] = curr[0] + t*(next[0]-curr[0])
					out[outCount][1] = curr[1] + t*(next[1]-curr[1])
					out[outCount][2] = zBound
					outCount++
				}
			} else if nextIn {
				t := (zBound - curr[2]) / (next[2] - curr[2])
				out[outCount][0] = curr[0] + t*(next[0]-curr[0])
				out[outCount][1] = curr[1] + t*(next[1]-curr[1])
				out[outCount][2] = zBound
				outCount++
			}
		}
		return outCount
	}

	var bufA, bufB [6][3]float64
	var poly2D [6][2]float64

	if s.eRadX <= 0.0 || s.eRadY <= 0.0 || s.eRadZ <= 0.0 {
		return true, false, 0, 0, 0, 0, 0
	}

	bufA[0][0], bufA[0][1], bufA[0][2] = rFace.Get0()
	bufA[1][0], bufA[1][1], bufA[1][2] = rFace.Get1()
	bufA[2][0], bufA[2][1], bufA[2][2] = rFace.Get2()

	invRX := 1.0 / s.eRadX
	invRY := 1.0 / s.eRadY
	invRZ := 1.0 / s.eRadZ
	qx3, qy3, qz3, _ := closestPointOnTriangleZero(
		(bufA[0][0]-s.tX)*invRX, (bufA[0][1]-s.tY)*invRY, (bufA[0][2]-s.tZ)*invRZ,
		(bufA[1][0]-s.tX)*invRX, (bufA[1][1]-s.tY)*invRY, (bufA[1][2]-s.tZ)*invRZ,
		(bufA[2][0]-s.tX)*invRX, (bufA[2][1]-s.tY)*invRY, (bufA[2][2]-s.tZ)*invRZ,
	)
	bestSq3D := qx3*qx3 + qy3*qy3 + qz3*qz3
	if bestSq3D >= 1.0-1e-9 && !tunneled {
		return false, false, 0, 0, 0, 0, 0
	}
	_, _, rNZ := rFace.GetNormal()
	rMinZ := math.Min(bufA[0][2], math.Min(bufA[1][2], bufA[2][2]))
	rMaxZ := math.Max(bufA[0][2], math.Max(bufA[1][2], bufA[2][2]))
	isTiltedOrCap := rMinZ > s.tZ || rMaxZ < s.tZ || math.Abs(rNZ) > 1e-3
	if !tunneled && (isTiltedOrCap || bestSq3D < 1.0-1e-9) {
		if bestSq3D <= 1e-12 {
			return true, false, 0, 0, 0, 0, 0
		}
		rZ := math.Sqrt(math.Max(0.0, 1.0-qz3*qz3))
		qXY3 := math.Hypot(qx3, qy3)
		if qXY3 <= 1e-6 || rZ <= qXY3 {
			return true, false, 0, 0, 0, 0, 0
		}
		scaleH := rZ/qXY3 - 1.0
		dxH := -qx3 * scaleH * s.eRadX
		dyH := -qy3 * scaleH * s.eRadY
		horizPen3D := math.Hypot(dxH, dyH)
		if horizPen3D <= 1e-12 {
			return false, false, 0, 0, 0, 0, 0
		}
		hx := dxH / horizPen3D
		hy := dyH / horizPen3D
		dot := 1.0
		if nXY := math.Hypot(nX, nY); nXY > 1e-9 {
			dot = hx*(nX/nXY) + hy*(nY/nXY)
		}
		if dot < -0.5 {
			return true, false, 0, 0, 0, 0, 0
		}
		if horizPen3D >= penetration-1e-4 && dot > 0.999 {
			return true, false, 0, 0, 0, 0, 0
		}
		if isTiltedOrCap && horizPen3D >= penetration*1.5 && dot > 0.7 {
			return true, false, 0, 0, 0, 0, 0
		}
		return true, true, hx, hy, horizPen3D, s.tX + qx3*s.eRadX, s.tY + qy3*s.eRadY
	}
	if tunneled && isTiltedOrCap {
		if rMinZ > s.tZ+s.eRadZ || rMaxZ < s.tZ-s.eRadZ {
			return false, false, 0, 0, 0, 0, 0
		}
		midX := 0.5 * (s.cX + s.tX)
		midY := 0.5 * (s.cY + s.tY)
		midZ := 0.5 * (s.cZ + s.tZ)
		mx3, my3, mz3, _ := closestPointOnTriangleZero(
			(bufA[0][0]-midX)*invRX, (bufA[0][1]-midY)*invRY, (bufA[0][2]-midZ)*invRZ,
			(bufA[1][0]-midX)*invRX, (bufA[1][1]-midY)*invRY, (bufA[1][2]-midZ)*invRZ,
			(bufA[2][0]-midX)*invRX, (bufA[2][1]-midY)*invRY, (bufA[2][2]-midZ)*invRZ,
		)
		if mx3*mx3+my3*my3+mz3*mz3 >= 1.0-1e-9 {
			return false, false, 0, 0, 0, 0, 0
		}
		return true, false, 0, 0, 0, 0, 0
	}

	n := clipPolyZSlab(&bufA, 3, &bufB, s.tZ-s.eRadZ, true)
	if n < 3 {
		return false, false, 0, 0, 0, 0, 0
	}
	n = clipPolyZSlab(&bufB, n, &bufA, s.tZ+s.eRadZ, false)
	if n < 3 {
		return false, false, 0, 0, 0, 0, 0
	}

	for i := 0; i < n; i++ {
		poly2D[i][0] = (bufA[i][0] - s.tX) * invRX
		poly2D[i][1] = (bufA[i][1] - s.tY) * invRY
	}

	//point in convexPoly2D
	const polyMax = 1e-12
	hasPos := false
	hasNeg := false
	areaAbsSum := 0.0
	hasHalt := false
	for i := 0; i < n; i++ {
		j := i + 1
		if j == n {
			j = 0
		}
		cross := poly2D[i][0]*poly2D[j][1] - poly2D[i][1]*poly2D[j][0]
		areaAbsSum += math.Abs(cross)
		if cross > polyMax {
			hasPos = true
		} else if cross < -polyMax {
			hasNeg = true
		}
		if hasPos && hasNeg {
			hasHalt = true
			break
		}
	}
	if !hasHalt && areaAbsSum > polyMax && (hasPos != hasNeg) {
		return true, false, 0, 0, 0, 0, 0
	}

	bestSq := math.MaxFloat64
	var qx, qy float64
	for i := 0; i < n; i++ {
		j := i + 1
		if j == n {
			j = 0
		}
		ax, ay := poly2D[i][0], poly2D[i][1]
		bx, by := poly2D[j][0], poly2D[j][1]
		abx, aby := bx-ax, by-ay
		abLenSq := abx*abx + aby*aby
		var t float64
		if abLenSq > 1e-18 {
			t = -(ax*abx + ay*aby) / abLenSq
			if t < 0.0 {
				t = 0.0
			} else if t > 1.0 {
				t = 1.0
			}
		}
		px := ax + t*abx
		py := ay + t*aby
		if dSq := px*px + py*py; dSq < bestSq {
			bestSq = dSq
			qx, qy = px, py
		}
	}

	if bestSq >= 1.0-1e-9 {
		return tunneled, false, 0, 0, 0, 0, 0
	}
	if bestSq <= 1e-12 {
		return true, false, 0, 0, 0, 0, 0
	}

	qLen := math.Sqrt(bestSq)
	scale := 1.0/qLen - 1.0
	dx := -qx * scale * s.eRadX
	dy := -qy * scale * s.eRadY
	horizPen := math.Hypot(dx, dy)
	if horizPen <= 1e-12 {
		return false, false, 0, 0, 0, 0, 0
	}
	hx := dx / horizPen
	hy := dy / horizPen

	nXY := math.Hypot(nX, nY)
	if nXY > 1e-9 {
		fnX, fnY := nX/nXY, nY/nXY
		dot := hx*fnX + hy*fnY
		// Reject pushouts pointing strongly behind a non-flipped wall plane (overshoot from the front side).
		if dot < -0.5 {
			return true, false, 0, 0, 0, 0, 0
		}
		if horizPen >= penetration-1e-4 && dot > 0.999 {
			return true, false, 0, 0, 0, 0, 0
		}
	}
	return true, true, hx, hy, horizPen, s.tX + qx*s.eRadX, s.tY + qy*s.eRadY
}

// closestPointOnSegmentZero returns the point on segment AB closest to the origin, returns A when the segment is degenerate.
func closestPointOnSegmentZero(ax, ay, az, bx, by, bz float64) (float64, float64, float64) {
	abx, aby, abz := bx-ax, by-ay, bz-az
	denom := abx*abx + aby*aby + abz*abz
	if denom <= 1e-18 {
		return ax, ay, az
	}
	t := -(ax*abx + ay*aby + az*abz) / denom
	if t <= 0.0 {
		return ax, ay, az
	}
	if t >= 1.0 {
		return bx, by, bz
	}
	return ax + t*abx, ay + t*aby, az + t*abz
}

// closestPointOnTriangleZero returns the closest point on 3D triangle (a, b, c) to the origin
// and reports whether that point lies on a vertex or edge boundary rather than the triangle interior.
func closestPointOnTriangleZero(ax, ay, az, bx, by, bz, cx, cy, cz float64) (qx, qy, qz float64, onBoundary bool) {
	abx, aby, abz := bx-ax, by-ay, bz-az
	acx, acy, acz := cx-ax, cy-ay, cz-az
	apx, apy, apz := -ax, -ay, -az
	d1 := abx*apx + aby*apy + abz*apz
	d2 := acx*apx + acy*apy + acz*apz
	if d1 <= 0.0 && d2 <= 0.0 {
		return ax, ay, az, true
	}

	bpx, bpy, bpz := -bx, -by, -bz
	d3 := abx*bpx + aby*bpy + abz*bpz
	d4 := acx*bpx + acy*bpy + acz*bpz
	if d3 >= 0.0 && d4 <= d3 {
		return bx, by, bz, true
	}

	vc := d1*d4 - d3*d2
	if vc <= 0.0 && d1 >= 0.0 && d3 <= 0.0 {
		v := d1 / (d1 - d3)
		return ax + v*abx, ay + v*aby, az + v*abz, true
	}

	cpx, cpy, cpz := -cx, -cy, -cz
	d5 := abx*cpx + aby*cpy + abz*cpz
	d6 := acx*cpx + acy*cpy + acz*cpz
	if d6 >= 0.0 && d5 <= d6 {
		return cx, cy, cz, true
	}

	vb := d5*d2 - d1*d6
	if vb <= 0.0 && d2 >= 0.0 && d6 <= 0.0 {
		w := d2 / (d2 - d6)
		return ax + w*acx, ay + w*acy, az + w*acz, true
	}

	va := d3*d6 - d5*d4
	if va <= 0.0 && (d4-d3) >= 0.0 && (d5-d6) >= 0.0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return bx + w*(cx-bx), by + w*(cy-by), bz + w*(cz-bz), true
	}

	denom := va + vb + vc
	if denom <= 1e-18 {
		q0x, q0y, q0z := closestPointOnSegmentZero(ax, ay, az, bx, by, bz)
		bestSq := q0x*q0x + q0y*q0y + q0z*q0z
		q1x, q1y, q1z := closestPointOnSegmentZero(ax, ay, az, cx, cy, cz)
		if dSq := q1x*q1x + q1y*q1y + q1z*q1z; dSq < bestSq {
			bestSq = dSq
			q0x, q0y, q0z = q1x, q1y, q1z
		}
		q2x, q2y, q2z := closestPointOnSegmentZero(bx, by, bz, cx, cy, cz)
		if dSq := q2x*q2x + q2y*q2y + q2z*q2z; dSq < bestSq {
			q0x, q0y, q0z = q2x, q2y, q2z
		}
		return q0x, q0y, q0z, true
	}

	invDenom := 1.0 / denom
	v := vb * invDenom
	w := vc * invDenom
	return ax + abx*v + acx*w, ay + aby*v + acy*w, az + abz*v + acz*w, false
}

// refineKeepUprightSlope validates finite 3D triangle vs ellipsoid contact for floor/ceiling faces
// and redirects side-grazed bevel or ledge hits into smooth separation without vertical catapults.
func (s *Cage) refineKeepUprightSlope(rFace IFace, bucket BucketType, nX, nY, nZ, penetration float64, tunneled bool) (valid, adjusted, cornerHit bool, cNX, cNY, cNZ, cPen, qxW, qyW, qzW float64) {
	if s.eRadX <= 0 || s.eRadY <= 0 || s.eRadZ <= 0 {
		return false, false, false, 0, 0, 0, 0, 0, 0, 0
	}
	invRX := 1.0 / s.eRadX
	invRY := 1.0 / s.eRadY
	invRZ := 1.0 / s.eRadZ

	v0x, v0y, v0z := rFace.Get0()
	v1x, v1y, v1z := rFace.Get1()
	v2x, v2y, v2z := rFace.Get2()

	qx, qy, qz, onBoundary := closestPointOnTriangleZero(
		(v0x-s.tX)*invRX, (v0y-s.tY)*invRY, (v0z-s.tZ)*invRZ,
		(v1x-s.tX)*invRX, (v1y-s.tY)*invRY, (v1z-s.tZ)*invRZ,
		(v2x-s.tX)*invRX, (v2y-s.tY)*invRY, (v2z-s.tZ)*invRZ,
	)

	bestSq := qx*qx + qy*qy + qz*qz
	if bestSq >= 1.0-1e-9 {
		return tunneled, false, false, 0, 0, 0, 0, 0, 0, 0
	}
	if bestSq <= 1e-12 {
		return true, false, false, 0, 0, 0, 0, 0, 0, 0
	}

	qxW = s.tX + qx*s.eRadX
	qyW = s.tY + qy*s.eRadY
	qzW = s.tZ + qz*s.eRadZ

	qLen := math.Sqrt(bestSq)
	scale := 1.0/qLen - 1.0
	dx := -qx * scale * s.eRadX
	dy := -qy * scale * s.eRadY
	dz := -qz * scale * s.eRadZ
	pen3D := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if pen3D <= 1e-12 {
		return false, false, false, 0, 0, 0, 0, 0, 0, 0
	}

	// Side/edge hits on ledges or sloped bevels should separate horizontally when Z separation fights the floor/ceiling.
	qXY := math.Hypot(qx, qy)
	if !tunneled && qXY > 1e-6 && math.Abs(qz) < 1.0-1e-9 {
		rZ := math.Sqrt(math.Max(0.0, 1.0-qz*qz))
		if rZ > qXY {
			scaleH := rZ/qXY - 1.0
			dxH := -qx * scaleH * s.eRadX
			dyH := -qy * scaleH * s.eRadY
			if penHoriz := math.Hypot(dxH, dyH); penHoriz > 1e-12 {
				wrongSideOfPlane := (bucket == BucketCeiling && qz < 0.0) || (bucket == BucketFloor && qz > 0.0)
				midBodyEdge := onBoundary && math.Abs(qz) < 0.707 && penHoriz < penetration*0.85
				if (wrongSideOfPlane && penHoriz <= pen3D*2.0) || midBodyEdge {
					return true, true, onBoundary, dxH / penHoriz, dyH / penHoriz, 0.0, penHoriz, qxW, qyW, qzW
				}
			}
		}
	}

	// For non-flat sloped ceiling/floor edges or oversized infinite-plane penetrations, use exact 3D finite-triangle separation.
	if math.Abs(nZ) < 0.999 && (onBoundary || pen3D < penetration*0.75) {
		nx3, ny3, nz3 := dx/pen3D, dy/pen3D, dz/pen3D
		if pen3D < penetration-satFilterEpsilon || (nx3*nX+ny3*nY+nz3*nZ) > 0.0 {
			return true, true, onBoundary, nx3, ny3, nz3, pen3D, qxW, qyW, qzW
		}
	}

	return true, false, false, 0, 0, 0, 0, 0, 0, 0
}

// growSeen expands the seen-session tracking slice to accommodate idx while preserving existing marks.
func (s *Cage) growSeen(idx uint64) {
	n := make([]uint64, idx*2)
	copy(n, s.seen)
	s.seen = n
}
