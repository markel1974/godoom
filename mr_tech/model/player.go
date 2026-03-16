package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/mathematic"
)

// EyeHeight represents the height of the eye level in units.
// DuckHeight represents the height in a ducking position in units.
// HeadMargin represents the margin or buffer space for the head in units.
// KneeHeight represents the height of the knee level in units.
const (
	EyeHeight  = 6
	DuckHeight = 2.5
	HeadMargin = 2
	KneeHeight = 2
)

// Player represents a player entity with position, velocity, angle, yaw, sector, states, and lighting attributes.
type Player struct {
	where          XYZ
	velocity       XYZ
	angle          float64
	angleSin       float64
	angleCos       float64
	yaw            float64
	yawState       float64
	radius         float64
	mass           float64
	sector         *Sector
	ducking        bool
	falling        bool
	lightIntensity float64
	debug          bool
}

// NewPlayer creates a new Player instance with initial position, angle, sector, and debug configuration.
func NewPlayer(cfg *ConfigPlayer, sector *Sector, debug bool) *Player {
	p := &Player{
		where:          XYZ{X: cfg.Position.X, Y: cfg.Position.Y, Z: sector.FloorY + EyeHeight},
		velocity:       XYZ{},
		yaw:            0,
		yawState:       0,
		radius:         cfg.Radius,
		mass:           cfg.Mass,
		sector:         sector,
		lightIntensity: 0.0039, // 1 / distance == 1 / 255
		debug:          debug,
	}
	p.SetAngle(cfg.Angle)
	return p
}

// AddAngle increments the player's current angle by the specified value and updates related trigonometric properties.
func (p *Player) AddAngle(angle float64) {
	p.SetAngle(p.angle + angle)
}

// SetAngle sets the player's viewing angle in radians, recalculating the sine and cosine of the angle for movement.
func (p *Player) SetAngle(angle float64) {
	p.angle = angle
	p.angleSin = math.Sin(p.angle)
	p.angleCos = math.Cos(p.angle)
}

// GetAngle returns the sine and cosine of the player's current angle as float64 values.
func (p *Player) GetAngle() (float64, float64) {
	return p.angleSin, p.angleCos
}

// SetYaw adjusts the player's yaw (vertical rotation) by modifying yawState and accounting for velocity along the Z-axis.
func (p *Player) SetYaw(y float64) {
	p.yawState = mathematic.ClampF(p.yawState-(y*0.05), -5, 5)
	p.yaw = p.yawState - (p.velocity.Z * 0.5)
}

// Move updates the player's velocity based on the given impulse and directional input flags (up, down, left, right).
func (p *Player) Move(impulse float64, up bool, down bool, left bool, right bool) {
	var moveX float64
	var moveY float64
	var acceleration float64
	//impulse := impulseNormal
	//if slow {
	//	impulse = impulseSlow
	//}
	if up || down || left || right {
		if up {
			moveX += p.angleCos * impulse
			moveY += p.angleSin * impulse
		}
		if down {
			moveX -= p.angleCos * impulse
			moveY -= p.angleSin * impulse
		}
		if left {
			moveX += p.angleSin * impulse
			moveY -= p.angleCos * impulse
		}
		if right {
			moveX -= p.angleSin * impulse
			moveY += p.angleCos * impulse
		}
		acceleration = 0.4
	} else {
		acceleration = 0.2
	}
	p.velocity.X = p.velocity.X*(1-acceleration) + (moveX * acceleration)
	p.velocity.Y = p.velocity.Y*(1-acceleration) + (moveY * acceleration)
}

// SetJump increases the player's Z velocity to simulate a jump and marks the player as falling.
func (p *Player) SetJump() {
	p.velocity.Z += 0.5
	p.falling = true
}

// SetDucking toggles the player's ducking state and sets falling to true if the player is ducking.
func (p *Player) SetDucking() {
	p.ducking = !p.ducking
	if p.ducking {
		p.falling = true
	}
}

// GetXY returns the X and Y coordinates of the player's current position.
func (p *Player) GetXY() (float64, float64) {
	return p.where.X, p.where.Y
}

// GetXYZ retrieves the player's current X, Y, and Z coordinates in the game world.
func (p *Player) GetXYZ() (float64, float64, float64) {
	return p.where.X, p.where.Y, p.where.Z
}

// SetXY updates the player's X and Y coordinates and sets the falling state to true.
func (p *Player) SetXY(x float64, y float64) {
	p.where.X = x
	p.where.Y = y
	p.falling = true
}

// AddXY applies incremental adjustments to the player's X and Y coordinates and sets the falling state to true.
func (p *Player) AddXY(x float64, y float64) {
	p.where.X += x
	p.where.Y += y
	p.falling = true
}

// GetZ retrieves the Z-coordinate (vertical position) of the player.
func (p *Player) GetZ() float64 {
	return p.where.Z
}

// GetLightIntensity returns the current light intensity value associated with the Player instance.
func (p *Player) GetLightIntensity() float64 {
	return p.lightIntensity
}

// SetLightIntensity sets the light intensity for the player by updating the lightIntensity property.
func (p *Player) SetLightIntensity(lightIntensity float64) {
	p.lightIntensity = lightIntensity
}

// GetRadius returns the radius of the player as a float64 value.
func (p *Player) GetRadius() float64 {
	return p.radius
}

// GetMass returns the mass of the player as a float64 value.
func (p *Player) GetMass() float64 {
	return p.mass
}

// GetVelocity returns the X and Y components of the player's velocity as two float64 values.
func (p *Player) GetVelocity() (float64, float64) {
	return p.velocity.X, p.velocity.Y
}

// GetSector returns the current sector the player is located in.
func (p *Player) GetSector() *Sector {
	return p.sector
}

// SetSector updates the Player's current sector to the specified Sector instance.
func (p *Player) SetSector(sector *Sector) {
	p.sector = sector
}

// GetYaw returns the current yaw value of the player.
func (p *Player) GetYaw() float64 {
	return p.yaw
}

// EyeHeight returns the height of the player's eyes, considering whether the player is ducking or standing upright.
func (p *Player) EyeHeight() float64 {
	if p.ducking {
		return DuckHeight
	}
	return EyeHeight
}

// VerticalCollision checks and resolves vertical collisions for the player, adjusting position and velocity based on sector bounds.
func (p *Player) VerticalCollision() {
	if p.falling {
		eyeHeight := p.EyeHeight()
		p.velocity.Z -= 0.05
		nextZ := p.where.Z + p.velocity.Z
		if p.velocity.Z < 0 && nextZ < p.sector.FloorY+eyeHeight {
			// down
			p.where.Z = p.sector.FloorY + eyeHeight
			p.velocity.Z = 0
			p.falling = false
		} else if p.velocity.Z > 0 && nextZ > p.sector.CeilY {
			// up
			p.velocity.Z = 0
			p.falling = true
		}
		if p.falling {
			p.where.Z += p.velocity.Z
		}
	}
}

// IsMoving determines whether the player is currently in motion based on its velocity in the X and Y axes.
func (p *Player) IsMoving() bool {
	return !(p.velocity.X == 0 && p.velocity.Y == 0)
}

// GetHeadPosition returns the Z-coordinate of the player's head, calculated as the player's current Z-position plus HeadMargin.
func (p *Player) GetHeadPosition() float64 {
	return p.where.Z + HeadMargin
}

// GetKneePosition calculates and returns the player's knee position based on their current Z-coordinate and eye height.
func (p *Player) GetKneePosition() float64 {
	return p.where.Z - p.EyeHeight() + KneeHeight
}

// MoveApply updates the player's position based on the given displacement and handles sector transitions when necessary.
func (p *Player) MoveApply(dx float64, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}

	// 1. Apply the atomic vector and obtain the final coordinates
	p.AddXY(dx, dy)
	px, py := p.GetXY()

	currentSector := p.GetSector()

	// 2. Spatial stability check: are we still inside the same sector?
	if PointInSegments(px, py, currentSector.Segments) {
		return
	}

	// 3. Portal Transition: The point has physically exited the root sector.
	// Navigate the topological graph by testing adjacent sectors (neighbors).
	for _, seg := range currentSector.Segments {
		neighbor := seg.Sector
		if neighbor != nil {
			if PointInSegments(px, py, neighbor.Segments) {
				p.SetSector(neighbor)
				if p.debug {
					fmt.Println("New Sector crossed via PIP:", neighbor.Id)
				}
				return
			}
		}
	}

	// 4. Architectural Fallback (Edge Case Perimeter)
	// If the point falls EXACTLY on the mathematical edge of a portal,
	// the FP precision of ray-casting might return 'false' for both sectors.
	// In this case, we force the update if the point is in the back half-plane of the portal.
	for _, seg := range currentSector.Segments {
		if seg.Sector != nil {
			if mathematic.PointSideF(px, py, seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y) < 0 {
				p.SetSector(seg.Sector)
				if p.debug {
					fmt.Println("New Sector crossed via Fallback:", seg.Sector.Id)
				}
				return
			}
		}
	}
}

// Compute updates the player's position and velocity based on collision detection and sector constraints.
func (p *Player) Compute(vi *ViewMatrix) {
	const maxIter = 3

	p.VerticalCollision()
	if !p.IsMoving() {
		return
	}

	headPos := p.GetHeadPosition()
	kneePos := p.GetKneePosition()
	dx, dy := p.GetVelocity()

	// Micro-loop per predizione collisioni multiple
	for iter := 0; iter < maxIter; iter++ {
		hit := false
		pSector := p.GetSector()
		px, py := vi.GetXY()
		p1 := px + dx
		p2 := py + dy

		for _, seg := range pSector.Segments {
			start := seg.Start
			end := seg.End

			if mathematic.IntersectLineSegmentsF(px, py, p1, p2, start.X, start.Y, end.X, end.Y) {
				holeLow := 9e9
				holeHigh := -9e9
				if seg.Sector != nil {
					holeLow = mathematic.MaxF(pSector.FloorY, seg.Sector.FloorY)
					holeHigh = mathematic.MinF(pSector.CeilY, seg.Sector.CeilY)
				}

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
					break // Vettore modificato, ri-valuta contro la geometria
				}
			}
		}
		if !hit {
			break // Traiettoria stabilizzata
		}
	}

	if math.Abs(p.velocity.X) < 0.001 {
		p.velocity.X = 0
	}
	if math.Abs(p.velocity.Y) < 0.001 {
		p.velocity.Y = 0
	}

	p.MoveApply(dx, dy)
}
