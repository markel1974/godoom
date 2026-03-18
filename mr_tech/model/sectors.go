package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Sectors represents a collection of Sector instances, organized in an AABBTree for efficient spatial queries.
type Sectors struct {
	sectors []*Sector
	tree    *physics.AABBTree
	cache   map[string]*Sector
}

// NewSectors initializes a new Sectors instance by building an AABB tree and a cache from the given slice of Sector pointers.
func NewSectors(sectors []*Sector) *Sectors {
	cache := make(map[string]*Sector)
	for _, sec := range sectors {
		cache[sec.Id] = sec
	}
	return &Sectors{sectors: sectors, tree: nil, cache: cache}
}

// CreateTree constructs a new AABB tree for spatial organization of sectors within the Sectors instance.
func (s *Sectors) CreateTree() {
	s.tree = physics.NewAABBTree(uint(len(s.sectors)))
	for _, sec := range s.sectors {
		sec.ComputeAABB()
		s.tree.InsertObject(sec)
	}
}

// GetSector retrieves a Sector instance by its unique identifier from the cache map. Returns nil if not found.
func (s *Sectors) GetSector(id string) *Sector {
	return s.cache[id]
}

// GetSectors retrieves the list of all sectors managed within the current Sectors instance.
func (s *Sectors) GetSectors() []*Sector {
	return s.sectors
}

// Len returns the number of sectors contained within the Sectors instance.
func (s *Sectors) Len() int {
	return len(s.sectors)
}

// SectorSearch attempts to locate a sector containing the point (px, py) within or near the provided sector.
func (s *Sectors) SectorSearch(sector *Sector, px, py float64) *Sector {
	if newSector := sector.LocatePoint(px, py); newSector != nil {
		return newSector
	}
	if newSector := s.QueryPoint(px, py); newSector != nil {
		return newSector
	}
	//fmt.Println("SectorSearch: No sector found for point (", px, ",", py, ")")
	return nil
}

// QueryOverlap performs an AABB overlap query to locate a Sector containing the point (px, py) or returns nil if not found.
func (s *Sectors) QueryOverlap(aabb physics.IAABB, px, py float64) *Sector {
	candidates := s.tree.QueryOverlaps(aabb)
	for _, c := range candidates {
		sector, ok := c.(*Sector)
		if !ok {
			continue
		}
		if target := sector.LocatePoint(px, py); target != nil {
			return target
		}
	}
	return nil
}

// QueryPoint performs a spatial query to determine if a point (px, py) lies within any sector and returns the matching Sector.
func (s *Sectors) QueryPoint(px, py float64) *Sector {
	candidates := s.tree.QueryPoint(px, py)
	for _, c := range candidates {
		if sector, ok := c.(*Sector); ok {
			if sector.ContainsPoint(px, py) {
				return sector
			}
		}
	}
	return nil
}

// MakeSegmentsCache builds a map of edgeKey to segment, representing all unique segments in the sectors.
func (s *Sectors) MakeSegmentsCache() map[EdgeKey]*EdgeSegment {
	t := make(map[EdgeKey]*EdgeSegment)
	for _, sect := range s.sectors {
		for np := 0; np < len(sect.Segments); np++ {
			seg := sect.Segments[np]
			hash := seg.MakeStraightEdgeKey()
			ld := &EdgeSegment{sector: sect, np: np, start: seg.Start, end: seg.End}
			if fld, ok := t[hash]; ok {
				if sect.Id != fld.sector.Id {
					//fmt.Println("line segment already added", sect.Id, fld.Sector.Id, hash, np)
				}
			} else {
				t[hash] = ld
			}
		}
	}
	return t
}
