package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

const (
	// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
	ImpactInelastic = 1
	// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
	ImpactElastic = 2
	// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
	ImpactStep = 3
)

const satFilterEpsilon = 0.05

// BucketType represents the type of bucket categorizing spatial elements such as walls, ceiling, and floor in 3D space.
type BucketType int

// BucketWallWest represents the wall bucket located in the -X direction.
// BucketWallEast represents the wall bucket located in the +X direction.
// BucketWallNorth represents the wall bucket located in the -Y direction.
// BucketWallSouth represents the wall bucket located in the +Y direction.
// BucketCeiling represents the ceiling bucket located in the -Z direction.
// BucketFloor represents the floor bucket located in the +Z direction.
const (
	BucketWallWest  = BucketType(0) // -X
	BucketWallEast  = BucketType(1) // +X
	BucketWallNorth = BucketType(2) // -Y
	BucketWallSouth = BucketType(3) // +Y
	BucketCeiling   = BucketType(4) // -Z
	BucketFloor     = BucketType(5) // +Z
)

// String returns the string representation of the BucketType value.
func (p BucketType) String() string {
	switch p {
	case BucketWallWest:
		return "BucketWallWest"
	case BucketWallEast:
		return "BucketWallEast"
	case BucketWallNorth:
		return "BucketWallNorth"
	case BucketWallSouth:
		return "BucketWallSouth"
	case BucketCeiling:
		return "BucketCeiling"
	case BucketFloor:
		return "BucketFloor"
	default:
		return "BucketUnknown"
	}
}

// IsWall checks if the BucketType represents any wall (west, east, north, or south).
func (p BucketType) IsWall() bool {
	return p == BucketWallWest || p == BucketWallEast || p == BucketWallNorth || p == BucketWallSouth
}

// BucketSize defines the size of a bucket as one more than the BucketFloor.
// FacesPerBucket specifies the fixed number of faces in each bucket.
// TotalSlots calculates the total number of slots using BucketSize and FacesPerBucket.
const (
	BucketSize     = BucketFloor + 1
	FacesPerBucket = 4
	TotalSlots     = BucketSize * FacesPerBucket
)

type ContactManifold struct {
	Hit           bool
	Bucket        BucketType
	NormalX       float64
	NormalY       float64
	NormalZ       float64
	PointX        float64 // Esatto punto di impatto 3D per calcolo momento torcente
	PointY        float64
	PointZ        float64
	Depth         float64 // Compenetrazione matematica lungo l'asse SAT vincente
	HitDist       float64 // Distanza originale mantenuta per compatibilità
	P0X, P0Y, P0Z float64 // Punto di ancoraggio piano originario
	MinOverlap    float64 // Ovelap AABB originale
	RMaxZ         float64 // RMaxZ originale
}

type CageFaces struct {
	lFace *Face
	rFace *Face
}

// CageEntry represents a single entry within a collision cage, storing data about collisions and their properties.
type CageEntry struct {
	bucket      BucketType
	rCage       *CollisionCage
	rFace       *Face
	lCage       *CollisionCage
	dist        float64
	penetration float64
	nX          float64
	nY          float64
	nZ          float64
	p0X         float64
	p0Y         float64
	p0Z         float64
	maxZ        float64
	iMode       int
}

// GetRemoteFace retrieves the remote Face associated with this CageEntry.
func (s *CageEntry) GetRemoteFace() *Face { return s.rFace }

// GetDistance returns the distance value of the CageEntry.
func (s *CageEntry) GetDistance() float64 { return s.dist }

// IsDynamic checks if the CageEntry has an associated remote collision cage (rCage) and returns true if it does.
func (s *CageEntry) IsDynamic() bool { return s.rCage != nil }

// GetMaxZ returns the maximum Z-coordinate value associated with the CageEntry instance.
func (s *CageEntry) GetMaxZ() float64 { return s.maxZ }

// GetBucket retrieves the collision bucket type associated with the CageEntry.
func (s *CageEntry) GetBucket() BucketType { return s.bucket }

// GetImpactMode returns the impact mode as an integer, representing the type of collision or interaction detected.
func (s *CageEntry) GetImpactMode() int {
	return s.iMode
}

// GetNormal returns the normal vector components (nX, nY, nZ) of the collision.
func (s *CageEntry) GetNormal() (float64, float64, float64) { return s.nX, s.nY, s.nZ }

// GetPenetration returns the penetration distance indicating the overlap depth with another object in the collision system.
func (s *CageEntry) GetPenetration() float64 { return s.penetration }

// NewCollisionFace creates and returns a pointer to a new, uninitialized CageEntry structure.
func NewCollisionFace() *CageEntry {
	return &CageEntry{}
}

// Rebuild reinitializes the CageEntry with the provided parameters to update collision and interaction state.
func (s *CageEntry) Rebuild(bucket BucketType, lCage *CollisionCage, rCage *CollisionCage, rFace *Face, dist, penetration, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode int) {
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

// Penetrable checks if the cage entry can be penetrated based on its impact mode and the state of the associated remote cage.
func (s *CageEntry) Penetrable() bool {
	if s.iMode == ImpactInelastic {
		return false
	}
	if s.rCage == nil {
		return true
	}
	// Passiamo il controllo al bucket dell'entità remota nella STESSA direzione.
	v := s.rCage.buckets[s.bucket]
	// Se il bucket remoto non è penetrabile (bloccato da muri o da altre casse bloccate)
	// allora questa specifica faccia non può essere penetrata/spinta.
	return v.Penetrable(s.lCage.object.GetEntity().GetId())
}

// CollisionBucket represents a storage container for collision detection entities within a specific spatial bucket.
type CollisionBucket struct {
	bucket           BucketType
	spare            [FacesPerBucket]*CageEntry
	container        [FacesPerBucket]*CageEntry
	containerCounter int
}

// NewCollisionBucket initializes a new CollisionBucket with the specified BucketType.
// It pre-allocates collision faces and resets container counters.
func NewCollisionBucket(bucket BucketType) *CollisionBucket {
	b := &CollisionBucket{
		bucket:           bucket,
		containerCounter: 0,
	}
	for i := 0; i < FacesPerBucket; i++ {
		b.spare[i] = NewCollisionFace()
		b.container[i] = nil
	}
	return b
}

// Rebuild resets the collision bucket's container counter to zero, clearing any previously stored entries.
func (b *CollisionBucket) Rebuild() {
	b.containerCounter = 0
}

// Count returns the number of entries currently stored in the CollisionBucket.
func (b *CollisionBucket) Count() int {
	return b.containerCounter
}

// Penetrable determines if the CollisionBucket can be traversed based on its contents and the provided entity ID.
// It checks for inelastic resistance or recursively evaluates dynamic entities. Returns false if any resistance is detected.
func (b *CollisionBucket) Penetrable(from uint64) bool {
	// Scorriamo TUTTE le entità in questo bucket (multi-pushing)
	for i := 0; i < b.containerCounter; i++ {
		entry := b.container[i]
		// Se tocchiamo direttamente un muro, il bucket è bloccato
		if entry.iMode == ImpactInelastic {
			return false
		}
		if entry.rCage != nil && entry.rCage.object.GetEntity().GetId() == from {
			continue
		}
		// Altrimenti, verifichiamo se l'entità dinamica che stiamo toccando
		// può essere spinta a sua volta (Attraversamento Ricorsivo del Grafo).
		// Se anche UNA SOLA entità è bloccata, tutto il nostro fronte di spinta è bloccato!
		if !entry.Penetrable() {
			return false
		}
	}
	// Se nessuna entità oppone resistenza anelastica infinita, il bucket cede.
	return true
}

// Add inserts or updates a collision entry in the bucket based on penetration depth and topological deduplication criteria.
func (b *CollisionBucket) Add(bucket BucketType, lCage *CollisionCage, rCage *CollisionCage, rFace *Face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, maxZ float64, iMode int) *CageEntry {
	for i := 0; i < b.containerCounter; i++ {
		existing := b.container[i]
		// 1. DEDUPLICAZIONE PER ENTITÀ DINAMICHE (Contact Reduction)
		// Se questa faccia appartiene allo STESSO oggetto dinamico che abbiamo già registrato in QUESTO bucket...
		if rCage != nil && existing.rCage != nil {
			if rCage.object.GetEntity().GetId() == existing.rCage.object.GetEntity().GetId() {
				if penetration > existing.penetration {
					existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
				}
				return nil // Interrompiamo: abbiamo già gestito questo oggetto in questo bucket
			}
		}
		// 2. DEDUPLICAZIONE TOPOLOGICA (Geometria Statica Coplanare)
		// Se il dot product è ~1.0, i due triangoli formano un piano continuo (Triangle Soup statica)
		if dot := (normalX * existing.nX) + (normalY * existing.nY) + (normalZ * existing.nZ); dot > 0.999 {
			// Aggiorniamo il vincolo solo se la nuova penetrazione è più profonda
			if penetration > existing.penetration {
				existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
			}
			return nil
		}
	}

	// Insert a new plane into the non-full bucket
	if b.containerCounter < FacesPerBucket {
		target := b.spare[b.containerCounter]
		target.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
		b.container[b.containerCounter] = target
		b.containerCounter++
		return target
	}

	// Replace the least relevant face
	minIdx := 0
	minPen := b.container[0].penetration
	for i := 1; i < FacesPerBucket; i++ {
		if b.container[i].penetration < minPen {
			minPen = b.container[i].penetration
			minIdx = i
		}
	}
	if penetration > minPen {
		b.container[minIdx].Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
	}
	return nil
}

// CollisionCage represents a spatial structure used for managing collision detection and resolution in a 3D environment.
type CollisionCage struct {
	seen                map[*CollisionCage]bool
	object              IThing
	buckets             [BucketSize]*CollisionBucket
	ellipsoid           *physics.Entity
	ellipsoidLocal      [4]*physics.Entity
	cX, cY, cZ          float64
	dX, dY, dZ          float64
	tX, tY, tZ          float64
	eRadX, eRadY, eRadZ float64
	volume              *Volume
	distance            float64
	slots               []*CageEntry
	slotsLen            int
	maxStep             float64
	faces               []*CageFaces
	facesIdx            int
}

// NewCollisionCage initializes and returns a pointer to a new CollisionCage instance for the provided IThing entity.
func NewCollisionCage(object IThing) *CollisionCage {
	c := &CollisionCage{
		seen:      make(map[*CollisionCage]bool),
		object:    object,
		ellipsoid: physics.NewEntity(0, 0, 0, 0),
		volume:    nil,
		slots:     make([]*CageEntry, TotalSlots),
		slotsLen:  0,
		faces:     make([]*CageFaces, 8),
		facesIdx:  0,
	}
	for i := 0; i < len(c.faces); i++ {
		c.faces[i] = &CageFaces{} // Inizializzazione puntatori per zero-allocation
	}
	for i := BucketType(0); i < BucketSize; i++ {
		c.buckets[i] = NewCollisionBucket(i)
	}
	for i := 0; i < len(c.ellipsoidLocal); i++ {
		c.ellipsoidLocal[i] = physics.NewEntity(0, 0, 0, 0)
	}
	return c
}

// Rebuild updates the collision cage's geometry, displacement, and internal buckets based on the provided maximum step size.
func (s *CollisionCage) Rebuild(maxStep float64) {
	entity := s.object.GetEntity()
	// Calculate Half-Extents
	s.eRadX, s.eRadY, s.eRadZ = entity.GetSizeCenter()
	// Calculate center for Broad-Phase
	s.cX, s.cY, s.cZ = entity.GetCenter()
	// Calculate Position
	s.dX, s.dY, s.dZ = entity.GetDisplacement()

	s.tX, s.tY, s.tZ = s.cX+s.dX, s.cY+s.dY, s.cZ+s.dZ

	// Calculate absolute extremes (Broad-Phase Swept Volume)
	minX := s.cX - s.eRadX + math.Min(0, s.dX)
	maxX := s.cX + s.eRadX + math.Max(0, s.dX)
	minY := s.cY - s.eRadY + math.Min(0, s.dY)
	maxY := s.cY + s.eRadY + math.Max(0, s.dY)
	minZ := s.cZ - s.eRadZ + math.Min(0, s.dZ)
	maxZ := s.cZ + s.eRadZ + math.Max(0, s.dZ)

	// Canonical mapping for Rect/AABB
	s.ellipsoid.Rebuild(minX, minY, minZ, maxX-minX, maxY-minY, maxZ-minZ)

	for i := 0; i < len(s.buckets); i++ {
		s.buckets[i].Rebuild()
	}

	s.volume = nil
	s.distance = math.MaxFloat64

	s.slotsLen = 0

	for k := range s.seen {
		delete(s.seen, k)
	}
	s.maxStep = maxStep
}

// AddFace adds a face to the CollisionCage, expanding the storage if necessary to accommodate new entries.
func (s *CollisionCage) AddFace(rFace *Face, lFace *Face) {
	if s.facesIdx >= len(s.faces) {
		n := make([]*CageFaces, len(s.faces)*2)
		copy(n, s.faces)
		// Popolamento dei nuovi puntatori dopo il resize
		for i := len(s.faces); i < len(n); i++ {
			n[i] = &CageFaces{}
		}
		s.faces = n
	}
	f := s.faces[s.facesIdx]
	f.rFace = rFace
	f.lFace = lFace
	s.facesIdx++
}

// Commit resolves collisions between the current CollisionCage and an optional target CollisionCage.
func (s *CollisionCage) Commit(rCage *CollisionCage) {
	lAABB := s.ellipsoid.GetAABB()

	var rOffX, rOffY, rOffZ float64
	var lOffX, lOffY, lOffZ float64
	if rCage != nil {
		rOffX, rOffY, rOffZ = rCage.object.GetEntity().GetCenter()
		lOffX, lOffY, lOffZ = s.object.GetEntity().GetCenter()
	}

	for x := 0; x < s.facesIdx; x++ {
		faces := s.faces[x]
		rFace := faces.rFace
		lFace := faces.lFace

		var b BucketType
		var dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ float64

		// DISPATCHER: Mesh-vs-Mesh (6 DOF) o Ellipsoid-vs-Plane
		if lFace != nil && rCage != nil {
			//manifold := s.computeManifold(rFace, lFace, rOffX, rOffY, rOffZ, lOffX, lOffY, lOffZ)
			//b = manifold.Bucket
			//dist = manifold.HitDist
			//pen = manifold.Depth
			//nX, nY, nZ = manifold.NormalX, manifold.NormalY, manifold.NormalZ
			//p0x, p0y, p0z = manifold.P0X, manifold.P0Y, manifold.P0Z
			//minOverlap = manifold.MinOverlap
			//rMaxZ = manifold.RMaxZ
			b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ = s.computeFaceMeshVsMesh(rFace, lFace, rOffX, rOffY, rOffZ, lOffX, lOffY, lOffZ)
		} else {
			b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ = s.computeFace(lAABB, rFace, rOffX, rOffY, rOffZ)
		}

		// Volume Priority per le mesh statiche
		iMode := ImpactElastic
		if rCage == nil {
			iMode = ImpactInelastic
			if dist < s.distance {
				if volume := rFace.GetParent(); volume != nil {
					s.volume = volume
					s.distance = dist
				}
			}
		}

		_, texKind := rFace.GetMaterialDetails()
		if texKind == int(config.MaterialKindSky) {
			continue // Skybox/transparent: ignore collision
		}

		if pen <= 0 {
			continue
		}

		// If the penetration calculated from the infinite half-space exceeds the physical AABB limit,
		// we are intersecting the projection of a phantom orthogonal plane. Discard it.
		if pen > minOverlap+satFilterEpsilon { // SAT filter (Anti-Phantom Plane)
			continue
		}

		if b.IsWall() {
			baseZ := s.GetBaseZ()
			if rMaxZ <= baseZ { // down-hill (going downhill)
				continue
			}
			stepZ := baseZ + s.maxStep
			if rMaxZ <= stepZ { // up-hill (climbable step)
				iMode = ImpactStep
			}
		}
		s.addToBucket(b, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, rMaxZ, iMode)
	}
	s.facesIdx = 0
}

// computeFace computes the collision interaction with a given face and returns bucket type, distances, penetration, normals, and vertex coordinates.
func (s *CollisionCage) computeFace(lAABB *physics.AABB, rFace *Face, offX, offY, offZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ
	// Translation (Local -> World)
	p0x, p0y, p0z := rFace.tri[0].X+offX, rFace.tri[0].Y+offY, rFace.tri[0].Z+offZ
	distStart := (s.cX-p0x)*nX + (s.cY-p0y)*nY + (s.cZ-p0z)*nZ
	var bucket BucketType
	// Universal normalization
	if solidWE || solidNS {
		// Facing Normalization: Forces the plane to oppose the thing
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
			distStart = -distStart
		}
		// Wall Bucket Assignment
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
		// Exact elevation evaluation (Plane Z at Center X,Y)
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
	// Minkowski / Support Mapping for Ellipsoids
	rayEff := math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))
	distTarget := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
	dist := distTarget - rayEff
	penetration := rayEff - distTarget

	// sat filter (Anti-Phantom Plane)
	rFaceAABB := rFace.GetAABB()
	// world space translation
	rMinX := rFaceAABB.GetMinX() + offX
	rMaxX := rFaceAABB.GetMaxX() + offX
	rMinY := rFaceAABB.GetMinY() + offY
	rMaxY := rFaceAABB.GetMaxY() + offY
	rMinZ := rFaceAABB.GetMinZ() + offZ
	rMaxZ := rFaceAABB.GetMaxZ() + offZ
	oX := math.Max(0.0, math.Min(lAABB.GetMaxX()-rMinX, rMaxX-lAABB.GetMinX()))
	oY := math.Max(0.0, math.Min(lAABB.GetMaxY()-rMinY, rMaxY-lAABB.GetMinY()))
	oZ := math.Max(0.0, math.Min(lAABB.GetMaxZ()-rMinZ, rMaxZ-lAABB.GetMinZ()))
	// La reale penetrazione volumetrica massima possibile per questa specifica faccia
	minOverlap := math.Min(oX, math.Min(oY, oZ))

	return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ
}

// computeFaceMeshVsMesh performs collision detection between two 3D faces, calculating penetration depth and overlap metrics.
// rFace and lFace represent the remote and local mesh faces, respectively.
// rOffX, rOffY, rOffZ specify the position offsets for the remote face in world space.
// lOffX, lOffY, lOffZ specify the position offsets for the local face in world space.
// Returns the collision bucket, hit distance, maximum penetration depth, normal vector, anchor point, and overlap metrics.
func (s *CollisionCage) computeFaceMeshVsMesh(rFace *Face, lFace *Face, rOffX, rOffY, rOffZ float64, lOffX, lOffY, lOffZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	// 1. Dati del piano remoto (Il Muro/Ostacolo)
	nX, nY, nZ := rFace.GetNormal()

	// Punto di ancoraggio del piano remoto in World Space
	p0x := rFace.tri[0].X + rOffX
	p0y := rFace.tri[0].Y + rOffY
	p0z := rFace.tri[0].Z + rOffZ

	// 2. Normalizzazione
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	distStart := (s.cX-p0x)*nX + (s.cY-p0y)*nY + (s.cZ-p0z)*nZ
	var bucket BucketType

	if solidWE || solidNS {
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
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
			}
		} else {
			bucket = BucketCeiling
			if nZ > 0 {
				nX, nY, nZ = -nX, -nY, -nZ
			}
		}
	}

	// 6 DOF: Calcolo Penetrazione Esatta Vertice-Piano
	maxPenetration := -math.MaxFloat64
	var hitDist float64

	for i := 0; i < 3; i++ {
		// Trasliamo il vertice della nostra mesh in World Space
		vx := lFace.tri[i].X + lOffX
		vy := lFace.tri[i].Y + lOffY
		vz := lFace.tri[i].Z + lOffZ
		// Distanza ortogonale del vertice dal piano remoto
		vDist := (vx-p0x)*nX + (vy-p0y)*nY + (vz-p0z)*nZ
		// La penetrazione è negativa rispetto alla distanza
		pen := -vDist
		if pen > maxPenetration {
			maxPenetration = pen
			hitDist = vDist
		}
	}

	// Se la compenetrazione massima è <= 0, scartiamo.
	if maxPenetration <= 0 {
		return bucket, hitDist, maxPenetration, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	// SAT Filter basato sulle AABB locali portate in World Space
	rFaceAABB := rFace.GetAABB()
	rMinX := rFaceAABB.GetMinX() + rOffX
	rMaxX := rFaceAABB.GetMaxX() + rOffX
	rMinY := rFaceAABB.GetMinY() + rOffY
	rMaxY := rFaceAABB.GetMaxY() + rOffY
	rMinZ := rFaceAABB.GetMinZ() + rOffZ
	rMaxZ := rFaceAABB.GetMaxZ() + rOffZ

	lFaceAABB := lFace.GetAABB()
	lMinX := lFaceAABB.GetMinX() + lOffX
	lMaxX := lFaceAABB.GetMaxX() + lOffX
	lMinY := lFaceAABB.GetMinY() + lOffY
	lMaxY := lFaceAABB.GetMaxY() + lOffY
	lMinZ := lFaceAABB.GetMinZ() + lOffZ
	lMaxZ := lFaceAABB.GetMaxZ() + lOffZ

	oX := math.Max(0.0, math.Min(lMaxX-rMinX, rMaxX-lMinX))
	oY := math.Max(0.0, math.Min(lMaxY-rMinY, rMaxY-lMinY))
	oZ := math.Max(0.0, math.Min(lMaxZ-rMinZ, rMaxZ-lMinZ))

	minOverlap := math.Min(oX, math.Min(oY, oZ))

	return bucket, hitDist, maxPenetration, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ
}

func (s *CollisionCage) computeManifold(rFace *Face, lFace *Face, rOffX, rOffY, rOffZ float64, lOffX, lOffY, lOffZ float64) ContactManifold {
	manifold := ContactManifold{Hit: false}

	// 1. FAST-FAIL: SAT Filter AABB spostato in testa
	rFaceAABB := rFace.GetAABB()
	rMinX, rMaxX := rFaceAABB.GetMinX()+rOffX, rFaceAABB.GetMaxX()+rOffX
	rMinY, rMaxY := rFaceAABB.GetMinY()+rOffY, rFaceAABB.GetMaxY()+rOffY
	rMinZ, rMaxZ := rFaceAABB.GetMinZ()+rOffZ, rFaceAABB.GetMaxZ()+rOffZ

	lFaceAABB := lFace.GetAABB()
	lMinX, lMaxX := lFaceAABB.GetMinX()+lOffX, lFaceAABB.GetMaxX()+lOffX
	lMinY, lMaxY := lFaceAABB.GetMinY()+lOffY, lFaceAABB.GetMaxY()+lOffY
	lMinZ, lMaxZ := lFaceAABB.GetMinZ()+lOffZ, lFaceAABB.GetMaxZ()+lOffZ

	oX := max(0.0, min(lMaxX-rMinX, rMaxX-lMinX))
	oY := max(0.0, min(lMaxY-rMinY, rMaxY-lMinY))
	oZ := max(0.0, min(lMaxZ-rMinZ, rMaxZ-lMinZ))

	if oX <= 0 || oY <= 0 || oZ <= 0 {
		return manifold // Nessuna intersezione spaziale, abort immediato
	}
	manifold.MinOverlap = min(oX, min(oY, oZ))
	manifold.RMaxZ = rMaxZ

	// 2. Setup Vertici in World Space
	var rV, lV [3]geometry.XYZ
	for i := 0; i < 3; i++ {
		rV[i] = geometry.XYZ{X: rFace.tri[i].X + rOffX, Y: rFace.tri[i].Y + rOffY, Z: rFace.tri[i].Z + rOffZ}
		lV[i] = geometry.XYZ{X: lFace.tri[i].X + lOffX, Y: lFace.tri[i].Y + lOffY, Z: lFace.tri[i].Z + lOffZ}
	}

	// 3. Normali e Calcolo Legacy Bucket
	nX, nY, nZ := rFace.GetNormal()
	manifold.P0X, manifold.P0Y, manifold.P0Z = rV[0].X, rV[0].Y, rV[0].Z

	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ
	distStart := (s.cX-manifold.P0X)*nX + (s.cY-manifold.P0Y)*nY + (s.cZ-manifold.P0Z)*nZ

	if solidWE || solidNS {
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
		}
		if solidWE {
			if nX < 0 {
				manifold.Bucket = BucketWallWest
			} else {
				manifold.Bucket = BucketWallEast
			}
		} else {
			if nY < 0 {
				manifold.Bucket = BucketWallNorth
			} else {
				manifold.Bucket = BucketWallSouth
			}
		}
	} else {
		planeZ := manifold.P0Z
		if nAbsZ > 1e-5 {
			planeZ = manifold.P0Z - (nX*(s.cX-manifold.P0X)+nY*(s.cY-manifold.P0Y))/nZ
		}
		if s.cZ >= planeZ {
			manifold.Bucket = BucketFloor
			if nZ < 0 {
				nX, nY, nZ = -nX, -nY, -nZ
			}
		} else {
			manifold.Bucket = BucketCeiling
			if nZ > 0 {
				nX, nY, nZ = -nX, -nY, -nZ
			}
		}
	}
	manifold.NormalX, manifold.NormalY, manifold.NormalZ = nX, nY, nZ

	// 4. SAT FULL 11-AXIS E CALCOLO PUNTO DI CONTATTO 3D
	lnX, lnY, lnZ := lFace.GetNormal()
	axes := [11]geometry.XYZ{
		{X: nX, Y: nY, Z: nZ},    // 0: Normale Remota (Faccia vs Vertice)
		{X: lnX, Y: lnY, Z: lnZ}, // 1: Normale Locale (Vertice vs Faccia)
	}

	rE := [3]geometry.XYZ{subXYZ(rV[1], rV[0]), subXYZ(rV[2], rV[1]), subXYZ(rV[0], rV[2])}
	lE := [3]geometry.XYZ{subXYZ(lV[1], lV[0]), subXYZ(lV[2], lV[1]), subXYZ(lV[0], lV[2])}

	axisIdx := 2
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			crossP := crossXYZ(rE[i], lE[j])
			// Se i due edge sono paralleli (cross quasi nullo), l'asse è degenere e viene scartato
			if dotXYZ(crossP, crossP) > 1e-8 {
				axes[axisIdx] = normalizeXYZ(crossP)
			}
			axisIdx++
		}
	}

	bestPenetration := math.MaxFloat64
	bestAxisIdx := -1

	// Risoluzione SAT (Proiezione di R e L su tutti gli 11 assi)
	for i := 0; i < 11; i++ {
		if axes[i].X == 0 && axes[i].Y == 0 && axes[i].Z == 0 {
			continue
		}
		rMin, rMax := projectTri(axes[i], rV)
		lMin, lMax := projectTri(axes[i], lV)

		// Check separazione
		if lMax < rMin || rMax < lMin {
			return manifold // Esiste un asse separatore. Hit = false.
		}

		// Calcolo compenetrazione
		pen := math.Min(rMax-lMin, lMax-rMin)
		if pen < bestPenetration {
			bestPenetration = pen
			bestAxisIdx = i
			// Assicuriamo che la normale punti da Remote verso Local
			if rMin < lMin {
				axes[i] = geometry.XYZ{X: -axes[i].X, Y: -axes[i].Y, Z: -axes[i].Z}
			}
		}
	}

	// 5. Estrazione Feature e Punto Esatto in base all'asse vincente
	manifold.Hit = true
	manifold.Depth = bestPenetration

	// Conserviamo il calcolo legacy per hitDist per non rompere il tuo StageResolve attuale
	for i := 0; i < 3; i++ {
		vDist := (lV[i].X-manifold.P0X)*nX + (lV[i].Y-manifold.P0Y)*nY + (lV[i].Z-manifold.P0Z)*nZ
		if -vDist > manifold.HitDist {
			manifold.HitDist = vDist // Legacy hitDist maintain
		}
	}

	if bestAxisIdx == 0 {
		// Urto planare (Vertice Locale penetra Faccia Remota)
		manifold.PointX, manifold.PointY, manifold.PointZ = findDeepestPoint(axes[0], lV)
	} else if bestAxisIdx == 1 {
		// Urto planare inverso (Vertice Remoto penetra Faccia Locale)
		manifold.PointX, manifold.PointY, manifold.PointZ = findDeepestPoint(geometry.XYZ{X: -axes[1].X, Y: -axes[1].Y, Z: -axes[1].Z}, rV)
	} else {
		// Urto Edge-Edge (Segmenti Sghembi). Ricaviamo gli indici rE e lE.
		eIdx := bestAxisIdx - 2
		rIdx, lIdx := eIdx/3, eIdx%3
		p1, p2 := closestPointSegmentSegment(rV[rIdx], rE[rIdx], lV[lIdx], lE[lIdx])
		// Il punto di contatto è il punto medio tra i due segmenti intersecanti
		manifold.PointX = (p1.X + p2.X) * 0.5
		manifold.PointY = (p1.Y + p2.Y) * 0.5
		manifold.PointZ = (p1.Z + p2.Z) * 0.5
	}

	return manifold
}

// addToBucket adds a CollisionCage or Face into the specified bucket with given parameters to manage collision resolution.
func (s *CollisionCage) addToBucket(bucket BucketType, rCage *CollisionCage, rFace *Face, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode int) {
	cage := s.buckets[bucket].Add(bucket, s, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode)
	if cage != nil {
		s.slots[s.slotsLen] = cage
		s.slotsLen++
	}
}

// TranslateWorldToLocal translates the AABB of a target `CollisionCage` from world space to local space for a given slot.
func (s *CollisionCage) TranslateWorldToLocal(slot int, deltaX, deltaY, deltaZ float64) *physics.Entity {
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

// TranslateCage computes the local translation of a target CollisionCage and returns relative position deltas in 3D space.
func (s *CollisionCage) TranslateCage(slot int, rCage *CollisionCage) (physics.IAABB, float64, float64, float64) {
	lCx, lCy, lCz := s.object.GetEntity().GetCenter()
	rCx, rCy, rCz := rCage.object.GetEntity().GetCenter()
	//lCx, lCy, lCz := s.GetAABB().GetCentroid()
	//rCx, rCy, rCz := rCage.GetAABB().GetCentroid()
	lEntityL := s.TranslateWorldToLocal(slot, rCx, rCy, rCz)
	return lEntityL, rCx - lCx, rCy - lCy, rCz - lCz
}

// Translate updates an ellipsoid slot's dimensions by applying delta values to the target AABB coordinates in world space.
func (s *CollisionCage) Translate(slot int, src physics.IAABB, deltaX, deltaY, deltaZ float64) *physics.Entity {
	w := s.ellipsoidLocal[slot]
	to := src.GetAABB()
	lMinX := to.GetMinX() + deltaX
	lMaxX := to.GetMaxX() + deltaX
	lMinY := to.GetMinY() + deltaY
	lMaxY := to.GetMaxY() + deltaY
	lMinZ := to.GetMinZ() + deltaZ
	lMaxZ := to.GetMaxZ() + deltaZ
	w.Rebuild(lMinX, lMinY, lMinZ, lMaxX-lMinX, lMaxY-lMinY, lMaxZ-lMinZ)
	return w
}

// HasSeen checks if the given CollisionCage has already been encountered in the current context.
func (s *CollisionCage) HasSeen(rCage *CollisionCage) bool {
	return s.seen[rCage]
}

// Seen marks the given CollisionCage as seen by the current CollisionCage.
func (s *CollisionCage) Seen(rCage *CollisionCage) {
	s.seen[rCage] = true
}

// GetBaseZ computes and returns the base Z-coordinate of the collision cage considering its center and Z-radius.
func (s *CollisionCage) GetBaseZ() float64 { return s.cZ - s.eRadZ }

// GetSlotsLen returns the count of active collision slots currently in use within the CollisionCage.
func (s *CollisionCage) GetSlotsLen() int { return s.slotsLen }

// GetSlot retrieves the CageEntry at the specified index from the slots array in the CollisionCage.
func (s *CollisionCage) GetSlot(i int) *CageEntry { return s.slots[i] }

// GetThing returns the IThing object associated with the CollisionCage.
func (s *CollisionCage) GetThing() IThing { return s.object }

// GetVolume retrieves the current volume associated with the collision cage. Returns nil if no volume is set.
func (s *CollisionCage) GetVolume() *Volume { return s.volume }

// GetRad returns the half-extents (eRadX, eRadY, eRadZ) of the collision cage's bounding ellipsoid.
func (s *CollisionCage) GetRad() (float64, float64, float64) { return s.eRadX, s.eRadY, s.eRadZ }

// GetC retrieves the central coordinates (cX, cY, cZ) of the CollisionCage object.
func (s *CollisionCage) GetC() (float64, float64, float64) { return s.cX, s.cY, s.cZ }

// GetDisplacement returns the displacement vector (dX, dY, dZ) of the CollisionCage.
func (s *CollisionCage) GetDisplacement() (float64, float64, float64) { return s.dX, s.dY, s.dZ }

// GetT retrieves the transformed coordinates (tX, tY, tZ) of the CollisionCage.
func (s *CollisionCage) GetT() (float64, float64, float64) { return s.tX, s.tY, s.tZ }

// BucketCount returns the total number of entries in the specified bucket.
func (s *CollisionCage) BucketCount(t BucketType) int { return s.buckets[t].Count() }

// GetAABB returns the axis-aligned bounding box (AABB) associated with the collision cage.
func (s *CollisionCage) GetAABB() *physics.AABB { return s.ellipsoid.GetAABB() }

// subXYZ subtracts two geometry.XYZ vectors component-wise and returns the resulting vector.
func subXYZ(a, b geometry.XYZ) geometry.XYZ {
	return geometry.XYZ{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

// dotXYZ computes the dot product of two 3D vectors represented by geometry.XYZ structs.
func dotXYZ(a, b geometry.XYZ) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// crossXYZ computes the cross product of two 3D vectors a and b, returning the resulting vector as geometry.XYZ.
func crossXYZ(a, b geometry.XYZ) geometry.XYZ {
	return geometry.XYZ{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}

// normalizeXYZ normalizes a 3D vector to unit length and returns the resulting vector.
func normalizeXYZ(v geometry.XYZ) geometry.XYZ {
	lenInv := 1.0 / math.Sqrt(v.X*v.X+v.Y*v.Y+v.Z*v.Z)
	return geometry.XYZ{X: v.X * lenInv, Y: v.Y * lenInv, Z: v.Z * lenInv}
}

// projectTri projects a triangle's vertices onto a given axis and returns the minimum and maximum projection values.
func projectTri(axis geometry.XYZ, v [3]geometry.XYZ) (float64, float64) {
	minP := dotXYZ(axis, v[0])
	maxP := minP
	for i := 1; i < 3; i++ {
		p := dotXYZ(axis, v[i])
		if p < minP {
			minP = p
		}
		if p > maxP {
			maxP = p
		}
	}
	return minP, maxP
}

// findDeepestPoint computes the vertex in a triangle with the smallest projection onto a given direction vector.
// dir is the reference direction for projection.
// v is an array of three 3D points representing the vertices of a triangle.
// Returns the X, Y, and Z coordinates of the vertex with the smallest projection.
func findDeepestPoint(dir geometry.XYZ, v [3]geometry.XYZ) (float64, float64, float64) {
	bestP := dotXYZ(dir, v[0])
	bestIdx := 0
	for i := 1; i < 3; i++ {
		p := dotXYZ(dir, v[i])
		if p < bestP {
			bestP = p
			bestIdx = i
		}
	}
	return v[bestIdx].X, v[bestIdx].Y, v[bestIdx].Z
}

// closestPointSegmentSegment calculates the closest points between two 3D line segments defined by their points and directions.
// p1 and d1 define the starting point and direction vector of the first segment, respectively.
// p2 and d2 define the starting point and direction vector of the second segment, respectively.
// Returns the closest points on the two segments as two geometry.XYZ instances.
func closestPointSegmentSegment(p1, d1, p2, d2 geometry.XYZ) (geometry.XYZ, geometry.XYZ) {
	r := subXYZ(p1, p2)
	a, e, f := dotXYZ(d1, d1), dotXYZ(d2, d2), dotXYZ(d2, r)
	c := dotXYZ(d1, r)
	b := dotXYZ(d1, d2)
	denom := a*e - b*b
	s, t := 0.0, 0.0
	if denom != 0.0 {
		s = max(0.0, min(1.0, (b*f-c*e)/denom))
	}
	t = (b*s + f) / e
	t = max(0.0, min(1.0, t))
	s = (b*t - c) / a
	s = max(0.0, min(1.0, s))
	c1 := geometry.XYZ{X: p1.X + d1.X*s, Y: p1.Y + d1.Y*s, Z: p1.Z + d1.Z*s}
	c2 := geometry.XYZ{X: p2.X + d2.X*t, Y: p2.Y + d2.Y*t, Z: p2.Z + d2.Z*t}
	return c1, c2
}
