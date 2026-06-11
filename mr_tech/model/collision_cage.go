package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
const ImpactStep = 1

// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
const ImpactInelastic = 2

// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
const ImpactElastic = 3

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

type ICageObject interface {
	GetEntity() *physics.Entity
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
	return v.Penetrable(s.lCage.GetObject().GetEntity().GetId())
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
		if entry.rCage != nil && entry.rCage.GetObject().GetEntity().GetId() == from {
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
			if rCage.GetObject().GetEntity().GetId() == existing.rCage.GetObject().GetEntity().GetId() {
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
	object              ICageObject
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
	slotsEmpty          []*CageEntry
	slotsLen            int
	maxStep             float64
	faces               []*Face
	facesIdx            int
}

// NewCollisionCage initializes and returns a pointer to a new CollisionCage instance for the provided IThing entity.
func NewCollisionCage(object ICageObject) *CollisionCage {
	c := &CollisionCage{
		seen:       make(map[*CollisionCage]bool),
		object:     object,
		ellipsoid:  physics.NewEntity(0, 0, 0, 0),
		volume:     nil,
		slots:      make([]*CageEntry, TotalSlots),
		slotsEmpty: make([]*CageEntry, TotalSlots),
		slotsLen:   0,
		faces:      make([]*Face, 8),
		facesIdx:   0,
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
	s.dX, s.dY, s.dZ = entity.GetDisplacement()
	// Estrazione origine (Bottom-Left)
	pX, pY, pZ := entity.GetBottomLeft()
	// Calcolo Half-Extents
	w, h, d := entity.GetSize()
	s.eRadX, s.eRadY, s.eRadZ = w*0.5, h*0.5, d*0.5
	// Calcolo del CENTRO per il Broad-Phase
	s.cX, s.cY, s.cZ = pX+s.eRadX, pY+s.eRadY, pZ+s.eRadZ

	s.tX, s.tY, s.tZ = s.cX+s.dX, s.cY+s.dY, s.cZ+s.dZ

	// Calculate absolute extremes (Broad-Phase Swept Volume)
	minX := s.cX - s.eRadX + min(0, s.dX) //- s.margin
	maxX := s.cX + s.eRadX + max(0, s.dX) //+ s.margin
	minY := s.cY - s.eRadY + min(0, s.dY) //- s.margin
	maxY := s.cY + s.eRadY + max(0, s.dY) //+ s.margin
	minZ := s.cZ - s.eRadZ + min(0, s.dZ) //- s.margin
	maxZ := s.cZ + s.eRadZ + max(0, s.dZ) //+ s.margin

	// Canonical mapping for Rect/AABB
	s.ellipsoid.Rebuild(minX, minY, minZ, maxX-minX, maxY-minY, maxZ-minZ)

	for i := 0; i < len(s.buckets); i++ {
		s.buckets[i].Rebuild()
	}

	s.volume = nil
	s.distance = math.MaxFloat64

	copy(s.slots, s.slotsEmpty)
	s.slotsLen = 0

	for k := range s.seen {
		delete(s.seen, k)
	}
	s.maxStep = maxStep
}

// AddFace adds a face to the CollisionCage, expanding the storage if necessary to accommodate new entries.
func (s *CollisionCage) AddFace(rFace *Face) {
	if s.facesIdx >= len(s.faces) {
		n := make([]*Face, len(s.faces)*2)
		copy(n, s.faces)
		s.faces = n
	}
	s.faces[s.facesIdx] = rFace
	s.facesIdx++
}

const epsilon = 0.05

// CommitStatic processes static collision detection between the entity and the environment using SAT filtering logic.
func (s *CollisionCage) CommitStatic() {
	lAABB := s.ellipsoid.GetAABB()
	for x := 0; x < s.facesIdx; x++ {
		face := s.faces[x]
		b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ := s.computeFace(lAABB, face, 0.0, 0.0, 0.0)
		// Se la compenetrazione calcolata dal semispazio infinito supera il limite fisico della AABB,
		// stiamo intersecando la proiezione di un piano ortogonale fantasma. Lo scartiamo.
		if pen > minOverlap+epsilon { // SAT filter (Anti-Phantom Plane)
			continue
		}
		// Volume Priority
		if dist < s.distance {
			if volume := face.GetParent(); volume != nil {
				s.volume = volume
				s.distance = dist
			}
		}
		_, texKind := face.GetMaterialDetails()
		if texKind == int(config.MaterialKindSky) {
			continue // Skybox/transparent: ignore collision
		}
		iMode := ImpactInelastic
		if b.IsWall() {
			baseZ := s.GetBaseZ()
			if rMaxZ <= baseZ { // down-hill (in discesa)
				continue
			}
			stepZ := baseZ + s.maxStep
			if rMaxZ <= stepZ { // up-hill (gradino superabile)
				iMode = ImpactStep
			}
		}
		s.addToBucket(b, nil, face, dist, pen, nX, nY, nZ, p0x, p0y, p0z, rMaxZ, iMode)
	}
	s.facesIdx = 0
}

// CommitDynamic processes elastic collisions between the current CollisionCage and a reference CollisionCage.
func (s *CollisionCage) CommitDynamic(rCage *CollisionCage) {
	lAABB := s.ellipsoid.GetAABB()
	offX, offY, offZ := rCage.ellipsoid.GetCenter()
	for x := 0; x < s.facesIdx; x++ {
		face := s.faces[x]
		_, texKind := face.GetMaterialDetails()
		if texKind == int(config.MaterialKindSky) {
			continue
		}
		b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ := s.computeFace(lAABB, face, offX, offY, offZ)
		if pen > minOverlap+epsilon { // SAT filter (Anti-Phantom Plane)
			continue
		}
		if b.IsWall() {
			if rMaxZ <= s.GetBaseZ() { // down-hill (in discesa)
				continue
			}
		}
		s.addToBucket(b, rCage, face, dist, pen, nX, nY, nZ, p0x, p0y, p0z, rMaxZ, ImpactElastic)
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
	// Support Mapping for AABB (Sorgente Rettangolare Completa)
	//rayEff := math.Abs(nX*s.eRadX) + math.Abs(nY*s.eRadY) + math.Abs(nZ*s.eRadZ)
	// Minkowski / Support Mapping for Ellipsoids
	rayEff := math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))
	distTarget := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
	dist := distTarget - rayEff
	penetration := rayEff - distTarget
	// Early-Exit Filtering: The plane exceeds the configured broad-margin
	//if dist > s.margin {
	//	return
	//}
	// If the face is NOT penetrated at the target (penetration <= 0), it is not needed by the Half-Space solver
	//if penetration <= 0 {
	//	return
	//}

	// sat filter (Anti-Phantom Plane)
	rFaceAABB := rFace.GetAABB()
	// world space translation
	rMinX := rFaceAABB.GetMinX() + offX
	rMaxX := rFaceAABB.GetMaxX() + offX
	rMinY := rFaceAABB.GetMinY() + offY
	rMaxY := rFaceAABB.GetMaxY() + offY
	rMinZ := rFaceAABB.GetMinZ() + offZ
	rMaxZ := rFaceAABB.GetMaxZ() + offZ
	oX := max(0.0, min(lAABB.GetMaxX()-rMinX, rMaxX-lAABB.GetMinX()))
	oY := max(0.0, min(lAABB.GetMaxY()-rMinY, rMaxY-lAABB.GetMinY()))
	oZ := max(0.0, min(lAABB.GetMaxZ()-rMinZ, rMaxZ-lAABB.GetMinZ()))
	// La reale penetrazione volumetrica massima possibile per questa specifica faccia
	minOverlap := min(oX, min(oY, oZ))

	return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ
}

// addToBucket adds a CollisionCage or Face into the specified bucket with given parameters to manage collision resolution.
func (s *CollisionCage) addToBucket(bucket BucketType, rCage *CollisionCage, rFace *Face, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode int) {
	cage := s.buckets[bucket].Add(bucket, s, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode)
	if cage != nil {
		s.slots[s.slotsLen] = cage
		s.slotsLen++
	}
}

// TranslateWorldToLocalAABB translates the AABB of a target `CollisionCage` from world space to local space for a given slot.
func (s *CollisionCage) TranslateWorldToLocalAABB(slot int, target *CollisionCage) *physics.Entity {
	from := s.ellipsoid.GetAABB()
	to := target.GetAABB() // target anchor
	offX := to.GetMinX()
	offY := to.GetMinY()
	offZ := to.GetMinZ()
	lMinX := from.GetMinX() - offX
	lMaxX := from.GetMaxX() - offX
	lMinY := from.GetMinY() - offY
	lMaxY := from.GetMaxY() - offY
	lMinZ := from.GetMinZ() - offZ
	lMaxZ := from.GetMaxZ() - offZ
	s.ellipsoidLocal[slot].Rebuild(lMinX, lMinY, lMinZ, lMaxX-lMinX, lMaxY-lMinY, lMaxZ-lMinZ)
	return s.ellipsoidLocal[slot]
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

// GetObject returns the ICageObject instance contained within the CollisionCage.
func (s *CollisionCage) GetObject() ICageObject { return s.object }

// GetMargin retrieves the margin value used in collision calculations for the CollisionCage.
//func (s *CollisionCage) GetMargin() float64 { return s.margin }

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
