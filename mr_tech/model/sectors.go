package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Sectors represents a collection of spatial sectors with a tree structure for efficient spatial queries.
type Sectors struct {
	container []*Sector
	tree      *physics.AABBTree
	cache     map[string]*Sector
	fullZ     bool
}

// NewSectors initializes and returns a new Sectors instance with a container of Sector objects and an optional fullZ flag.
func NewSectors(container []*Sector, fullZ bool) *Sectors {
	cache := make(map[string]*Sector)
	for _, sec := range container {
		cache[sec.GetId()] = sec
	}
	vs := &Sectors{
		container: container,
		cache:     cache,
		tree:      physics.NewAABBTree(uint(len(container)), 4.0),
		fullZ:     fullZ,
	}
	return vs
}

// Setup initializes and rebuilds sectors in the container, inserting them into the spatial tree if needed.
func (s *Sectors) Setup() {
	for _, sector := range s.container {
		if sector.Rebuild() {
			s.tree.InsertObject(sector)
		}
	}
}

// GetSectors returns the list of sectors contained within the Sectors object.
func (s *Sectors) GetSectors() []*Sector {
	return s.container
}

// QueryPoint performs a 2D point query to find and return the Sector containing the point (px, py), or nil if none is found.
func (s *Sectors) QueryPoint(px, py float64) *Sector {
	var target *Sector = nil
	s.tree.QueryPoint2d(px, py, func(object physics.IAABB) bool {
		sector := object.(*Sector)
		if target == nil {
			target = sector
		}
		if sector.PointInLineSide(px, py) {
			target = sector
			return true
		}
		return false
	})
	return target
}
