package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Calibration represents parameters used for rendering and camera positioning in a 3D spatial context.
type Calibration struct {
	OrthoSize  float32
	MapCenterX float32
	MapCenterZ float32
	LightCamY  float32
	ZNearRoom  float32
	ZFarRoom   float32
}

// Volumes is a collection of 3D navigable spaces managed within a hierarchical bounding volume tree for spatial queries.
type Volumes struct {
	container []*Volume
	tree      *physics.AABBTree
	cache     map[string]*Volume
}

// NewVolumes creates a new Volumes instance, initializing its container and cache with the given Volume list.
func NewVolumes(container []*Volume) *Volumes {
	cache := make(map[string]*Volume)
	for _, sec := range container {
		cache[sec.GetId()] = sec
	}
	return &Volumes{container: container, tree: nil, cache: cache}
}

func (s *Volumes) CreateTree() {
	s.tree = physics.NewAABBTree(uint(len(s.container)))
	for _, sec := range s.container {
		sec.Rebuild()
		s.tree.InsertObject(sec)
	}
}

// GetVolume retrieves a Volume from the cache using its unique identifier.
// Returns nil if the Volume is not found in the cache.
func (s *Volumes) GetVolume(id string) *Volume {
	return s.cache[id]
}

// GetCalibration computes and returns a Calibration object based on the spatial properties of the AABBTree's root element.
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

// GetVolumes returns the list of all volumes managed by the Volumes instance.
func (s *Volumes) GetVolumes() []*Volume {
	return s.container
}

// Len returns the total number of volumes currently stored in the container.
func (s *Volumes) Len() int {
	return len(s.container)
}

// Query retrieves all volumes from the tree that overlap with the given Axis-Aligned Bounding Box (AABB).
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

// SearchVolume2d searches for a 2D volume containing the point (px, py) starting from the given sector or the volume tree.
func (s *Volumes) SearchVolume2d(sector *Volume, px, py float64) *Volume {
	if newSector := sector.LocatePoint(px, py, 0); newSector != nil {
		return newSector
	}
	if newSector := s.QueryPoint2d(px, py); newSector != nil {
		return newSector
	}
	return nil
}

// QueryOverlap2d checks for overlapping volumes within a 2D plane at the given point and returns the first matching volume.
func (s *Volumes) QueryOverlap2d(aabb physics.IAABB, px, py float64) *Volume {
	var target *Volume = nil
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		volume, ok := object.(*Volume)
		if !ok {
			return false
		}
		if t1 := volume.LocatePoint(px, py, 0); target != t1 {
			target = t1
			return true
		}
		return false
	})
	return target
}

// QueryPoint2d queries the spatial tree to find a 2D volume that contains the specified point (px, py).
func (s *Volumes) QueryPoint2d(px, py float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint(px, py, func(object physics.IAABB) bool {
		if volume, ok := object.(*Volume); ok {
			if volume.ContainsPoint(px, py, 0) {
				target = volume
				return true
			}
		}
		return false
	})
	return target
}

// SearchVolume searches for a volume containing the point (px, py, pz) starting from the currentVolume and returns it.
func (s *Volumes) SearchVolume(currentVolume *Volume, px, py, pz float64) *Volume {
	if newVolume := currentVolume.LocatePoint(px, py, pz); newVolume != nil {
		return newVolume
	}
	if newVolume := s.QueryPoint(px, py, pz); newVolume != nil {
		return newVolume
	}
	return nil
}

// QueryPoint locates and returns the first Volume containing the specified 3D point (px, py, pz), or nil if no match is found.
func (s *Volumes) QueryPoint(px, py, pz float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint(px, py, func(object physics.IAABB) bool {
		if vol, ok := object.(*Volume); ok {
			if vol.ContainsPoint(px, py, pz) {
				target = vol
				return true
			}
		}
		return false
	})
	return target
}

// QueryOverlap checks for overlaps within the given AABB and returns the first volume containing the provided point (px, py, pz).
func (s *Volumes) QueryOverlap(aabb physics.IAABB, px, py, pz float64) *Volume {
	var target *Volume = nil
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		vol, ok := object.(*Volume)
		if !ok {
			return false
		}
		if t1 := vol.LocatePoint(px, py, pz); target != t1 {
			target = t1
			return true
		}
		return false
	})
	return target
}
