package collisions

const maxPushDepth = 8

// Entry represents a collision entry used during spatial computations in the collision detection system.
type Entry struct {
	bucket        BucketType
	rCage         *Cage
	rFace         IFace
	lCage         *Cage
	iMode         ImpactMode
	dist          float64
	penetration   float64
	nX            float64
	nY            float64
	nZ            float64
	p0X           float64
	p0Y           float64
	p0Z           float64
	maxZ          float64
	accNormal     float64
	accTx         float64
	accTy         float64
	accTz         float64
	restTarget    float64
	hasRestTarget bool
	warmConsumed  bool
	warmEligible  bool
	isCorner      bool
	origNX        float64
	origNY        float64
	origNZ        float64
	origAX        float64
	origAY        float64
	origAZ        float64
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

// GetSolverState returns pointers to accumulated normal/tangential impulses and target restitution state for PGS iterations.
func (s *Entry) GetSolverState() (*float64, *float64, *float64, *float64, *float64, *bool) {
	return &s.accNormal, &s.accTx, &s.accTy, &s.accTz, &s.restTarget, &s.hasRestTarget
}

// GetWarmImpulse returns the stored accumulated normal and tangential impulses for this entry.
func (s *Entry) GetWarmImpulse() (float64, float64, float64, float64) {
	return s.accNormal, s.accTx, s.accTy, s.accTz
}

// SetWarmImpulse seeds the entry's accumulated normal impulse and re-projects tangential impulse onto the contact plane.
func (s *Entry) SetWarmImpulse(accNormal, accTx, accTy, accTz float64) {
	tDotN := accTx*s.nX + accTy*s.nY + accTz*s.nZ
	s.accNormal = accNormal
	s.accTx = accTx - tDotN*s.nX
	s.accTy = accTy - tDotN*s.nY
	s.accTz = accTz - tDotN*s.nZ
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
	s.accNormal = 0
	s.accTx = 0
	s.accTy = 0
	s.accTz = 0
	s.restTarget = 0
	s.hasRestTarget = false
	s.warmConsumed = false
	s.warmEligible = false
	s.isCorner = false
	s.origNX, s.origNY, s.origNZ = 0, 0, 0
	s.origAX, s.origAY, s.origAZ = 0, 0, 0
}

// SetCornerOrigin records whether this entry came from a wall corner refinement and stores the unrefined triangle plane.
func (s *Entry) SetCornerOrigin(isCorner bool, nx, ny, nz, ax, ay, az float64) {
	s.isCorner = isCorner
	s.origNX, s.origNY, s.origNZ = nx, ny, nz
	s.origAX, s.origAY, s.origAZ = ax, ay, az
}

// Penetrable reports whether this entry can be penetrated or pushed, following remote cages up to depth.
func (s *Entry) Penetrable(depth int) bool {
	var visited [maxPushDepth]uint64
	var visitedLen int
	if s.lCage != nil {
		if ent := s.lCage.GetEntity(); ent != nil {
			visited[0] = ent.GetId()
			visitedLen = 1
		}
	}
	return s.penetrableInternal(depth, &visited, visitedLen)
}

func (s *Entry) penetrableInternal(depth int, visited *[maxPushDepth]uint64, visitedLen int) bool {
	if s.iMode == ImpactInelastic {
		return false
	}
	if s.rCage == nil || depth >= maxPushDepth {
		return true
	}
	rEnt := s.rCage.GetEntity()
	if rEnt == nil {
		return true
	}
	rId := rEnt.GetId()

	// Check if this remote entity is already in the visited chain (cycle detection)
	for i := 0; i < visitedLen; i++ {
		if visited[i] == rId {
			return false // Closed loop in push direction cannot yield freely
		}
	}

	if visitedLen < maxPushDepth {
		visited[visitedLen] = rId
		visitedLen++
	}

	// Pass control to remote entity's bucket along the SAME push direction
	v := s.rCage.buckets[s.bucket]
	if v == nil {
		return true
	}
	return v.penetrableInternal(depth+1, visited, visitedLen)
}
