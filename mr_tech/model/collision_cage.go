package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// BucketType represents an enumeration for categorizing different types of spatial regions in a collision detection system.
type BucketType int

// BucketWallWest represents the bucket type for the west wall (-X).
// BucketWallEast represents the bucket type for the east wall (+X).
// BucketWallNorth represents the bucket type for the north wall (-Y).
// BucketWallSouth represents the bucket type for the south wall (+Y).
// BucketCeiling represents the bucket type for the ceiling (-Z).
// BucketFloor represents the bucket type for the floor (+Z).
const (
	BucketWallWest  = BucketType(0) // -X
	BucketWallEast  = BucketType(1) // +X
	BucketWallNorth = BucketType(2) // -Y
	BucketWallSouth = BucketType(3) // +Y
	BucketCeiling   = BucketType(4) // -Z
	BucketFloor     = BucketType(5) // +Z
)

// String returns the string representation of a BucketType.
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

// BucketSize represents the total number of slots in a single bucket, calculated as BucketFloor + 1.
// FacesPerBucket defines the number of faces associated with each bucket.
// TotalSlots represents the overall slots available across all buckets, calculated as BucketSize * FacesPerBucket.
const (
	BucketSize     = BucketFloor + 1
	FacesPerBucket = 4
	TotalSlots     = BucketSize * FacesPerBucket
)

// CageEntry represents a collision entry containing face geometry, distance, normal, penetration, and other flags.
type CageEntry struct {
	face        *Face
	dist        float64
	penetration float64
	nX          float64
	nY          float64
	nZ          float64
	p0X         float64
	p0Y         float64
	p0Z         float64
	isWall      bool
	maxZ        float64
}

// GetFace retrieves the Face instance associated with the CageEntry. Returns nil if no Face is set.
func (s *CageEntry) GetFace() *Face { return s.face }

// GetDistance returns the distance value (`dist`) associated with the CageEntry instance.
func (s *CageEntry) GetDistance() float64 { return s.dist }

// IsWall returns true if the CageEntry is classified as a wall, false otherwise.
func (s *CageEntry) IsWall() bool { return s.isWall }

// GetMaxZ returns the maximum Z-coordinate value associated with the CageEntry instance.
func (s *CageEntry) GetMaxZ() float64 { return s.maxZ }

// GetNormal returns the normal vector components (nX, nY, nZ) of the `CageEntry`.
func (s *CageEntry) GetNormal() (float64, float64, float64) { return s.nX, s.nY, s.nZ }

// GetPenetration returns the penetration depth value for the current CageEntry instance.
func (s *CageEntry) GetPenetration() float64 { return s.penetration }

// NewCollisionFace creates and returns a new instance of CageEntry with default, uninitialized values.
func NewCollisionFace() *CageEntry {
	return &CageEntry{}
}

// Rebuild updates the CageEntry fields with the provided values for geometry, collision, and wall properties.
func (s *CageEntry) Rebuild(face *Face, dist, penetration, nX, nY, nZ, p0x, p0y, p0z float64, isWall bool, maxZ float64) {
	s.face = face
	s.dist = dist
	s.penetration = penetration
	s.nX, s.nY, s.nZ = nX, nY, nZ
	s.p0X, s.p0Y, s.p0Z = p0x, p0y, p0z
	s.isWall = isWall
	s.maxZ = maxZ
}

// CollisionBucket represents a data structure that manages collision entries for a specific bucket type in a defined space.
type CollisionBucket struct {
	bucket    BucketType
	container [FacesPerBucket]*CageEntry
	spare     [FacesPerBucket]*CageEntry
	empty     [FacesPerBucket]*CageEntry
	counter   int
}

// NewCollisionBucket initializes a new CollisionBucket with preallocated CageEntry arrays and a specified BucketType.
func NewCollisionBucket(bucket BucketType) *CollisionBucket {
	b := &CollisionBucket{
		bucket:  bucket,
		counter: 0,
	}
	for i := 0; i < FacesPerBucket; i++ {
		b.spare[i] = NewCollisionFace()
		b.container[i] = nil
		b.empty[i] = nil
	}
	return b
}

// Rebuild resets the CollisionBucket's counter to zero and clears its container by copying from the empty array.
func (b *CollisionBucket) Rebuild() {
	b.counter = 0
	copy(b.container[:], b.empty[:])
}

// Count returns the number of entries currently stored in the CollisionBucket.
func (b *CollisionBucket) Count() int {
	return b.counter
}

// Add inserts a face into the CollisionBucket, replacing the lowest-priority entry if the bucket is full.
func (b *CollisionBucket) Add(face *Face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, isWall bool, fMaxZ float64) *CageEntry {
	// If the bucket is not full, add and expose the new pointer
	if b.counter < FacesPerBucket {
		target := b.spare[b.counter]
		target.Rebuild(face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, isWall, fMaxZ)
		b.container[b.counter] = target
		b.counter++
		return target
	}

	// Find the currently stored face with the LOWEST PENETRATION (the least priority)
	minIdx := 0
	minPen := b.container[0].penetration
	for i := 1; i < FacesPerBucket; i++ {
		if b.container[i].penetration < minPen {
			minPen = b.container[i].penetration
			minIdx = i
		}
	}

	// If the new face penetrates deeper than the one with minimum penetration, overwrite it
	if penetration > minPen {
		b.container[minIdx].Rebuild(face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, isWall, fMaxZ)
	}

	// Return nil because the replaced face already has its pointer exposed in the slots array
	return nil
}

// CollisionCage provides a structure for managing collision detection and resolution for a 3D entity.
type CollisionCage struct {
	thing               IThing
	buckets             [BucketSize]*CollisionBucket
	ellipsoid           *physics.Entity
	ellipsoidLocal      *physics.Entity
	margin              float64
	cX, cY, cZ          float64
	dX, dY, dZ          float64
	tX, tY, tZ          float64
	eRadX, eRadY, eRadZ float64
	volume              *Volume
	distance            float64
	slots               []*CageEntry
	slotsEmpty          []*CageEntry
	slotsLen            int
}

// NewCollisionCage creates and initializes a new CollisionCage with the given IThing and margin values.
func NewCollisionCage(thing IThing, margin float64) *CollisionCage {
	c := &CollisionCage{
		thing:          thing,
		margin:         margin,
		ellipsoid:      physics.NewEntity(0, 0, 0, 0),
		ellipsoidLocal: physics.NewEntity(0, 0, 0, 0),
		volume:         nil,
		slots:          make([]*CageEntry, TotalSlots),
		slotsEmpty:     make([]*CageEntry, TotalSlots),
		slotsLen:       0,
	}
	for i := BucketType(0); i < BucketSize; i++ {
		c.buckets[i] = NewCollisionBucket(i)
	}
	return c
}

// Rebuild updates the internal state of the CollisionCage, recalculating volumes, bounds, and resetting temporary data.
func (s *CollisionCage) Rebuild(cx, cy, cz, dx, dy, dz, eRadX, eRadY, eRadZ float64) {
	s.cX, s.cY, s.cZ = cx, cy, cz
	s.dX, s.dY, s.dZ = dx, dy, dz
	s.eRadX, s.eRadY, s.eRadZ = eRadX, eRadY, eRadZ
	s.tX, s.tY, s.tZ = cx+dx, cy+dy, cz+dz

	// Calculate absolute extremes (Broad-Phase Swept Volume)
	minX := cx - eRadX + math.Min(0, dx) - s.margin
	maxX := cx + eRadX + math.Max(0, dx) + s.margin
	minY := cy - eRadY + math.Min(0, dy) - s.margin
	maxY := cy + eRadY + math.Max(0, dy) + s.margin
	minZ := cz - eRadZ + math.Min(0, dz) - s.margin
	maxZ := cz + eRadZ + math.Max(0, dz) + s.margin

	// Canonical mapping for Rect/AABB
	s.ellipsoid.Rebuild(minX, minY, minZ, maxX-minX, maxY-minY, maxZ-minZ)

	for i := 0; i < len(s.buckets); i++ {
		s.buckets[i].Rebuild()
	}

	s.volume = nil
	s.distance = math.MaxFloat64

	copy(s.slots, s.slotsEmpty)
	s.slotsLen = 0
}

// GetBaseZ calculates and returns the lower Z-bound of the collision cage based on its center and radius.
func (s *CollisionCage) GetBaseZ() float64 { return s.cZ - s.eRadZ }

// GetSlotsLen returns the number of occupied slots in the CollisionCage.
func (s *CollisionCage) GetSlotsLen() int { return s.slotsLen }

// GetSlot retrieves the CageEntry at the specified index from the slots list.
func (s *CollisionCage) GetSlot(i int) *CageEntry { return s.slots[i] }

// GetThing retrieves the IThing instance associated with the CollisionCage.
func (s *CollisionCage) GetThing() IThing { return s.thing }

// GetMargin retrieves the margin value used in collision calculations for the CollisionCage.
func (s *CollisionCage) GetMargin() float64 { return s.margin }

// GetVolume returns the Volume instance associated with the CollisionCage, or nil if no Volume is assigned.
func (s *CollisionCage) GetVolume() *Volume { return s.volume }

// GetRad retrieves the radii of the ellipsoid along the X, Y, and Z axes as a tuple of float64 values.
func (s *CollisionCage) GetRad() (float64, float64, float64) { return s.eRadX, s.eRadY, s.eRadZ }

// GetC retrieves the current center coordinates (cX, cY, cZ) of the CollisionCage.
func (s *CollisionCage) GetC() (float64, float64, float64) { return s.cX, s.cY, s.cZ }

// GetD returns the displacement vector components (dX, dY, dZ) of the CollisionCage.
func (s *CollisionCage) GetD() (float64, float64, float64) { return s.dX, s.dY, s.dZ }

// GetT retrieves the transformed coordinates (tX, tY, tZ) of the collision cage.
func (s *CollisionCage) GetT() (float64, float64, float64) { return s.tX, s.tY, s.tZ }

// BucketCount returns the number of elements currently stored in the specified bucket type.
func (s *CollisionCage) BucketCount(t BucketType) int { return s.buckets[t].Count() }

// GetAABB retrieves the axis-aligned bounding box (AABB) of the CollisionCage's ellipsoid entity.
func (s *CollisionCage) GetAABB() *physics.AABB { return s.ellipsoid.GetAABB() }

// GetEntity returns the ellipsoid entity associated with the CollisionCage.
func (s *CollisionCage) GetEntity() *physics.Entity { return s.ellipsoid }

// Translate updates the local ellipsoid's position relative to a target and returns the updated physics.Entity.
func (s *CollisionCage) Translate(targetX, targetY, targetZ float64) *physics.Entity {
	cageAABB := s.ellipsoid.GetAABB()
	lMinX := cageAABB.GetMinX() - targetX
	lMaxX := cageAABB.GetMaxX() - targetX
	lMinY := cageAABB.GetMinY() - targetY
	lMaxY := cageAABB.GetMaxY() - targetY
	lMinZ := cageAABB.GetMinZ() - targetZ
	lMaxZ := cageAABB.GetMaxZ() - targetZ
	s.ellipsoidLocal.Rebuild(lMinX, lMinY, lMinZ, lMaxX-lMinX, lMaxY-lMinY, lMaxZ-lMinZ)
	return s.ellipsoidLocal
}

// AddFace processes a Face to determine its type, position, and potential collision influence within the CollisionCage.
// face is the Face object to process.
// offX, offY, offZ specify the offsets to transform the face into world space.
// isVolume indicates whether the Face should be prioritized as a volumetric obstacle.
func (s *CollisionCage) AddFace(face *Face, offX, offY, offZ float64, isVolume bool) {
	nAbsX, nAbsY, nAbsZ := face.normalAbs.X, face.normalAbs.Y, face.normalAbs.Z

	wallWE := nAbsX > nAbsY && nAbsX > nAbsZ
	wallNS := nAbsY > nAbsZ
	isWall := wallWE || wallNS

	// Translation (from Local to World Space)
	p0x, p0y, p0z := face.tri[0].X+offX, face.tri[0].Y+offY, face.tri[0].Z+offZ
	nX, nY, nZ := face.normal.X, face.normal.Y, face.normal.Z

	distStart := (s.cX-p0x)*nX + (s.cY-p0y)*nY + (s.cZ-p0z)*nZ
	var bucket BucketType

	if isWall {
		// Facing Normalization: Forces the plane to oppose the player
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
			distStart = -distStart
		}
		// Wall Bucket Assignment
		if wallWE {
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
		if math.Abs(nZ) > 1e-5 {
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

	// Volume Priority
	if isVolume && dist < s.distance {
		if volume := face.GetParent(); volume != nil {
			s.volume = volume
			s.distance = dist
		}
	}

	// TODO BETTER IMPLEMENTATION!
	_, texKind := face.GetMaterialDetails()
	if texKind == int(config.MaterialKindSky) {
		return // Skybox/transparent: ignore collision
	}

	// Early-Exit Filtering: The plane exceeds the configured broad-margin
	if dist > s.margin {
		return
	}

	// If the face is NOT penetrated at the target (penetration <= 0), it is not needed by the Half-Space solver
	if penetration <= 0 {
		return
	}

	fMaxZ := face.GetAABB().GetMaxZ() + offZ
	cage := s.buckets[bucket].Add(face, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, isWall, fMaxZ)

	if cage != nil {
		s.slots[s.slotsLen] = cage
		s.slotsLen++
	}
}
