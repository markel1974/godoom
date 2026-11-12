package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// ImpactMode identifies how a collision impact should be resolved and prioritized.
type ImpactMode int

type IObject interface {
	GetAABB() *aabb.AABB
	GetEntity() *rigid_body.Entity
}

type IFace interface {
	GetAABB() *aabb.AABB
	GetNormal() (float64, float64, float64)
	GetNormalAbs() (float64, float64, float64)
	Translate(int, float64, float64, float64) (float64, float64, float64)
	GetParent() IObject
	Get0() (float64, float64, float64)
	Get1() (float64, float64, float64)
	Get2() (float64, float64, float64)
	IsSolid() bool
}

// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
const (
	// ImpactInelastic represents a collision mode where objects experience infinite resistance to penetration or displacement.
	ImpactInelastic = ImpactMode(1)
	// ImpactElastic represents a collision impact mode where entities respond elastically, maintaining relative motion post-impact.
	ImpactElastic = ImpactMode(2)
	// ImpactStep is a constant representing a collision mode where the entity encounters a step or small height difference.
	ImpactStep = ImpactMode(3)
)

// Priority returns the collision ordering rank for the impact mode, with unknown modes ranked lowest.
func (iMode ImpactMode) Priority() int {
	switch iMode {
	case ImpactInelastic:
		return 3
	case ImpactElastic:
		return 2
	case ImpactStep:
		return 1
	default:
		return 0
	}
}

// satFilterEpsilon defines the tolerance threshold used in SAT to filter phantom plane projections in collision detection.
const satFilterEpsilon = 0.05

// planeEpsilon is a tolerance value used for floating-point precision when comparing coplanar planes or distances.
const planeEpsilon = 0.01

// toiTieEpsilon represents a small positive threshold used to handle numerical imprecision in time of impact calculations.
const toiTieEpsilon = 1e-4

// TotalSlots represents the total available slots, computed as BucketSize * FacesPerBucket.
const TotalSlots = BucketSize * FacesPerBucket

// Cage represents a data structure for managing collision information for a physical entity.
type Cage struct {
	seen                map[*Cage]bool
	object              IObject
	buckets             [BucketSize]*Bucket
	ellipsoid           *rigid_body.Entity
	ellipsoidLocal      [4]*rigid_body.Entity
	cX, cY, cZ          float64
	dX, dY, dZ          float64
	tX, tY, tZ          float64
	eRadX, eRadY, eRadZ float64
	volume              IObject
	distance            float64
	slots               []*Entry
	slotsLen            int
	maxStep             float64
	faces               []*Faces
	facesIdx            int
	hasFloor            bool
	applied             bool
	appliedX            float64
	appliedY            float64
	appliedZ            float64
}

// NewCage creates and returns a new instance of Cage initialized with the specified IThing object.
func NewCage(object IObject) *Cage {
	c := &Cage{
		seen:      make(map[*Cage]bool),
		object:    object,
		ellipsoid: rigid_body.NewEntity(0, 0, 0, 0),
		volume:    nil,
		slots:     make([]*Entry, TotalSlots),
		slotsLen:  0,
		faces:     make([]*Faces, 8),
		facesIdx:  0,
		hasFloor:  false,
	}
	for i := 0; i < len(c.faces); i++ {
		c.faces[i] = NewFaces()
	}
	for i := BucketType(0); i < BucketSize; i++ {
		c.buckets[i] = NewBucket(i)
	}
	for i := 0; i < len(c.ellipsoidLocal); i++ {
		c.ellipsoidLocal[i] = rigid_body.NewEntity(0, 0, 0, 0)
	}
	return c
}

func (s *Cage) GetObject() IObject {
	return s.object
}

// Rebuild recalculates the collision cage dimensions, position, and broad-phase swept volume using the given maxStep value.
func (s *Cage) Rebuild(maxStep float64) {
	lEnt := s.object.GetEntity()
	// Calculate Position
	s.dX, s.dY, s.dZ = lEnt.GetFrameDisplacement()

	// Default fallback: entity's oriented extents
	s.eRadX, s.eRadY, _ = lEnt.GetOrientedHalfExtents()
	_, _, s.eRadZ = lEnt.GetSizeCenter()
	s.cX, s.cY, s.cZ = lEnt.GetCenter()

	if objAABB := s.object.GetAABB(); objAABB != nil {
		minX := objAABB.GetMinX()
		maxX := objAABB.GetMaxX()
		minY := objAABB.GetMinY()
		maxY := objAABB.GetMaxY()
		minZ := objAABB.GetMinZ()
		maxZ := objAABB.GetMaxZ()
		if maxX > minX && maxY > minY && maxZ > minZ {
			s.cX = (minX + maxX) * 0.5
			s.cY = (minY + maxY) * 0.5
			s.cZ = (minZ + maxZ) * 0.5
			s.eRadX = (maxX - minX) * 0.5
			s.eRadY = (maxY - minY) * 0.5
			s.eRadZ = (maxZ - minZ) * 0.5
		}
	}

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
	s.hasFloor = false
	s.applied = false

	clear(s.seen)
	s.maxStep = maxStep
}

// AddFace adds a pair of faces (rFace and lFace) to the Cage, resizing the internal slice if needed.
func (s *Cage) AddFace(rFace IFace, lFace IFace) {
	if s.facesIdx >= len(s.faces) {
		n := make([]*Faces, len(s.faces)*2)
		copy(n, s.faces)
		for i := len(s.faces); i < len(n); i++ {
			n[i] = NewFaces()
		}
		s.faces = n
	}
	f := s.faces[s.facesIdx]
	f.rFace = rFace
	f.lFace = lFace
	s.facesIdx++
}

// Commit processes collision data between two CollisionCages, resolving interactions based on material and displacement factors.
func (s *Cage) Commit(rCage *Cage) {
	lAABB := s.ellipsoid.GetAABB()

	var rOffX, rOffY, rOffZ float64
	var lOffX, lOffY, lOffZ float64
	if rCage != nil {
		rEnt := rCage.object.GetEntity()
		rOffX, rOffY, rOffZ = rEnt.GetCenter()
		rDX, rDY, rDZ := rEnt.GetFrameDisplacement()
		rOffX += rDX
		rOffY += rDY
		rOffZ += rDZ

		lEnt := s.object.GetEntity()
		lOffX, lOffY, lOffZ = lEnt.GetCenter()
		lOffX += s.dX
		lOffY += s.dY
		lOffZ += s.dZ
	}

	for x := 0; x < s.facesIdx; x++ {
		faces := s.faces[x]
		rFace := faces.rFace
		lFace := faces.lFace

		if !rFace.IsSolid() {
			continue // Skybox or non-solid surface: ignore collision and volume tracking
		}

		var b BucketType
		var dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ float64

		// DISPATCHER: Mesh-vs-Mesh (6 DOF) o Ellipsoid-vs-Plane
		if lFace != nil && rCage != nil {
			//b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ := s.computeManifold(rFace, lFace, rOffX, rOffY, rOffZ, lOffX, lOffY, lOffZ)
			b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ = s.computeFaceMeshVsMesh(rCage, rFace, lFace, rOffX, rOffY, rOffZ, lOffX, lOffY, lOffZ)
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

		// Floor support detection: resting contacts within ground tolerance maintain hasFloor = true
		const floorGroundTolerance = 0.25
		if b == BucketFloor {
			if dist <= floorGroundTolerance {
				s.hasFloor = true
			}
		} else if b == BucketCeiling {
			if rCage != nil && dist <= floorGroundTolerance {
				rCage.hasFloor = true
			}
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
			minBaseZ := math.Min(baseZ, baseZ+s.dZ)
			if rMaxZ <= minBaseZ+satFilterEpsilon { // down-hill (going downhill) or within floor penetration slop
				continue
			}
			stepZ := baseZ + s.maxStep
			if s.maxStep > 0 && rMaxZ <= stepZ { // up-hill (climbable step)
				iMode = ImpactStep
			}
		}

		s.addToBucket(b, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, rMaxZ, iMode)
		if rCage != nil && iMode == ImpactElastic {
			recipFace := lFace
			if recipFace == nil {
				recipFace = rFace
			}
			lMaxZ := lAABB.GetMaxZ()
			oppB := b.Opposite()
			rCage.addToBucket(oppB, s, recipFace, dist, pen, -nX, -nY, -nZ, p0x, p0y, p0z, lMaxZ, ImpactElastic)
			if oppB == BucketFloor {
				rCage.hasFloor = true
			}
		}
		switch b {
		case BucketFloor:
			s.hasFloor = true
		case BucketCeiling:
			if rCage != nil {
				rCage.hasFloor = true
			}
		}
	}
	s.facesIdx = 0
}

// addToBucket adds a collision cage to the specified bucket and updates the slot list if a new cage is created.
func (s *Cage) addToBucket(bucket BucketType, rCage *Cage, rFace IFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) bool {
	cage := s.buckets[bucket].Add(bucket, s, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode)
	if cage != nil {
		s.slots[s.slotsLen] = cage
		s.slotsLen++
		return true
	}
	return false
}

// TranslateWorldToLocal transforms the world-space AABB of the ellipsoid to local-space relative to the given delta values.
func (s *Cage) TranslateWorldToLocal(slot int, deltaX, deltaY, deltaZ float64) *rigid_body.Entity {
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
func (s *Cage) TranslateCage(slot int, rCage *Cage) (aabb.IObject, float64, float64, float64) {
	lEnt := s.object.GetEntity()
	lOx, lOy, lOz := lEnt.GetCenter()
	lOx += s.dX
	lOy += s.dY
	lOz += s.dZ
	rEnt := rCage.object.GetEntity()
	rOx, rOy, rOz := rEnt.GetCenter()
	rDX, rDY, rDZ := rEnt.GetFrameDisplacement()
	rOx += rDX
	rOy += rDY
	rOz += rDZ

	from := s.ellipsoid.GetAABB()
	wMinX, wMaxX := from.GetMinX()-rOx, from.GetMaxX()-rOx
	wMinY, wMaxY := from.GetMinY()-rOy, from.GetMaxY()-rOy
	wMinZ, wMaxZ := from.GetMinZ()-rOz, from.GetMaxZ()-rOz

	wCx, wCy, wCz := (wMinX+wMaxX)*0.5, (wMinY+wMaxY)*0.5, (wMinZ+wMaxZ)*0.5
	wHx, wHy, wHz := (wMaxX-wMinX)*0.5, (wMaxY-wMinY)*0.5, (wMaxZ-wMinZ)*0.5

	a00, a01, a02, a10, a11, a12, a20, a21, a22 := rEnt.GetAxes()
	locCx := wCx*a00 + wCy*a01 + wCz*a02
	locCy := wCx*a10 + wCy*a11 + wCz*a12
	locCz := wCx*a20 + wCy*a21 + wCz*a22

	locHx := math.Abs(a00)*wHx + math.Abs(a01)*wHy + math.Abs(a02)*wHz
	locHy := math.Abs(a10)*wHx + math.Abs(a11)*wHy + math.Abs(a12)*wHz
	locHz := math.Abs(a20)*wHx + math.Abs(a21)*wHy + math.Abs(a22)*wHz

	w := s.ellipsoidLocal[slot]
	w.Rebuild(locCx-locHx, locCy-locHy, locCz-locHz, locHx*2.0, locHy*2.0, locHz*2.0)
	return w, rOx - lOx, rOy - lOy, rOz - lOz
}

// Translate updates the local ellipsoid of the given slot based on the provided AABB source and displacement values.
func (s *Cage) Translate(slot int, src aabb.IObject, deltaX, deltaY, deltaZ float64) *rigid_body.Entity {
	w := s.ellipsoidLocal[slot]
	to := src.GetAABB()
	rLocCx := (to.GetMinX() + to.GetMaxX()) * 0.5
	rLocCy := (to.GetMinY() + to.GetMaxY()) * 0.5
	rLocCz := (to.GetMinZ() + to.GetMaxZ()) * 0.5
	rLocHx := (to.GetMaxX() - to.GetMinX()) * 0.5
	rLocHy := (to.GetMaxY() - to.GetMinY()) * 0.5
	rLocHz := (to.GetMaxZ() - to.GetMinZ()) * 0.5

	wCx, wCy, wCz := rLocCx+deltaX, rLocCy+deltaY, rLocCz+deltaZ
	wHx, wHy, wHz := rLocHx, rLocHy, rLocHz

	if f, ok := src.(IFace); ok {
		nX, nY, nZ := f.GetNormal()
		if p := f.GetParent(); p != nil {

			if rEnt := p.GetEntity(); rEnt != nil {
				rA00, rA01, rA02, rA10, rA11, rA12, rA20, rA21, rA22 := rEnt.GetAxes()
				wCx = rA00*rLocCx + rA10*rLocCy + rA20*rLocCz + deltaX
				wCy = rA01*rLocCx + rA11*rLocCy + rA21*rLocCz + deltaY
				wCz = rA02*rLocCx + rA12*rLocCy + rA22*rLocCz + deltaZ
				wHx = math.Abs(rA00)*rLocHx + math.Abs(rA10)*rLocHy + math.Abs(rA20)*rLocHz
				wHy = math.Abs(rA01)*rLocHx + math.Abs(rA11)*rLocHy + math.Abs(rA21)*rLocHz
				wHz = math.Abs(rA02)*rLocHx + math.Abs(rA12)*rLocHy + math.Abs(rA22)*rLocHz
				fnX, fnY, fnZ := nX, nY, nZ
				nX = rA00*fnX + rA10*fnY + rA20*fnZ
				nY = rA01*fnX + rA11*fnY + rA21*fnZ
				nZ = rA02*fnX + rA12*fnY + rA22*fnZ
			}
		}
		wHx += math.Abs(nX) * s.eRadX * 2.0
		wHy += math.Abs(nY) * s.eRadY * 2.0
		wHz += math.Abs(nZ) * s.eRadZ * 2.0
	}

	lEnt := s.object.GetEntity()
	lA00, lA01, lA02, lA10, lA11, lA12, lA20, lA21, lA22 := lEnt.GetAxes()
	lLocCx := wCx*lA00 + wCy*lA01 + wCz*lA02
	lLocCy := wCx*lA10 + wCy*lA11 + wCz*lA12
	lLocCz := wCx*lA20 + wCy*lA21 + wCz*lA22

	lLocHx := math.Abs(lA00)*wHx + math.Abs(lA01)*wHy + math.Abs(lA02)*wHz
	lLocHy := math.Abs(lA10)*wHx + math.Abs(lA11)*wHy + math.Abs(lA12)*wHz
	lLocHz := math.Abs(lA20)*wHx + math.Abs(lA21)*wHy + math.Abs(lA22)*wHz

	w.Rebuild(lLocCx-lLocHx, lLocCy-lLocHy, lLocCz-lLocHz, lLocHx*2.0, lLocHy*2.0, lLocHz*2.0)
	return w
}

// HasSeen checks if the given Cage has already been marked as seen in the current Cage.
func (s *Cage) HasSeen(rCage *Cage) bool {
	return s.seen[rCage]
}

// Seen marks a given Cage as observed by the current Cage within its internal tracking.
func (s *Cage) Seen(rCage *Cage) {
	s.seen[rCage] = true
}

// GetBaseZ computes the base Z value of the collision cage by subtracting the ellipsoid radius in Z direction from its center Z coordinate.
func (s *Cage) GetBaseZ() float64 { return s.cZ - s.eRadZ }

// HasBucket reports whether the collision cage contains a bucket of the specified type.
func (s *Cage) HasBucket(t BucketType) bool {
	for z := 0; z < s.slotsLen; z++ {
		if s.slots[z].bucket == t {
			return true
		}
	}
	return false
}

// GetSlotsLen returns the current number of active slots in the Cage.
func (s *Cage) GetSlotsLen() int { return s.slotsLen }

// GetSlot returns the Entry at the specified index in the slots slice.
func (s *Cage) GetSlot(i int) *Entry { return s.slots[i] }

// GetThing returns the object associated with the cage.
func (s *Cage) GetThing() IObject { return s.object }

// GetVolume returns the volume associated with the Cage.
func (s *Cage) GetVolume() IObject { return s.volume }

// GetRad retrieves the ellipsoid radii along the X, Y, and Z axes.
func (s *Cage) GetRad() (float64, float64, float64) { return s.eRadX, s.eRadY, s.eRadZ }

// GetC returns the center coordinates (cX, cY, cZ) of the Cage.
func (s *Cage) GetC() (float64, float64, float64) { return s.cX, s.cY, s.cZ }

// GetDisplacement returns the displacement of the collision cage along the x, y, and z axes.
func (s *Cage) GetDisplacement() (float64, float64, float64) { return s.dX, s.dY, s.dZ }

// GetT retrieves the current transformed coordinates (tX, tY, tZ) of the Cage.
func (s *Cage) GetT() (float64, float64, float64) { return s.tX, s.tY, s.tZ }

// BucketCount returns the number of entries in the specified bucket.
func (s *Cage) BucketCount(t BucketType) int { return s.buckets[t].Count() }

// GetAABB returns the axis-aligned bounding box (AABB) representing the broad-phase collision geometry of the cage.
func (s *Cage) GetAABB() *aabb.AABB { return s.ellipsoid.GetAABB() }

// HasFloor reports whether the collision cage has a floor.
func (s *Cage) HasFloor() bool {
	return s.hasFloor
}

// GetEntity returns the physics entity associated with the collision cage.
func (s *Cage) GetEntity() *rigid_body.Entity {
	return s.object.GetEntity()
}

// SetApplied marks the collision cage as having applied its frame displacement and records entity center for tracking push-out offset.
func (s *Cage) SetApplied() {
	s.applied = true
	lEnt := s.object.GetEntity()
	s.appliedX, s.appliedY, s.appliedZ = lEnt.GetCenter()
}

// GetPushOffset returns the accumulated positional push-out offset of the entity in the current frame.
func (s *Cage) GetPushOffset() (float64, float64, float64) {
	lEnt := s.object.GetEntity()
	curX, curY, curZ := lEnt.GetCenter()
	if s.applied {
		return curX - s.appliedX, curY - s.appliedY, curZ - s.appliedZ
	}
	return curX - s.cX, curY - s.cY, curZ - s.cZ
}

// computeFace calculates the penetration and distance of a face relative to an ellipsoid's motion and assigns a collision bucket.
func (s *Cage) computeFace(lAABB *aabb.AABB, rFace IFace, offX, offY, offZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return 0, math.MaxFloat64, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	// Translation (Local -> World)
	p0x, p0y, p0z := rFace.Translate(0, offX, offY, offZ)
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
	if oX <= 0 || oY <= 0 {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	if penetration <= 0 {
		return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	oZ := math.Max(0.0, math.Min(lAABB.GetMaxZ()-rMinZ, rMaxZ-lAABB.GetMinZ()))
	if oZ <= 0 {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	// Calcolo TOI (Time Of Impact) tra il moto dell'ellissoide e la faccia statica
	rCx := (rMinX + rMaxX) * 0.5
	rCy := (rMinY + rMaxY) * 0.5
	rCz := (rMinZ + rMaxZ) * 0.5
	rRadX := (rMaxX - rMinX) * 0.5
	rRadY := (rMaxY - rMinY) * 0.5
	rRadZ := (rMaxZ - rMinZ) * 0.5

	relX := s.cX - rCx
	relY := s.cY - rCy
	relZ := s.cZ - rCz

	sumRadX := s.eRadX + rRadX
	sumRadY := s.eRadY + rRadY
	sumRadZ := s.eRadZ + rRadZ

	tEnter, tExit, _, _, _, ok := computeTOI(relX, relY, relZ, sumRadX, sumRadY, sumRadZ, s.dX, s.dY, s.dZ)
	if !ok {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	impDot := s.dX*nX + s.dY*nY + s.dZ*nZ
	tPlaneEnter, tPlaneExit, okPlane := sweptAxisInterval(distStart, rayEff, impDot)
	if !okPlane {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	overlapStart := math.Max(tEnter, tPlaneEnter)
	overlapEnd := math.Min(math.Min(1.0, tExit), tPlaneExit)
	if overlapStart > overlapEnd+toiTieEpsilon || overlapEnd < -toiTieEpsilon || overlapStart > 1.0+toiTieEpsilon {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	// Separazione tra la compenetrazione già presente a t = 0 (pen0) e l'affondamento prodotto dal moto nel frame (deltaPen).
	// Un piano infinito è fantasma se la compenetrazione proiettata eccede l'overlap fisico effettivo dell'AABB
	// lungo l'asse principale della normale del piano.
	pen0 := math.Max(0.0, rayEff-distStart)
	deltaPen := math.Max(0.0, penetration-pen0)

	normOX := oX
	if solidWE && nAbsX > 1e-5 && nAbsX < 1.0 {
		normOX = oX / nAbsX
	}
	normOY := oY
	if solidNS && nAbsY > 1e-5 && nAbsY < 1.0 {
		normOY = oY / nAbsY
	}
	normOZ := oZ
	if !solidWE && !solidNS && nAbsZ > 1e-5 && nAbsZ < 1.0 {
		normOZ = oZ / nAbsZ
	}
	var axisOverlap float64
	if solidWE {
		axisOverlap = normOX
	} else if solidNS {
		axisOverlap = normOY
	} else {
		axisOverlap = normOZ
	}
	minOverlap := axisOverlap + deltaPen

	if penetration > minOverlap+satFilterEpsilon {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}
	if penetration > minOverlap {
		penetration = minOverlap
	}

	return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ
}

// computeFaceMeshVsMesh determines the intersection and response between two mesh faces in 3D space with offsets.
// It calculates collision parameters such as entry/exit times, normals, overlap, and separation distance between meshes.
func (s *Cage) computeFaceMeshVsMesh(rCage *Cage, rFace IFace, lFace IFace, rOffX, rOffY, rOffZ float64, lOffX, lOffY, lOffZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	// Dati del piano remoto (Il Muro/Ostacolo)
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	rQ := rCage.GetEntity().GetQRotation()
	nX, nY, nZ = rQ.Rotate(nX, nY, nZ)
	nAbsX, nAbsY, nAbsZ = math.Abs(nX), math.Abs(nY), math.Abs(nZ)

	// Punto di ancoraggio del piano remoto in World Space a fine frame (t = 1),
	// poiché rOffX, rOffY, rOffZ includono già lo spostamento rDX, rDY, rDZ.
	lp0x, lp0y, lp0z := rFace.Get0()
	rp0x, rp0y, rp0z := rQ.Rotate(lp0x, lp0y, lp0z)
	p0x, p0y, p0z := rp0x+rOffX, rp0y+rOffY, rp0z+rOffZ

	rCx0, rCy0, rCz0 := rCage.GetC()
	rDX, rDY, rDZ := rCage.GetDisplacement()
	rTx, rTy, rTz := rCage.GetT()
	rRadX, rRadY, rRadZ := rCage.GetRad()

	relX := s.cX - rCx0
	relY := s.cY - rCy0
	relZ := s.cZ - rCz0

	// Orientiamo la normale verso l'esterno del volume remoto (valutato in modo coerente a t = 1).
	// Scartiamo solo i piani visivi interni privi di tag fisico (es. quad billboard centrale a Y=0).
	outDot := (p0x-rTx)*nX + (p0y-rTy)*nY + (p0z-rTz)*nZ
	if math.Abs(outDot) <= planeEpsilon {
		if relX*nX+relY*nY+relZ*nZ < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
		}
	} else if outDot < 0 {
		nX, nY, nZ = -nX, -nY, -nZ
	}

	// Classificazione dell'asse dominante della faccia remota e canonizzazione sul volume AABB
	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	if solidWE {
		if nX < 0 {
			nX = -1.0
		} else {
			nX = 1.0
		}
		nY, nZ = 0.0, 0.0
		nAbsX, nAbsY, nAbsZ = 1.0, 0.0, 0.0
		p0x, p0y, p0z = rTx+nX*rRadX, rTy, rTz
	} else if solidNS {
		if nY < 0 {
			nY = -1.0
		} else {
			nY = 1.0
		}
		nX, nZ = 0.0, 0.0
		nAbsX, nAbsY, nAbsZ = 0.0, 1.0, 0.0
		p0x, p0y, p0z = rTx, rTy+nY*rRadY, rTz
	} else {
		if nZ < 0 {
			nZ = -1.0
		} else {
			nZ = 1.0
		}
		nX, nY = 0.0, 0.0
		nAbsX, nAbsY, nAbsZ = 0.0, 0.0, 1.0
		p0x, p0y, p0z = rTx, rTy, rTz+nZ*rRadZ
	}

	// Punto di ancoraggio del piano remoto a inizio frame (t = 0)
	p0x0 := p0x - rDX
	p0y0 := p0y - rDY
	p0z0 := p0z - rDZ

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

	tEnter, tExit, tEnterX, tEnterY, tEnterZ, ok := computeTOI(relX, relY, relZ, sumRadX, sumRadY, sumRadZ, impX, impY, impZ)
	if !ok {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	// Overlap bilaterale a t = 1 (per contatti discreti/persistenti) e penetrazione unidirezionale dal lato d'ingresso
	// (che resta positiva anche in caso di tunneling oltre il centro o oltre la parete opposta a t = 1).
	relTx := s.tX - rTx
	relTy := s.tY - rTy
	relTz := s.tZ - rTz

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

	// Distanza iniziale a t = 0 del centro locale rispetto al piano remoto a t = 0
	distStart := (s.cX-p0x0)*nX + (s.cY-p0y0)*nY + (s.cZ-p0z0)*nZ
	rayEff := math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))

	var activeX, activeY, activeZ bool

	// In modalità discreta (volumi già sovrapposti a t = 0 su tutti gli assi), se a t = 1 si sono separati non c'è impatto
	boxOX := sumRadX - math.Abs(relTx)
	boxOY := sumRadY - math.Abs(relTy)
	boxOZ := sumRadZ - math.Abs(relTz)

	sepX := math.Abs(relX) - sumRadX
	sepY := math.Abs(relY) - sumRadY
	sepZ := math.Abs(relZ) - sumRadZ
	sweptMode := sepX > planeEpsilon || sepY > planeEpsilon || sepZ > planeEpsilon
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
		// Determinazione degli assi temporali attivi (indipendente dall'asse dominante della faccia)
		activeX = sepX > planeEpsilon && tEnterX >= tEnter-toiTieEpsilon && relX*impX < -planeEpsilon
		activeY = sepY > planeEpsilon && tEnterY >= tEnter-toiTieEpsilon && relY*impY < -planeEpsilon
		activeZ = sepZ > planeEpsilon && tEnterZ >= tEnter-toiTieEpsilon && relZ*impZ < -planeEpsilon
	} else {
		// In modalità discreta (compenetrazione già presente a t = 0), scartiamo solo se l'intero supporto a t = 0 era già oltre il piano
		if distStart+rayEff < -planeEpsilon {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		if boxOX <= 0 || boxOY <= 0 || boxOZ <= 0 {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		// Determinazione degli assi MTV attivi (indipendente dall'asse dominante della faccia)
		prevOX := math.Max(0.0, -sepX)
		prevOY := math.Max(0.0, -sepY)
		prevOZ := math.Max(0.0, -sepZ)
		activeX = isDiscreteMTV(prevOX, boxOX, relX, impX, prevOY, boxOY, prevOZ, boxOZ)
		activeY = isDiscreteMTV(prevOY, boxOY, relY, impY, prevOX, boxOX, prevOZ, boxOZ)
		activeZ = isDiscreteMTV(prevOZ, boxOZ, relZ, impZ, prevOX, boxOX, prevOY, boxOY)
	}

	// Verifica di compatibilità geometrica tra la normale del piano remoto e gli assi di contatto attivi
	var axisOverlap float64
	matchedAxis := false

	pickOverlap := func(entryO, boxO float64) float64 {
		if sweptMode {
			return entryO
		}
		return boxO
	}

	if solidWE {
		if activeX && relX*nX > 0 && (!sweptMode || impX*nX < -planeEpsilon) {
			axisOverlap = pickOverlap(entryOX, boxOX)
			matchedAxis = true
		}
	} else if solidNS {
		if activeY && relY*nY > 0 && (!sweptMode || impY*nY < -planeEpsilon) {
			axisOverlap = pickOverlap(entryOY, boxOY)
			matchedAxis = true
		}
	} else {
		const contactSlop = 0.25
		if (activeZ || (!sweptMode && boxOX > 0 && boxOY > 0 && boxOZ >= -contactSlop)) && relZ*nZ > 0 && (!sweptMode || impZ*nZ < -planeEpsilon) {
			axisOverlap = pickOverlap(entryOZ, boxOZ)
			matchedAxis = true
		}
	}
	if !matchedAxis || axisOverlap < -0.25 {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	maxPenetration := axisOverlap
	hitDist := -axisOverlap

	// Classificazione finale del Bucket sulla normale di contatto già determinata e orientata

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
			bucket = BucketCeiling
		}
	}

	tri0X, tri0Y, tri0Z := rFace.Get0()
	tri1X, tri1Y, tri1Z := rFace.Get1()
	tri2X, tri2Y, tri2Z := rFace.Get2()
	_, _, z0 := rQ.Rotate(tri0X, tri0Y, tri0Z)
	_, _, z1 := rQ.Rotate(tri1X, tri1Y, tri1Z)
	_, _, z2 := rQ.Rotate(tri2X, tri2Y, tri2Z)
	rMaxZ := math.Max(z0, math.Max(z1, z2)) + rOffZ

	return bucket, hitDist, maxPenetration, nX, nY, nZ, p0x, p0y, p0z, axisOverlap, rMaxZ
}

/*
// computeManifold calculates the collision manifold between two 3D faces with offset transformations.
// Returns a ContactManifold containing collision details or non-collision state.
func (s *CollisionCage) computeManifold(rFace *Face, lFace *Face, rOffX, rOffY, rOffZ float64, lOffX, lOffY, lOffZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
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
	//cm := ContactManifold{Hit: false}

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

	//b = cm.Bucket
	//dist = cm.HitDist
	//pen = cm.Depth
	//nX, nY, nZ = cm.NormalX, manifold.NormalY, manifold.NormalZ
	//p0x, p0y, p0z = cm.P0X, manifold.P0Y, manifold.P0Z
	//minOverlap = cm.MinOverlap
	//rMaxZ = cm.RMaxZ
	return cm
}
*/
