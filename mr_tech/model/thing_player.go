package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
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

// ThingPlayer represents a player entity with position, velocity, angle, yaw, sector, states, and lighting attributes.
type ThingPlayer struct {
	id             string
	kind           int
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
	sectors        *Sectors
	entities       *Entities
	entity         *physics.Entity
	identifier     int
	debug          bool
}

// NewThingPlayer creates a new ThingPlayer instance with initial position, angle, sector, and debug configuration.
func NewThingPlayer(cfg *ConfigPlayer, sector *Sector, sectors *Sectors, entities *Entities, debug bool) *ThingPlayer {
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	x := cfg.Position.X - cfg.Radius
	y := cfg.Position.Y - cfg.Radius

	p := &ThingPlayer{
		id:             "PLAYER",
		kind:           0,
		where:          XYZ{X: cfg.Position.X, Y: cfg.Position.Y, Z: sector.FloorY + EyeHeight},
		velocity:       XYZ{},
		yaw:            0,
		yawState:       0,
		radius:         cfg.Radius,
		mass:           cfg.Mass,
		sector:         sector,
		lightIntensity: 0.0039, // 1 / distance == 1 / 255
		sectors:        sectors,
		entities:       entities,
		debug:          debug,
		identifier:     -1,
		entity:         physics.NewEntity(x, y, w, h, cfg.Mass),
	}
	p.SetAngle(cfg.Angle)
	p.entities.AddThing(p)
	return p
}

// GetId retrieves the unique identifier of the ThingPlayer instance.
func (p *ThingPlayer) GetId() string {
	return p.id
}

// GetKind retrieves the kind or type classification of the ThingPlayer as an integer value.
func (p *ThingPlayer) GetKind() ThingType {
	return ThingPlayerDef
}

// GetAnimation retrieves the current animation associated with the ThingPlayer instance.
func (p *ThingPlayer) GetAnimation() *textures.Animation {
	return nil
}

func (p *ThingPlayer) GetAABB() *physics.AABB {
	return p.entity.GetAABB()
}

func (p *ThingPlayer) GetEntity() *physics.Entity {
	return p.entity
}

// GetLight returns a pointer to the Light object associated with the ThingPlayer.
func (p *ThingPlayer) GetLight() *Light {
	return nil
}

// GetFloorY returns the Y-coordinate of the floor associated with the player's current sector.
func (p *ThingPlayer) GetFloorY() float64 {
	return p.sector.FloorY
}

// GetCeilY returns the ceiling Y coordinate of the sector the player is currently located in.
func (p *ThingPlayer) GetCeilY() float64 {
	return p.sector.CeilY
}

// SetIdentifier updates the value of the identifier for the ThingPlayer instance.
func (p *ThingPlayer) SetIdentifier(identifier int) {
	p.identifier = identifier
}

// GetIdentifier returns the unique identifier associated with the ThingPlayer instance.
func (p *ThingPlayer) GetIdentifier() int {
	return p.identifier
}

// Compute performs a calculation based on the provided float64 arguments using the ThingPlayer instance.
func (p *ThingPlayer) Compute(_ float64, _ float64) {
}

// EntityUpdate updates the state of the entity associated with the ThingPlayer and returns whether the update was successful.
func (p *ThingPlayer) EntityUpdate() bool {
	return p.entity.Update()
}

// AddAngle increments the player's current angle by the specified value and updates related trigonometric properties.
func (p *ThingPlayer) AddAngle(angle float64) {
	p.SetAngle(p.angle + angle)
}

// SetAngle sets the player's viewing angle in radians, recalculating the sine and cosine of the angle for movement.
func (p *ThingPlayer) SetAngle(angle float64) {
	p.angle = angle
	p.angleSin = math.Sin(p.angle)
	p.angleCos = math.Cos(p.angle)
}

func (p *ThingPlayer) GetRealAngle() float64 {
	return p.angle
}

// GetAngle returns the sine and cosine of the player's current angle as float64 values.
func (p *ThingPlayer) GetAngle() (float64, float64) {
	return p.angleSin, p.angleCos
}

// SetYaw adjusts the player's yaw (vertical rotation) by modifying yawState and accounting for velocity along the Z-axis.
func (p *ThingPlayer) SetYaw(y float64) {
	p.yawState = mathematic.ClampF(p.yawState-(y*0.05), -5, 5)
	p.yaw = p.yawState - (p.velocity.Z * 0.5)
}

// Move updates the player's velocity based on the given impulse and directional input flags (up, down, left, right).
func (p *ThingPlayer) Move(impulse float64, up bool, down bool, left bool, right bool) {
	var moveX float64
	var moveY float64
	var acceleration float64
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
func (p *ThingPlayer) SetJump() {
	p.velocity.Z += 0.5
	p.falling = true
}

// SetDucking toggles the player's ducking state and sets falling to true if the player is ducking.
func (p *ThingPlayer) SetDucking() {
	p.ducking = !p.ducking
	if p.ducking {
		p.falling = true
	}
}

// GetPosition returns the X and Y coordinates of the player's current position.
func (p *ThingPlayer) GetPosition() (float64, float64) {
	return p.where.X, p.where.Y
}

// GetXYZ retrieves the player's current X, Y, and Z coordinates in the game world.
func (p *ThingPlayer) GetXYZ() (float64, float64, float64) {
	return p.where.X, p.where.Y, p.where.Z
}

// SetXY updates the player's X and Y coordinates and sets the falling state to true.
func (p *ThingPlayer) SetXY(x float64, y float64) {
	p.where.X = x
	p.where.Y = y
	p.falling = true
}

// AddXY applies incremental adjustments to the player's X and Y coordinates and sets the falling state to true.
func (p *ThingPlayer) AddXY(x float64, y float64) {
	p.where.X += x
	p.where.Y += y
	p.falling = true
}

// GetZ retrieves the Z-coordinate (vertical position) of the player.
func (p *ThingPlayer) GetZ() float64 {
	return p.where.Z
}

// GetLightIntensity returns the current light intensity value associated with the ThingPlayer instance.
func (p *ThingPlayer) GetLightIntensity() float64 {
	return p.lightIntensity
}

// SetLightIntensity sets the light intensity for the player by updating the lightIntensity property.
func (p *ThingPlayer) SetLightIntensity(lightIntensity float64) {
	p.lightIntensity = lightIntensity
}

// GetRadius returns the radius of the player as a float64 value.
func (p *ThingPlayer) GetRadius() float64 {
	return p.radius
}

// GetMass returns the mass of the player as a float64 value.
func (p *ThingPlayer) GetMass() float64 {
	return p.mass
}

// GetVelocity returns the X and Y components of the player's velocity as two float64 values.
func (p *ThingPlayer) GetVelocity() (float64, float64) {
	return p.velocity.X, p.velocity.Y
}

// GetSector returns the current sector the player is located in.
func (p *ThingPlayer) GetSector() *Sector {
	return p.sector
}

// SetSector updates the ThingPlayer's current sector to the specified Sector instance.
func (p *ThingPlayer) SetSector(sector *Sector) {
	p.sector = sector
}

// GetYaw returns the current yaw value of the player.
func (p *ThingPlayer) GetYaw() float64 {
	return p.yaw
}

// IsMoving determines whether the player is currently in motion based on its velocity in the X and Y axes.
func (p *ThingPlayer) IsMoving() bool {
	return !(p.velocity.X == 0 && p.velocity.Y == 0)
}

// PhysicsApply applies the physics calculations to adjust the player's position relative to the associated entity.
func (p *ThingPlayer) PhysicsApply() {
	pX, pY := p.GetPosition()
	eX, eY := p.entity.GetCenterXY()
	dx := eX - pX
	dy := eY - pY
	if math.Abs(dx) > minMovement || math.Abs(dy) > minMovement {
		p.MoveApply(dx, dy)
	}
}

// MoveApply updates the player's position based on the given displacement and handles sector transitions when necessary.
func (p *ThingPlayer) MoveApply(dx float64, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}
	// Apply the atomic vector and obtain the final coordinates
	p.AddXY(dx, dy)
	px, py := p.GetPosition()

	// Spatial stability check: are we still inside the same sector?
	if newSector := p.sectors.SectorSearch(p.sector, px, py); newSector != nil {
		p.sector = newSector
	}

	p.entity.Vx, p.entity.Vy = p.GetVelocity()

	p.entities.UpdateThing(p, px, py)
}

// Update updates the player's position and velocity based on collision detection and sector constraints.
func (p *ThingPlayer) Update(vi *ViewMatrix) {
	p.verticalCollision()
	if !p.IsMoving() {
		return
	}

	headPos := p.getHeadPosition()
	kneePos := p.getKneePosition()
	viewX, viewY := vi.GetXY()
	velX, velY := p.GetVelocity()
	pX := viewX + velX
	pY := viewY + velY

	velX, velY = p.wallSlidingEffect(viewX, viewY, pX, pY, velX, velY, headPos, kneePos)

	if math.Abs(p.velocity.X) < 0.001 {
		p.velocity.X = 0
	}
	if math.Abs(p.velocity.Y) < 0.001 {
		p.velocity.Y = 0
	}

	p.MoveApply(velX, velY)
}

// GetHeadPosition returns the Z-coordinate of the player's head, calculated as the player's current Z-position plus HeadMargin.
func (p *ThingPlayer) getHeadPosition() float64 {
	return p.where.Z + HeadMargin
}

// GetKneePosition calculates and returns the player's knee position based on their current Z-coordinate and eye height.
func (p *ThingPlayer) getKneePosition() float64 {
	return p.where.Z - p.eyeHeight() + KneeHeight
}

// EyeHeight returns the height of the player's eyes, considering whether the player is ducking or standing upright.
func (p *ThingPlayer) eyeHeight() float64 {
	if p.ducking {
		return DuckHeight
	}
	return EyeHeight
}

// VerticalCollision checks and resolves vertical collisions for the player, adjusting position and velocity based on sector bounds.
func (p *ThingPlayer) verticalCollision() {
	if p.falling {
		eyeHeight := p.eyeHeight()
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

// wallSlidingEffect adjusts the velocity when sliding along a wall to simulate a wall-sliding effect with slight separation.
// Takes the current view coordinates, position, velocity, head and knee positions, and returns the modified velocity.
func (p *ThingPlayer) wallSlidingEffect(viewX float64, viewY float64, pX float64, pY float64, velX float64, velY float64, headPos float64, kneePos float64) (float64, float64) {
	const epsilon = 0.005
	seg1 := p.sector.CheckSegmentsClearance(viewX, viewY, pX, pY, headPos, kneePos, p.entity.GetWidth()/2)
	if seg1 != nil {
		xd := seg1.End.X - seg1.Start.X
		yd := seg1.End.Y - seg1.Start.Y
		if lenSq := xd*xd + yd*yd; lenSq > 0 {
			dot := velX*xd + velY*yd
			velX = (xd * dot) / lenSq
			velY = (yd * dot) / lenSq
			invLen := 1.0 / math.Sqrt(lenSq)
			nx := -yd * invLen
			ny := xd * invLen
			velX += nx * epsilon
			velY += ny * epsilon
		}
	}
	return velX, velY
}
