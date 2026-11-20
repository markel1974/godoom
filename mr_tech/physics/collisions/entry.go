package collisions

import "github.com/markel1974/godoom/mr_tech/physics/rigid_body"

const maxPushDepth = 8

// Entry represents a collision entry used during spatial computations in the collision detection system.
type Entry struct {
	bucket           BucketType
	lCage            *Cage
	rCage            *Cage
	rFace            IFace
	iMode            ImpactMode
	dist             float64
	penetration      float64
	nX               float64
	nY               float64
	nZ               float64
	p0X              float64
	p0Y              float64
	p0Z              float64
	maxZ             float64
	acc              rigid_body.Accumulator
	warmConsumed     bool
	warmEligible     bool
	isCorner         bool
	origNX           float64
	origNY           float64
	origNZ           float64
	origAX           float64
	origAY           float64
	origAZ           float64
	penetrableCached int8
}

// NewCageEntry creates and returns a new instance of Entry initialized with default zero values.
func NewCageEntry() *Entry {
	return &Entry{}
}

// GetRemoteFace returns the remote face associated with the Entry structure.
func (s *Entry) GetRemoteFace() IFace { return s.rFace }

// GetDistance returns the precomputed distance value associated with the Entry instance.
func (s *Entry) GetDistance() float64 { return s.dist }

// IsDynamic determines if the instance is associated with a remote collision cage.
func (s *Entry) IsDynamic() bool { return s.rCage != nil }

// GetMaxZ returns the maximum Z-coordinate of the Entry.
func (s *Entry) GetMaxZ() float64 { return s.maxZ }

// GetBucket retrieves the BucketType associated with the Entry instance.
func (s *Entry) GetBucket() BucketType { return s.bucket }

// GetImpactMode returns the impact mode of the collision as an integer value.
func (s *Entry) GetImpactMode() ImpactMode {
	return s.iMode
}

// GetNormal returns the components of the contact normal vector (nX, nY, nZ) as a tuple of three float64 values.
func (s *Entry) GetNormal() (float64, float64, float64) { return s.nX, s.nY, s.nZ }

// GetPoint returns the contact point coordinates (p0X, p0Y, p0Z) as a tuple of three float64 values.
func (s *Entry) GetPoint() (float64, float64, float64) { return s.p0X, s.p0Y, s.p0Z }

// GetPenetration returns the penetration depth of the collision measured by the cage entry.
func (s *Entry) GetPenetration() float64 { return s.penetration }

// GetLCage returns the local collision cage associated with the entry.
func (s *Entry) GetLCage() *Cage { return s.lCage }

// GetRCage returns the remote collision cage associated with the entry.
func (s *Entry) GetRCage() *Cage { return s.rCage }

// GetSolverState returns the embedded accumulated impulse and restitution target state for PGS iterations.
func (s *Entry) GetSolverState() *rigid_body.Accumulator {
	return &s.acc
}

// GetWarmImpulse returns the stored accumulated normal and tangential impulses for this entry.
func (s *Entry) GetWarmImpulse() (float64, float64, float64, float64) {
	return s.acc.Get()
}

// SetWarmImpulse seeds the entry's accumulated normal impulse and re-projects tangential impulse onto the contact plane.
func (s *Entry) SetWarmImpulse(accNormal, accTx, accTy, accTz float64) {
	tDotN := accTx*s.nX + accTy*s.nY + accTz*s.nZ
	s.acc.Set(accNormal, accTx-tDotN*s.nX, accTy-tDotN*s.nY, accTz-tDotN*s.nZ)
}

// Rebuild reinitializes the Entry with the given parameters and updates all its attributes accordingly.
func (s *Entry) Rebuild(bucket BucketType, lCage *Cage, rCage *Cage, rFace IFace, dist, penetration, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) {
	s.bucket = bucket
	s.lCage = lCage
	s.rCage = rCage
	s.rFace = rFace
	s.dist = dist
	s.penetration = penetration
	s.nX, s.nY, s.nZ = nX, nY, nZ
	s.p0X, s.p0Y, s.p0Z = p0x, p0y, p0z
	s.maxZ = maxZ
	s.iMode = iMode
	s.acc.Reset()
	s.warmConsumed = false
	s.warmEligible = false
	s.isCorner = false
	s.origNX, s.origNY, s.origNZ = 0, 0, 0
	s.origAX, s.origAY, s.origAZ = 0, 0, 0
	s.penetrableCached = 0
}

// SetCornerOrigin records whether this entry came from a wall corner refinement and stores the unrefined triangle plane.
func (s *Entry) SetCornerOrigin(isCorner bool, nx, ny, nz, ax, ay, az float64) {
	s.isCorner = isCorner
	s.origNX, s.origNY, s.origNZ = nx, ny, nz
	s.origAX, s.origAY, s.origAZ = ax, ay, az
}

// Penetrable reports whether this entry can be penetrated or pushed, following remote cages up to depth.
func (s *Entry) Penetrable(depth int) bool {
	if depth == 0 && s.penetrableCached != 0 {
		return s.penetrableCached == 1
	}
	res := s.evalPenetrable(depth)
	if depth == 0 {
		if res {
			s.penetrableCached = 1
		} else {
			s.penetrableCached = -1
		}
	}
	return res
}

func (s *Entry) evalPenetrable(depth int) bool {
	if s.iMode == ImpactInelastic {
		return false
	}
	if s.rCage == nil {
		return s.iMode == ImpactStep
	}
	if depth >= maxPushDepth {
		return true
	}
	rEnt := s.rCage.GetEntity()
	if rEnt == nil {
		return false
	}
	rId := rEnt.GetId()
	lId := noEntityId
	if s.lCage != nil {
		if lEnt := s.lCage.GetEntity(); lEnt != nil {
			lId = lEnt.GetId()
			if lId == rId {
				return false
			}
		}
	}

	nextDepth := depth + 1
	if nextDepth >= maxPushDepth {
		return true
	}
	v := s.rCage.buckets[s.bucket]
	if v == nil {
		return true
	}
	if v.inelasticCount > 0 {
		return false
	}
	if v.dynamicCount == 0 {
		return true
	}
	if ok, val := v.cache.Check(lId, nextDepth); ok {
		return val
	}
	res := v.doPenetrable(nextDepth, lId, rId)
	v.cache.Store(lId, nextDepth, res)
	return res
}
