package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
const (
	// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
	ImpactInelastic = 1
	// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
	ImpactElastic = 2
	// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
	ImpactStep = 3
)

// satFilterEpsilon defines the tolerance threshold used in SAT to filter phantom plane projections in collision detection.
const satFilterEpsilon = 0.05

// planeEpsilon is a tolerance value used for floating-point precision when comparing coplanar planes or distances.
const planeEpsilon = 0.01

// BucketType represents a categorized type for spatial partitioning or collision detection in 3D environments.
type BucketType int

// BucketWallWest represents the west (-X) wall bucket type.
// BucketWallEast represents the east (+X) wall bucket type.
// BucketWallNorth represents the north (-Y) wall bucket type.
// BucketWallSouth represents the south (+Y) wall bucket type.
// BucketCeiling represents the ceiling (-Z) bucket type.
// BucketFloor represents the floor (+Z) bucket type.
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

// IsWall determines if the BucketType value represents a wall (west, east, north, or south).
func (p BucketType) IsWall() bool {
	return p == BucketWallWest || p == BucketWallEast || p == BucketWallNorth || p == BucketWallSouth
}

// BucketSize defines the size of a single bucket, calculated as BucketFloor + 1.
// FacesPerBucket specifies the number of faces allocated per bucket.
// TotalSlots represents the total available slots, computed as BucketSize * FacesPerBucket.
const (
	BucketSize     = BucketFloor + 1
	FacesPerBucket = 4
	TotalSlots     = BucketSize * FacesPerBucket
)

// ContactManifold represents the result of a collision detection process, including contact details and metadata.
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

// CageFaces represents a pair of adjacent faces (left and right) within a collision cage structure.
type CageFaces struct {
	lFace *Face
	rFace *Face
}

// CageEntry represents a collision entry used during spatial computations in the collision detection system.
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

// GetRemoteFace returns the remote face associated with the CageEntry structure.
func (s *CageEntry) GetRemoteFace() *Face { return s.rFace }

// GetDistance returns the precomputed distance value associated with the CageEntry instance.
func (s *CageEntry) GetDistance() float64 { return s.dist }

// IsDynamic determines if the instance is associated with a remote collision cage.
func (s *CageEntry) IsDynamic() bool { return s.rCage != nil }

// GetMaxZ returns the maximum Z-coordinate of the CageEntry.
func (s *CageEntry) GetMaxZ() float64 { return s.maxZ }

// GetBucket retrieves the BucketType associated with the CageEntry instance.
func (s *CageEntry) GetBucket() BucketType { return s.bucket }

// GetImpactMode returns the impact mode of the collision as an integer value.
func (s *CageEntry) GetImpactMode() int {
	return s.iMode
}

// GetNormal returns the components of the contact normal vector (nX, nY, nZ) as a tuple of three float64 values.
func (s *CageEntry) GetNormal() (float64, float64, float64) { return s.nX, s.nY, s.nZ }

// GetPenetration returns the penetration depth of the collision measured by the cage entry.
func (s *CageEntry) GetPenetration() float64 { return s.penetration }

// NewCollisionFace creates and returns a new instance of CageEntry initialized with default zero values.
func NewCollisionFace() *CageEntry {
	return &CageEntry{}
}

// Rebuild reinitializes the CageEntry with the given parameters and updates all its attributes accordingly.
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

// Penetrable determines if the CageEntry can be penetrated based on its collision mode and the state of adjacent entities.
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

// CollisionBucket holds collision data for a specific bucket, including spare and active cage entries and counters.
type CollisionBucket struct {
	bucket           BucketType
	spare            [FacesPerBucket]*CageEntry
	container        [FacesPerBucket]*CageEntry
	containerCounter int
}

// NewCollisionBucket creates a new CollisionBucket for a given BucketType, initializing its spare and container arrays.
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

// Rebuild resets the containerCounter to 0, clearing the current collision state for the bucket.
func (b *CollisionBucket) Rebuild() {
	b.containerCounter = 0
}

// Count returns the total number of entries currently stored in the CollisionBucket.
func (b *CollisionBucket) Count() int {
	return b.containerCounter
}

// Penetrable determines if the CollisionBucket can be penetrated starting from a specified entity ID.
// Returns false if any entity in the bucket provides infinite inelastic resistance or cannot be recursively pushed.
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

// Add inserts or updates a collision entry in the bucket based on specific conditions, ensuring deduplication and prioritization.
func (b *CollisionBucket) Add(bucket BucketType, lCage *CollisionCage, rCage *CollisionCage, rFace *Face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, maxZ float64, iMode int) *CageEntry {
	for i := 0; i < b.containerCounter; i++ {
		existing := b.container[i]
		// DEDUPLICAZIONE PER ENTITÀ DINAMICHE (Contact Reduction)
		// Se questa faccia appartiene allo STESSO oggetto dinamico che abbiamo già registrato in QUESTO bucket...
		if rCage != nil && existing.rCage != nil {
			if rCage.object.GetEntity().GetId() == existing.rCage.object.GetEntity().GetId() {
				if penetration > existing.penetration {
					existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
				}
				return nil // Interrompiamo: abbiamo già gestito questo oggetto in questo bucket
			}
		}
		// DEDUPLICAZIONE TOPOLOGICA (Geometria Statica Coplanare)
		// Se il dot product è ~1.0, i due triangoli formano un piano continuo (Triangle Soup statica)
		if dot := (normalX * existing.nX) + (normalY * existing.nY) + (normalZ * existing.nZ); dot > 0.999 {
			planeDistance := math.Abs(
				(p0x-existing.p0X)*existing.nX +
					(p0y-existing.p0Y)*existing.nY +
					(p0z-existing.p0Z)*existing.nZ,
			)
			if planeDistance < planeEpsilon {
				// Un vincolo anelastico (muro solido) ha sempre priorità su uno scalino (ImpactStep) coplanare
				if existing.iMode == ImpactStep && iMode == ImpactInelastic {
					existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
					return nil
				}
				if existing.iMode == ImpactInelastic && iMode == ImpactStep {
					return nil
				}
				if penetration > existing.penetration || (iMode == ImpactStep && math.Abs(penetration-existing.penetration) <= planeEpsilon && maxZ > existing.maxZ) {
					existing.Rebuild(bucket, lCage, rCage, rFace, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, maxZ, iMode)
				}
				return nil
			}
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

// CollisionCage represents a data structure for managing collision information for a physical entity.
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

// NewCollisionCage creates and returns a new instance of CollisionCage initialized with the specified IThing object.
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

// Rebuild recalculates the collision cage dimensions, position, and broad-phase swept volume using the given maxStep value.
func (s *CollisionCage) Rebuild(maxStep float64) {
	entity := s.object.GetEntity()
	// Calculate Half-Extents
	s.eRadX, s.eRadY, s.eRadZ = entity.GetSizeCenter()
	// Calculate center for Broad-Phase
	s.cX, s.cY, s.cZ = entity.GetCenter()
	// Calculate Position
	s.dX, s.dY, s.dZ = entity.GetFrameDisplacement()

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

// AddFace adds a pair of faces (rFace and lFace) to the CollisionCage, resizing the internal slice if needed.
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

// Commit processes collision data between two CollisionCages, resolving interactions based on material and displacement factors.
func (s *CollisionCage) Commit(rCage *CollisionCage) {
	lAABB := s.ellipsoid.GetAABB()

	var rOffX, rOffY, rOffZ float64
	var lOffX, lOffY, lOffZ float64
	if rCage != nil {
		rOffX, rOffY, rOffZ = rCage.object.GetVerticesCenter()
		rDX, rDY, rDZ := rCage.object.GetEntity().GetFrameDisplacement()
		rOffX += rDX
		rOffY += rDY
		rOffZ += rDZ

		lOffX, lOffY, lOffZ = s.object.GetVerticesCenter()
		lOffX += s.dX
		lOffY += s.dY
		lOffZ += s.dZ
	}

	for x := 0; x < s.facesIdx; x++ {
		faces := s.faces[x]
		rFace := faces.rFace
		lFace := faces.lFace

		//TODO SMALL PATCH - FULL IMPLEMENTATION REQUIRED
		if mat := rFace.material; mat != nil && mat.Kind() == int(config.MaterialKindLiquid) {
			//fmt.Println("Liquid material detected")
			continue
		}

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

// computeFace calculates the penetration and distance of a face relative to an ellipsoid's motion and assigns a collision bucket.
func (s *CollisionCage) computeFace(lAABB *aabb.AABB, rFace *Face, offX, offY, offZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return 0, math.MaxFloat64, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

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

	if penetration <= 0 {
		return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	oX := math.Max(0.0, math.Min(lAABB.GetMaxX()-rMinX, rMaxX-lAABB.GetMinX()))
	oY := math.Max(0.0, math.Min(lAABB.GetMaxY()-rMinY, rMaxY-lAABB.GetMinY()))
	oZ := math.Max(0.0, math.Min(lAABB.GetMaxZ()-rMinZ, rMaxZ-lAABB.GetMinZ()))
	if oX <= 0 || oY <= 0 || oZ <= 0 {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	// Separazione tra la compenetrazione già presente a t = 0 (pen0) e l'affondamento prodotto dal moto nel frame (deltaPen).
	// Un piano infinito è fantasma solo se pen0 (a t = 0) eccede l'overlap tangenziale con l'AABB della faccia;
	// l'affondamento deltaPen accumulato nel frame e il transito sulle cuciture tra triangoli adiacenti restano preservati.
	pen0 := math.Max(0.0, rayEff-distStart)
	deltaPen := math.Max(0.0, penetration-pen0)

	var minOverlap float64
	if solidWE {
		var normOX float64
		if nX > 0 {
			normOX = math.Max(oX, rMaxX-(s.tX-s.eRadX))
		} else {
			normOX = math.Max(oX, (s.tX+s.eRadX)-rMinX)
		}
		minOverlap = math.Min(normOX, math.Min(oY, oZ)+deltaPen)
	} else if solidNS {
		var normOY float64
		if nY > 0 {
			normOY = math.Max(oY, rMaxY-(s.tY-s.eRadY))
		} else {
			normOY = math.Max(oY, (s.tY+s.eRadY)-rMinY)
		}
		minOverlap = math.Min(normOY, math.Min(oX, oZ)+deltaPen)
	} else {
		var normOZ float64
		if nZ > 0 {
			normOZ = math.Max(oZ, rMaxZ-(s.tZ-s.eRadZ))
		} else {
			normOZ = math.Max(oZ, (s.tZ+s.eRadZ)-rMinZ)
		}
		minOverlap = math.Min(normOZ, math.Min(oX, oY)+deltaPen)
	}

	return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ
}

// sweptAxisInterval calculates the time interval of penetration in a swept axis-aligned scenario.
// Returns start and end of interval, and a boolean indicating if penetration occurs.
func sweptAxisInterval(rel, sumRad, imp float64) (float64, float64, bool) {
	if math.Abs(imp) <= 1e-9 {
		if math.Abs(rel) > sumRad+planeEpsilon {
			return 0, 0, false
		}
		return -math.MaxFloat64, math.MaxFloat64, true
	}
	t1 := (-sumRad - rel) / imp
	t2 := (sumRad - rel) / imp
	if t1 < t2 {
		return t1, t2, true
	}
	return t2, t1, true
}

// computeFaceMeshVsMesh determines the intersection and response between two mesh faces in 3D space with offsets.
// It calculates collision parameters such as entry/exit times, normals, overlap, and separation distance between meshes.
func (s *CollisionCage) computeFaceMeshVsMesh(rFace *Face, lFace *Face, rOffX, rOffY, rOffZ float64, lOffX, lOffY, lOffZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	// Dati del piano remoto (Il Muro/Ostacolo)
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	// Punto di ancoraggio del piano remoto in World Space a fine frame (t = 1),
	// poiché rOffX, rOffY, rOffZ includono già lo spostamento rDX, rDY, rDZ.
	p0x := rFace.tri[0].X + rOffX
	p0y := rFace.tri[0].Y + rOffY
	p0z := rFace.tri[0].Z + rOffZ

	rParent := rFace.GetParent()
	if rParent == nil {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}
	rEnt := rParent.GetEntity()
	rCx0, rCy0, rCz0 := rEnt.GetCenter()
	rDX, rDY, rDZ := rEnt.GetFrameDisplacement()
	rTx := rCx0 + rDX
	rTy := rCy0 + rDY
	rTz := rCz0 + rDZ
	rRadX, rRadY, rRadZ := rEnt.GetSizeCenter()

	// Punto di ancoraggio del piano remoto a inizio frame (t = 0)
	p0x0 := p0x - rDX
	p0y0 := p0y - rDY
	p0z0 := p0z - rDZ

	relX := s.cX - rCx0
	relY := s.cY - rCy0
	relZ := s.cZ - rCz0

	// Orientiamo la normale verso l'esterno del volume remoto (valutato in modo coerente a t = 1).
	// Scartiamo solo i piani visivi interni privi di tag fisico (es. quad billboard centrale a Y=0).
	outDot := (p0x-rTx)*nX + (p0y-rTy)*nY + (p0z-rTz)*nZ
	if math.Abs(outDot) <= planeEpsilon {
		if rFace.GetTag() == "" {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		if relX*nX+relY*nY+relZ*nZ < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
		}
	} else if outDot < 0 {
		nX, nY, nZ = -nX, -nY, -nZ
	}

	// La faccia remota deve essere rivolta verso il centro relativo iniziale dell'entità locale
	if relX*nX+relY*nY+relZ*nZ <= 0 {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	// Spostamento relativo nel frame corrente (Locale rispetto a Remoto)
	impX := s.dX - rDX
	impY := s.dY - rDY
	impZ := s.dZ - rDZ
	impDot := impX*nX + impY*nY + impZ*nZ
	if impDot > planeEpsilon {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	sumRadX := s.eRadX + rRadX
	sumRadY := s.eRadY + rRadY
	sumRadZ := s.eRadZ + rRadZ

	// Intervalli temporali completi [tEnter_i, tExit_i] su tutti e tre gli assi (inclusi gli assi già sovrapposti a t = 0)
	const toiTieEpsilon = 1e-4
	tEnterX, tExitX, okX := sweptAxisInterval(relX, sumRadX, impX)
	if !okX {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}
	tEnterY, tExitY, okY := sweptAxisInterval(relY, sumRadY, impY)
	if !okY {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}
	tEnterZ, tExitZ, okZ := sweptAxisInterval(relZ, sumRadZ, impZ)
	if !okZ {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	tEnter := math.Max(tEnterX, math.Max(tEnterY, tEnterZ))
	tExit := math.Min(tExitX, math.Min(tExitY, tExitZ))

	// Condizione necessaria e sufficiente di intersezione temporale AABB in t in [0, 1]
	if tEnter > tExit+toiTieEpsilon || tExit < -toiTieEpsilon || tEnter > 1.0+toiTieEpsilon {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	// Overlap bilaterale a t = 1 (per contatti discreti/persistenti) e penetrazione unidirezionale dal lato d'ingresso
	// (che resta positiva anche in caso di tunneling oltre il centro o oltre la parete opposta a t = 1).
	relTx := s.tX - rTx
	relTy := s.tY - rTy
	relTz := s.tZ - rTz

	boxOX := sumRadX - math.Abs(relTx)
	boxOY := sumRadY - math.Abs(relTy)
	boxOZ := sumRadZ - math.Abs(relTz)

	entryOX := sumRadX - relTx
	if relX < 0 || (relX == 0 && nX < 0) {
		entryOX = sumRadX + relTx
	}
	entryOY := sumRadY - relTy
	if relY < 0 || (relY == 0 && nY < 0) {
		entryOY = sumRadY + relTy
	}
	entryOZ := sumRadZ - relTz
	if relZ < 0 || (relZ == 0 && nZ < 0) {
		entryOZ = sumRadZ + relTz
	}

	sepX := math.Abs(relX) - sumRadX
	sepY := math.Abs(relY) - sumRadY
	sepZ := math.Abs(relZ) - sumRadZ

	prevOX := math.Max(0.0, -sepX)
	prevOY := math.Max(0.0, -sepY)
	prevOZ := math.Max(0.0, -sepZ)

	sweptMode := sepX > planeEpsilon || sepY > planeEpsilon || sepZ > planeEpsilon

	// Distanza iniziale a t = 0 del centro locale rispetto al piano remoto a t = 0
	distStart := (s.cX-p0x0)*nX + (s.cY-p0y0)*nY + (s.cZ-p0z0)*nZ
	rayEff := math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))

	if sweptMode {
		// In modalità swept il centro locale deve partire dal semispazio esterno del piano
		if distStart < -planeEpsilon {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		// Verifica che il contatto col piano avvenga all'interno della finestra di sovrapposizione AABB [tEnter, min(1, tExit)]
		suppStart := distStart - rayEff
		tWindowEnd := math.Min(1.0, tExit)
		if suppStart > planeEpsilon {
			if impDot >= -1e-9 {
				return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
			}
			tEnterPlane := suppStart / (-impDot)
			if tEnterPlane < tEnter-toiTieEpsilon || tEnterPlane > tWindowEnd+toiTieEpsilon {
				return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
			}
		} else if distStart+tEnter*impDot < -planeEpsilon {
			// A t = tEnter il centro locale aveva già oltrepassato il prolungamento infinito del piano (faccia fantasma)
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
	} else {
		// In modalità discreta (compenetrazione già presente a t = 0), scartiamo solo se l'intero supporto a t = 0 era già oltre il piano
		if distStart+rayEff < -planeEpsilon {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
	}

	// Determinazione degli assi temporali/MTV attivi (indipendente dall'asse dominante della faccia)
	var activeX, activeY, activeZ bool

	if sweptMode {
		activeX = sepX > planeEpsilon && tEnterX >= tEnter-toiTieEpsilon && relX*impX < -planeEpsilon
		activeY = sepY > planeEpsilon && tEnterY >= tEnter-toiTieEpsilon && relY*impY < -planeEpsilon
		activeZ = sepZ > planeEpsilon && tEnterZ >= tEnter-toiTieEpsilon && relZ*impZ < -planeEpsilon
	} else {
		// In modalità discreta (volumi già sovrapposti a t = 0 su tutti gli assi), se a t = 1 si sono separati non c'è impatto
		if boxOX <= 0 || boxOY <= 0 || boxOZ <= 0 {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		isDiscreteMTV := func(prevA, boxA, relA, impA, prevB, boxB, prevC, boxC float64) bool {
			if prevA > prevB+satFilterEpsilon || prevA > prevC+satFilterEpsilon {
				return false
			}
			tieB := math.Abs(prevA-prevB) <= satFilterEpsilon
			tieC := math.Abs(prevA-prevC) <= satFilterEpsilon
			if tieB && boxA > boxB+satFilterEpsilon && relA*impA >= -planeEpsilon {
				return false
			}
			if tieC && boxA > boxC+satFilterEpsilon && relA*impA >= -planeEpsilon {
				return false
			}
			return true
		}
		activeX = isDiscreteMTV(prevOX, boxOX, relX, impX, prevOY, boxOY, prevOZ, boxOZ)
		activeY = isDiscreteMTV(prevOY, boxOY, relY, impY, prevOX, boxOX, prevOZ, boxOZ)
		activeZ = isDiscreteMTV(prevOZ, boxOZ, relZ, impZ, prevOX, boxOX, prevOY, boxOY)
	}

	// Verifica di compatibilità geometrica tra la normale del piano remoto e gli assi di contatto attivi
	var axisOverlap float64
	matchedAxis := false

	pickOverlap := func(entryO, boxO, nComp float64) float64 {
		ov := boxO
		if sweptMode {
			ov = entryO
		}
		if nComp > planeEpsilon && nComp < 1.0 {
			return ov / nComp
		}
		return ov
	}

	if activeX && nAbsX > planeEpsilon && relX*nX > 0 && (!sweptMode || impX*nX < -planeEpsilon) {
		axisOverlap = pickOverlap(entryOX, boxOX, nAbsX)
		matchedAxis = true
	}
	if activeY && nAbsY > planeEpsilon && relY*nY > 0 && (!sweptMode || impY*nY < -planeEpsilon) {
		ov := pickOverlap(entryOY, boxOY, nAbsY)
		if !matchedAxis || ov > axisOverlap {
			axisOverlap = ov
			matchedAxis = true
		}
	}
	if activeZ && nAbsZ > planeEpsilon && relZ*nZ > 0 && (!sweptMode || impZ*nZ < -planeEpsilon) {
		ov := pickOverlap(entryOZ, boxOZ, nAbsZ)
		if !matchedAxis || ov > axisOverlap {
			axisOverlap = ov
			matchedAxis = true
		}
	}
	if !matchedAxis || axisOverlap <= 0 {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	// Calcolo Penetrazione Esatta Vertice-Piano (valutata a t = 1 per il push-out di StageApply)
	maxPenetration := -math.MaxFloat64
	var hitDist float64

	for i := 0; i < 3; i++ {
		vx := lFace.tri[i].X + lOffX
		vy := lFace.tri[i].Y + lOffY
		vz := lFace.tri[i].Z + lOffZ
		vDist := (vx-p0x)*nX + (vy-p0y)*nY + (vz-p0z)*nZ
		pen := -vDist
		if pen > maxPenetration {
			maxPenetration = pen
			hitDist = vDist
		}
	}

	if maxPenetration <= 0 || maxPenetration > axisOverlap+satFilterEpsilon {
		return 0, hitDist, maxPenetration, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	// Classificazione finale del Bucket sulla normale di contatto già determinata e orientata
	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	var bucket BucketType
	if solidWE {
		if nX < 0 {
			bucket = BucketWallWest
		} else {
			bucket = BucketWallEast
		}
	} else if solidNS {
		if nY < 0 {
			bucket = BucketWallNorth
		} else {
			bucket = BucketWallSouth
		}
	} else {
		if nZ > 0 {
			bucket = BucketFloor
		} else {
			if impZ <= planeEpsilon {
				return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
			}
			bucket = BucketCeiling
		}
	}

	rFaceAABB := rFace.GetAABB()
	rMaxZ := rFaceAABB.GetMaxZ() + rOffZ

	return bucket, hitDist, maxPenetration, nX, nY, nZ, p0x, p0y, p0z, axisOverlap, rMaxZ
}

// computeManifold calculates the collision manifold between two 3D faces with offset transformations.
// Returns a ContactManifold containing collision details or non-collision state.
func (s *CollisionCage) computeManifold(rFace *Face, lFace *Face, rOffX, rOffY, rOffZ float64, lOffX, lOffY, lOffZ float64) ContactManifold {
	cm := ContactManifold{Hit: false}

	// FAST-FAIL: SAT Filter AABB spostato in testa
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
		return cm // Nessuna intersezione spaziale, abort immediato
	}
	cm.MinOverlap = min(oX, min(oY, oZ))
	cm.RMaxZ = rMaxZ

	// Setup Vertici in World Space
	var rV, lV [3]geometry.XYZ
	for i := 0; i < 3; i++ {
		rV[i] = geometry.XYZ{X: rFace.tri[i].X + rOffX, Y: rFace.tri[i].Y + rOffY, Z: rFace.tri[i].Z + rOffZ}
		lV[i] = geometry.XYZ{X: lFace.tri[i].X + lOffX, Y: lFace.tri[i].Y + lOffY, Z: lFace.tri[i].Z + lOffZ}
	}

	// Normali e Calcolo Legacy Bucket
	nX, nY, nZ := rFace.GetNormal()
	cm.P0X, cm.P0Y, cm.P0Z = rV[0].X, rV[0].Y, rV[0].Z

	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ
	distStart := (s.cX-cm.P0X)*nX + (s.cY-cm.P0Y)*nY + (s.cZ-cm.P0Z)*nZ

	if solidWE || solidNS {
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
		}
		if solidWE {
			if nX < 0 {
				cm.Bucket = BucketWallWest
			} else {
				cm.Bucket = BucketWallEast
			}
		} else {
			if nY < 0 {
				cm.Bucket = BucketWallNorth
			} else {
				cm.Bucket = BucketWallSouth
			}
		}
	} else {
		planeZ := cm.P0Z
		if nAbsZ > 1e-5 {
			planeZ = cm.P0Z - (nX*(s.cX-cm.P0X)+nY*(s.cY-cm.P0Y))/nZ
		}
		if s.cZ >= planeZ {
			cm.Bucket = BucketFloor
			if nZ < 0 {
				nX, nY, nZ = -nX, -nY, -nZ
			}
		} else {
			cm.Bucket = BucketCeiling
			if nZ > 0 {
				nX, nY, nZ = -nX, -nY, -nZ
			}
		}
	}
	cm.NormalX, cm.NormalY, cm.NormalZ = nX, nY, nZ
	// SAT FULL 11-AXIS E CALCOLO PUNTO DI CONTATTO 3D
	lnX, lnY, lnZ := lFace.GetNormal()
	axes := [11]geometry.XYZ{
		{X: nX, Y: nY, Z: nZ},    // 0: Normale Remota (Faccia vs Vertice)
		{X: lnX, Y: lnY, Z: lnZ}, // 1: Normale Locale (Vertice vs Faccia)
	}

	rE := [3]geometry.XYZ{geometry.SubXYZ(rV[1], rV[0]), geometry.SubXYZ(rV[2], rV[1]), geometry.SubXYZ(rV[0], rV[2])}
	lE := [3]geometry.XYZ{geometry.SubXYZ(lV[1], lV[0]), geometry.SubXYZ(lV[2], lV[1]), geometry.SubXYZ(lV[0], lV[2])}
	axisIdx := 2
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			crossP := geometry.CrossXYZ(rE[i], lE[j])
			// Se i due edge sono paralleli (cross quasi nullo), l'asse è degenere e viene scartato
			if geometry.DotXYZ(crossP, crossP) > 1e-8 {
				axes[axisIdx] = geometry.NormalizeXYZ(crossP)
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
		rMin, rMax := geometry.TriangleProject(rV, axes[i])
		lMin, lMax := geometry.TriangleProject(lV, axes[i])
		// Check separazione
		if lMax < rMin || rMax < lMin {
			return cm // Esiste un asse separatore. Hit = false.
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
	// Estrazione Feature e Punto Esatto in base all'asse vincente
	cm.Hit = true
	cm.Depth = bestPenetration
	// Conserviamo il calcolo legacy per hitDist per non rompere il tuo StageResolve attuale
	for i := 0; i < 3; i++ {
		vDist := (lV[i].X-cm.P0X)*nX + (lV[i].Y-cm.P0Y)*nY + (lV[i].Z-cm.P0Z)*nZ
		if -vDist > cm.HitDist {
			cm.HitDist = vDist // Legacy hitDist maintain
		}
	}
	if bestAxisIdx == 0 {
		// Urto planare (Vertice Locale penetra Faccia Remota)
		cm.PointX, cm.PointY, cm.PointZ = geometry.TriangleFindDeepestPoint(lV, axes[0])
	} else if bestAxisIdx == 1 {
		// Urto planare inverso (Vertice Remoto penetra Faccia Locale)
		cm.PointX, cm.PointY, cm.PointZ = geometry.TriangleFindDeepestPoint(rV, geometry.XYZ{X: -axes[1].X, Y: -axes[1].Y, Z: -axes[1].Z})
	} else {
		// Urto Edge-Edge (Segmenti Sghembi). Ricaviamo gli indici rE e lE.
		eIdx := bestAxisIdx - 2
		rIdx, lIdx := eIdx/3, eIdx%3
		p1, p2 := geometry.ClosestPointSegmentSegment(rV[rIdx], rE[rIdx], lV[lIdx], lE[lIdx])
		// Il punto di contatto è il punto medio tra i due segmenti intersecanti
		cm.PointX = (p1.X + p2.X) * 0.5
		cm.PointY = (p1.Y + p2.Y) * 0.5
		cm.PointZ = (p1.Z + p2.Z) * 0.5
	}
	return cm
}

// addToBucket adds a collision cage to the specified bucket and updates the slot list if a new cage is created.
func (s *CollisionCage) addToBucket(bucket BucketType, rCage *CollisionCage, rFace *Face, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode int) {
	cage := s.buckets[bucket].Add(bucket, s, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode)
	if cage != nil {
		s.slots[s.slotsLen] = cage
		s.slotsLen++
	}
}

// TranslateWorldToLocal transforms the world-space AABB of the ellipsoid to local-space relative to the given delta values.
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

// TranslateCage translates the collision cage based on the given slot and reference cage, returning the local AABB and offsets.
func (s *CollisionCage) TranslateCage(slot int, rCage *CollisionCage) (aabb.IAABB, float64, float64, float64) {
	lOx, lOy, lOz := s.object.GetVerticesCenter()
	lOx += s.dX
	lOy += s.dY
	lOz += s.dZ
	rOx, rOy, rOz := rCage.object.GetVerticesCenter()
	rDX, rDY, rDZ := rCage.object.GetEntity().GetFrameDisplacement()
	rOx += rDX
	rOy += rDY
	rOz += rDZ
	lEntityL := s.TranslateWorldToLocal(slot, rOx, rOy, rOz)
	return lEntityL, rOx - lOx, rOy - lOy, rOz - lOz
}

// Translate updates the local ellipsoid of the given slot based on the provided AABB source and displacement values.
func (s *CollisionCage) Translate(slot int, src aabb.IAABB, deltaX, deltaY, deltaZ float64) *physics.Entity {
	w := s.ellipsoidLocal[slot]
	to := src.GetAABB()
	lMinX := to.GetMinX() + deltaX
	lMaxX := to.GetMaxX() + deltaX
	lMinY := to.GetMinY() + deltaY
	lMaxY := to.GetMaxY() + deltaY
	lMinZ := to.GetMinZ() + deltaZ
	lMaxZ := to.GetMaxZ() + deltaZ
	if f, ok := src.(*Face); ok {
		nX, nY, nZ := f.GetNormal()
		padX := math.Abs(nX) * s.eRadX * 2.0
		padY := math.Abs(nY) * s.eRadY * 2.0
		padZ := math.Abs(nZ) * s.eRadZ * 2.0
		lMinX -= padX
		lMaxX += padX
		lMinY -= padY
		lMaxY += padY
		lMinZ -= padZ
		lMaxZ += padZ
	}
	w.Rebuild(lMinX, lMinY, lMinZ, lMaxX-lMinX, lMaxY-lMinY, lMaxZ-lMinZ)
	return w
}

// HasSeen checks if the given CollisionCage has already been marked as seen in the current CollisionCage.
func (s *CollisionCage) HasSeen(rCage *CollisionCage) bool {
	return s.seen[rCage]
}

// Seen marks a given CollisionCage as observed by the current CollisionCage within its internal tracking.
func (s *CollisionCage) Seen(rCage *CollisionCage) {
	s.seen[rCage] = true
}

// GetBaseZ computes the base Z value of the collision cage by subtracting the ellipsoid radius in Z direction from its center Z coordinate.
func (s *CollisionCage) GetBaseZ() float64 { return s.cZ - s.eRadZ }

// GetSlotsLen returns the current number of active slots in the CollisionCage.
func (s *CollisionCage) GetSlotsLen() int { return s.slotsLen }

// GetSlot returns the CageEntry at the specified index in the slots slice.
func (s *CollisionCage) GetSlot(i int) *CageEntry { return s.slots[i] }

// GetThing retrieves the IThing object associated with the CollisionCage.
func (s *CollisionCage) GetThing() IThing { return s.object }

// GetVolume returns the volume associated with the CollisionCage.
func (s *CollisionCage) GetVolume() *Volume { return s.volume }

// GetRad retrieves the ellipsoid radii along the X, Y, and Z axes.
func (s *CollisionCage) GetRad() (float64, float64, float64) { return s.eRadX, s.eRadY, s.eRadZ }

// GetC returns the center coordinates (cX, cY, cZ) of the CollisionCage.
func (s *CollisionCage) GetC() (float64, float64, float64) { return s.cX, s.cY, s.cZ }

// GetDisplacement returns the displacement of the collision cage along the x, y, and z axes.
func (s *CollisionCage) GetDisplacement() (float64, float64, float64) { return s.dX, s.dY, s.dZ }

// GetT retrieves the current transformed coordinates (tX, tY, tZ) of the CollisionCage.
func (s *CollisionCage) GetT() (float64, float64, float64) { return s.tX, s.tY, s.tZ }

// BucketCount returns the number of entries in the specified bucket.
func (s *CollisionCage) BucketCount(t BucketType) int { return s.buckets[t].Count() }

// GetAABB returns the axis-aligned bounding box (AABB) representing the broad-phase collision geometry of the cage.
func (s *CollisionCage) GetAABB() *aabb.AABB { return s.ellipsoid.GetAABB() }
