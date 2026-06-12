package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// Volumes represents a collection of 3D or 2D world, utilizing a hierarchical spatial structure and caching for efficiency.
type Volumes struct {
	container []*Volume
	tree      *physics.AABBTree
	cache     map[string]*Volume
}

// NewVolumes initializes a Volumes structure with a container of Volume instances and a cache for quick access by ID.
func NewVolumes(container []*Volume) *Volumes {
	cache := make(map[string]*Volume)
	for _, sec := range container {
		cache[sec.GetId()] = sec
	}
	vs := &Volumes{
		container: container,
		cache:     cache,
		tree:      physics.NewAABBTree(uint(len(container)), 4.0),
	}
	return vs
}

// Setup constructs a new AABB tree based on the current container and populates it with rebuilt location objects.
func (s *Volumes) Setup() {
	for _, volume := range s.container {
		if volume.Rebuild() {
			s.tree.InsertObject(volume)
		}
	}
}

// GetVolume retrieves a Volume from the cache using the provided unique identifier.
func (s *Volumes) GetVolume(id string) *Volume {
	return s.cache[id]
}

// GetVolumes returns all Volume objects managed by the Volumes instance.
func (s *Volumes) GetVolumes() []*Volume {
	return s.container
}

// Len returns the number of Volume objects contained within the Volumes container.
func (s *Volumes) Len() int {
	return len(s.container)
}

// Query retrieves a list of world that overlap with the specified axis-aligned bounding box (AABB).
func (s *Volumes) Query(aabb physics.IAABB) []*Volume {
	var target []*Volume
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		sector, ok := object.(*Volume)
		if !ok {
			return false
		}
		target = append(target, sector)
		return false
	})
	return target
}

// QueryCollisionCage evaluates 3D collision data within a given cage and applies spatial filters, assigning results into buckets.
func (s *Volumes) QueryCollisionCage(cage *CollisionCage) {
	s.tree.QueryOverlaps(cage, func(object physics.IAABB) bool {
		vol := object.(*Volume)
		vol.QueryOverlaps(cage, func(otherEnt physics.IAABB) bool {
			face := otherEnt.(*Face)
			cage.AddFace(face)
			return false
		})
		cage.Commit(nil)
		return false
	})
}

// QueryFrustum performs a spatial query using a frustum, invoking the callback for each intersected object in the tree.
func (s *Volumes) QueryFrustum(frustum *physics.Frustum, callback func(object physics.IAABB) bool) {
	s.tree.QueryFrustum(frustum, callback)
}

// QueryMultiFrustum performs a spatial query using two frustums, invoking the callback for each overlapping object.
func (s *Volumes) QueryMultiFrustum(front, rear *physics.Frustum, callback func(object physics.IAABB) bool) {
	s.tree.QueryMultiFrustum(front, rear, callback)
}

// QueryRay performs a raycasting query starting from origin (oX, oY, oZ) in direction (dirX, dirY, dirZ) up to maxDistance.
// It invokes the callback for each intersected object, passing the object and intersection distance as arguments.
func (s *Volumes) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object physics.IAABB, distance float64) (float64, bool)) {
	s.tree.QueryRay(oX, oY, oZ, dirX, dirY, dirZ, maxDistance, callback)
}

// QueryPoint identifies the 3D location and specific face at the given point (px, py, pz) in world coordinates.
func (s *Volumes) QueryPoint(px, py, pz float64) (*Volume, *Face) {
	var bestVol *Volume
	var bestFace *Face
	var minZDist = math.MaxFloat64
	// Broad-Phase Globale: troviamo i volumi il cui AABB 3D contiene il punto
	s.tree.QueryPoint3d(px, py, pz, func(object physics.IAABB) bool {
		volume := object.(*Volume)
		if bestVol == nil {
			bestVol = volume
		}
		// Broad-Phase Locale
		volume.facesTree.QueryPoint2d(px, py, func(object physics.IAABB) bool {
			face := object.(*Face)
			if bestFace == nil {
				bestFace = face
			}
			normX, normY, normZ := face.GetNormal()
			// Filtro Topologico: Selezioniamo solo i pavimenti (Normal Z negativa)
			if normZ >= -0.001 {
				return false
			}
			// Proiezione 2D
			if face.PointInside2d(px, py) {
				// CALCOLO Z ESATTO SUL TRIANGOLO (Plane Equation)
				// Z = V.z - (Nx*(Px - V.x) + Ny*(Py - V.y)) / Nz
				v0 := face.tri[0]
				floorZ := v0.Z - (normX*(px-v0.X)+normY*(py-v0.Y))/normZ
				// Distanza verticale al pavimento
				zDist := pz - floorZ
				// Se il pavimento è SOTTO il player (o entro un piccolo margine di compenetrazione/step)
				// E se è il più vicino che abbiamo trovato finora
				if zDist >= -0.5 && zDist < minZDist {
					minZDist = zDist
					bestVol = volume
					bestFace = face
				}
				return false
			}
			return false
		})
		// Continuiamo sempre la query globale per coprire il caso di AABB di volumi sovrapposti
		return false
	})

	return bestVol, bestFace
}

// QueryAABB performs a spatial query, invoking the callback for each Volume that overlaps with the specified AABB.
func (s *Volumes) QueryAABB(aabb physics.IAABB, callback func(vol *Volume)) {
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			callback(vol)
		}
		return false
	})
}
