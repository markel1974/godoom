package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Calibration represents calibration parameters for rendering and spatial configuration.
type Calibration struct {
	OrthoSize  float32
	MapCenterX float32
	MapCenterZ float32
	LightCamY  float32
	ZNearRoom  float32
	ZFarRoom   float32
}

// Volumes represents a collection of 3D or 2D volumes, utilizing a hierarchical spatial structure and caching for efficiency.
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
	return &Volumes{container: container, tree: nil, cache: cache}
}

// CreateTree constructs a new AABB tree based on the current container and populates it with rebuilt volume objects.
func (s *Volumes) CreateTree() {
	s.tree = physics.NewAABBTree(uint(len(s.container)))
	for _, sec := range s.container {
		sec.Rebuild()
		s.tree.InsertObject(sec)
	}
}

// GetVolume retrieves a Volume from the cache using the provided unique identifier.
func (s *Volumes) GetVolume(id string) *Volume {
	return s.cache[id]
}

// GetCalibration computes calibration details based on the spatial properties of the root node in the AABB tree.
func (s *Volumes) GetCalibration() *Calibration {
	root, ok := s.tree.GetRoot()
	if !ok {
		return nil
	}
	c := &Calibration{}
	// 2. OrthoSize è esattamente la metà dell'asse maggiore
	width := root.GetWidth()
	depth := root.GetDepth()
	if width > depth {
		c.OrthoSize = float32(width / 2.0)
	} else {
		c.OrthoSize = float32(depth / 2.0)
	}
	c.MapCenterX = float32(root.GetMinX() + (width / 2.0))
	c.MapCenterZ = float32(root.GetMinZ() + (depth / 2.0))
	// La telecamera si posiziona appena sopra il punto più alto della mappa
	c.LightCamY = float32(root.GetMaxY()) //+ 2.0
	// Distanze di proiezione relative dalla telecamera
	c.ZNearRoom = 1.0
	c.ZFarRoom = float32(root.GetMaxY() - root.GetMinY())
	return c
}

// GetVolumes returns all Volume objects managed by the Volumes instance.
func (s *Volumes) GetVolumes() []*Volume {
	return s.container
}

// Len returns the number of Volume objects contained within the Volumes container.
func (s *Volumes) Len() int {
	return len(s.container)
}

// Query retrieves a list of volumes that overlap with the specified axis-aligned bounding box (AABB).
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

// SearchVolume2d searches for a 2D volume in the provided sector or the tree using the given x and y coordinates.
func (s *Volumes) SearchVolume2d(sector *Volume, px, py float64) *Volume {
	if newSector := sector.LocatePoint2d(px, py); newSector != nil {
		return newSector
	}
	if newSector := s.QueryPoint2d(px, py); newSector != nil {
		return newSector
	}
	return nil
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

// QueryOverlap2d checks for overlaps with the given 2D AABB and identifies which volume contains the specified point.
func (s *Volumes) QueryOverlap2d(aabb physics.IAABB, px, py float64) *Volume {
	var target *Volume = nil
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		volume, ok := object.(*Volume)
		if !ok {
			return false
		}
		if t1 := volume.LocatePoint2d(px, py); target != t1 {
			target = t1
			return true
		}
		return false
	})
	return target
}

// QueryPoint2d performs a 2D query to find a Volume containing the point (px, py).
// Returns the first matching Volume or nil if no match is found.
func (s *Volumes) QueryPoint2d(px, py float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint(px, py, func(object physics.IAABB) bool {
		if volume, ok := object.(*Volume); ok {
			if volume.ContainsPoint2d(px, py) {
				target = volume
				return true
			}
		}
		return false
	})
	return target
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

// SearchVolume3d searches for the Volume containing the point (px, py, pz) starting from the given currentVolume.
// Returns the located Volume if found, otherwise nil.
func (s *Volumes) SearchVolume3d(currentVolume *Volume, px, py, baseZ, topZ, maxStep float64) *Volume {
	if newVolume := currentVolume.LocatePoint3d(px, py, baseZ, topZ, maxStep); newVolume != nil {
		return newVolume
	}
	if newVolume := s.QueryPoint3d(px, py, baseZ, topZ, maxStep); newVolume != nil {
		return newVolume
	}
	return nil
}

// QueryPoint3d searches for a volume containing the specified 3D point (px, py, pz) and returns the matched Volume, or nil if not found.
func (s *Volumes) QueryPoint3d(px, py, baseZ, topZ, maxStep float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint(px, py, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			if vol.ContainsPoint2d(px, py) {
				if vol.IsValidZ(baseZ, topZ, maxStep) {
					target = vol
					return true
				}
			}
		}
		return false
	})
	return target
}

// QueryOverlap3d performs a spatial query to find the first volume overlapping the given AABB and containing the specified point.
func (s *Volumes) QueryOverlap3d(aabb physics.IAABB, px, py, baseZ, topZ, maxStep float64) *Volume {
	var target *Volume = nil

	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			// Sfruttiamo il nuovo LocatePoint3d per il test di intrusione esatto
			if t1 := vol.LocatePoint3d(px, py, baseZ, topZ, maxStep); t1 != nil {
				target = t1
				return true // Volume valido trovato, interrompe la ricerca
			}
		}
		return false
	})

	return target
}
