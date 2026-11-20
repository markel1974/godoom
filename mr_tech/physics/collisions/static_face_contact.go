package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// flatWallPlane represents a planar wall contact defined by a normal and a point on the plane.
type flatWallPlane struct {
	nX, nY, nZ float64
	pX, pY, pZ float64
}

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

	flatWalls    [32]flatWallPlane
	flatWallsLen int

	clampNormals    [16][3]float64
	clampNormalsLen int

	clampCorners    [16]IFace
	clampCornersLen int
}

// NewStaticFaceContact creates a new StaticFaceContact with zeroed contact state.
func NewStaticFaceContact() *StaticFaceContact {
	return &StaticFaceContact{}
}

// Reset clears cached clamp normals, flat walls, and clamp corners.
func (fc *StaticFaceContact) Reset() {
	fc.clampNormalsLen = 0
	fc.flatWallsLen = 0
	fc.clampCornersLen = 0
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

// Commit evaluates static face contacts for the cage and records valid collision manifolds, steps, floors, and volume hits.
func (fc *StaticFaceContact) Commit(s *Cage, lAABB *aabb.AABB) {
	baseZ := s.GetBaseZ()
	minBaseZ := math.Min(baseZ, baseZ+s.dZ)
	stepZ := baseZ + s.maxStep
	for x := 0; x < s.facesIdx; x++ {
		rFace := s.faces[x].rFace
		if rFace == nil || !rFace.IsSolid() {
			continue // Skybox or non-solid surface: ignore collision and volume tracking[cite: 6]
		}

		if !s.evaluateStaticFace(lAABB, rFace, fc) {
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
			if fc.maxZ <= minBaseZ+satFilterEpsilon { // down-hill or within floor penetration slop[cite: 6]
				continue
			}
			if s.maxStep > 0 && fc.maxZ <= stepZ { // up-hill (climbable step)[cite: 6]
				iMode = ImpactStep
			}
		}

		if fc.bucket.IsWall() {
			if iMode == ImpactStep {
				if fc.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) ||
					s.hasCoplanarInelasticSlot(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
					continue
				}
			} else {
				if fc.cornerHit && fc.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
					continue
				}
				s.pruneCoplanarStepSlots(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
				if !fc.cornerHit {
					fc.recordFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
					s.pruneCoplanarCornerHits(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
				}
			}
		}

		s.addStaticTriangleManifold(rFace, fc, iMode)

		if fc.bucket == BucketFloor {
			s.hasFloor = true
		}
	}
}

// FlushClamp resolves queued corner clamp contacts against the cage and returns whether any contact moved it.
// It clears the queued clamp corners after processing.
func (fc *StaticFaceContact) FlushClamp(s *Cage, lAABB *aabb.AABB) bool {
	moved := false
	for i := 0; i < fc.clampCornersLen; i++ {
		if !s.evaluateStaticFace(lAABB, fc.clampCorners[i], fc) || fc.pen <= 0.0 {
			continue
		}
		if fc.bucket.IsWall() {
			if fc.clampWallShouldSkip(s) ||
				fc.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
				continue
			}
		}
		if fc.clampApplyContact(s) {
			moved = true
		}
	}
	fc.clampCornersLen = 0
	return moved
}

// Clamp evaluates a solid face against the cage AABB and applies penetration correction when needed.
// Clamp marks floor contact as grounded, skips invalid wall clamps, and defers corner wall contacts.
func (fc *StaticFaceContact) Clamp(s *Cage, lAABB *aabb.AABB, rFace IFace) bool {
	if !rFace.IsSolid() {
		return false
	}
	if !s.evaluateStaticFace(lAABB, rFace, fc) {
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
		if fc.clampWallShouldSkip(s) {
			return false
		}
		if fc.cornerHit {
			if fc.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
				return false
			}
			if fc.clampCornersLen < len(fc.clampCorners) {
				fc.clampCorners[fc.clampCornersLen] = rFace
				fc.clampCornersLen++
				return false
			}
		} else {
			fc.recordFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
		}
	}
	return fc.clampApplyContact(s)
}

// clampWallShouldSkip reports whether a wall contact should be ignored as stepable or ground-adjacent.
func (fc *StaticFaceContact) clampWallShouldSkip(s *Cage) bool {
	curBaseZ := s.tZ - s.eRadZ
	preBaseZ := s.GetBaseZ()
	if fc.maxZ <= curBaseZ+satFilterEpsilon && preBaseZ+s.maxStep >= fc.maxZ-satFilterEpsilon {
		return true
	}
	return preBaseZ >= fc.maxZ-satFilterEpsilon && s.dZ < -satFilterEpsilon && fc.maxZ-curBaseZ <= floorGroundTolerance
}

// clampApplyContact clamps velocity, resolves penetration, updates cage position, and reports whether movement occurred.
func (fc *StaticFaceContact) clampApplyContact(s *Cage) bool {
	ent := s.entity
	fc.addClampNormal(fc.nX, fc.nY, fc.nZ)
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

// hasCoplanarFlatWall reports whether a recorded flat wall shares the given normal and plane within tolerance.
func (fc *StaticFaceContact) hasCoplanarFlatWall(nX, nY, nZ, ax, ay, az float64) bool {
	for i := 0; i < fc.flatWallsLen; i++ {
		w := &fc.flatWalls[i]
		if nX*w.nX+nY*w.nY+nZ*w.nZ > 0.999 &&
			math.Abs((ax-w.pX)*w.nX+(ay-w.pY)*w.nY+(az-w.pZ)*w.nZ) < planeEpsilon {
			return true
		}
	}
	return false
}

// recordFlatWall stores a flat wall plane unless an equivalent coplanar wall is already tracked.
// It ignores new walls when the fixed wall buffer is full.
func (fc *StaticFaceContact) recordFlatWall(nX, nY, nZ, ax, ay, az float64) {
	if fc.hasCoplanarFlatWall(nX, nY, nZ, ax, ay, az) || fc.flatWallsLen >= len(fc.flatWalls) {
		return
	}
	fc.flatWalls[fc.flatWallsLen] = flatWallPlane{
		nX: nX, nY: nY, nZ: nZ,
		pX: ax, pY: ay, pZ: az,
	}
	fc.flatWallsLen++
}

// addClampNormal stores a clamp normal if no nearly identical normal is already recorded.
// It silently ignores duplicates and additions beyond the fixed clamp-normal capacity.
func (fc *StaticFaceContact) addClampNormal(nX, nY, nZ float64) {
	for i := 0; i < fc.clampNormalsLen; i++ {
		c := &fc.clampNormals[i]
		if c[0]*nX+c[1]*nY+c[2]*nZ > 0.999 {
			return
		}
	}
	if fc.clampNormalsLen < len(fc.clampNormals) {
		fc.clampNormals[fc.clampNormalsLen][0] = nX
		fc.clampNormals[fc.clampNormalsLen][1] = nY
		fc.clampNormals[fc.clampNormalsLen][2] = nZ
		fc.clampNormalsLen++
	}
}
