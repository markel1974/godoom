package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

const (
	staticFaceFlatWallInitCap = 32
	staticFaceClampMax        = 16
	staticFaceCacheInitCap    = 64
)

// StaticFaceContact stores reusable contact data for resolving collisions between a moving cage and static faces.
type StaticFaceContact struct {
	bucket                 BucketType
	dist                   float64
	pen                    float64
	nX, nY, nZ             float64
	anchorX                float64
	anchorY                float64
	anchorZ                float64
	origNX, origNY, origNZ float64
	origAX, origAY, origAZ float64
	contactX               float64
	contactY               float64
	contactZ               float64
	minX, maxX             float64
	minY, maxY             float64
	minZ, maxZ             float64
	cornerHit              bool
	hasContact             bool
	wallTopLip             bool

	flatWalls    []FlatWallPlane
	flatWallsLen int

	clampNormals    [][3]float64
	clampNormalsLen int

	clampCorners    []*Face
	clampCornersLen int

	cachedFaces    []*Face
	cachedFacesLen int
	evalFace       *Face
	cachedValid    bool
	cacheMinX      float64
	cacheMinY      float64
	cacheMinZ      float64
	cacheMaxX      float64
	cacheMaxY      float64
	cacheMaxZ      float64
	prism          TrianglePrism
}

// NewStaticFaceContact creates a new StaticFaceContact with zeroed contact state.
func NewStaticFaceContact() *StaticFaceContact {
	clampCorners := make([]*Face, staticFaceClampMax)
	for i := range clampCorners {
		clampCorners[i] = NewFace()
	}
	cachedFaces := make([]*Face, staticFaceCacheInitCap)
	for i := range cachedFaces {
		cachedFaces[i] = NewFace()
	}
	return &StaticFaceContact{
		flatWalls:    make([]FlatWallPlane, staticFaceFlatWallInitCap),
		clampNormals: make([][3]float64, staticFaceClampMax),
		clampCorners: clampCorners,
		cachedFaces:  cachedFaces,
		evalFace:     NewFace(),
	}
}

// Reset clears cached clamp normals, flat walls, clamp corners, and per-frame static face cache.
func (fc *StaticFaceContact) Reset() {
	fc.clampNormalsLen = 0
	fc.flatWallsLen = 0
	fc.clampCornersLen = 0
	fc.cachedFacesLen = 0
	fc.cachedValid = false
}

// ResetFlatWalls clears the recorded flat wall planes for the contact.
func (fc *StaticFaceContact) ResetFlatWalls() {
	fc.flatWallsLen = 0
}

// ResetClampCorners clears the queued clamp corner contacts.
func (fc *StaticFaceContact) ResetClampCorners() {
	fc.clampCornersLen = 0
}

// IsBlockedAlong reports whether the direction is blocked by any stored clamp normal below threshold.
func (fc *StaticFaceContact) IsBlockedAlong(dirX, dirY, dirZ float64, threshold float64) bool {
	for i := 0; i < fc.clampNormalsLen; i++ {
		c := &fc.clampNormals[i]
		if dirX*c[0]+dirY*c[1]+dirZ*c[2] < threshold {
			return true
		}
	}
	return false
}

// Project projects the support point onto the contact plane and clamps it to the contact bounds.
func (fc *StaticFaceContact) Project(supX, supY, supZ float64) (float64, float64, float64) {
	distSup := (supX-fc.anchorX)*fc.nX + (supY-fc.anchorY)*fc.nY + (supZ-fc.anchorZ)*fc.nZ
	px := math.Max(fc.minX, math.Min(fc.maxX, supX-fc.nX*distSup))
	py := math.Max(fc.minY, math.Min(fc.maxY, supY-fc.nY*distSup))
	pz := math.Max(fc.minZ, math.Min(fc.maxZ, supZ-fc.nZ*distSup))
	return px, py, pz
}

// Commit evaluates static face contacts for the cage and records valid collision manifolds, steps, floors, and volume hits.
func (fc *StaticFaceContact) Commit(s *Cage, lAABB *aabb.AABB) {
	baseZ := s.GetBaseZ()
	minBaseZ := math.Min(baseZ, baseZ+s.dZ)
	stepZ := baseZ + s.maxStep
	for x := 0; x < s.facesIdx; x++ {
		rFace := s.faces[x].rFace
		if !rFace.IsSolid() {
			continue // Skybox or non-solid surface: ignore collision and volume tracking[cite: 6]
		}

		if !s.evalStaticFace(lAABB, rFace, fc) {
			continue
		}

		// Volume Priority per le mesh statiche[cite: 6]
		if fc.dist < s.distance {
			if volume := rFace.GetParent(); volume != nil {
				s.volumeObj = volume.GetObject()
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
			if fc.wallTopLip {
				continue
			}
			curBaseZ := s.tZ - s.eRadZ
			if fc.maxZ <= minBaseZ+satFilterEpsilon { // down-hill or within floor penetration slop[cite: 6]
				continue
			}
			if baseZ >= fc.maxZ-satFilterEpsilon && s.dZ < -satFilterEpsilon && fc.maxZ-curBaseZ <= floorGroundTolerance {
				continue
			}
			if s.maxStep > 0 && fc.maxZ <= stepZ { // up-hill (climbable step)[cite: 6]
				iMode = ImpactStep
			}
		}

		if fc.bucket.IsWall() {
			if iMode == ImpactStep {
				if fc.HasCoplanarFlatWall() ||
					fc.hasCoplanarInelasticSlot(s.GetSlots()) {
					continue
				}
			} else {
				if fc.cornerHit && fc.HasCoplanarFlatWall() {
					continue
				}
				fc.pruneCoplanarStepSlots(s)
				if !fc.cornerHit {
					fc.RecordFlatWall()
					fc.pruneCoplanarCornerHits(s)
				}
			}
		}

		s.addStaticTriangleManifold(rFace, fc, iMode)

		if fc.bucket == BucketFloor {
			s.hasFloor = true
		}
	}
}

// HasCoplanarFlatWall reports whether a recorded flat wall shares the given normal and plane within tolerance.
func (fc *StaticFaceContact) HasCoplanarFlatWall() bool {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	for i := 0; i < fc.flatWallsLen; i++ {
		if fc.flatWalls[i].IsCoplanar(nX, nY, nZ, ax, ay, az) {
			return true
		}
	}
	return false
}

// RecordFlatWall stores a flat wall plane unless an equivalent coplanar wall is already tracked.
func (fc *StaticFaceContact) RecordFlatWall() {
	if fc.HasCoplanarFlatWall() {
		return
	}
	if fc.flatWallsLen >= len(fc.flatWalls) {
		fc.growFlatWalls()
	}
	fc.flatWalls[fc.flatWallsLen].Assign(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
	fc.flatWallsLen++
}

// AddClampCorner queues a face for corner clamping, expanding the reusable buffer if needed.
func (fc *StaticFaceContact) AddClampCorner(rFace IFace) bool {
	if fc.clampCornersLen >= len(fc.clampCorners) {
		fc.growClampCorners()
	}
	fc.clampCorners[fc.clampCornersLen].SetIFace(rFace)
	fc.clampCornersLen++
	return true
}

// AddClampNormal stores a clamp normal if no nearly identical normal is already recorded, expanding the buffer if needed.
func (fc *StaticFaceContact) AddClampNormal(nX, nY, nZ float64) {
	for i := 0; i < fc.clampNormalsLen; i++ {
		c := &fc.clampNormals[i]
		if c[0]*nX+c[1]*nY+c[2]*nZ > 0.999 {
			return
		}
	}
	if fc.clampNormalsLen >= len(fc.clampNormals) {
		fc.growClampNormals()
	}
	fc.clampNormals[fc.clampNormalsLen][0] = nX
	fc.clampNormals[fc.clampNormalsLen][1] = nY
	fc.clampNormals[fc.clampNormalsLen][2] = nZ
	fc.clampNormalsLen++
}

// hasCoplanarInelasticSlot reports whether the cage has an inelastic wall slot
// on the original plane defined by the given normal and anchor.
func (fc *StaticFaceContact) hasCoplanarInelasticSlot(s *Slots) bool {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	for i := 0; i < s.Len(); i++ {
		slot := s.At(i)
		if slot.iMode == ImpactInelastic && slot.bucket.IsWall() && slot.MatchesOrigPlane(nX, nY, nZ, ax, ay, az) {
			return true
		}
	}
	return false
}

// pruneCoplanarStepSlots removes wall step slots whose original plane matches the given normal and anchor.
func (fc *StaticFaceContact) pruneCoplanarStepSlots(s *Cage) {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	slots := s.GetSlots()
	for i := 0; i < slots.Len(); {
		slot := slots.At(i)
		if slot.iMode == ImpactStep && slot.bucket.IsWall() && slot.MatchesOrigPlane(nX, nY, nZ, ax, ay, az) {
			s.buckets[slot.bucket].RemoveEntry(slot)
			slots.Decrease(i)
			s.cradle.SetDirty()
			continue
		}
		i++
	}
}

// pruneCoplanarCornerHits removes corner-hit slots whose original plane matches the contact plane.
func (fc *StaticFaceContact) pruneCoplanarCornerHits(s *Cage) {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	slots := s.GetSlots()
	for i := 0; i < slots.Len(); {
		slot := slots.At(i)
		if slot.isCorner && slot.MatchesOrigPlane(nX, nY, nZ, ax, ay, az) {
			s.buckets[slot.bucket].RemoveEntry(slot)
			slots.Decrease(i)
			s.cradle.SetDirty()
			continue
		}
		i++
	}
}

// growFlatWalls expands the reusable flatWalls buffer while preserving recorded wall planes.
func (fc *StaticFaceContact) growFlatWalls() {
	n := make([]FlatWallPlane, nextGrowCap(len(fc.flatWalls), staticFaceFlatWallInitCap))
	copy(n, fc.flatWalls)
	fc.flatWalls = n
}

// growClampCorners expands the reusable clampCorners buffer while preserving queued faces.
func (fc *StaticFaceContact) growClampCorners() {
	oldLen := len(fc.clampCorners)
	n := make([]*Face, nextGrowCap(oldLen, staticFaceClampMax))
	copy(n, fc.clampCorners)
	for i := oldLen; i < len(n); i++ {
		n[i] = NewFace()
	}
	fc.clampCorners = n
}

// growClampNormals expands the reusable clampNormals buffer while preserving recorded normals.
func (fc *StaticFaceContact) growClampNormals() {
	n := make([][3]float64, nextGrowCap(len(fc.clampNormals), staticFaceClampMax))
	copy(n, fc.clampNormals)
	fc.clampNormals = n
}

// growCachedFaces expands the reusable cachedFaces buffer while preserving collected static faces.
func (fc *StaticFaceContact) growCachedFaces() {
	oldLen := len(fc.cachedFaces)
	n := make([]*Face, nextGrowCap(oldLen, staticFaceCacheInitCap))
	copy(n, fc.cachedFaces)
	for i := oldLen; i < len(n); i++ {
		n[i] = NewFace()
	}
	fc.cachedFaces = n
}

// BeginStaticFaceCache initializes the per-frame static broadphase cache for the given AABB bounds.
func (fc *StaticFaceContact) BeginStaticFaceCache(lAABB *aabb.AABB) {
	fc.cacheMinX, fc.cacheMinY, fc.cacheMinZ, fc.cacheMaxX, fc.cacheMaxY, fc.cacheMaxZ = lAABB.GetMinMax()
	fc.cachedFacesLen = 0
	fc.cachedValid = false
}

// AddCachedFace stores a solid static candidate face in the per-frame broadphase cache.
func (fc *StaticFaceContact) AddCachedFace(rFace IFace) {
	if !rFace.IsSolid() {
		return
	}
	if fc.cachedFacesLen >= len(fc.cachedFaces) {
		fc.growCachedFaces()
	}
	fc.cachedFaces[fc.cachedFacesLen].SetIFace(rFace)
	fc.cachedFacesLen++
}

// CommitStaticFaceCache marks the current cached static face set valid for its recorded AABB bounds.
func (fc *StaticFaceContact) CommitStaticFaceCache() {
	fc.cachedValid = true
}

// HasStaticFaceCache reports whether the cached static face set completely covers lAABB.
func (fc *StaticFaceContact) HasStaticFaceCache(lAABB *aabb.AABB) bool {
	if !fc.cachedValid {
		return false
	}
	minX, minY, minZ, maxX, maxY, maxZ := lAABB.GetMinMax()
	return minX >= fc.cacheMinX && maxX <= fc.cacheMaxX &&
		minY >= fc.cacheMinY && maxY <= fc.cacheMaxY &&
		minZ >= fc.cacheMinZ && maxZ <= fc.cacheMaxZ
}

// nextGrowCap returns the next buffer capacity, doubling cur while honoring initialCap as a minimum.
func nextGrowCap(cur, initialCap int) int {
	newCap := max(cur*2, initialCap)
	return newCap
}
