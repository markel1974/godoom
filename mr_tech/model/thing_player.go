package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
)

const minPitch = -5.0
const maxPitch = 5.0

// ThingPlayer represents a controllable entity with movement, physics, and gameplay-related properties.
type ThingPlayer struct {
	kind           int
	angleSin       float64
	angleCos       float64
	pitch          float64
	pitchState     float64
	ducking        bool
	lightIntensity float64
	bobbing        *Bobbing
	debug          bool
	*ThingBase
}

// NewThingPlayer creates and initializes a new ThingPlayer entity using the provided configuration, world, and things.
// It ensures the player is placed in a valid sector and properly configures position, angle, and other properties.
// Returns the initialized ThingPlayer or an error if the player's sector is not found or configuration fails.
func NewThingPlayer(things *Things, c *config.Player, volumes *Volumes, debug bool) *ThingPlayer {
	volume := volumes.LocateVolume(c.Position.X, c.Position.Y, c.Position.Z)
	if volume == nil {
		fmt.Printf("can't find player location at %f, %f\n", c.Position.X, c.Position.Y)
		return nil
	}
	c.Kind = config.ThingPlayerDef
	if c.Height <= 0 {
		c.Height = 8.0
	}
	if c.Speed <= 0 {
		c.Speed = 60.0
	}
	c.Position = geometry.XYZ{X: c.Position.X, Y: c.Position.Y, Z: c.Position.Z}
	p := &ThingPlayer{
		kind:           0,
		pitch:          0,
		pitchState:     0,
		ThingBase:      NewThingBase(things, c.Thing, c.Position, nil, volume),
		bobbing:        NewBobbing(c.Bobbing),
		lightIntensity: 0.0039,
		debug:          debug,
	}
	p.id = "PLAYER"
	p.SetAngle(c.Angle)
	return p
}

func (p *ThingPlayer) IsActive() bool {
	return true
}

func (p *ThingPlayer) PostMessage(ec *ThingEvent) {
	p.inbox <- ec
}

func (p *ThingPlayer) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-p.inbox:
				switch evt.GetKind() {
				case StageThinking:
					p.Compute(evt.GetCoords())
				case StagePhysics:
					p.PhysicsApply()
				}
				evt.Done()
			case <-p.done:
				return
			}
		}
	}()
}

// GetLight retrieves the Light object associated with the ThingPlayer instance. Returns a pointer to the Light.
func (p *ThingPlayer) GetLight() *Light {
	return nil
}

func (p *ThingPlayer) Compute(playerX float64, playerY float64, playerZ float64) {
	//nothing to do
}

// AddAngle updates the player's current angle by adding the given angle in radians.
func (p *ThingPlayer) AddAngle(angle float64) {
	p.SetAngle(p.angle + angle)
}

// SetAngle sets the angle of the player and updates sine and cosine values based on the new angle.
func (p *ThingPlayer) SetAngle(angle float64) {
	p.angle = angle
	p.angleSin = math.Sin(p.angle)
	p.angleCos = math.Cos(p.angle)
}

// GetAngleFull returns the sine and cosine of the player's current angle as two float64 values.
func (p *ThingPlayer) GetAngleFull() (float64, float64) {
	return p.angleSin, p.angleCos
}

// SetPitch adjusts the player's pitch angle within a defined range and updates the current pitch state.
func (p *ThingPlayer) SetPitch(y float64) {
	p.pitchState = mathematic.ClampF(p.pitchState-(y*0.05), minPitch, maxPitch)
	// Svincolamento totale dalla fisica: lo sguardo è assoluto
	p.pitch = p.pitchState
}

// Move applies a directional impulse to the player based on input flags (up, down, left, right) and a given impulse magnitude.
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
	if mag := math.Sqrt(fx*fx + fy*fy); mag > 0 {
		// 1. Vettore direzione puro (Normalizzato)
		dirX := fx / mag
		dirY := fy / mag
		p.MoveTowards(dirX, dirY, p.speed*impulse, p.speed)
	}
}

// SetJump applies an upward force to make the ThingPlayer jump.
func (p *ThingPlayer) SetJump(multi bool) {
	onGround := true
	mass := p.entity.GetMass()
	fz := mass * 100
	if !multi {
		onGround = p.entity.IsOnGround()
		fz = mass * 1000
	}
	if onGround {
		p.entity.AddForce(0.0, 0.0, fz)
		p.entity.SetOnGround(false)
		p.bobbing.InjectVerticalImpulse(-1.5)
	}
}

// SetDucking toggles the player's ducking state between true and false.
func (p *ThingPlayer) SetDucking() {
	p.ducking = !p.ducking
}

// GetBob retrieves the current bobbing values of the ThingPlayer as three float64 components (x, y, z).
func (p *ThingPlayer) GetBob() (float64, float64, float64) {
	return p.bobbing.Get()
}

// GetSway retrieves the flashlight's current state as two float64 values.
func (p *ThingPlayer) GetSway() (float64, float64) {
	return p.bobbing.GetSway()
}

// GetPosition returns the player's current X, Y, and Z coordinates, adjusting for the eye height based on their state.
func (p *ThingPlayer) GetPosition() (float64, float64, float64) {
	return p.pos.X, p.pos.Y, p.pos.Z
}

// GetVisualPosition calculates and returns the player's visual position as X, Y, and Z coordinates.
func (p *ThingPlayer) GetVisualPosition() (float64, float64, float64) {
	visualX, visualY, visualZ := p.pos.X, p.pos.Y, p.pos.Z
	angleSin, angleCos := p.GetAngleFull()
	bobX, bobY, _ := p.GetBob()
	visualZ += p.getEyeHeight() + bobY + p.bobbing.GetJump()
	rightX := angleSin
	rightY := -angleCos
	visualX += bobX * rightX
	visualY += bobX * rightY
	return visualX, visualY, visualZ
}

// GetLightIntensity retrieves the current light intensity value associated with the player.
func (p *ThingPlayer) GetLightIntensity() float64 {
	return p.lightIntensity
}

// SetLightIntensity sets the light intensity level for the ThingPlayer instance.
func (p *ThingPlayer) SetLightIntensity(lightIntensity float64) {
	p.lightIntensity = lightIntensity
}

// GetRadius returns the radius of the ThingPlayer entity.
func (p *ThingPlayer) GetRadius() float64 {
	return p.entity.GetWidth() / 2
}

// GetMass returns the mass of the ThingPlayer as a float64.
func (p *ThingPlayer) GetMass() float64 {
	return p.entity.GetMass()
}

// GetVelocity returns the current velocity components (vx, vy, vz) of the player as float64 values.
func (p *ThingPlayer) GetVelocity() (float64, float64, float64) {
	return p.entity.GetVx(), p.entity.GetVy(), p.entity.GetVz()
}

// GetPitch returns the current pitch angle of the ThingPlayer as a float64 value.
func (p *ThingPlayer) GetPitch() float64 {
	return p.pitch
}

// IsMoving returns true if the ThingPlayer's associated entity is currently in motion, and false otherwise.
func (p *ThingPlayer) IsMoving() bool {
	return p.entity.IsMoving()
}

// PhysicsApply applies physics computations to the player by calculating the head position and updating its motion state.
func (p *ThingPlayer) PhysicsApply() {
	wasGrounded := p.entity.IsOnGround()
	prevVz := p.entity.GetVz()

	p.doPhysics()

	// Trigger: Atterraggio rilevato dal solver
	isGrounded := p.entity.IsOnGround()
	if !wasGrounded && isGrounded {
		// Inietta la velocità terminale reale calcolata dall'integratore per schiacciare la molla
		p.bobbing.InjectVerticalImpulse(prevVz)
	}
	// Fattore di allineamento per portare il 2.9 a ~60.0 fps o 120 fps...

	const dt = 0.016 //0.016 per 60fps

	//fmt.Printf("Vx: %f, Vy: %f, Speed: %f\n", p.entity.GetVx(), p.entity.GetVy(), p.speed)

	p.bobbing.Compute(dt, p.speed, p.entity.GetVx(), p.entity.GetVy())
}

// getEyeHeight computes the eye height of the player by considering their base height and ducking state.
func (p *ThingPlayer) getEyeHeight() float64 {
	if p.ducking {
		return p.entity.GetDepth() * 0.25
	}
	return p.entity.GetDepth() * 0.80
}

// Throw creates and spawns a projectile at a position based on the player's orientation and camera position.
func (p *ThingPlayer) Throw() {
	camX, camY, camZ := p.GetVisualPosition()
	diameter := p.entity.GetWidth()
	spawnX := camX + (p.angleCos * diameter)
	spawnY := camY + (p.angleSin * diameter)
	spawnZ := camZ - (p.getEyeHeight() * 0.5)
	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	p.LaunchObject(spawnPos, p.angle, -p.pitch)
}

// Fire triggers a hitscan action from the player's position along a calculated direction vector.
func (p *ThingPlayer) Fire() {
	// 1. Origine della vista (include Bobbing e Crouch)
	camX, camY, camZ := p.GetVisualPosition()
	// 2. Calcolo del vettore direzione (Radianti)
	// Usiamo lo yaw invertito per la telecamera e lo scaliamo per il FOV
	pitchRad := -p.pitch
	dirX := p.angleCos * math.Cos(pitchRad)
	dirY := p.angleSin * math.Cos(pitchRad)
	dirZ := math.Sin(pitchRad)
	// 3. Punto di spawn fuori dalla hitbox del player
	// Usiamo il raggio dinamico per evitare l'auto-collisione nel BVH
	weaponForward := p.GetRadius() * 2.0
	spawnX := camX + (p.angleCos * weaponForward)
	spawnY := camY + (p.angleSin * weaponForward)
	spawnZ := camZ - (p.getEyeHeight() * 0.5)

	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	p.FireHitscan(spawnPos, dirX, dirY, dirZ)
}
