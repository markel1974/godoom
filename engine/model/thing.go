package model

import "github.com/markel1974/godoom/engine/textures"

// Thing represents a compiled game entity with its physical position and resolved sector.
type Thing struct {
	Id        string
	Position  XY
	Angle     float64
	Type      int
	Sector    *Sector
	Animation *textures.Animation
}
