package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// BucketType represents a categorization of spatial partitions such as walls, ceilings, and floors in a 3D space.
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

// String returns the string representation of the BucketType enumeration.
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

// BucketSize defines the size of a bucket, calculated as BucketFloor increased by 1.
// FacesPerBucket specifies the number of faces assigned to each bucket.
const (
	BucketSize     = BucketFloor + 1
	FacesPerBucket = 4
)

// _emptyBucketFaces is a pre-defined array of nil CageEntry pointers representing an empty state for bucket initialization.
var _emptyBucketFaces = [FacesPerBucket]*CageEntry{nil, nil, nil, nil}

// CageEntry represents a structure containing collision-related attributes, such as the face, distance, and effective radius.
type CageEntry struct {
	face    *Face
	dist    float64
	rEff    float64
	normalX float64
	normalY float64
	normalZ float64
}

// GetFace retrieves the Face instance associated with the CageEntry. Returns nil if no Face is set.
func (s *CageEntry) GetFace() *Face {
	return s.face
}

// GetDist returns the distance value stored in the CageEntry instance.
func (s *CageEntry) GetDist() float64 {
	return s.dist
}

// GetREff retrieves the effective radius (rEff) of the CageEntry.
func (s *CageEntry) GetREff() float64 {
	return s.rEff
}

// GetNormal returns the normal vector components (normalX, normalY, normalZ) of the CageEntry as a tuple.
func (s *CageEntry) GetNormal() (float64, float64, float64) {
	return s.normalX, s.normalY, s.normalZ
}

// NewCollisionFace creates and returns a new instance of CageEntry with uninitialized fields.
func NewCollisionFace() *CageEntry {
	return &CageEntry{}
}

// Rebuild updates the CageEntry instance with new face and attributes: distance, effective radius, and normal vector components.
func (s *CageEntry) Rebuild(face *Face, dist, rEff, normalX, normalY, normalZ float64) {
	s.face = face
	s.dist = dist
	s.rEff = rEff
	s.normalX = normalX
	s.normalY = normalY
	s.normalZ = normalZ
}

// CollisionCage represents a structure for handling collision constraints in a 3D space through a bucketed system.
// It tracks faces, active constraints, and spatial properties of an ellipsoid with associated margins.
type CollisionCage struct {
	faces               [BucketSize][FacesPerBucket]*CageEntry
	counts              [BucketSize]int // Quanti vincoli attivi per bucket
	spare               [BucketSize][FacesPerBucket]*CageEntry
	ellipsoid           *physics.Entity
	margin              float64
	cX, cY, cZ          float64
	dX, dY, dZ          float64
	tX, tY, tZ          float64
	eRadX, eRadY, eRadZ float64
}

// NewCollisionCage creates a new CollisionCage with specified margin, restitution, and friction coefficients.
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

// Rebuild recalculates CollisionCage properties based on center, direction, radii, and swept volume parameters.
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
	s.ellipsoid.Rebuild(x, y, w, h, z, d)
	// Fast reset
	for i := 0; i < 6; i++ {
		s.counts[i] = 0
		copy(s.faces[i][:], _emptyBucketFaces[:])
	}
}

// GetMargin returns the margin value associated with the CollisionCage.
func (s *CollisionCage) GetMargin() float64 {
	return s.margin
}

// GetRad returns the radii of the ellipsoid in the X, Y, and Z dimensions.
func (s *CollisionCage) GetRad() (float64, float64, float64) {
	return s.eRadX, s.eRadY, s.eRadZ
}

// GetC returns the origin coordinates (cX, cY, cZ) of the collision cage.
func (s *CollisionCage) GetC() (float64, float64, float64) {
	return s.cX, s.cY, s.cZ
}

// GetD retrieves the displacement vector (dX, dY, dZ) describing the cage's offset relative to its origin.
func (s *CollisionCage) GetD() (float64, float64, float64) {
	return s.dX, s.dY, s.dZ
}

// GetT returns the target coordinates (tX, tY, tZ) of the CollisionCage as a tuple of three float64 values.
func (s *CollisionCage) GetT() (float64, float64, float64) {
	return s.tX, s.tY, s.tZ
}

// GetFaces returns a 2D array of pointers to CageEntry, representing the faces organized by bucket and index.
func (s *CollisionCage) GetFaces() [BucketSize][FacesPerBucket]*CageEntry {
	return s.faces
}

// GetAABB returns the axis-aligned bounding box (AABB) of the collision cage by delegating to the ellipsoid entity.
func (s *CollisionCage) GetAABB() *physics.AABB {
	return s.ellipsoid.GetAABB()
}

// GetEntity returns the physics.Entity instance associated with the CollisionCage.
func (s *CollisionCage) GetEntity() *physics.Entity {
	return s.ellipsoid
}

// AddFace adds a face to a suitable collision bucket based on constraints such as orientation, distance, and margin.
func (s *CollisionCage) AddFace(face *Face, maxCliff float64) {
	//TODO MOVE IN REBUILD
	baseCliff := s.cZ - s.eRadZ

	absX, absY, absZ := face.normalAbs.X, face.normalAbs.Y, face.normalAbs.Z
	other := face.GetAABB()
	fMaxZ := other.GetMaxZ()

	// CLIFF CULLING
	wallWE := absX > absY && absX > absZ
	wallNS := absY > absZ
	isWall := wallWE || wallNS

	if isWall && fMaxZ <= baseCliff+maxCliff {
		//fmt.Println("FILTRO WALL ATTIVO, RETURNING", fMaxZ, baseCliff+maxCliff)
		return
	}
	p0x, p0y, p0z := face.tri[0].X, face.tri[0].Y, face.tri[0].Z
	nX, nY, nZ := face.normal.X, face.normal.Y, face.normal.Z

	// ==========================================
	// ORIENTAMENTO E ASSEGNAZIONE BUCKET SIMULTANEA
	// ==========================================
	distStart := (s.cX-p0x)*nX + (s.cY-p0y)*nY + (s.cZ-p0z)*nZ
	var bucket BucketType

	if isWall {
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
			distStart = -distStart
		}
		// Assegnazione bucket per i muri in base alla normale finale
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
		// PIANI ORIZZONTALI
		planeZ := p0z
		if math.Abs(nZ) > 1e-5 {
			planeZ = p0z - (nX*(s.cX-p0x)+nY*(s.cY-p0y))/nZ
		}
		if s.cZ >= planeZ-maxCliff {
			bucket = BucketFloor // È matematicamente un Pavimento
			if nZ < 0 {
				nX, nY, nZ = -nX, -nY, -nZ
				distStart = -distStart
			}
		} else {
			bucket = BucketCeiling // È matematicamente un Soffitto
			if nZ > 0 {
				nX, nY, nZ = -nX, -nY, -nZ
				distStart = -distStart
			}
		}
	}

	rEff := math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))
	distTarget := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
	distSurfTarget := distTarget - rEff

	if distSurfTarget > s.margin {
		//fmt.Println("FILTRO MARGIN ATTIVO, RETURNING", distSurfTarget, margin)
		return
	}

	s.add(bucket, face, distSurfTarget, rEff, nX, nY, nZ)

	/*
		self := s.ellipsoid.GetAABB()
		minX, minY, minZ := self.GetMinX(), self.GetMinY(), self.GetMinZ()
		maxX, maxY, maxZ := self.GetMaxX(), self.GetMaxY(), self.GetMaxZ()

		fMinX, fMinY, fMinZ := other.GetMinX(), other.GetMinY(), other.GetMinZ()
		fMaxX, fMaxY := other.GetMaxX(), other.GetMaxY()

		if maxX >= fMinX-s.margin && minX <= fMaxX+s.margin &&
			maxY >= fMinY-s.margin && minY <= fMaxY+s.margin &&
			maxZ >= fMinZ-s.margin && minZ <= fMaxZ+s.margin {
			s.add(bucket, face, distSurfTarget, rEff, nX, nY, nZ)
		} else {
			fmt.Println("HEREE!!!!!!")
		}

	*/
}

// add inserts a face into the specified bucket or replaces the furthest face if the bucket is full and the new face is closer.
func (s *CollisionCage) add(bucket BucketType, face *Face, dist, rEff, normalX, normalY, normalZ float64) {
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
