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
const totalSlots = BucketSize * FacesPerBucket

// Cage represents a data structure for managing collision information for a physical entity.[cite: 6]
type Cage struct {
	seen                []uint64
	session             uint64
	id                  uint64
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

//var _counter uint64

// NewCage creates and returns a new instance of Cage initialized with the specified IThing object.[cite: 6]
func NewCage(id uint64, object IObject) *Cage {
	c := &Cage{
		id:        id,
		seen:      make([]uint64, max(id*2, 1024)),
		object:    object,
		ellipsoid: rigid_body.NewEntity(0, 0, 0, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0),
		volume:    nil,
		slots:     make([]*Entry, totalSlots),
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
		c.ellipsoidLocal[i] = rigid_body.NewEntity(0, 0, 0, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0)
	}
	return c
}

// GetObject returns the object associated with the cage.[cite: 6]
func (s *Cage) GetObject() IObject {
	return s.object
}

// Rebuild recalculates the collision cage dimensions, position, and broad-phase swept volume using the given maxStep value.[cite: 6]
func (s *Cage) Rebuild(maxStep float64, session uint64) {
	s.session = session
	lEnt := s.object.GetEntity()
	// Calculate Position
	s.dX, s.dY, s.dZ = lEnt.GetFrameDisplacement()

	// Oriented extents from entity OBB (kept synchronized by Entity.Update).
	s.eRadX, s.eRadY, s.eRadZ = lEnt.GetOrientedHalfExtents()
	s.cX, s.cY, s.cZ = lEnt.GetCenter()

	if lEnt.KeepUpright() {
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

	//clear(s.seen)
	s.maxStep = maxStep
}

// AddFace adds a pair of faces (rFace and lFace) to the Cage, resizing the internal slice if needed.[cite: 6]
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

// Commit processes collision data between two CollisionCages, resolving interactions based on material and displacement factors.[cite: 6]
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
			continue // Skybox or non-solid surface: ignore collision and volume tracking[cite: 6]
		}

		var b BucketType
		var dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ float64

		// DISPATCHER: Mesh-vs-Mesh (6 DOF) o Ellipsoid-vs-Plane[cite: 6]
		if lFace != nil && rCage != nil {
			b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ = s.computeFaceMeshVsMesh(rCage, rFace, lFace, rOffX, rOffY, rOffZ, lOffX, lOffY, lOffZ)
		} else {
			b, dist, pen, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ = s.computeFace(lAABB, rFace, rOffX, rOffY, rOffZ)
		}

		// Volume Priority per le mesh statiche[cite: 6]
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

		// Floor support detection: resting contacts within ground tolerance maintain hasFloor = true[cite: 6]
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
		// we are intersecting the projection of a phantom orthogonal plane. Discard it.[cite: 6]
		if pen > minOverlap+satFilterEpsilon { // SAT filter (Anti-Phantom Plane)[cite: 6]
			continue
		}

		if b.IsWall() {
			baseZ := s.GetBaseZ()
			minBaseZ := math.Min(baseZ, baseZ+s.dZ)
			if rMaxZ <= minBaseZ+satFilterEpsilon { // down-hill (going downhill) or within floor penetration slop[cite: 6]
				continue
			}
			stepZ := baseZ + s.maxStep
			if s.maxStep > 0 && rMaxZ <= stepZ { // up-hill (climbable step)[cite: 6]
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

// addToBucket adds a collision cage to the specified bucket and updates the slot list if a new cage is created.[cite: 6]
func (s *Cage) addToBucket(bucket BucketType, rCage *Cage, rFace IFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z float64, maxZ float64, iMode ImpactMode) bool {
	cage := s.buckets[bucket].Add(bucket, s, rCage, rFace, dist, pen, nX, nY, nZ, p0x, p0y, p0z, maxZ, iMode)
	if cage != nil {
		s.slots[s.slotsLen] = cage
		s.slotsLen++
		return true
	}
	return false
}

// TranslateWorldToLocal transforms the world-space AABB of the ellipsoid to local-space relative to the given delta values.[cite: 6]
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

// TranslateCage translates the collision cage based on the given slot and reference cage, returning the local AABB and offsets.[cite: 6]
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

// Translate updates the local ellipsoid of the given slot based on the provided AABB source and displacement values.[cite: 6]
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

// HasSeen checks if the given Cage has already been marked as seen in the current Cage.[cite: 6]
func (s *Cage) HasSeen(rCage *Cage) bool {
	if rCage.id >= uint64(len(s.seen)) {
		s.growSeen(rCage.id)
	}
	return s.seen[rCage.id] == s.session
}

// Seen marks a given Cage as observed by the current Cage within its internal tracking.[cite: 6]
func (s *Cage) Seen(rCage *Cage) {
	if rCage.id >= uint64(len(s.seen)) {
		s.growSeen(rCage.id)
	}
	s.seen[rCage.id] = s.session
}

// GetBaseZ computes the base Z value of the collision cage by subtracting the ellipsoid radius in Z direction from its center Z coordinate.[cite: 6]
func (s *Cage) GetBaseZ() float64 { return s.cZ - s.eRadZ }

// HasBucket reports whether the collision cage contains a bucket of the specified type.[cite: 6]
func (s *Cage) HasBucket(t BucketType) bool {
	for z := 0; z < s.slotsLen; z++ {
		if s.slots[z].bucket == t {
			return true
		}
	}
	return false
}

// GetSlotsLen returns the current number of active slots in the Cage.[cite: 6]
func (s *Cage) GetSlotsLen() int { return s.slotsLen }

// GetSlot returns the Entry at the specified index in the slots slice.[cite: 6]
func (s *Cage) GetSlot(i int) *Entry { return s.slots[i] }

// GetThing returns the object associated with the cage.[cite: 6]
func (s *Cage) GetThing() IObject { return s.object }

// GetVolume returns the volume associated with the Cage.[cite: 6]
func (s *Cage) GetVolume() IObject { return s.volume }

// GetRad retrieves the ellipsoid radii along the X, Y, and Z axes.[cite: 6]
func (s *Cage) GetRad() (float64, float64, float64) { return s.eRadX, s.eRadY, s.eRadZ }

// GetC returns the center coordinates (cX, cY, cZ) of the Cage.[cite: 6]
func (s *Cage) GetC() (float64, float64, float64) { return s.cX, s.cY, s.cZ }

// GetDisplacement returns the displacement of the collision cage along the x, y, and z axes.[cite: 6]
func (s *Cage) GetDisplacement() (float64, float64, float64) { return s.dX, s.dY, s.dZ }

// GetT retrieves the current transformed coordinates (tX, tY, tZ) of the Cage.[cite: 6]
func (s *Cage) GetT() (float64, float64, float64) { return s.tX, s.tY, s.tZ }

// BucketCount returns the number of entries in the specified bucket.[cite: 6]
func (s *Cage) BucketCount(t BucketType) int { return s.buckets[t].Count() }

// GetAABB returns the axis-aligned bounding box (AABB) representing the broad-phase collision geometry of the cage.[cite: 6]
func (s *Cage) GetAABB() *aabb.AABB { return s.ellipsoid.GetAABB() }

// HasFloor reports whether the collision cage has a floor.[cite: 6]
func (s *Cage) HasFloor() bool {
	return s.hasFloor
}

// GetEntity returns the physics entity associated with the collision cage.[cite: 6]
func (s *Cage) GetEntity() *rigid_body.Entity {
	return s.object.GetEntity()
}

// SetApplied marks the collision cage as having applied its frame displacement and records entity center for tracking push-out offset.[cite: 6]
func (s *Cage) SetApplied() {
	s.applied = true
	lEnt := s.object.GetEntity()
	s.appliedX, s.appliedY, s.appliedZ = lEnt.GetCenter()
}

// GetPushOffset returns the accumulated positional push-out offset of the entity in the current frame.[cite: 6]
func (s *Cage) GetPushOffset() (float64, float64, float64) {
	lEnt := s.object.GetEntity()
	curX, curY, curZ := lEnt.GetCenter()
	if s.applied {
		return curX - s.appliedX, curY - s.appliedY, curZ - s.appliedZ
	}
	return curX - s.cX, curY - s.cY, curZ - s.cZ
}

// computeFace calculates the penetration and distance of a face relative to an ellipsoid's motion and assigns a collision bucket.[cite: 6]
func (s *Cage) computeFace(lAABB *aabb.AABB, rFace IFace, offX, offY, offZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return 0, math.MaxFloat64, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	// Translation (Local -> World)[cite: 6]
	p0x, p0y, p0z := rFace.Translate(0, offX, offY, offZ)
	distStart := (s.cX-p0x)*nX + (s.cY-p0y)*nY + (s.cZ-p0z)*nZ

	var bucket BucketType
	// Universal normalization[cite: 6]
	if solidWE || solidNS {
		// Facing Normalization: Forces the plane to oppose the thing[cite: 6]
		if distStart < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
			distStart = -distStart
		}
		// Wall Bucket Assignment[cite: 6]
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
		// Exact elevation evaluation (Plane Z at Center X,Y)[cite: 6]
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

	rFaceAABB := rFace.GetAABB()
	rMinX := rFaceAABB.GetMinX() + offX
	rMaxX := rFaceAABB.GetMaxX() + offX
	rMinY := rFaceAABB.GetMinY() + offY
	rMaxY := rFaceAABB.GetMaxY() + offY
	rMinZ := rFaceAABB.GetMinZ() + offZ
	rMaxZ := rFaceAABB.GetMaxZ() + offZ

	return s.resolveFaceGeometry(
		bucket,
		nX, nY, nZ, nAbsX, nAbsY, nAbsZ,
		p0x, p0y, p0z, distStart,
		rMinX, rMaxX, rMinY, rMaxY, rMinZ, rMaxZ,
		s.dX, s.dY, s.dZ,
		0, 0, 0,
		solidWE, solidNS,
	)
}

const satMinAxisNormal = 0.05

// supportRadiusAlong computes the half-extent of the local collider along unit normal (nX, nY, nZ).
// Free 6-DoF rigid bodies use their oriented bounding box projection; KeepUpright entities use ellipsoid radii.
func (s *Cage) supportRadiusAlong(nX, nY, nZ float64) float64 {
	if lEnt := s.object.GetEntity(); lEnt != nil && !lEnt.KeepUpright() {
		ax0x, ax0y, ax0z, ax1x, ax1y, ax1z, ax2x, ax2y, ax2z := lEnt.GetAxes()
		wx, wy, wz := lEnt.GetSize()
		return math.Abs(nX*ax0x+nY*ax0y+nZ*ax0z)*(wx*0.5) +
			math.Abs(nX*ax1x+nY*ax1y+nZ*ax1z)*(wy*0.5) +
			math.Abs(nX*ax2x+nY*ax2y+nZ*ax2z)*(wz*0.5)
	}
	return math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))
}

// satMinAxisOverlap computes the minimum separation distance along unit normal n across all active AABB overlap axes.
func satMinAxisOverlap(nAbsX, nAbsY, nAbsZ, oX, oY, oZ float64, solidWE, solidNS bool) float64 {
	axisOverlap := math.MaxFloat64
	if nAbsX >= satMinAxisNormal {
		if ox := oX / nAbsX; ox < axisOverlap {
			axisOverlap = ox
		}
	}
	if nAbsY >= satMinAxisNormal {
		if oy := oY / nAbsY; oy < axisOverlap {
			axisOverlap = oy
		}
	}
	if nAbsZ >= satMinAxisNormal {
		if oz := oZ / nAbsZ; oz < axisOverlap {
			axisOverlap = oz
		}
	}
	if axisOverlap == math.MaxFloat64 {
		if solidWE {
			return oX
		}
		if solidNS {
			return oY
		}
		return oZ
	}
	return axisOverlap
}

func (s *Cage) resolveFaceGeometry(
	bucket BucketType,
	nX, nY, nZ, nAbsX, nAbsY, nAbsZ float64,
	p0x, p0y, p0z float64,
	distStart float64,
	rMinX, rMaxX, rMinY, rMaxY, rMinZ, rMaxZ float64,
	impX, impY, impZ float64,
	rDX, rDY, rDZ float64,
	solidWE, solidNS bool,
) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	rayEff := s.supportRadiusAlong(nX, nY, nZ)
	distTarget := (s.tX-p0x)*nX + (s.tY-p0y)*nY + (s.tZ-p0z)*nZ
	dist := distTarget - rayEff
	penetration := rayEff - distTarget

	// Relative swept bounds of s at t=1 and t=0 (shifted by remote frame displacement rD).
	startRelX := s.tX - impX
	startRelY := s.tY - impY
	startRelZ := s.tZ - impZ
	sMinX := math.Min(s.tX, startRelX) - s.eRadX
	sMaxX := math.Max(s.tX, startRelX) + s.eRadX
	sMinY := math.Min(s.tY, startRelY) - s.eRadY
	sMaxY := math.Max(s.tY, startRelY) + s.eRadY
	sMinZ := math.Min(s.tZ, startRelZ) - s.eRadZ
	sMaxZ := math.Max(s.tZ, startRelZ) + s.eRadZ

	oX := math.Max(0.0, math.Min(sMaxX-rMinX, rMaxX-sMinX))
	oY := math.Max(0.0, math.Min(sMaxY-rMinY, rMaxY-sMinY))
	if oX <= 0 || oY <= 0 {
		return 0, math.MaxFloat64, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	if penetration <= 0 {
		return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	oZ := math.Max(0.0, math.Min(sMaxZ-rMinZ, rMaxZ-sMinZ))
	if oZ <= 0 {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	// Calcolo TOI tra il volume locale e la faccia nello spazio relativo
	rCx := (rMinX+rMaxX)*0.5 - rDX
	rCy := (rMinY+rMaxY)*0.5 - rDY
	rCz := (rMinZ+rMaxZ)*0.5 - rDZ
	rRadX := (rMaxX - rMinX) * 0.5
	rRadY := (rMaxY - rMinY) * 0.5
	rRadZ := (rMaxZ - rMinZ) * 0.5

	relX := s.cX - rCx
	relY := s.cY - rCy
	relZ := s.cZ - rCz

	sumRadX := s.eRadX + rRadX
	sumRadY := s.eRadY + rRadY
	sumRadZ := s.eRadZ + rRadZ

	tEnter, tExit, _, _, _, ok := computeTOI(relX, relY, relZ, sumRadX, sumRadY, sumRadZ, impX, impY, impZ)
	if !ok {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	impDot := impX*nX + impY*nY + impZ*nZ
	tPlaneEnter, tPlaneExit, okPlane := sweptAxisInterval(distStart, rayEff, impDot)
	if !okPlane {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	overlapStart := math.Max(tEnter, tPlaneEnter)
	overlapEnd := math.Min(math.Min(1.0, tExit), tPlaneExit)
	if overlapStart > overlapEnd+toiTieEpsilon || overlapEnd < -toiTieEpsilon || overlapStart > 1.0+toiTieEpsilon {
		return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
	}

	pen0 := math.Max(0.0, rayEff-distStart)
	deltaPen := math.Max(0.0, penetration-pen0)
	axisOverlap := satMinAxisOverlap(nAbsX, nAbsY, nAbsZ, oX, oY, oZ, solidWE, solidNS)
	minOverlap := axisOverlap + deltaPen

	if penetration > minOverlap {
		// Reject only back-side phantom planes; front-side contacts on narrow spikes clamp to SAT axis overlap.
		if distTarget < -satFilterEpsilon && penetration > minOverlap+satFilterEpsilon {
			return bucket, dist, 0, nX, nY, nZ, p0x, p0y, p0z, 0, rMaxZ
		}
		penetration = minOverlap
	}

	supX, supY, supZ := s.object.GetEntity().GetSupportPoint(-nX, -nY, -nZ)
	supX += s.dX
	supY += s.dY
	supZ += s.dZ
	distSup := (supX-p0x)*nX + (supY-p0y)*nY + (supZ-p0z)*nZ

	p0x = math.Max(rMinX, math.Min(rMaxX, supX-nX*distSup))
	p0y = math.Max(rMinY, math.Min(rMaxY, supY-nY*distSup))
	p0z = math.Max(rMinZ, math.Min(rMaxZ, supZ-nZ*distSup))

	return bucket, dist, penetration, nX, nY, nZ, p0x, p0y, p0z, minOverlap, rMaxZ
}

// computeFaceMeshVsMesh determines the intersection and response between two mesh faces in 3D space with offsets.
// It calculates collision parameters such as entry/exit times, normals, overlap, and separation distance between meshes.[cite: 6]
func (s *Cage) computeFaceMeshVsMesh(rCage *Cage, rFace IFace, lFace IFace, rOffX, rOffY, rOffZ float64, lOffX, lOffY, lOffZ float64) (BucketType, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	// Dati del piano remoto (Il Muro/Ostacolo)[cite: 6]
	nX, nY, nZ := rFace.GetNormal()
	nAbsX, nAbsY, nAbsZ := rFace.GetNormalAbs()
	if nAbsX == 0 && nAbsY == 0 && nAbsZ == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}

	rQ := rCage.GetEntity().GetQRotation()
	nX, nY, nZ = rQ.Rotate(nX, nY, nZ)
	nAbsX, nAbsY, nAbsZ = math.Abs(nX), math.Abs(nY), math.Abs(nZ)

	// Punto di ancoraggio del piano remoto in World Space a fine frame (t = 1),
	// poiché rOffX, rOffY, rOffZ includono già lo spostamento rDX, rDY, rDZ.[cite: 6]
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
	// Scartiamo solo i piani visivi interni privi di tag fisico (es. quad billboard centrale a Y=0).[cite: 6]
	outDot := (p0x-rTx)*nX + (p0y-rTy)*nY + (p0z-rTz)*nZ
	if math.Abs(outDot) <= planeEpsilon {
		if relX*nX+relY*nY+relZ*nZ < 0 {
			nX, nY, nZ = -nX, -nY, -nZ
		}
	} else if outDot < 0 {
		nX, nY, nZ = -nX, -nY, -nZ
	}

	solidWE := nAbsX > nAbsY && nAbsX > nAbsZ
	solidNS := nAbsY > nAbsZ

	rEnt := rCage.GetEntity()
	if !rEnt.KeepUpright() || math.Abs(outDot) > planeEpsilon {
		v1x, v1y, v1z := rQ.Rotate(rFace.Get1())
		v2x, v2y, v2z := rQ.Rotate(rFace.Get2())
		rMinX := math.Min(rp0x, math.Min(v1x, v2x)) + rOffX
		rMaxX := math.Max(rp0x, math.Max(v1x, v2x)) + rOffX
		rMinY := math.Min(rp0y, math.Min(v1y, v2y)) + rOffY
		rMaxY := math.Max(rp0y, math.Max(v1y, v2y)) + rOffY
		rMinZ := math.Min(rp0z, math.Min(v1z, v2z)) + rOffZ
		rMaxZ := math.Max(rp0z, math.Max(v1z, v2z)) + rOffZ

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
			if nZ >= 0 {
				bucket = BucketFloor
			} else {
				bucket = BucketCeiling
			}
		}

		distStart := (s.cX-(p0x-rDX))*nX + (s.cY-(p0y-rDY))*nY + (s.cZ-(p0z-rDZ))*nZ
		return s.resolveFaceGeometry(
			bucket,
			nX, nY, nZ, nAbsX, nAbsY, nAbsZ,
			p0x, p0y, p0z, distStart,
			rMinX, rMaxX, rMinY, rMaxY, rMinZ, rMaxZ,
			s.dX-rDX, s.dY-rDY, s.dZ-rDZ,
			rDX, rDY, rDZ,
			solidWE, solidNS,
		)
	}

	// Billboard / upright sprite fallback: canonizzazione sul volume AABB[cite: 6]

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

	// Punto di ancoraggio del piano remoto a inizio frame (t = 0)[cite: 6]
	p0x0 := p0x - rDX
	p0y0 := p0y - rDY
	p0z0 := p0z - rDZ

	// La faccia remota deve essere rivolta verso il centro relativo iniziale dell'entità locale[cite: 6]
	if relX*nX+relY*nY+relZ*nZ <= 0 {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	// Spostamento relativo nel frame corrente (Locale rispetto a Remoto)[cite: 6]
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
	// (che resta positiva anche in caso di tunneling oltre il centro o oltre la parete opposta a t = 1).[cite: 6]
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

	// Distanza iniziale a t = 0 del centro locale rispetto al piano remoto a t = 0[cite: 6]
	distStart := (s.cX-p0x0)*nX + (s.cY-p0y0)*nY + (s.cZ-p0z0)*nZ
	rayEff := math.Sqrt((nX*s.eRadX)*(nX*s.eRadX) + (nY*s.eRadY)*(nY*s.eRadY) + (nZ*s.eRadZ)*(nZ*s.eRadZ))

	var activeX, activeY, activeZ bool

	// In modalità discreta (volumi già sovrapposti a t = 0 su tutti gli assi), se a t = 1 si sono separati non c'è impatto[cite: 6]
	boxOX := sumRadX - math.Abs(relTx)
	boxOY := sumRadY - math.Abs(relTy)
	boxOZ := sumRadZ - math.Abs(relTz)

	sepX := math.Abs(relX) - sumRadX
	sepY := math.Abs(relY) - sumRadY
	sepZ := math.Abs(relZ) - sumRadZ
	sweptMode := sepX > planeEpsilon || sepY > planeEpsilon || sepZ > planeEpsilon
	if sweptMode {
		// In modalità swept il centro locale deve partire dal semispazio esterno del piano[cite: 6]
		if distStart < -planeEpsilon {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		// Verifica che il contatto col piano avvenga all'interno della finestra di sovrapposizione AABB [tEnter, min(1, tExit)][cite: 6]
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
			// A t = tEnter il centro locale aveva già oltrepassato il prolungamento infinito del piano (faccia fantasma)[cite: 6]
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		// Determinazione degli assi temporali attivi (indipendente dall'asse dominante della faccia)[cite: 6]
		activeX = sepX > planeEpsilon && tEnterX >= tEnter-toiTieEpsilon && relX*impX < -planeEpsilon
		activeY = sepY > planeEpsilon && tEnterY >= tEnter-toiTieEpsilon && relY*impY < -planeEpsilon
		activeZ = sepZ > planeEpsilon && tEnterZ >= tEnter-toiTieEpsilon && relZ*impZ < -planeEpsilon
	} else {
		// In modalità discreta (compenetrazione già presente a t = 0), scartiamo solo se l'intero supporto a t = 0 era già oltre il piano[cite: 6]
		if distStart+rayEff < -planeEpsilon {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		if boxOX <= 0 || boxOY <= 0 || boxOZ <= 0 {
			return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
		}
		// Determinazione degli assi MTV attivi (indipendente dall'asse dominante della faccia)[cite: 6]
		prevOX := math.Max(0.0, -sepX)
		prevOY := math.Max(0.0, -sepY)
		prevOZ := math.Max(0.0, -sepZ)
		activeX = isDiscreteMTV(prevOX, boxOX, relX, impX, prevOY, boxOY, prevOZ, boxOZ)
		activeY = isDiscreteMTV(prevOY, boxOY, relY, impY, prevOX, boxOX, prevOZ, boxOZ)
		activeZ = isDiscreteMTV(prevOZ, boxOZ, relZ, impZ, prevOX, boxOX, prevOY, boxOY)
	}

	// Verifica di compatibilità geometrica tra la normale del piano remoto e gli assi di contatto attivi[cite: 6]
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
		// Correzione applicata: rimosso il bypass che forzava l'asse Z su qualsiasi spuntone laterale
		if activeZ && relZ*nZ > 0 && (!sweptMode || impZ*nZ < -planeEpsilon) {
			axisOverlap = pickOverlap(entryOZ, boxOZ)
			matchedAxis = true
		}
	}
	if !matchedAxis || axisOverlap < -0.25 {
		return 0, 0, 0, nX, nY, nZ, p0x, p0y, p0z, 0, 0
	}

	maxPenetration := axisOverlap
	hitDist := -axisOverlap

	var contactX, contactY, contactZ float64
	if ent := s.object.GetEntity(); ent != nil && !ent.KeepUpright() {
		supX, supY, supZ := ent.GetSupportPoint(-nX, -nY, -nZ)
		contactX = math.Max(rTx-rRadX, math.Min(rTx+rRadX, supX+s.dX))
		contactY = math.Max(rTy-rRadY, math.Min(rTy+rRadY, supY+s.dY))
		contactZ = math.Max(rTz-rRadZ, math.Min(rTz+rRadZ, supZ+s.dZ))
	} else {
		// Calcolo del baricentro della patch di contatto sulle coordinate tangenziali al piano d'impatto.
		// In caso di tunneling ad alta velocità in sweptMode (intervallo a t = 1 disgiunto),
		// effettua il clamp del centro locale sui bordi del volume remoto.[cite: 6]
		patchMinX := math.Max(s.tX-s.eRadX, rTx-rRadX)
		patchMaxX := math.Min(s.tX+s.eRadX, rTx+rRadX)
		patchMinY := math.Max(s.tY-s.eRadY, rTy-rRadY)
		patchMaxY := math.Min(s.tY+s.eRadY, rTy+rRadY)
		patchMinZ := math.Max(s.tZ-s.eRadZ, rTz-rRadZ)
		patchMaxZ := math.Min(s.tZ+s.eRadZ, rTz+rRadZ)

		contactX = math.Max(rTx-rRadX, math.Min(rTx+rRadX, s.tX))
		if patchMaxX >= patchMinX {
			contactX = (patchMinX + patchMaxX) * 0.5
		}
		contactY = math.Max(rTy-rRadY, math.Min(rTy+rRadY, s.tY))
		if patchMaxY >= patchMinY {
			contactY = (patchMinY + patchMaxY) * 0.5
		}
		contactZ = math.Max(rTz-rRadZ, math.Min(rTz+rRadZ, s.tZ))
		if patchMaxZ >= patchMinZ {
			contactZ = (patchMinZ + patchMaxZ) * 0.5
		}
	}

	// Classificazione finale del Bucket e assegnazione coordinate tangenziali del punto di contatto[cite: 6]
	var bucket BucketType
	if solidWE {
		p0y, p0z = contactY, contactZ
		if nX < 0 {
			bucket = BucketWallWest
		} else {
			bucket = BucketWallEast
		}
	} else if solidNS {
		p0x, p0z = contactX, contactZ
		if nY < 0 {
			bucket = BucketWallNorth
		} else {
			bucket = BucketWallSouth
		}
	} else {
		p0x, p0y = contactX, contactY
		if nZ > 0 {
			bucket = BucketFloor
		} else {
			bucket = BucketCeiling
		}
	}

	z0 := rQ.RotateZ(rFace.Get0())
	z1 := rQ.RotateZ(rFace.Get1())
	z2 := rQ.RotateZ(rFace.Get2())
	rMaxZ := math.Max(z0, math.Max(z1, z2)) + rOffZ

	return bucket, hitDist, maxPenetration, nX, nY, nZ, p0x, p0y, p0z, axisOverlap, rMaxZ
}

// growSeen expands the seen-session tracking slice to accommodate idx while preserving existing marks.
func (s *Cage) growSeen(idx uint64) {
	n := make([]uint64, idx*2)
	copy(n, s.seen)
	s.seen = n
}
