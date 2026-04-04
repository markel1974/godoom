package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Calibration represents the parameters used for setting up rendering configurations in a 3D engine.
type Calibration struct {
	OrthoSize  float32
	MapCenterX float32
	MapCenterZ float32
	LightCamY  float32
	ZNearRoom  float32
	ZFarRoom   float32
}

// Volumes is a collection of Sector instances, organized with spatial indexing and caching for optimized queries.
type Volumes struct {
	container []*Volume
	tree      *physics.AABBTree
	cache     map[string]*Volume
}

// NewVolumes initializes a Volumes structure with a cache mapping sector IDs to their respective Sector objects.
func NewVolumes(container []*Volume) *Volumes {
	cache := make(map[string]*Volume)
	for _, sec := range container {
		cache[sec.GetId()] = sec
	}
	return &Volumes{container: container, tree: nil, cache: cache}
}

// CreateTree constructs a new AABBTree and populates it with sectors after computing their axis-aligned bounding boxes.
func (s *Volumes) CreateTree() {
	s.tree = physics.NewAABBTree(uint(len(s.container)))
	for _, sec := range s.container {
		sec.Rebuild()
		s.tree.InsertObject(sec)
	}
}

// GetVolume retrieves a Sector from the cache using the given id. Returns nil if the id is not found.
func (s *Volumes) GetVolume(id string) *Volume {
	return s.cache[id]
}

// GetCalibration computes and returns a calibration object based on the spatial properties of the sector tree's root node.
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

// GetVolumes returns the list of sectors managed by the Volumes instance.
func (s *Volumes) GetVolumes() []*Volume {
	return s.container
}

// Len returns the number of sectors in the Volumes collection.
func (s *Volumes) Len() int {
	return len(s.container)
}

// SearchVolume searches for a sector containing the point (px, py), starting from the given sector and querying the tree if needed.
// It returns the sector containing the point or nil if no matching sector is found.
func (s *Volumes) SearchVolume(sector *Volume, px, py float64) *Volume {
	//TODO missing z
	if newSector := sector.LocatePoint(px, py, 0); newSector != nil {
		return newSector
	}
	if newSector := s.QueryPoint(px, py); newSector != nil {
		return newSector
	}
	//fmt.Println("SearchVolume: No sector found for point (", px, ",", py, ")")
	return nil
}

// Query retrieves all sectors that overlap with the given Axis-Aligned Bounding Box (AABB).
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

// QueryOverlap identifies a Sector containing a given point (px, py) within an AABB, if such a Sector exists.
// It searches the AABB tree for overlaps and attempts to locate the point within the overlapping sectors.
func (s *Volumes) QueryOverlap(aabb physics.IAABB, px, py float64) *Volume {
	var target *Volume = nil
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		volume, ok := object.(*Volume)
		if !ok {
			return false
		}
		//todo missing z
		if t1 := volume.LocatePoint(px, py, 0); target != t1 {
			target = t1
			return true
		}
		return false
	})
	return target
}

// QueryPoint searches for the sector containing the specified point (px, py) and returns it, or nil if not found.
func (s *Volumes) QueryPoint(px, py float64) *Volume {
	var target *Volume = nil
	s.tree.QueryPoint(px, py, func(object physics.IAABB) bool {
		if volume, ok := object.(*Volume); ok {
			//todo missing z
			if volume.ContainsPoint(px, py, 0) {
				target = volume
				return true
			}
		}
		return false
	})
	return target
}
