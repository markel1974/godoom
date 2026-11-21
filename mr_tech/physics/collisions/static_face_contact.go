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

func (fw *flatWallPlane) Assign(nx, ny, nz float64, px, py, pz float64) {
	fw.nX, fw.nY, fw.nZ = nx, ny, nz
	fw.pX, fw.pY, fw.pZ = px, py, pz
}

func (fw *flatWallPlane) IsCoplanar(nx, ny, nz float64, ax, ay, az float64) bool {
	if nx*fw.nX+ny*fw.nY+nz*fw.nZ > 0.999 && math.Abs((ax-fw.pX)*fw.nX+(ay-fw.pY)*fw.nY+(az-fw.pZ)*fw.nZ) < planeEpsilon {
		return true
	}
	return false
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
			if fc.maxZ <= minBaseZ+satFilterEpsilon { // down-hill or within floor penetration slop[cite: 6]
				continue
			}
			if s.maxStep > 0 && fc.maxZ <= stepZ { // up-hill (climbable step)[cite: 6]
				iMode = ImpactStep
			}
		}

		if fc.bucket.IsWall() {
			if iMode == ImpactStep {
				if fc.hasCoplanarFlatWall() ||
					fc.hasCoplanarInelasticSlot(s) {
					continue
				}
			} else {
				if fc.cornerHit && fc.hasCoplanarFlatWall() {
					continue
				}
				fc.pruneCoplanarStepSlots(s)
				if !fc.cornerHit {
					fc.recordFlatWall()
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

// hasCoplanarFlatWall reports whether a recorded flat wall shares the given normal and plane within tolerance.
func (fc *StaticFaceContact) hasCoplanarFlatWall() bool {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	for i := 0; i < fc.flatWallsLen; i++ {
		if fc.flatWalls[i].IsCoplanar(nX, nY, nZ, ax, ay, az) {
			return true
		}
	}
	return false
}

// recordFlatWall stores a flat wall plane unless an equivalent coplanar wall is already tracked.
// It ignores new walls when the fixed wall buffer is full.
func (fc *StaticFaceContact) recordFlatWall() {
	if fc.hasCoplanarFlatWall() || fc.flatWallsLen >= len(fc.flatWalls) {
		return
	}
	ptr := &fc.flatWalls[fc.flatWallsLen]
	ptr.Assign(fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ)
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

// hasCoplanarInelasticSlot reports whether the cage has an inelastic wall slot
// on the original plane defined by the given normal and anchor.
func (fc *StaticFaceContact) hasCoplanarInelasticSlot(s *Cage) bool {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	for i := 0; i < s.slotsLen; i++ {
		slot := s.slots[i]
		if slot.iMode == ImpactInelastic && slot.bucket.IsWall() && slot.MatchesOrigPlane(nX, nY, nZ, ax, ay, az) {
			return true
		}
	}
	return false
}

// pruneCoplanarStepSlots removes wall step slots whose original plane matches the given normal and anchor.
func (fc *StaticFaceContact) pruneCoplanarStepSlots(s *Cage) {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	for i := 0; i < s.slotsLen; {
		slot := s.slots[i]
		if slot.iMode == ImpactStep && slot.bucket.IsWall() && slot.MatchesOrigPlane(nX, nY, nZ, ax, ay, az) {
			s.buckets[slot.bucket].RemoveEntry(slot)
			s.slotsLen--
			s.slots[i] = s.slots[s.slotsLen]
			continue
		}
		i++
	}
}

// pruneCoplanarCornerHits removes corner-hit slots whose original plane matches the contact plane.
func (fc *StaticFaceContact) pruneCoplanarCornerHits(s *Cage) {
	nX, nY, nZ, ax, ay, az := fc.origNX, fc.origNY, fc.origNZ, fc.origAX, fc.origAY, fc.origAZ
	for i := 0; i < s.slotsLen; {
		slot := s.slots[i]
		if slot.isCorner && slot.MatchesOrigPlane(nX, nY, nZ, ax, ay, az) {
			s.buckets[slot.bucket].RemoveEntry(slot)
			s.slotsLen--
			s.slots[i] = s.slots[s.slotsLen]
			continue
		}
		i++
	}
}
