package jedi

import "github.com/markel1974/godoom/mr_tech/model/geometry"

// Sector represents a space or room in a level, defined by boundaries, height, and associated properties.
type Sector struct {
	Id             int
	FloorY         float64
	CeilingY       float64
	FloorTexture   int
	CeilingTexture int
	LightLevel     float64
	Vertices       []geometry.XY
	Walls          []*Wall
	Flags          [3]int
}

// NewSector creates and initializes a new Sector instance with the specified ID.
func NewSector(id int) *Sector {
	return &Sector{Id: id}
}

// IsSky determines if the sector is marked as a sky sector based on its flag configuration.
func (s *Sector) IsSky() bool {
	return (s.Flags[0] & 1) != 0
}

// IsAbyss checks whether the sector has the "abyss" property by evaluating specific flag settings.
func (s *Sector) IsAbyss() bool {
	return (s.Flags[0] & 2) != 0
}
