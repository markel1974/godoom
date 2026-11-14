package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// Volumes represents a collection of 3D or 2D world, utilizing a hierarchical spatial structure and caching for efficiency.
type Volumes struct {
	container []*volumes.Volume
	tree      *aabb.Tree
	cache     map[string]*volumes.Volume
}

// NewVolumes initializes a Volumes structure with a container of Volume instances and a cache for quick access by ID.
func NewVolumes(container []*volumes.Volume) *Volumes {
	cache := make(map[string]*volumes.Volume)
	for _, sec := range container {
		cache[sec.GetId()] = sec
	}
	const fatMargin = 2.0
	vs := &Volumes{
		container: container,
		cache:     cache,
		tree:      aabb.NewTree(uint(len(container)), fatMargin),
	}
	return vs
}

// Setup constructs a new AABB tree based on the current container and populates it with rebuilt location objects.
func (s *Volumes) Setup() error {
	for _, volume := range s.container {
		if volume.Rebuild() {
			s.tree.InsertObject(volume)
		}
	}
	return nil
}

func (s *Volumes) GetRoot() (*aabb.AABB, bool) {
	return s.tree.GetRoot()
}

// GetVolume retrieves a Volume from the cache using the provided unique identifier.
func (s *Volumes) GetVolume(id string) *volumes.Volume {
	return s.cache[id]
}

// GetVolumes returns all Volume objects managed by the Volumes instance.
func (s *Volumes) GetVolumes() []*volumes.Volume {
	return s.container
}

// Len returns the number of Volume objects contained within the Volumes container.
func (s *Volumes) Len() int {
	return len(s.container)
}

// Query retrieves a list of world that overlap with the specified axis-aligned bounding box (AABB).
func (s *Volumes) Query(targetObj aabb.IObject) []*volumes.Volume {
	var target []*volumes.Volume
	s.tree.QueryOverlaps(targetObj, func(object aabb.IObject) bool {
		sector, ok := object.(*volumes.Volume)
		if !ok {
			return false
		}
		target = append(target, sector)
		return false
	})
	return target
}

// QueryCollisionCage evaluates 3D collision data within a given cage and applies spatial filters, assigning results into buckets.
func (s *Volumes) QueryCollisionCage(cage *collisions.Cage, maxOverlaps int) {
	counter := 0
	s.tree.QueryOverlaps(cage, func(object aabb.IObject) bool {
		vol := object.(*volumes.Volume)
		vol.QueryOverlaps(cage, func(otherEnt aabb.IObject) bool {
			rFace := otherEnt.(*volumes.Face)
			cage.AddFace(rFace, nil)
			counter++
			if counter > maxOverlaps {
				return true
			}
			return false
		})
		cage.Commit(nil)
		if counter > maxOverlaps {
			return true
		}
		return false
	})
}

// ClampStaticCage projects any entity moved or pushed during the step strictly outside solid world faces.
func (s *Volumes) ClampStaticCage(cage *collisions.Cage) {
	const maxPasses = 4
	for pass := 0; pass < maxPasses; pass++ {
		moved := false
		origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ, lAABB := cage.BeginStaticClamp()
		s.tree.QueryOverlaps(cage, func(object aabb.IObject) bool {
			vol := object.(*volumes.Volume)
			vol.QueryOverlaps(cage, func(otherEnt aabb.IObject) bool {
				if cage.ClampStaticFace(lAABB, otherEnt.(*volumes.Face)) {
					moved = true
				}
				return false
			})
			return false
		})
		cage.EndStaticClamp(origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ)
		if !moved {
			break
		}
	}
	cage.CommitStaticClamp()
}

// QueryFrustum performs a spatial query using a frustum, invoking the callback for each intersected object in the tree.
func (s *Volumes) QueryFrustum(frustum *aabb.Frustum, callback func(object aabb.IObject) bool) {
	s.tree.QueryFrustum(frustum, callback)
}

// QueryMultiFrustum performs a spatial query using two frustums, invoking the callback for each overlapping object.
func (s *Volumes) QueryMultiFrustum(front, rear *aabb.Frustum, callback func(object aabb.IObject, isFront bool, isBack bool) bool) {
	s.tree.QueryMultiFrustum(front, rear, callback)
}

// QueryRay performs a raycasting query starting from origin (oX, oY, oZ) in direction (dirX, dirY, dirZ) up to maxDistance.
// It invokes the callback for each intersected object, passing the object and intersection distance as arguments.
func (s *Volumes) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object aabb.IObject, distance float64) (float64, bool)) {
	s.tree.QueryRay(oX, oY, oZ, dirX, dirY, dirZ, maxDistance, callback)
}

// QueryPoint identifies the 3D location and specific face at the given point (px, py, pz) in world coordinates.
func (s *Volumes) QueryPoint(px, py, pz float64) (*volumes.Volume, *volumes.Face) {
	var bestVol *volumes.Volume
	var bestFace *volumes.Face
	var minZDist = math.MaxFloat64
	// Broad-Phase Globale: troviamo i volumi il cui AABB 3D contiene il punto
	s.tree.QueryPoint3d(px, py, pz, func(object aabb.IObject) bool {
		volume := object.(*volumes.Volume)
		if bestVol == nil {
			bestVol = volume
		}
		// Broad-Phase Locale
		volume.GetFacesTree().QueryPoint2d(px, py, func(object aabb.IObject) bool {
			face := object.(*volumes.Face)
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
				v0X, v0Y, v0Z := face.Get0()
				floorZ := v0Z - (normX*(px-v0X)+normY*(py-v0Y))/normZ
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
func (s *Volumes) QueryAABB(targetObj aabb.IObject, callback func(vol *volumes.Volume)) {
	s.tree.QueryOverlaps(targetObj, func(object aabb.IObject) bool {
		if vol, ok := object.(*volumes.Volume); ok {
			callback(vol)
		}
		return false
	})
}
