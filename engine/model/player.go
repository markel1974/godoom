package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/engine/mathematic"
)

// EyeHeight represents the height of the eye level in a given context.
// DuckHeight represents the height of a ducking position in a given context.
// HeadMargin specifies the additional margin to account for the head in calculations.
// KneeHeight represents the height of the knee level in a given context.
const (
	EyeHeight  = 6
	DuckHeight = 2.5
	HeadMargin = 2
	KneeHeight = 2
)

// Player represents the state and properties of a player in the game, including position, velocity, angle, and interactions.
type Player struct {
	where          XYZ
	velocity       XYZ
	angle          float64
	angleSin       float64
	angleCos       float64
	yaw            float64
	yawState       float64
	sector         *Sector
	ducking        bool
	falling        bool
	lightIntensity float64
	debug          bool
}

// NewPlayer initializes and returns a new Player instance at the specified position, angle, and sector.
func NewPlayer(cfg *ConfigPlayer, sector *Sector, debug bool) *Player {
	p := &Player{
		where:          XYZ{X: cfg.Position.X, Y: cfg.Position.Y, Z: sector.FloorY + EyeHeight},
		velocity:       XYZ{},
		yaw:            0,
		yawState:       0,
		sector:         sector,
		lightIntensity: 0.0039, // 1 / distance == 1 / 255
		debug:          debug,
	}
	p.SetAngle(cfg.Angle)
	return p
}

// AddAngle adjusts the player's current angle by adding the specified value and updates related angle properties.
func (p *Player) AddAngle(angle float64) {
	p.SetAngle(p.angle + angle)
}

// SetAngle updates the player's angle and recalculates its sine and cosine values for movement computations.
func (p *Player) SetAngle(angle float64) {
	p.angle = angle
	p.angleSin = math.Sin(p.angle)
	p.angleCos = math.Cos(p.angle)
}

// GetAngle returns the sine and cosine values of the player's current angle as two float64 values.
func (p *Player) GetAngle() (float64, float64) {
	return p.angleSin, p.angleCos
}

// SetYaw adjusts the player's yaw and yawState within a constrained range, also factoring in the Z velocity component.
func (p *Player) SetYaw(y float64) {
	p.yawState = mathematic.ClampF(p.yawState-(y*0.05), -5, 5)
	p.yaw = p.yawState - (p.velocity.Z * 0.5)
}

// Move adjusts the player's velocity based on directional input, considering speed and acceleration factors.
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

// SetJump alters the player's vertical velocity to initiate a jump and sets the falling state to true.
func (p *Player) SetJump() {
	p.velocity.Z += 0.5
	p.falling = true
}

// SetDucking toggles the player's ducking state and sets the falling state to true if ducking is enabled.
func (p *Player) SetDucking() {
	p.ducking = !p.ducking
	if p.ducking {
		p.falling = true
	}
}

// GetXY retrieves the current X and Y coordinates of the player.
func (p *Player) GetXY() (float64, float64) {
	return p.where.X, p.where.Y
}

// GetXYZ retrieves the current X, Y, and Z coordinates of the player as three float64 values.
func (p *Player) GetXYZ() (float64, float64, float64) {
	return p.where.X, p.where.Y, p.where.Z
}

// SetXY updates the player's X and Y coordinates and marks the player as falling.
func (p *Player) SetXY(x float64, y float64) {
	p.where.X = x
	p.where.Y = y
	p.falling = true
}

// AddXY increments the Player's position by the specified x and y values and sets the falling state to true.
func (p *Player) AddXY(x float64, y float64) {
	p.where.X += x
	p.where.Y += y
	p.falling = true
}

// GetZ retrieves the z-coordinate of the player's current position.
func (p *Player) GetZ() float64 {
	return p.where.Z
}

// GetLightIntensity returns the maximum distance the player can illuminate based on their current settings.
func (p *Player) GetLightIntensity() float64 {
	return p.lightIntensity
}

// SetLightIntensity updates the player's maximum illumination distance to the specified value.
func (p *Player) SetLightIntensity(lightIntensity float64) {
	p.lightIntensity = lightIntensity
}

// GetVelocity returns the X and Y components of the player's velocity as two separate float64 values.
func (p *Player) GetVelocity() (float64, float64) {
	return p.velocity.X, p.velocity.Y
}

// GetSector retrieves the current sector associated with the player.
func (p *Player) GetSector() *Sector {
	return p.sector
}

// SetSector updates the player's current sector to the specified sector model.
func (p *Player) SetSector(sector *Sector) {
	p.sector = sector
}

// GetYaw retrieves the current yaw (rotation about the vertical axis) of the player in radians.
func (p *Player) GetYaw() float64 {
	return p.yaw
}

// EyeHeight returns the player's current eye height. It changes based on whether the player is ducking or not.
func (p *Player) EyeHeight() float64 {
	if p.ducking {
		return DuckHeight
	}
	return EyeHeight
}

// VerticalCollision handles vertical movement and collision detection for the player based on velocity and sector boundaries.
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

// IsMoving checks if the player is currently in motion based on the velocity components. Returns true if moving, false otherwise.
func (p *Player) IsMoving() bool {
	return !(p.velocity.X == 0 && p.velocity.Y == 0)
}

// GetHeadPosition returns the player's head position (Z coordinate) by adding HeadMargin to the current Z position.
func (p *Player) GetHeadPosition() float64 {
	return p.where.Z + HeadMargin
}

// GetKneePosition calculates and returns the vertical position of the player's knees relative to the world coordinates.
func (p *Player) GetKneePosition() float64 {
	return p.where.Z - p.EyeHeight() + KneeHeight
}

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

func (p *Player) Compute2(vi *ViewItem) {
	p.VerticalCollision()
	if !p.IsMoving() {
		return
	}

	headPos := p.GetHeadPosition()
	kneePos := p.GetKneePosition()
	dx, dy := p.GetVelocity()
	pSector := p.GetSector()

	px, py := vi.GetXY()
	p1 := px + dx
	p2 := py + dy

	// Check if the player is about to cross one of the sector's edges
	for _, seg := range pSector.Segments {
		start := seg.Start
		end := seg.End

		if mathematic.IntersectLineSegmentsF(px, py, p1, p2, start.X, start.Y, end.X, end.Y) {
			// Check where the hole is.
			holeLow := 9e9
			holeHigh := -9e9
			if seg.Sector != nil {
				holeLow = mathematic.MaxF(pSector.FloorY, seg.Sector.FloorY)
				holeHigh = mathematic.MinF(pSector.CeilY, seg.Sector.CeilY)
			}

			// Check whether we're bumping into a wall
			if holeHigh < headPos || holeLow > kneePos {
				// Bumps into a wall! Slide along the wall
				xd := end.X - start.X
				yd := end.Y - start.Y
				lenSq := xd*xd + yd*yd

				if lenSq > 0 {
					dot := dx*xd + dy*yd
					dx = (xd * dot) / lenSq
					dy = (yd * dot) / lenSq

					// Calcolo della normale interna (winding CCW) e applicazione dell'epsilon
					invLen := 1.0 / math.Sqrt(lenSq)
					nx := -yd * invLen
					ny := xd * invLen

					epsilon := 0.005
					dx += nx * epsilon
					dy += ny * epsilon
				}
			}
			break
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
