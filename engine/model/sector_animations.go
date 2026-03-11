package model

import "github.com/markel1974/godoom/engine/textures"

// SectorAnimations represents the animation properties for a sector, including floor and ceiling animations and scaling.
type SectorAnimations struct {
	floor       *textures.Animation
	ceil        *textures.Animation
	scaleFactor float64
}

// NewSectorAnimations initializes and returns a new SectorAnimations instance with the given floor, ceiling, and scaleFactor.
func NewSectorAnimations(floor *textures.Animation, ceil *textures.Animation, scaleFactor float64) *SectorAnimations {
	return &SectorAnimations{
		floor:       floor,
		ceil:        ceil,
		scaleFactor: scaleFactor,
	}
}

// Clone creates and returns a deep copy of the SectorAnimations object, duplicating its floor, ceil, and scaleFactor fields.
func (cst *SectorAnimations) Clone() *SectorAnimations {
	out := &SectorAnimations{}
	out.floor = cst.floor
	out.ceil = cst.ceil
	out.scaleFactor = cst.scaleFactor
	return out
}

// Floor returns the animation associated with the floor texture for the sector.
func (cst *SectorAnimations) Floor() *textures.Animation {
	return cst.floor
}

// Ceil retrieves the animation associated with the ceiling of the sector.
func (cst *SectorAnimations) Ceil() *textures.Animation {
	return cst.ceil
}

// ScaleFactor returns the scaling factor applied to the sector's animations.
func (cst *SectorAnimations) ScaleFactor() float64 {
	return cst.scaleFactor
}
