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

// Sectors is a collection of Sector instances, organized with spatial indexing and caching for optimized queries.
type Sectors struct {
	sectors []*Sector
	tree    *physics.AABBTree
	cache   map[string]*Sector
}

// NewSectors initializes a Sectors structure with a cache mapping sector IDs to their respective Sector objects.
func NewSectors(sectors []*Sector) *Sectors {
	cache := make(map[string]*Sector)
	for _, sec := range sectors {
		cache[sec.GetId()] = sec
	}
	return &Sectors{sectors: sectors, tree: nil, cache: cache}
}

// CreateTree constructs a new AABBTree and populates it with sectors after computing their axis-aligned bounding boxes.
func (s *Sectors) CreateTree() {
	s.tree = physics.NewAABBTree(uint(len(s.sectors)))
	for _, sec := range s.sectors {
		sec.Rebuild()
		s.tree.InsertObject(sec)
	}
}

// GetSector retrieves a Sector from the cache using the given id. Returns nil if the id is not found.
func (s *Sectors) GetSector(id string) *Sector {
	return s.cache[id]
}

// GetCalibration computes and returns a calibration object based on the spatial properties of the sector tree's root node.
func (s *Sectors) GetCalibration() *Calibration {
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

// GetSectors returns the list of sectors managed by the Sectors instance.
func (s *Sectors) GetSectors() []*Sector {
	return s.sectors
}

// Len returns the number of sectors in the Sectors collection.
func (s *Sectors) Len() int {
	return len(s.sectors)
}

// SectorSearch searches for a sector containing the point (px, py), starting from the given sector and querying the tree if needed.
// It returns the sector containing the point or nil if no matching sector is found.
func (s *Sectors) SectorSearch(sector *Sector, px, py float64) *Sector {
	//TODO missing z
	if newSector := sector.LocatePoint(px, py, 0); newSector != nil {
		return newSector
	}
	if newSector := s.QueryPoint(px, py); newSector != nil {
		return newSector
	}
	//fmt.Println("SectorSearch: No sector found for point (", px, ",", py, ")")
	return nil
}

// Query retrieves all sectors that overlap with the given Axis-Aligned Bounding Box (AABB).
func (s *Sectors) Query(aabb physics.IAABB) []*Sector {
	var target []*Sector
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		sector, ok := object.(*Sector)
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
func (s *Sectors) QueryOverlap(aabb physics.IAABB, px, py float64) *Sector {
	var target *Sector = nil
	s.tree.QueryOverlaps(aabb, func(object physics.IAABB) bool {
		sector, ok := object.(*Sector)
		if !ok {
			return false
		}
		//todo missing z
		if t1 := sector.LocatePoint(px, py, 0); target != t1 {
			target = t1
			return true
		}
		return false
	})
	return target
}

// QueryPoint searches for the sector containing the specified point (px, py) and returns it, or nil if not found.
func (s *Sectors) QueryPoint(px, py float64) *Sector {
	var target *Sector = nil
	s.tree.QueryPoint(px, py, func(object physics.IAABB) bool {
		if sector, ok := object.(*Sector); ok {
			//todo missing z
			if sector.ContainsPoint(px, py, 0) {
				target = sector
				return true
			}
		}
		return false
	})
	return target
}
