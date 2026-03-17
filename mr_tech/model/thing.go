package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Thing represents a physical or logical entity in the environment with attributes like position, mass, and associated data.
type Thing struct {
	id        string
	position  XY
	mass      float64
	radius    float64
	height    float64
	angle     float64
	kind      int
	speed     float64
	sector    *Sector
	animation *textures.Animation
	sectors   *Sectors
	entities  *Entities
	entity    *physics.Entity
}

// NewThing creates and initializes a new Thing instance with the provided configuration, animation, sectors, and entities.
func NewThing(cfg *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *Thing {
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	x := cfg.Position.X - cfg.Radius
	y := cfg.Position.Y - cfg.Radius
	thing := &Thing{
		id:        cfg.Id,
		position:  cfg.Position,
		angle:     cfg.Angle,
		kind:      cfg.Kind,
		mass:      cfg.Mass,
		radius:    cfg.Radius,
		height:    cfg.Height,
		speed:     cfg.Speed,
		sector:    sector,
		animation: anim,
		sectors:   sectors,
		entities:  entities,
		entity:    physics.NewEntity(x, y, w, h, cfg.Mass),
	}
	thing.entities.AddEntity(thing.id, thing.entity)
	return thing
}

// GetId returns the unique identifier of the Thing as a string.
func (t *Thing) GetId() string {
	return t.id
}

// GetAnimation retrieves the animation associated with the Thing and returns it as a pointer to textures.Animation.
func (t *Thing) GetAnimation() *textures.Animation {
	return t.animation
}

// GetPosition retrieves the current position of the Thing as an XY value.
func (t *Thing) GetPosition() XY {
	return t.position
}

// GetLight retrieves the light source associated with the Thing's current sector and returns it as a pointer to Light.
func (t *Thing) GetLight() *Light {
	return t.sector.Light
}

// GetFloorY returns the Y-coordinate of the floor in the Thing's current sector as a float64.
func (t *Thing) GetFloorY() float64 {
	return t.sector.FloorY
}

// GetCeilY retrieves the ceiling height of the sector associated with the Thing and returns it as a float64.
func (t *Thing) GetCeilY() float64 {
	return t.sector.CeilY
}

// Move updates the Thing's direction and position based on the player's coordinates and its current speed.
func (t *Thing) Move(playerX float64, playerY float64) {
	if t.speed == 0 {
		return
	}
	dx := playerX - t.position.X
	dy := playerY - t.position.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 25.0 {
		invDist := 1.0 / dist
		dirX := dx * invDist * t.speed
		dirY := dy * invDist * t.speed
		t.modifyDirection(dirX, dirY)
	}
}

// MoveEntityApply processes the logical and physical movement of the entity, adjusting positions and sector affiliations.
func (t *Thing) MoveEntityApply() {
	tPx := t.entity.GetCenterX()
	tPy := t.entity.GetCenterY()

	// 1. Delta Passivo (rimbalzi calcolati da SetupCollision)
	tDx := tPx - t.position.X
	tDy := tPy - t.position.Y

	// 2. Delta Attivo (Kinematic Drive) aggiunto solo se c'è intenzionalità
	if t.entity.G > 0 {
		tDx += t.entity.Vx
		tDy += t.entity.Vy
	}

	if math.Abs(tDx) > 0.001 || math.Abs(tDy) > 0.001 {
		// 3. Traslazione del modello logico
		x, y := t.clipMovement(tDx, tDy)
		t.position.X += x
		t.position.Y += y
		if newSector := t.sectors.SectorSearch(t.sector, t.position.X, t.position.Y); newSector != nil {
			t.sector = newSector
		}

		// 4. Retro-Correzione (Sync-Back) AABB fisico
		r := t.entity.GetWidth() / 2.0
		t.entity.MoveTo(t.position.X-r, t.position.Y-r)
		t.entities.UpdateObject(t.entity)
	}
}

// clipMovement adjusts movement vectors to handle collisions with environment walls or obstacles in a 2D space.
// It takes initial deltas in X and Y directions (dx, dy) and returns the adjusted movement vector after collision checks.
func (t *Thing) clipMovement(dx float64, dy float64) (float64, float64) {
	const maxIter = 3

	// Le Thing poggiano sul pavimento. Simuliamo l'altezza testa/ginocchia per i dislivelli
	headPos := t.sector.FloorY + t.height
	kneePos := t.sector.FloorY + 2.0

	for iter := 0; iter < maxIter; iter++ {
		hit := false
		px, py := t.position.X, t.position.Y
		p1 := px + dx
		p2 := py + dy

		for _, seg := range t.sector.Segments {
			start := seg.Start
			end := seg.End

			if mathematic.IntersectLineSegmentsF(px, py, p1, p2, start.X, start.Y, end.X, end.Y) {
				holeLow := 9e9
				holeHigh := -9e9
				if seg.Sector != nil {
					holeLow = mathematic.MaxF(t.sector.FloorY, seg.Sector.FloorY)
					holeHigh = mathematic.MinF(t.sector.CeilY, seg.Sector.CeilY)
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

// modifyDirection adjusts the velocity of the entity towards the specified direction with a constant acceleration factor.
func (t *Thing) modifyDirection(dirX, dirY float64) {
	const acceleration = 0.15
	t.entity.Vx = t.entity.Vx*(1-acceleration) + (dirX * acceleration)
	t.entity.Vy = t.entity.Vy*(1-acceleration) + (dirY * acceleration)
	if t.entity.GForce == 0 {
		t.entity.GForce = 1.0
	}
	if t.entity.Friction < 0.2 {
		t.entity.Friction = 0.99
	}
}
