package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

type BucketType int

const (
	BucketWallWest  = BucketType(0) // -X
	BucketWallEast  = BucketType(1) // +X
	BucketWallNorth = BucketType(2) // -Y
	BucketWallSouth = BucketType(3) // +Y
	BucketCeiling   = BucketType(4) // -Z
	BucketFloor     = BucketType(5) // +Z
)

// String returns the string representation of the BucketType value. For unrecognized values, it returns "BucketType(%d)".
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

// BucketSize defines the size of a bucket used in the operation.
// FacesPerBucket specifies the number of faces contained within a single bucket.
const (
	BucketSize     = BucketFloor + 1
	FacesPerBucket = 4
)

// _emptyBucketFaces is a placeholder array used to initialize or reset bucket face buffers with nil CageEntry values.
var _emptyBucketFaces = [FacesPerBucket]*CageEntry{nil, nil, nil, nil}

// CageEntry represents an entry in a spatial cage, holding geometric and distance attributes for a specific face.
type CageEntry struct {
	face    *Face
	dist    float64
	rEff    float64
	normalX float64
	normalY float64
	normalZ float64
}

// GetFace returns the face associated with this CageEntry.
func (s *CageEntry) GetFace() *Face {
	return s.face
}

// GetDist returns the distance value of this CageEntry.
func (s *CageEntry) GetDist() float64 {
	return s.dist
}

// GetREff returns the effective radius of this CageEntry.
func (s *CageEntry) GetREff() float64 {
	return s.rEff
}

// GetNormal returns the normal vector of this CageEntry.
func (s *CageEntry) GetNormal() (float64, float64, float64) {
	return s.normalX, s.normalY, s.normalZ
}

// NewCollisionFace creates and returns a new instance of CageEntry with default zero-initialized values.
func NewCollisionFace() *CageEntry {
	return &CageEntry{}
}

// Rebuild updates the CageEntry fields with the given face, distance, effective radius, and normal vector.
func (s *CageEntry) Rebuild(face *Face, dist, rEff, normalX, normalY, normalZ float64) {
	s.face = face
	s.dist = dist
	s.rEff = rEff
	s.normalX = normalX
	s.normalY = normalY
	s.normalZ = normalZ
}

// CollisionCage represents a structure for managing collision detection using bounding ellipsoids and spatial buckets.
// It maintains a set of active and spare collision constraints associated with predefined buckets and faces.
// The type is primarily used for organizing and resolving collisions efficiently in 3D space.
type CollisionCage struct {
	faces     [BucketSize][FacesPerBucket]*CageEntry
	counts    [BucketSize]int // Quanti vincoli attivi per bucket
	spare     [BucketSize][FacesPerBucket]*CageEntry
	ellipsoid *physics.Entity
	margin    float64
	c         geometry.XYZ
	d         geometry.XYZ
	t         geometry.XYZ
	eRad      geometry.XYZ
}

// NewCollisionCage initializes and returns a pointer to a new CollisionCage with the specified margin, restitution, and friction.
func NewCollisionCage(margin float64, restitution, friction float64) *CollisionCage {
	c := &CollisionCage{
		margin:    margin,
		ellipsoid: physics.NewEntity(0, 0, 0, 0, 0, 0, -1, restitution, friction),
	}
	for i := BucketType(0); i < BucketSize; i++ {
		for j := 0; j < FacesPerBucket; j++ {
			c.spare[i][j] = NewCollisionFace()
		}
	}
	return c
}

// Rebuild recalculates the CollisionCage's internal state based on the given parameters cx, cy, cz, dx, dy, dz, eRadX, eRadY, and eRadZ.
func (s *CollisionCage) Rebuild(cx, cy, cz, dx, dy, dz, eRadX, eRadY, eRadZ float64) {
	s.c.X, s.c.Y, s.c.Z = cx, cy, cz
	s.d.X, s.d.Y, s.d.Z = dx, dy, dz
	s.eRad.X, s.eRad.Y, s.eRad.Z = eRadX, eRadY, eRadZ
	s.t.X, s.t.Y, s.t.Z = cx+dx, cy+dy, cz+dz
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
	s.ellipsoid.Rebuild(x, y, w, h, z, d)
	// Fast reset
	for i := 0; i < 6; i++ {
		s.counts[i] = 0
		copy(s.faces[i][:], _emptyBucketFaces[:])
	}
}

// AddFace adds a new face to the specified bucket or replaces the farthest face if the bucket is full and the new face is closer.
func (s *CollisionCage) AddFace(bucket BucketType, face *Face, dist, rEff, normalX, normalY, normalZ float64) {
	if count := s.counts[bucket]; count < FacesPerBucket {
		target := s.spare[bucket][count]
		target.Rebuild(face, dist, rEff, normalX, normalY, normalZ)
		s.faces[bucket][count] = target
		s.counts[bucket]++
		return
	}
	// Buffer full: find the weakest constraint (farthest)
	maxIdx := 0
	maxDist := s.faces[bucket][0].dist
	for i := 1; i < FacesPerBucket; i++ {
		if s.faces[bucket][i].dist > maxDist {
			maxDist = s.faces[bucket][i].dist
			maxIdx = i
		}
	}
	// Replace it if this new plane is closer (more dangerous)
	if dist < maxDist {
		s.faces[bucket][maxIdx].Rebuild(face, dist, rEff, normalX, normalY, normalZ)
	}
}

// GetMargin retrieves the margin value of the CollisionCage, used in collision and constraint calculations.
func (s *CollisionCage) GetMargin() float64 {
	return s.margin
}

// GetRad retrieves the ellipsoid radii stored in the CollisionCage as a geometry.XYZ vector.
func (s *CollisionCage) GetRad() (float64, float64, float64) {
	return s.eRad.X, s.eRad.Y, s.eRad.Z
}

// GetC returns the central position of the collision cage as a geometry.XYZ object.
func (s *CollisionCage) GetC() (float64, float64, float64) {
	return s.c.X, s.c.Y, s.c.Z
}

// GetD returns the displacement vector (d) of the CollisionCage, representing the directional offset in 3D space.
func (s *CollisionCage) GetD() (float64, float64, float64) {
	return s.d.X, s.d.Y, s.d.Z
}

// GetT returns the target position vector `t` defined for the CollisionCage in 3D space.
func (s *CollisionCage) GetT() (float64, float64, float64) {
	return s.t.X, s.t.Y, s.t.Z
}

// GetFaces returns a 2D array representing the faces of the collision cage, organized by buckets.
func (s *CollisionCage) GetFaces() [BucketSize][FacesPerBucket]*CageEntry {
	return s.faces
}

// GetAABB returns the axis-aligned bounding box (AABB) of the collision cage using the underlying ellipsoid entity.
func (s *CollisionCage) GetAABB() *physics.AABB {
	return s.ellipsoid.GetAABB()
}

// GetEntity returns the physics.Entity instance associated with the CollisionCage.
func (s *CollisionCage) GetEntity() *physics.Entity {
	return s.ellipsoid
}
