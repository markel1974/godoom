package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

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
}

func NewStaticFaceContact() *StaticFaceContact {
	return &StaticFaceContact{}
}

// StaticFacesCommit evaluates accumulated static world triangles and inserts single-point or clipped multi-point contacts.
func (s *Cage) StaticFacesCommit(lAABB *aabb.AABB) {
	baseZ := s.GetBaseZ()
	minBaseZ := math.Min(baseZ, baseZ+s.dZ)
	stepZ := baseZ + s.maxStep

	fc := s.staticContact
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

		s.addStaticTriangleManifold(rFace, fc, iMode)

		if fc.bucket == BucketFloor {
			s.hasFloor = true
		}
	}
}

// StaticFaceFlushClamp resolves deferred wall corner contacts not overridden by a coplanar flat wall in the same pass.
func (s *Cage) StaticFaceFlushClamp(lAABB *aabb.AABB) bool {
	moved := false
	fc := s.staticContact
	for i := 0; i < s.clampCornersLen; i++ {
		if !s.evaluateStaticFace(lAABB, s.clampCorners[i], fc) || fc.pen <= 0.0 {
			continue
		}
		if fc.bucket.IsWall() {
			if s.staticFaceClampWallShouldSkip(fc) ||
				s.hasCoplanarFlatWall(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ) {
				continue
			}
		}
		if s.staticFaceClampApplyContact(fc) {
			moved = true
		}
	}
	s.clampCornersLen = 0
	return moved
}

// StaticFaceClamp tests one static face against the entity sweep [clamp -> cur] and immediately projects out
// any wall, floor, or ceiling penetration without modifying bucket warm-start buffers.
func (s *Cage) StaticFaceClamp(lAABB *aabb.AABB, rFace IFace) bool {
	if !rFace.IsSolid() {
		return false
	}
	fc := s.staticContact
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
		if s.staticFaceClampWallShouldSkip(fc) {
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
	return s.staticFaceClampApplyContact(fc)
}

// clampStaticWallShouldSkip reports whether a wall contact is low enough to be ignored during static clamping.
func (s *Cage) staticFaceClampWallShouldSkip(fc *StaticFaceContact) bool {
	curBaseZ := s.tZ - s.eRadZ
	preBaseZ := s.GetBaseZ()
	if fc.maxZ <= curBaseZ+satFilterEpsilon && preBaseZ+s.maxStep >= fc.maxZ-satFilterEpsilon {
		return true
	}
	return preBaseZ >= fc.maxZ-satFilterEpsilon && s.dZ < -satFilterEpsilon && fc.maxZ-curBaseZ <= floorGroundTolerance
}

// staticClampApplyContact clamps velocity, resolves penetration, updates cage displacement, and reports whether it moved.
func (s *Cage) staticFaceClampApplyContact(fc *StaticFaceContact) bool {
	ent := s.entity
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
