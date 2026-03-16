package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/mathematic"
)

// ConfigThing represents a game entity with physical properties, animation, and positional data within a specific sector.
type ConfigThing struct {
	Id        string
	Position  XY
	Angle     float64
	Mass      float64
	Radius    float64
	Height    float64
	Kind      int
	Sector    string
	Animation *ConfigAnimation
}

// NewConfigThing creates and initializes a new ConfigThing with the given properties representing an object in the game world.
// id is the unique identifier for the thing.
// pos specifies the position of the thing in 2D space.
// angle represents the orientation of the thing in degrees.
// kind is an integer representing the type or category of the thing.
// sector assigns the thing to a specific sector in the level layout.
// mass defines the weight of the thing, used in physics calculations.
// radius and height define the dimensions of the thing for collision and spatial representation.
// anim is the animation configuration associated with the thing, such as frames, kind, and scale.
func NewConfigThing(id string, pos XY, angle float64, kind int, sector string, mass, radius, height float64, anim *ConfigAnimation) *ConfigThing {
	return &ConfigThing{
		Id:        id,
		Position:  pos,
		Angle:     angle,
		Kind:      kind,
		Sector:    sector,
		Mass:      mass,
		Radius:    radius,
		Height:    height,
		Animation: anim,
	}
}

// MoveApply updates the position of the Thing by applying movement deltas, adjusting its sector if necessary.
func (t *Thing) MoveApply(dx float64, dy float64) {
	t.Position.X += dx
	t.Position.Y += dy

	if PointInSegments(t.Position.X, t.Position.Y, t.Sector.Segments) {
		return
	}

	for _, seg := range t.Sector.Segments {
		neighbor := seg.Sector
		if neighbor != nil {
			if PointInSegments(t.Position.X, t.Position.Y, neighbor.Segments) {
				t.Sector = neighbor
				return
			}
		}
	}

	for _, seg := range t.Sector.Segments {
		if seg.Sector != nil {
			if mathematic.PointSideF(t.Position.X, t.Position.Y, seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y) < 0 {
				t.Sector = seg.Sector
				return
			}
		}
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
