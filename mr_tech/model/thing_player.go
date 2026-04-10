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
	angle          float64
	angleSin       float64
	angleCos       float64
	yaw            float64
	yawState       float64
	radius         float64
	mass           float64
	volume         *Volume
	ducking        bool
	lightIntensity float64
	sectors        *Volumes
	entities       *Entities
	entity         *physics.Entity
	identifier     int
	bobbing        *Bobbing
	wall           *ThingWall
	debug          bool
	height         float64
	eyeHeight      float64
	maxStep        float64
	headMargin     float64
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
		yaw:            0,
		yawState:       0,
		bobbing:        NewBobbing(2.6, 0.9, 0.03, 0.015, 0.15, 0.10),
		radius:         cfg.Radius,
		mass:           cfg.Mass,
		volume:         volume,
		lightIntensity: 0.0039,
		sectors:        volumes,
		entities:       entities,
		debug:          debug,
		identifier:     -1,
		wall:           NewThingWall(volumes, 0, 0),
		height:         height,
		eyeHeight:      height * 0.80,
		maxStep:        height * 0.50,
		headMargin:     height * 0.25,
		duckHeight:     height * 0.25,
	}
	z := volume.GetMinZ()
	d := p.eyeHeight + p.headMargin
	restitution := cfg.Restitution
	p.where = geometry.XYZ{X: cfg.Position.X, Y: cfg.Position.Y, Z: z} // volume.GetMinZ()+ p.eyeHeight}
	p.entity = physics.NewEntity(x, y, z, w, h, d, cfg.Mass, restitution, 0.9)
	//p.entity.SetGForce(0.2)
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
//func (p *ThingPlayer) EntityUpdate() bool {
//	return p.entity.Update()
//}

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
	return p.where.X, p.where.Y, p.getEyeHeight(p.where.Z)
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
	return p.entity.IsMoving()
}

func (p *ThingPlayer) PhysicsApply() {
	//TODO UNIFICARE CON THINKBASE
	headPos := p.getHeadHeight(p.where.Z)
	feetPos := p.getFeetHeight(p.where.Z)
	playerHeight := headPos - feetPos
	dx, dy, dz := p.entity.GetDisplacement()
	nextX := p.where.X + dx
	nextY := p.where.Y + dy
	nextZ := p.where.Z + dz
	// 2. CONTINUOUS COLLISION DETECTION (Sweep XY Elevato)
	elevatedBaseZ := feetPos + p.maxStep
	face, nx, ny, nz := p.wall.ClosestFace(p.where.X, p.where.Y, p.where.Z, nextX, nextY, nextZ, dx, dy, dz, headPos, elevatedBaseZ, p.radius)
	if face != nil {
		//p.entity.ResolveImpact(p.wall.GetEntity(), nx, ny, nz)
		//dx, dy, dz = p.entity.GetDisplacement()
		//nextX, nextY, nextZ = p.where.X+dx, p.where.Y+dy, p.where.Z+dz
		p.entity.ResolveImpact(p.wall.GetEntity(), nx, ny, nz)
		// 2. Raffinamento KCC: Clip della velocità residua
		// Impedisce che la restitution o errori di precisione facciano "staccare" il player dalle rampe
		vx, vy, vz := p.entity.GetVelocity()
		// Se atterriamo su un piano calpestabile (nz >= 0.7) e stiamo cadendo
		if nz >= 0.7 && vz < 0 {
			vz = 0 // Stop verticale immediato
		}
		// Proiezione del vettore velocità per scivolare sulla normale
		newVx, newVy, newVz := p.entity.ClipVelocity(vx, vy, vz, nx, ny, nz)
		p.entity.SetVx(newVx)
		p.entity.SetVy(newVy)
		p.entity.SetVz(newVz)
		// 3. Ricalcolo del displacement finale per il frame corrente
		dx, dy, dz = p.entity.GetDisplacement()
		nextX, nextY, nextZ = p.where.X+dx, p.where.Y+dy, p.where.Z+dz
	}
	// 3. TRANSIZIONE DI SETTORE
	topZ := p.getHeadHeight(nextZ) //nextZ + playerHeight
	newVolume := p.sectors.SearchVolume3d(p.volume, nextX, nextY, nextZ, topZ, p.maxStep)
	if newVolume != nil && newVolume != p.volume {
		if p.entity.GetVz() <= 0 {
			actualStep := newVolume.GetMinZ() - p.volume.GetMinZ()
			if actualStep > 0 || (actualStep < 0 && math.Abs(actualStep) < p.maxStep) {
				// Snap diretto senza offset: la nostra Z coincide col pavimento!
				nextZ = newVolume.GetMinZ()
				p.entity.SetVz(0.0)
			}
		}
		p.volume = newVolume
	}
	// 4. LIMITI TOPOLOGICI VERTICALI (Floor / Ceil Hard Clamp)
	floorZ := p.volume.GetMinZ()
	ceilZ := p.volume.GetMaxZ()
	if nextZ < floorZ {
		p.entity.ResolveImpact(p.wall.GetEntity(), 0, 0, 1)
		nextZ = floorZ // Snap matematico ai piedi
	} else if (nextZ + playerHeight) > ceilZ {
		p.entity.ResolveImpact(p.wall.GetEntity(), 0, 0, -1)
		nextZ = ceilZ - playerHeight // La Z torna giù lasciando lo spazio esatto per l'altezza
	}
	// 5. APPLICAZIONE FINALE
	p.where.X, p.where.Y, p.where.Z = nextX, nextY, nextZ
	//esegue anche il moveTo
	p.entities.UpdateThing(p, p.where.X, p.where.Y, p.where.Z)
	p.bobbing.Compute(p.entity.GetVx(), p.entity.GetVy())
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
func (p *ThingPlayer) getFeetHeight(base float64) float64 {
	return base
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
