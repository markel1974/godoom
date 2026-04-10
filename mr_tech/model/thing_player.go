package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
)

// ThingPlayer represents a dynamic entity within a 3D environment capable of movement and interaction.
type ThingPlayer struct {
	id             string
	kind           int
	angle          float64
	angleSin       float64
	angleCos       float64
	yaw            float64
	yawState       float64
	ducking        bool
	lightIntensity float64
	bobbing        *Bobbing
	debug          bool
	eyeHeight      float64
	headMargin     float64
	duckHeight     float64
	*ThingBase
}

// NewThingPlayer creates and initializes a new ThingPlayer instance using the provided configuration and environment data.
func NewThingPlayer(cfg *config.ConfigPlayer, volumes *Volumes, entities *Entities, debug bool) (*ThingPlayer, error) {
	volume := volumes.QueryPoint2d(cfg.Position.X, cfg.Position.Y)
	if volume == nil {
		return nil, fmt.Errorf("can't find player sector at %f, %f", cfg.Position.X, cfg.Position.Y)
	}
	cfg.Kind = config.ThingPlayerDef
	if cfg.Height <= 0 {
		cfg.Height = 8
	}
	cfg.Position = geometry.XYZ{X: cfg.Position.X, Y: cfg.Position.Y, Z: volume.GetMinZ()}
	p := &ThingPlayer{
		id:             "PLAYER",
		kind:           0,
		yaw:            0,
		yawState:       0,
		ThingBase:      NewThingBase(cfg.ConfigThing, cfg.Position, nil, volume, volumes, entities),
		bobbing:        NewBobbing(2.6, 0.9, 0.03, 0.015, 0.15, 0.10),
		lightIntensity: 0.0039,
		debug:          debug,
		eyeHeight:      cfg.Height * 0.80,
		headMargin:     cfg.Height * 0.25,
		duckHeight:     cfg.Height * 0.25,
	}
	p.SetAngle(cfg.Angle)
	p.entities.AddThing(p)
	return p, nil
}

// GetLight retrieves the current light source associated with the ThingPlayer instance.
func (p *ThingPlayer) GetLight() *Light {
	return nil
}

// Compute performs a calculation or operation based on the given float64 input parameters.
func (p *ThingPlayer) Compute(_ float64, _ float64, _ float64) {
}

// AddAngle adds the given angle (in degrees) to the current angle of the ThingPlayer object.
func (p *ThingPlayer) AddAngle(angle float64) {
	p.SetAngle(p.angle + angle)
}

// SetAngle sets the angle of the ThingPlayer and calculates its sine and cosine values.
func (p *ThingPlayer) SetAngle(angle float64) {
	p.angle = angle
	p.angleSin = math.Sin(p.angle)
	p.angleCos = math.Cos(p.angle)
}

// GetRealAngle returns the current angle of the ThingPlayer as a float64.
func (p *ThingPlayer) GetRealAngle() float64 {
	return p.angle
}

// GetAngle returns the sine and cosine values of the player's angle as two float64 values.
func (p *ThingPlayer) GetAngle() (float64, float64) {
	return p.angleSin, p.angleCos
}

// SetYaw adjusts the yawState and yaw of the ThingPlayer based on the input value and current velocity.
func (p *ThingPlayer) SetYaw(y float64) {
	p.yawState = mathematic.ClampF(p.yawState-(y*0.05), -5, 5)
	p.yaw = p.yawState - (p.entity.GetVz() * 0.5)
}

func (p *ThingPlayer) Move(impulse float64, up, down, left, right bool) {
	if !up && !down && !left && !right {
		return
	}
	var fx, fy float64
	if up {
		fx += p.angleCos
		fy += p.angleSin
	}
	if down {
		fx -= p.angleCos
		fy -= p.angleSin
	}
	if left {
		fx += p.angleSin
		fy -= p.angleCos
	}
	if right {
		fx -= p.angleSin
		fy += p.angleCos
	}
	mag := math.Sqrt(fx*fx + fy*fy)
	if mag > 0 {
		const forceScale = 100.0
		p.entity.AddForce((fx/mag)*impulse*forceScale, (fy/mag)*impulse*forceScale, 0.0)
	}
}

// SetJump modifies the player's vertical velocity to simulate a jump and marks the player as falling.
func (p *ThingPlayer) SetJump() {
	p.entity.AddForce(0.0, 0.0, 100)
}

// SetDucking toggles the ducking state of the ThingPlayer and sets falling to true if ducking becomes active.
func (p *ThingPlayer) SetDucking() {
	p.ducking = !p.ducking
}

// GetBobPhase returns the current bob value and bob phase of the ThingPlayer.
func (p *ThingPlayer) GetBobPhase() (float64, float64) {
	return p.bobbing.GetBob(), p.bobbing.GetPhase()
}

// GetPosition retrieves the X, Y, and Z coordinates of the ThingPlayer's current position.
func (p *ThingPlayer) GetPosition() (float64, float64, float64) {
	return p.pos.X, p.pos.Y, p.getEyeHeight(p.pos.Z)
}

// GetLightIntensity retrieves the current light intensity value for the ThingPlayer instance.
func (p *ThingPlayer) GetLightIntensity() float64 {
	return p.lightIntensity
}

// SetLightIntensity sets the light intensity for the ThingPlayer to the specified value.
func (p *ThingPlayer) SetLightIntensity(lightIntensity float64) {
	p.lightIntensity = lightIntensity
}

// GetRadius returns the radius of the ThingPlayer.
func (p *ThingPlayer) GetRadius() float64 {
	return p.radius
}

// GetMass returns the mass of the ThingPlayer as a float64.
func (p *ThingPlayer) GetMass() float64 {
	return p.mass
}

// GetVelocity returns the current velocity of the ThingPlayer as three float64 components: X, Y, and Z.
func (p *ThingPlayer) GetVelocity() (float64, float64, float64) {
	return p.entity.GetVx(), p.entity.GetVy(), p.entity.GetVz()
}

// GetYaw returns the current yaw angle of the ThingPlayer as a float64.
func (p *ThingPlayer) GetYaw() float64 {
	return p.yaw
}

// IsMoving returns true if the ThingPlayer is currently moving in the X or Y direction, otherwise false.
func (p *ThingPlayer) IsMoving() bool {
	return p.entity.IsMoving()
}

func (p *ThingPlayer) PhysicsApply() {
	headPos := p.getHeadHeight(0)
	p.doPhysics(headPos)
	p.bobbing.Compute(p.entity.GetVx(), p.entity.GetVy())
}

// OnCollide handles the collision event between the current ThingPlayer and another object implementing IThing.
func (p *ThingPlayer) OnCollide(other IThing) {
	//TODO IMPLEMENT
}

// getHeadPosition calculates the player's head position by adding a fixed margin to the player's current Z coordinate.
func (p *ThingPlayer) getHeadHeight(base float64) float64 {
	return p.getEyeHeight(base) + p.headMargin
}

// eyeHeight calculates the player's eye height based on their current state (e.g., ducking or standing).
func (p *ThingPlayer) getEyeHeight(base float64) float64 {
	if p.ducking {
		return base + p.duckHeight
	}
	return base + p.eyeHeight
}
