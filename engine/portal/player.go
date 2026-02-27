package portal

import (
	"math"

	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
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
	where         model.XYZ
	velocity      model.XYZ
	angle         float64
	angleSin      float64
	angleCos      float64
	yaw           float64
	yawState      float64
	sector        *model.Sector
	ducking       bool
	falling       bool
	lightDistance float64
}

// NewPlayer initializes and returns a new Player instance at the specified position, angle, and sector.
func NewPlayer(x float64, y float64, z float64, angle float64, sector *model.Sector) *Player {
	p := &Player{
		where:         model.XYZ{X: x, Y: y, Z: z + EyeHeight},
		velocity:      model.XYZ{},
		yaw:           0,
		yawState:      0,
		sector:        sector,
		lightDistance: 0.0039, // 1 / distance == 1 / 255
	}
	p.SetAngle(angle)
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

// GetAngle returns the player's current angle, as well as its sine and cosine values.
func (p *Player) GetAngle() (float64, float64, float64) {
	return p.angle, p.angleSin, p.angleCos
}

// SetYaw adjusts the player's yaw and yawState within a constrained range, also factoring in the Z velocity component.
func (p *Player) SetYaw(y float64) {
	p.yawState = mathematic.ClampF(p.yawState-(y*0.05), -5, 5)
	p.yaw = p.yawState - (p.velocity.Z * 0.5)
}

// Move adjusts the player's velocity based on directional input, considering speed and acceleration factors.
func (p *Player) Move(up bool, down bool, left bool, right bool, slow bool) {
	var moveX float64
	var moveY float64
	var acceleration float64
	impulse := 0.2
	if slow {
		impulse = 0.01
	}
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

// GetCoords retrieves the current X and Y coordinates of the player.
func (p *Player) GetCoords() (float64, float64) {
	return p.where.X, p.where.Y
}

// SetCoords updates the player's X and Y coordinates and marks the player as falling.
func (p *Player) SetCoords(x float64, y float64) {
	p.where.X = x
	p.where.Y = y
	p.falling = true
}

// AddCoords increments the Player's position by the specified x and y values and sets the falling state to true.
func (p *Player) AddCoords(x float64, y float64) {
	p.where.X += x
	p.where.Y += y
	p.falling = true
}

// GetZ retrieves the z-coordinate of the player's current position.
func (p *Player) GetZ() float64 {
	return p.where.Z
}

// GetLightDistance returns the maximum distance the player can illuminate based on their current settings.
func (p *Player) GetLightDistance() float64 {
	return p.lightDistance
}

// SetLightDistance updates the player's maximum illumination distance to the specified value.
func (p *Player) SetLightDistance(lightDistance float64) {
	p.lightDistance = lightDistance
}

// GetVelocity returns the X and Y components of the player's velocity as two separate float64 values.
func (p *Player) GetVelocity() (float64, float64) {
	return p.velocity.X, p.velocity.Y
}

// GetSector retrieves the current sector associated with the player.
func (p *Player) GetSector() *model.Sector {
	return p.sector
}

// SetSector updates the player's current sector to the specified sector model.
func (p *Player) SetSector(sector *model.Sector) {
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
		if p.velocity.Z < 0 && nextZ < p.sector.Floor+eyeHeight {
			// down
			p.where.Z = p.sector.Floor + eyeHeight
			p.velocity.Z = 0
			p.falling = false
		} else if p.velocity.Z > 0 && nextZ > p.sector.Ceil {
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

// Update resets very small velocity values to zero to avoid unintended movement or floating-point precision issues.
func (p *Player) Update() {
	if math.Abs(p.velocity.X) < 0.001 {
		p.velocity.X = 0
	}
	if math.Abs(p.velocity.Y) < 0.001 {
		p.velocity.Y = 0
	}
}
