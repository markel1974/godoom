package streaming

import (
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// SectorState represents the lifecycle status of a streaming world cell.
type SectorState int

const (
	// SectorStateUnloaded indicates the sector is not present in the active spatial tree.
	SectorStateUnloaded SectorState = iota
	// SectorStateActive indicates all detailed volumes and collision meshes are live in the spatial tree.
	SectorStateActive
	// SectorStateDistant indicates only the simplified skyline/proxy mesh is live for rendering.
	SectorStateDistant
)

// Sector represents one discrete spatial cell of the open-world map.
type Sector struct {
	Coord SectorCoord
	State SectorState

	// Detailed volumes for high-detail geometry, physics collision, and full rendering
	Volumes *volumes.Volumes

	// Optional low-poly proxy volume used for distant skyline rendering
	ProxyVolume *volumes.Volume

	// Combined AABB bounding the entire sector's geometry
	bounds *aabb.AABB
}

// NewSector creates an empty sector at the specified grid coordinate.
func NewSector(coord SectorCoord, cellSize float64) *Sector {
	minX, minY, maxX, maxY := coord.SectorToWorldBounds(cellSize)
	box := aabb.NewAABB()
	box.Rebuild(minX, minY, -1000.0, maxX, maxY, 2000.0)

	return &Sector{
		Coord:   coord,
		State:   SectorStateUnloaded,
		Volumes: volumes.NewVolumes(0),
		bounds:  box,
	}
}

// AddVolume attaches a detailed geometry volume to this sector and expands the sector's bounding box.
func (s *Sector) AddVolume(vol *volumes.Volume) {
	if vol == nil {
		return
	}
	s.Volumes.Add(vol)

	volBox := vol.GetAABB()
	if volBox != nil {
		s.bounds.MergeSelf(volBox)
	}
}

// SetProxyVolume assigns a simplified low-poly mesh volume for distant skyline visibility.
func (s *Sector) SetProxyVolume(proxy *volumes.Volume) {
	s.ProxyVolume = proxy
}

// GetAABB satisfies the aabb.IObject interface.
func (s *Sector) GetAABB() *aabb.AABB {
	return s.bounds
}

// VolumesLen returns the count of detailed volumes in this sector.
func (s *Sector) VolumesLen() int {
	if s.Volumes == nil {
		return 0
	}
	return s.Volumes.Len()
}
