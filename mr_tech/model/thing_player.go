package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
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
	where          geometry.XYZ
	velocity       geometry.XYZ
	angle          float64
	angleSin       float64
	angleCos       float64
	yaw            float64
	yawState       float64
	radius         float64
	mass           float64
	volume         *Volume
	ducking        bool
	falling        bool
	lightIntensity float64
	sectors        *Sectors
	entities       *Entities
	entity         *physics.Entity
	identifier     int
	bob            float64
	bobPhase       float64
	debug          bool
}

// NewThingPlayer creates a new ThingPlayer instance with initial position, angle, sector, and debug configuration.
func NewThingPlayer(cfg *config.ConfigPlayer, sectors *Sectors, entities *Entities, debug bool) (*ThingPlayer, error) {
	volume := sectors.QueryPoint(cfg.Position.X, cfg.Position.Y)
	if volume == nil {
		return nil, fmt.Errorf("can't find player sector at %f, %f", cfg.Position.X, cfg.Position.Y)
	}

	w := cfg.Radius * 2
	h := cfg.Radius * 2
	x := cfg.Position.X - cfg.Radius
	y := cfg.Position.Y - cfg.Radius

	p := &ThingPlayer{
		id:             "PLAYER",
		kind:           0,
		where:          geometry.XYZ{X: cfg.Position.X, Y: cfg.Position.Y, Z: volume.GetFloorY() + EyeHeight},
		velocity:       geometry.XYZ{},
		yaw:            0,
		yawState:       0,
		bob:            0,
		bobPhase:       0,
		radius:         cfg.Radius,
		mass:           cfg.Mass,
		volume:         volume,
		lightIntensity: 0.0039, // 1 / distance == 1 / 255
		sectors:        sectors,
		entities:       entities,
		debug:          debug,
		identifier:     -1,
		entity:         physics.NewEntity(x, y, w, h, cfg.Mass),
	}
	p.SetAngle(cfg.Angle)
	p.entities.AddThing(p)
	return p, nil
}

// GetId retrieves the unique identifier of the ThingPlayer instance.
func (p *ThingPlayer) GetId() string {
	return p.id
}

// GetKind retrieves the kind or type classification of the ThingPlayer as an integer value.
func (p *ThingPlayer) GetKind() config.ThingType {
	return config.ThingPlayerDef
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
	return p.volume.GetFloorY()
}

// GetCeilY returns the ceiling Y coordinate of the sector the player is currently located in.
func (p *ThingPlayer) GetCeilY() float64 {
	return p.volume.GetCeilY()
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
	vx := p.velocity.X*(1-acceleration) + (moveX * acceleration)
	vy := p.velocity.Y*(1-acceleration) + (moveY * acceleration)
	p.velocity.X = vx
	p.velocity.Y = vy
	if speed := math.Sqrt(vx*vx + vy*vy); speed > 0.05 {
		p.bobPhase += speed * 0.7 // Frequenza dei passi legata alla velocità reale
	} else {
		p.bobPhase *= 0.85 // Smorzamento elastico quando ti fermi
	}
	p.bob = math.Sin(p.bobPhase) * 0.9
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

// GetBobPhase returns the current bob and bob phase values as a pair of float64.
func (p *ThingPlayer) GetBobPhase() (float64, float64) {
	return p.bob, p.bobPhase
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

// GetVolume returns the current volume the player is located in.
func (p *ThingPlayer) GetVolume() *Volume {
	return p.volume
}

// SetVolume updates the ThingPlayer's current sector to the specified Sector instance.
func (p *ThingPlayer) SetVolume(volume *Volume) {
	p.volume = volume
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
	if newVolume := p.sectors.SearchVolume(p.volume, px, py); newVolume != nil {
		p.volume = newVolume
	}

	vx, vy := p.GetVelocity()
	p.entity.SetVx(vx)
	p.entity.SetVy(vy)

	p.entities.UpdateThing(p, px, py)
}

// Update updates the player's position and velocity based on collision detection and sector constraints.
func (p *ThingPlayer) Update(vi *ViewMatrix) {
	p.verticalMovementApply()
	if !p.IsMoving() {
		return
	}
	viewX, viewY := vi.GetXY()
	velX, velY := p.GetVelocity()
	velX, velY = p.adjustPassage(viewX, viewY, velX, velY)
	if math.Abs(p.velocity.X) < 0.001 {
		p.velocity.X = 0
	}
	if math.Abs(p.velocity.Y) < 0.001 {
		p.velocity.Y = 0
	}
	p.MoveApply(velX, velY)
}

// IsActive checks if the ThingPlayer instance is currently active and returns true if it is, otherwise false.
func (p *ThingPlayer) IsActive() bool {
	return true
}

// SetActive sets the active state of the ThingPlayer based on the provided boolean value.
func (p *ThingPlayer) SetActive(active bool) {
}

// OnCollide handles the collision event between the ThingPlayer and another object of type IThing.
func (p *ThingPlayer) OnCollide(other IThing) {
	//TODO IMPLEMENT
}

// checkWall adjusts the player's velocity when colliding with walls based on position, velocity, and collision detection logic.
func (p *ThingPlayer) adjustPassage(viewX, viewY, velX, velY float64) (float64, float64) {
	top := p.getHeadPosition()
	bottom := p.getKneePosition()
	pX := viewX + velX
	pY := viewY + velY
	radius := p.entity.GetWidth() / 2
	velX, velY = WallSlidingEffect(p.volume, viewX, viewY, pX, pY, velX, velY, top, bottom, radius)
	return velX, velY
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
func (p *ThingPlayer) verticalMovementApply() {
	if p.falling {
		eyeHeight := p.eyeHeight()
		p.velocity.Z -= 0.05
		nextZ := p.where.Z + p.velocity.Z
		if p.velocity.Z < 0 && nextZ < p.volume.GetFloorY()+eyeHeight {
			// down
			p.where.Z = p.volume.GetFloorY() + eyeHeight
			p.velocity.Z = 0
			p.falling = false
		} else if p.velocity.Z > 0 && nextZ > p.volume.GetCeilY() {
			// up
			p.velocity.Z = 0
			p.falling = true
		}
		if p.falling {
			p.where.Z += p.velocity.Z
		}
	}
}
