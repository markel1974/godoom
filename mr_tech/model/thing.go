package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/mathematic"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Thing represents an object in the environment with physical, positional, and type-specific attributes.
type Thing struct {
	Id        string
	Position  XY
	Mass      float64
	Radius    float64
	Height    float64
	Angle     float64
	Type      int
	Speed     float64
	Sector    *Sector
	Animation *textures.Animation
	sectors   *Sectors
}

// NewThing initializes and returns a new Thing instance based on the provided configuration, animation, and sector data.
func NewThing(ct *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors) *Thing {
	thing := &Thing{
		Id:        ct.Id,
		Position:  ct.Position,
		Angle:     ct.Angle,
		Type:      ct.Kind,
		Mass:      ct.Mass,
		Radius:    ct.Radius,
		Height:    ct.Height,
		Speed:     ct.Speed,
		Sector:    sector,
		Animation: anim,
		sectors:   sectors,
	}
	return thing
}

// MoveApply adjusts the position of the Thing by adding the given delta values dx and dy to its X and Y coordinates.
func (t *Thing) MoveApply(dx float64, dy float64) {
	t.Position.X += dx
	t.Position.Y += dy
}

// ClipMovement restricts the movement of a Thing based on collisions with walls and height differences in its sector.
func (t *Thing) ClipMovement(dx float64, dy float64) (float64, float64) {
	const maxIter = 3

	// Le Thing poggiano sul pavimento. Simuliamo l'altezza testa/ginocchia per i dislivelli
	headPos := t.Sector.FloorY + t.Height
	kneePos := t.Sector.FloorY + 2.0

	for iter := 0; iter < maxIter; iter++ {
		hit := false
		px, py := t.Position.X, t.Position.Y
		p1 := px + dx
		p2 := py + dy

		for _, seg := range t.Sector.Segments {
			start := seg.Start
			end := seg.End

			if mathematic.IntersectLineSegmentsF(px, py, p1, p2, start.X, start.Y, end.X, end.Y) {
				holeLow := 9e9
				holeHigh := -9e9
				if seg.Sector != nil {
					holeLow = mathematic.MaxF(t.Sector.FloorY, seg.Sector.FloorY)
					holeHigh = mathematic.MinF(t.Sector.CeilY, seg.Sector.CeilY)
				}

				// Se il segmento è un muro solido o un gradino troppo alto/basso
				if holeHigh < headPos || holeLow > kneePos {
					xd := end.X - start.X
					yd := end.Y - start.Y
					lenSq := xd*xd + yd*yd

					if lenSq > 0 {
						dot := dx*xd + dy*yd
						dx = (xd * dot) / lenSq
						dy = (yd * dot) / lenSq

						invLen := 1.0 / math.Sqrt(lenSq)
						nx := -yd * invLen
						ny := xd * invLen

						epsilon := 0.005
						dx += nx * epsilon
						dy += ny * epsilon
					}
					hit = true
					break // Vettore deviato, ricalcola contro gli altri muri
				}
			}
		}
		if !hit {
			break
		}
	}
	return dx, dy
}
