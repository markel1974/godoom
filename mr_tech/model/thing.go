package model

import "github.com/markel1974/godoom/mr_tech/textures"

// Thing represents a compiled game entity with its physical position and resolved sector.
type Thing struct {
	Id        string
	Position  XY
	Mass      float64
	Radius    float64
	Height    float64
	Angle     float64
	Type      int
	Sector    *Sector
	Animation *textures.Animation
}
