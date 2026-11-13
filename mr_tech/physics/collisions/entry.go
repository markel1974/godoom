package collisions

// Entry represents a collision entry used during spatial computations in the collision detection system.
type Entry struct {
	bucket      BucketType
	rCage       *Cage
	rFace       IFace
	lCage       *Cage
	dist        float64
	penetration float64
	nX          float64
	nY          float64
	nZ          float64
	p0X         float64
	p0Y         float64
	p0Z         float64
	maxZ        float64
	iMode       ImpactMode
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
}

const maxPushDepth = 8

// Penetrable determines if the Entry can be penetrated based on its collision mode and the state of adjacent entities.
func (s *Entry) Penetrable() bool {
	return s.penetrable(0)
}

func (s *Entry) penetrable(depth int) bool {
	if s.iMode == ImpactInelastic {
		return false
	}
	if s.rCage == nil || depth >= maxPushDepth {
		return true
	}
	// Passiamo il controllo al bucket dell'entità remota nella STESSA direzione.
	v := s.rCage.buckets[s.bucket]
	// Se il bucket remoto non è penetrabile (bloccato da muri o da altre casse bloccate)
	// allora questa specifica faccia non può essere penetrata/spinta.
	return v.penetrable(s.lCage.object.GetEntity().GetId(), depth+1)
}
