package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/mathematic"
	"github.com/markel1974/godoom/mr_tech/textures"
)

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

// MoveApply updates the position of the Thing by applying movement deltas, adjusting its sector if necessary.
func (t *Thing) MoveApply(tDx float64, tDy float64) {
	dx, dy := t.ClipMovement(tDx, tDy)
	t.Position.X += dx
	t.Position.Y += dy
	if newSector := t.Sector.LocateSector(t.Position.X, t.Position.Y); newSector != nil {
		t.Sector = newSector
	}
}

// ClipMovement adjusts the movement vector (dx, dy) to prevent collisions with walls or impassable terrain boundaries.
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
