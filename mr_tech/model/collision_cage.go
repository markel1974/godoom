package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// BucketType represents distinct categories for organizing collision buckets in a spatial structure.
type BucketType int

// BucketWallWest represents the west wall bucket type (-X).
// BucketWallEast represents the east wall bucket type (+X).
// BucketWallNorth represents the north wall bucket type (-Y).
// BucketWallSouth represents the south wall bucket type (+Y).
// BucketCeiling represents the ceiling bucket type (-Z).
// BucketFloor represents the floor bucket type (+Z).
const (
	BucketWallWest  = BucketType(0) // -X
	BucketWallEast  = BucketType(1) // +X
	BucketWallNorth = BucketType(2) // -Y
	BucketWallSouth = BucketType(3) // +Y
	BucketCeiling   = BucketType(4) // -Z
	BucketFloor     = BucketType(5) // +Z
)

// String returns the string representation of the BucketType value. Maps integer values to their respective type names.
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

// BucketSize represents the total count of slots available in a single bucket, derived from BucketFloor + 1.
// FacesPerBucket defines the fixed number of faces available in each bucket for allocation.
// TotalSlots is the total number of slots across all faces in a bucket, computed as BucketSize * FacesPerBucket.
const (
	BucketSize     = BucketFloor + 1
	FacesPerBucket = 4
	TotalSlots     = BucketSize * FacesPerBucket
)

// CageEntry represents a collision entry with geometric and physical properties for query results.
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

// GetFace retrieves the Face object associated with the CageEntry.
func (s *CageEntry) GetFace() *Face {
	return s.face
}

// GetDistance returns the stored distance value (dist) for the CageEntry instance.
func (s *CageEntry) GetDistance() float64 {
	return s.dist
}

// IsWall returns true if the CageEntry represents a wall, otherwise false.
func (s *CageEntry) IsWall() bool {
	return s.isWall
}

// GetMaxZ returns the maximum Z-coordinate value associated with the CageEntry instance.
func (s *CageEntry) GetMaxZ() float64 {
	return s.maxZ
}

// GetNormal returns the normal vector components (nX, nY, nZ) of the CageEntry as three float64 values.
func (s *CageEntry) GetNormal() (float64, float64, float64) {
	return s.nX, s.nY, s.nZ
}

// GetPenetration returns the penetration depth associated with the CageEntry instance.
func (s *CageEntry) GetPenetration() float64 {
	return s.penetration
}

// NewCollisionFace creates and returns a new instance of CageEntry with default values.
func NewCollisionFace() *CageEntry {
	return &CageEntry{}
}

// Rebuild updates the CageEntry's properties with the provided face, distance, penetration, normals, position, and other flags.
func (s *CageEntry) Rebuild(face *Face, dist, penetration, nX, nY, nZ, p0x, p0y, p0z float64, isWall bool, maxZ float64) {
	s.face = face
	s.dist = dist
	s.penetration = penetration
	s.nX, s.nY, s.nZ = nX, nY, nZ
	s.p0X, s.p0Y, s.p0Z = p0x, p0y, p0z
	s.isWall = isWall
	s.maxZ = maxZ
}

// CollisionBucket is a data structure for managing collision faces within a specific bucket type.
// It stores active collision entries, spare entries, empty entries, and a count of active entries.
type CollisionBucket struct {
	bucket    BucketType
	container [FacesPerBucket]*CageEntry
	spare     [FacesPerBucket]*CageEntry
	empty     [FacesPerBucket]*CageEntry
	counter   int
}

// NewCollisionBucket creates and initializes a new CollisionBucket for the specified BucketType.
func NewCollisionBucket(bucket BucketType) *CollisionBucket {
	b := &CollisionBucket{
		bucket:  bucket,
		counter: 0,
	}
	for i := 0; i < FacesPerBucket; i++ {
		b.container[i] = nil
	}
	for i := 0; i < FacesPerBucket; i++ {
		b.spare[i] = NewCollisionFace()
	}
	for i := 0; i < FacesPerBucket; i++ {
		b.empty[i] = nil
	}
	return b
}

// Rebuild resets the CollisionBucket by clearing its container and resetting the counter to 0.
func (b *CollisionBucket) Rebuild() {
	b.counter = 0
	copy(b.container[:], b.empty[:])
}

// Count returns the current number of entries in the CollisionBucket.
func (b *CollisionBucket) Count() int {
	return b.counter
}

// Add inserts a Face into the CollisionBucket and returns a CageEntry if successful; otherwise, updates existing entries.
func (b *CollisionBucket) Add(face *Face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z float64, isWall bool, fMaxZ float64) *CageEntry {
	if b.counter < FacesPerBucket {
		target := b.spare[b.counter]
		target.Rebuild(face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, isWall, fMaxZ)
		b.container[b.counter] = target
		b.counter++
		return target
	}
	maxIdx := 0
	maxDist := b.container[0].dist
	for i := 1; i < FacesPerBucket; i++ {
		if b.container[i].dist > maxDist {
			maxDist = b.container[i].dist
			maxIdx = i
		}
	}
	if dist < maxDist {
		b.container[maxIdx].Rebuild(face, dist, penetration, normalX, normalY, normalZ, p0x, p0y, p0z, isWall, fMaxZ)
	}
	return nil
}

// CollisionCage represents a spatial collider for managing physical interactions and detecting collisions in 3D space.
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

// NewCollisionCage initializes and returns a new CollisionCage with the given IThing instance and margin value.
// It sets up all required properties, including buckets, ellipsoid entities, and slot arrays.
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
	for i := BucketType(0); i < TotalSlots; i++ {
		c.slotsEmpty[i] = nil
	}
	return c
}

// Rebuild updates CollisionCage attributes, recalculates bounds and extremes, resets buckets, and clears cached values.
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
	x := minX
	y := minY
	z := minZ
	w := maxX - minX
	h := maxY - minY
	d := maxZ - minZ
	//x y z == BOTTOM LEFT
	s.ellipsoid.Rebuild(x, y, z, w, h, d)

	for i := 0; i < len(s.buckets); i++ {
		s.buckets[i].Rebuild()
	}

	s.volume = nil
	s.distance = math.MaxFloat64

	copy(s.slots, s.slotsEmpty)
	s.slotsLen = 0
}

// GetBaseZ calculates and returns the base Z coordinate of the collision cage by subtracting the Z radius from its center Z.
func (s *CollisionCage) GetBaseZ() float64 {
	return s.cZ - s.eRadZ
}

// GetSlotsLen returns the current number of slots in use within the CollisionCage.
func (s *CollisionCage) GetSlotsLen() int {
	return s.slotsLen
}

// GetSlot retrieves the CageEntry at the specified index from the CollisionCage's slots array.
func (s *CollisionCage) GetSlot(i int) *CageEntry {
	return s.slots[i]
}

// GetThing returns the IThing instance associated with the CollisionCage.
func (s *CollisionCage) GetThing() IThing {
	return s.thing
}

// GetMargin retrieves the margin value used in the CollisionCage for various calculations.
func (s *CollisionCage) GetMargin() float64 {
	return s.margin
}

// GetVolume retrieves the Volume associated with the CollisionCage instance.
func (s *CollisionCage) GetVolume() *Volume {
	return s.volume
}

// GetRad returns the radii of the collision cage along the X, Y, and Z axes.
func (s *CollisionCage) GetRad() (float64, float64, float64) {
	return s.eRadX, s.eRadY, s.eRadZ
}

// GetC returns the central coordinates (cX, cY, cZ) of the CollisionCage as a tuple of three float64 values.
func (s *CollisionCage) GetC() (float64, float64, float64) {
	return s.cX, s.cY, s.cZ
}

// GetD returns the displacement values (dX, dY, dZ) of the CollisionCage.
func (s *CollisionCage) GetD() (float64, float64, float64) {
	return s.dX, s.dY, s.dZ
}

// GetT retrieves the translation components (tX, tY, tZ) of the CollisionCage.
func (s *CollisionCage) GetT() (float64, float64, float64) {
	return s.tX, s.tY, s.tZ
}

// BucketCount returns the number of elements in the bucket of the specified type t.
func (s *CollisionCage) BucketCount(t BucketType) int {
	return s.buckets[t].Count()
}

// GetAABB returns the axis-aligned bounding box (AABB) of the collision cage using its ellipsoid entity.
func (s *CollisionCage) GetAABB() *physics.AABB {
	return s.ellipsoid.GetAABB()
}

// GetEntity returns the ellipsoid entity associated with the CollisionCage.
func (s *CollisionCage) GetEntity() *physics.Entity {
	return s.ellipsoid
}

// Translate adjusts the `CollisionCage`'s local transformation using the target coordinates and updates its AABB.
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

// AddFace processes a face, calculates its distance to the collision cage, and classifies it into a bucket for collision checks.
// face: the Face object to add.
// offX, offY, offZ: offset values to translate the face's position.
// isVolume: a flag indicating whether to perform volume computation.
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
		// Vertical planes
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
			distStart = -distStart
		}
		// Bucket assignment for walls based on final normal
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
		// Horizontal Planes
		planeZ := p0z
		if math.Abs(nZ) > 1e-5 {
			planeZ = p0z - (nX*(s.cX-p0x)+nY*(s.cY-p0y))/nZ
		}
		if s.cZ >= planeZ {
			bucket = BucketFloor // is mathematically a Floor
			if nZ < 0 {
				nX, nY, nZ = -nX, -nY, -nZ
				distStart = -distStart
			}
		} else {
			bucket = BucketCeiling // is mathematically a Ceiling
			if nZ > 0 {
				nX, nY, nZ = -nX, -nY, -nZ
				distStart = -distStart
			}
		}
	}

	rayEff := math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))
	distTarget := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
	dist := distTarget - rayEff

	//Important can't leave this method before volume computation
	if isVolume {
		if dist < s.distance {
			if volume := face.GetParent(); volume != nil {
				s.volume = volume
				s.distance = dist
			}
		}
	}

	//TODO MIGLIORARE MATERIALS IN MODO DA DEFINIRE LA TRASPARENZA
	_, texKind := face.GetMaterialDetails()
	if texKind == int(config.MaterialKindSky) {
		return //transparent
	}

	if dist > s.margin {
		return //outside margin
	}

	if distTarget >= rayEff {
		return //no penetration
	}

	penetration := rayEff - distTarget
	fMaxZ := face.GetAABB().GetMaxZ() + offZ
	cage := s.buckets[bucket].Add(face, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, isWall, fMaxZ)
	if cage != nil {
		s.slots[s.slotsLen] = cage
		s.slotsLen++
	}
}
