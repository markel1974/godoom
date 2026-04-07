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

// ThingPlayer represents a dynamic entity within a 3D environment capable of movement and interaction.
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
	sectors        *Volumes
	entities       *Entities
	entity         *physics.Entity
	identifier     int
	bob            float64
	bobPhase       float64
	wallPhysics    *WallPhysics
	debug          bool
	height         float64
	eyeHeight      float64
	maxStep        float64
	headMargin     float64
	kneeHeight     float64
	duckHeight     float64
}

// NewThingPlayer creates and initializes a new ThingPlayer instance using the provided configuration and environment data.
func NewThingPlayer(cfg *config.ConfigPlayer, volumes *Volumes, entities *Entities, debug bool) (*ThingPlayer, error) {
	volume := volumes.QueryPoint2d(cfg.Position.X, cfg.Position.Y)
	if volume == nil {
		return nil, fmt.Errorf("can't find player sector at %f, %f", cfg.Position.X, cfg.Position.Y)
	}
	diameter := cfg.Radius * 2
	w := diameter
	h := diameter
	x := cfg.Position.X - cfg.Radius
	y := cfg.Position.Y - cfg.Radius
	height := 8.0
	if cfg.Height > 0 {
		height = cfg.Height
	}
	p := &ThingPlayer{
		id:             "PLAYER",
		kind:           0,
		velocity:       geometry.XYZ{},
		yaw:            0,
		yawState:       0,
		bob:            0,
		bobPhase:       0,
		radius:         cfg.Radius,
		mass:           cfg.Mass,
		volume:         volume,
		lightIntensity: 0.0039,
		sectors:        volumes,
		entities:       entities,
		debug:          debug,
		identifier:     -1,
		wallPhysics:    NewWallPhysics(volumes),
		height:         height,
		eyeHeight:      height * 0.80,
		maxStep:        height * 0.50,
		headMargin:     height * 0.25,
		kneeHeight:     height * 0.25,
		duckHeight:     height * 0.25,
	}
	d := p.eyeHeight + p.headMargin
	p.where = geometry.XYZ{X: cfg.Position.X, Y: cfg.Position.Y, Z: volume.GetMinZ() + p.eyeHeight}
	p.entity = physics.NewEntity(x, y, volume.GetMinZ(), w, h, d, cfg.Mass)
	p.SetAngle(cfg.Angle)
	p.entities.AddThing(p)
	return p, nil
}

// GetId returns the unique identifier of the ThingPlayer instance.
func (p *ThingPlayer) GetId() string {
	return p.id
}

// GetKind returns the type of the ThingPlayer as defined in the ThingType configuration.
func (p *ThingPlayer) GetKind() config.ThingType {
	return config.ThingPlayerDef
}

// GetAnimation retrieves the current animation associated with the ThingPlayer instance. Returns a pointer to Animation.
func (p *ThingPlayer) GetAnimation() *textures.Animation {
	return nil
}

// GetAABB returns the Axis-Aligned Bounding Box (AABB) of the ThingPlayer entity.
func (p *ThingPlayer) GetAABB() *physics.AABB {
	return p.entity.GetAABB()
}

// GetEntity retrieves the physics.Entity instance associated with the ThingPlayer.
func (p *ThingPlayer) GetEntity() *physics.Entity {
	return p.entity
}

// GetLight retrieves the current light source associated with the ThingPlayer instance.
func (p *ThingPlayer) GetLight() *Light {
	return nil
}

// GetMinZ returns the minimum Z value from the associated volume of the ThingPlayer instance.
func (p *ThingPlayer) GetMinZ() float64 {
	return p.volume.GetMinZ()
}

// GetMaxZ returns the maximum Z value from the ThingPlayer's volume.
func (p *ThingPlayer) GetMaxZ() float64 {
	return p.volume.GetMaxZ()
}

// SetIdentifier sets the identifier for the ThingPlayer instance to the provided integer value.
func (p *ThingPlayer) SetIdentifier(identifier int) {
	p.identifier = identifier
}

// GetIdentifier returns the unique identifier associated with the ThingPlayer instance.
func (p *ThingPlayer) GetIdentifier() int {
	return p.identifier
}

// Compute performs a calculation or operation based on the given float64 input parameters.
func (p *ThingPlayer) Compute(_ float64, _ float64, _ float64) {
}

// EntityUpdate updates the entity associated with the ThingPlayer and returns a boolean indicating success or failure.
func (p *ThingPlayer) EntityUpdate() bool {
	return p.entity.Update()
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
	p.yaw = p.yawState - (p.velocity.Z * 0.5)
}

// Move applies movement to the ThingPlayer based on directional inputs and impulse.
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

// SetJump modifies the player's vertical velocity to simulate a jump and marks the player as falling.
func (p *ThingPlayer) SetJump() {
	p.velocity.Z += 0.5
	p.falling = true
}

// SetDucking toggles the ducking state of the ThingPlayer and sets falling to true if ducking becomes active.
func (p *ThingPlayer) SetDucking() {
	p.ducking = !p.ducking
	if p.ducking {
		p.falling = true
	}
}

// GetBobPhase returns the current bob value and bob phase of the ThingPlayer.
func (p *ThingPlayer) GetBobPhase() (float64, float64) {
	return p.bob, p.bobPhase
}

// GetPosition retrieves the X, Y, and Z coordinates of the ThingPlayer's current position.
func (p *ThingPlayer) GetPosition() (float64, float64, float64) {
	return p.where.X, p.where.Y, p.where.Z
}

// SetXYZ sets the X, Y, and Z coordinates of the ThingPlayer and marks it as falling.
func (p *ThingPlayer) SetXYZ(x float64, y float64, z float64) {
	p.where.X = x
	p.where.Y = y
	p.where.Z = z
	p.falling = true
}

// AddXYZ updates the X, Y, and Z coordinates by the specified values and sets the falling state to true.
func (p *ThingPlayer) AddXYZ(x float64, y float64, z float64) {
	p.where.X += x
	p.where.Y += y
	p.where.Z += z
	p.falling = true
}

// GetZ returns the Z coordinate of the ThingPlayer's current position.
func (p *ThingPlayer) GetZ() float64 {
	return p.where.Z
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
	return p.velocity.X, p.velocity.Y, p.velocity.Z
}

// GetVolume returns the current volume associated with the ThingPlayer.
func (p *ThingPlayer) GetVolume() *Volume {
	return p.volume
}

// SetVolume updates the player's current volume with the provided Volume instance.
func (p *ThingPlayer) SetVolume(volume *Volume) {
	p.volume = volume
}

// GetYaw returns the current yaw angle of the ThingPlayer as a float64.
func (p *ThingPlayer) GetYaw() float64 {
	return p.yaw
}

// IsMoving returns true if the ThingPlayer is currently moving in the X or Y direction, otherwise false.
func (p *ThingPlayer) IsMoving() bool {
	return !(p.velocity.X == 0 && p.velocity.Y == 0 && p.velocity.Z == 0)
}

// PhysicsApply updates the player's position based on physics calculations relative to its associated entity.
func (p *ThingPlayer) PhysicsApply() {
	pX, pY, pZ := p.GetPosition()
	eX, eY, eZ := p.entity.GetCenter()

	dx := eX - pX
	dy := eY - pY

	// Confronto rigoroso Base-su-Base per l'asse Z
	entityBaseZ := eZ - (p.entity.GetDepth() / 2.0)
	playerBaseZ := pZ - p.getEyeHeight()
	dz := entityBaseZ - playerBaseZ

	if math.Abs(dx) > minMovement || math.Abs(dy) > minMovement || math.Abs(dz) > minMovement {
		p.MoveApply(dx, dy, dz)
	}
}

// MoveApply adjusts the player's position by the specified delta values and updates related systems accordingly.
func (p *ThingPlayer) MoveApply(dx float64, dy float64, dz float64) {
	if dx == 0 && dy == 0 && dz == 0 {
		return
	}
	// 1. Aggiornamento posizione logica (Eye Level)
	p.AddXYZ(dx, dy, dz)
	px, py, pz := p.GetPosition()

	// 2. Spatial stability check in 3D
	// Usiamo il baricentro (metà altezza sotto gli occhi) per la query del volume
	feetZ := pz - p.getEyeHeight()
	topZ := feetZ + p.entity.GetDepth() // o p.entity.GetDepth()
	stepHeight := p.kneeHeight          // Definito come 2.0 nelle tue costanti
	if newVolume := p.sectors.SearchVolume3d(p.volume, px, py, feetZ, topZ, stepHeight); newVolume != nil && newVolume != p.volume {
		p.volume = newVolume
	}
	// 3. Sincronizzazione velocità nel motore impulsivo
	p.entity.SetVx(p.velocity.X)
	p.entity.SetVy(p.velocity.Y)
	p.entity.SetVz(p.velocity.Z)
	// 4. Update AABB Tree: passiamo la quota dei PIEDI (pz - eyeHeight)
	// affinché il Rect.point.z (base) sia allineato al pavimento del settore.
	p.entities.UpdateThing(p, px, py, pz-p.getEyeHeight())
}

// Update updates the player's position and velocity based on movement and physics calculations.
func (p *ThingPlayer) Update(vi *ViewMatrix) {
	p.verticalMovementApply()
	// Se siamo fermi orizzontalmente ma stiamo cadendo, IsMoving deve essere true
	if !p.IsMoving() && !p.falling {
		return
	}
	viewX, viewY, viewZ := p.GetPosition()
	velX, velY, velZ := p.GetVelocity()
	zTop := viewZ + p.headMargin
	zBottom := viewZ - p.getEyeHeight() + p.kneeHeight
	zMinLimit := p.volume.GetMinZ() + p.getEyeHeight()
	zMaxLimit := p.volume.GetMaxZ() - p.headMargin
	velX, velY, velZ, _ = p.wallPhysics.AdjustVelocity(viewX, viewY, viewZ, velX, velY, velZ, zTop, zBottom, zMinLimit, zMaxLimit, p.radius, false)
	// Applichiamo i delta finali filtrati
	p.MoveApply(velX, velY, velZ)
	// Smorzamento inerziale della velocità Z (opzionale per salti più naturali)
	if !p.falling {
		p.velocity.Z = 0
	}
}

// IsActive determines whether the ThingPlayer instance is currently active. Returns true if active, otherwise false.
func (p *ThingPlayer) IsActive() bool {
	return true
}

// SetActive sets the active state of the ThingPlayer, enabling or disabling its activity based on the given boolean value.
func (p *ThingPlayer) SetActive(active bool) {
}

// OnCollide handles the collision event between the current ThingPlayer and another object implementing IThing.
func (p *ThingPlayer) OnCollide(other IThing) {
	//TODO IMPLEMENT
}

// getHeadPosition calculates the player's head position by adding a fixed margin to the player's current Z coordinate.
func (p *ThingPlayer) getHeadPosition() float64 {
	return p.where.Z + p.headMargin
}

// getKneePosition calculates and returns the Z-coordinate of the player's knee position based on current attributes.
func (p *ThingPlayer) getKneePosition() float64 {
	return p.where.Z - p.getEyeHeight() + p.kneeHeight
}

// eyeHeight calculates the player's eye height based on their current state (e.g., ducking or standing).
func (p *ThingPlayer) getEyeHeight() float64 {
	if p.ducking {
		return p.duckHeight
	}
	return p.eyeHeight
}

// verticalMovementApply handles the vertical movement logic for a player, applying gravity and collision constraints.
func (p *ThingPlayer) verticalMovementApply() {
	eyeHeight := p.getEyeHeight()
	floorZ := p.volume.GetMinZ() + eyeHeight
	ceilZ := p.volume.GetMaxZ() - p.headMargin

	// Applichiamo la gravità se non siamo appoggiati al suolo
	if p.where.Z > floorZ || p.velocity.Z > 0 {
		p.falling = true
		p.velocity.Z -= 0.02 // Gravità ridotta per fluidità
	}

	// Clipping preventivo della velocità
	nextZ := p.where.Z + p.velocity.Z
	if nextZ < floorZ {
		p.velocity.Z = floorZ - p.where.Z
		p.falling = false
	} else if nextZ > ceilZ {
		p.velocity.Z = 0
	}
}
