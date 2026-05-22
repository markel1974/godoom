package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
)

// ThingPlayer represents a controllable entity with movement, physics, and gameplay-related properties.
type ThingPlayer struct {
	kind           int
	angleSin       float64
	angleCos       float64
	pitch          float64
	pitchState     float64
	pitchMin       float64
	pitchMax       float64
	pitchSens      float64
	ducking        bool
	lightIntensity float64
	bobbing        *Bobbing
	flash          *Flash
	debug          bool
	*ThingBase
}

// NewThingPlayer creates and initializes a new ThingPlayer entity using the provided configuration, world, and things.
// It ensures the player is placed in a valid sector and properly configures position, angle, and other properties.
// Returns the initialized ThingPlayer or an error if the player's sector is not found or configuration fails.
func NewThingPlayer(things *Things, c *config.Player, volumes *Volumes, debug bool) *ThingPlayer {
	volume := volumes.LocateVolume(c.Position.X, c.Position.Y, c.Position.Z)
	if volume == nil {
		fmt.Printf("can't find 3d player location at X: %f Y: %f Z: %f\n", c.Position.X, c.Position.Y, c.Position.Z)
		return nil
	}
	c.Kind = config.ThingPlayerDef
	if c.Height <= 0 {
		panic("player height must be positive")
	}
	if c.Speed <= 0 {
		panic("player speed must be positive")
	}
	if c.Mass <= 0 {
		panic("player mass must be positive")
	}
	c.Id = "PLAYER"

	c.Position = geometry.XYZ{X: c.Position.X, Y: c.Position.Y, Z: c.Position.Z}
	thing := &ThingPlayer{
		kind:           0,
		pitch:          0,
		pitchState:     0,
		bobbing:        NewBobbing(c.Bobbing),
		lightIntensity: 0.0039,
		debug:          debug,
		flash:          NewFlash(c.Flash),
		pitchMin:       -5.0,
		pitchMax:       5.0,
		pitchSens:      0.05,
	}
	thing.ThingBase = NewThingBase(thing, things, c.Thing, volume)
	thing.GetEntity().MoveTo(c.Position.X, c.Position.Y, c.Position.Z)
	thing.SetAngle(c.Angle)
	return thing
}

// IsActive determines whether the ThingPlayer instance is currently active, returning true if active, otherwise false.
func (p *ThingPlayer) IsActive() bool {
	return true
}

// PostMessage sends the provided ThingEvent to the ThingPlayer's inbox channel for processing.
func (p *ThingPlayer) PostMessage(ec *ThingEvent) {
	p.inbox <- ec
}

// StartLoop initializes and starts a concurrent processing loop for handling incoming events and player state updates.
func (p *ThingPlayer) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-p.inbox:
				switch evt.GetKind() {
				case StageThinking:
					p.StageThinking(evt.GetCoords())
				case StageCompute:
					p.StageCompute()
				case StageResolve:
					p.StageResolve(evt.GetSolverJitter())
				case StageApply:
					p.StageApply()
				}
				evt.Done()
			case <-p.done:
				return
			}
		}
	}()
}

// GetLight retrieves the light object associated with the ThingPlayer.
func (p *ThingPlayer) GetLight() *Light {
	return nil
}

// AddAngle adjusts the player's current angle by the specified value in radians, updating related trigonometric properties.
func (p *ThingPlayer) AddAngle(angle float64) {
	p.SetAngle(p.angle + angle)
}

// SetAngle updates the player's orientation angle and recalculates the sine and cosine of the angle.
func (p *ThingPlayer) SetAngle(angle float64) {
	p.angle = angle
	p.angleSin = math.Sin(p.angle)
	p.angleCos = math.Cos(p.angle)
}

// GetAngleFull returns the sine and cosine of the current angle of the ThingPlayer as two float64 values.
func (p *ThingPlayer) GetAngleFull() (float64, float64) {
	return p.angleSin, p.angleCos
}

// SetPitch adjusts the player's pitch angle within the allowed range, affecting vertical view orientation.
func (p *ThingPlayer) SetPitch(y float64) {
	p.pitchState = mathematic.ClampF(p.pitchState-(y*p.pitchSens), p.pitchMin, p.pitchMax)
	p.pitch = p.pitchState
}

// SetPitchOptions configures the minimum, maximum, and sensitivity values for pitch adjustment.
func (p *ThingPlayer) SetPitchOptions(min, max, sens float64) {
	p.pitchMin = min
	p.pitchMax = max
	p.pitchSens = sens
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
		// Vettore direzione puro (Normalizzato)
		dirX := fx / mag
		dirY := fy / mag
		p.MoveTowards(dirX, dirY, p.speed*impulse, p.speed)
	}
}

// SetJump applies an upward force to make the ThingPlayer jump.
func (p *ThingPlayer) SetJump(multi bool) {
	onGround := true
	entity := p.GetEntity()
	mass := entity.GetMass()
	fz := mass * p.jumpForce
	if !multi {
		onGround = entity.IsOnGround()
	} else {
		fz *= 0.2
	}
	if onGround {
		entity.AddForce(0.0, 0.0, fz)
		entity.SetOnGround(false)
		p.bobbing.InjectVerticalImpulse(-1.5)
	}
}

// SetDucking toggles the player's ducking state between true and false.
func (p *ThingPlayer) SetDucking() {
	entity := p.GetEntity()
	p.ducking = !p.ducking
	if p.ducking {
		entity.SetSize(entity.GetWidth(), entity.GetHeight(), entity.GetDepth()*0.5)
		//p.pos.Z -= p.entity.GetDepth() * 0.25
	} else {
		//p.pos.Z += p.entity.GetDepth() * 0.25
		entity.SetSize(entity.GetWidth(), entity.GetHeight(), entity.GetDepth()*2.0)
	}
}

// GetFlash retrieves the flash instance associated with the ThingPlayer.
func (p *ThingPlayer) GetFlash() *Flash {
	return p.flash
}

// GetBob retrieves the current bobbing values of the ThingPlayer as three float64 components (x, y, z).
func (p *ThingPlayer) GetBob() (float64, float64, float64) {
	return p.bobbing.Get()
}

// GetSway retrieves the flashlight's current state as two float64 values.
func (p *ThingPlayer) GetSway() (float64, float64, float64) {
	return p.bobbing.GetSway()
}

func (p *ThingPlayer) GetTilt() float64 {
	//if !p.entity.IsOnGround() {
	//	return 0.0
	//}

	entity := p.GetEntity()
	rawTilt := p.bobbing.GetTilt()
	// 3. Calcoliamo la velocità reale sul piano
	vx := entity.GetVx()
	vy := entity.GetVy()
	currentSpeed := math.Sqrt(vx*vx + vy*vy)
	// 4. Creiamo la maschera (Ratio)
	// p.speed è la tua velocità massima (es. 60.0).
	// Vogliamo che il tilt sia al 100% già quando siamo a mezza velocità
	ratio := currentSpeed / (p.speed * 0.5)
	if ratio > 1.0 {
		ratio = 1.0
	} else if ratio < 0.05 {
		// Deadzone: Ignoriamo le micro-vibrazioni del solutore PGS quando siamo "fermi"
		ratio = 0.0
	}
	// 5. Applichiamo la maschera! Da fermo, ratio sarà 0.0 annullando l'idleDrift.
	return rawTilt * ratio
}

// GetVisualPosition calculates and returns the player's visual position as X, Y, and Z coordinates.
func (p *ThingPlayer) GetVisualPosition() (float64, float64, float64) {
	visualX, visualY, visualZ := p.GetBottomCenter() //p.pos.X, p.pos.Y, p.pos.Z
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

// GetPitch returns the current pitch angle of the ThingPlayer as a float64 value.
func (p *ThingPlayer) GetPitch() float64 {
	return p.pitch
}

// StageThinking updates the player's internal state based on the provided x, y, and z coordinates.
func (p *ThingPlayer) StageThinking(playerX float64, playerY float64, playerZ float64) {
	//
}

// StageApply processes the physics-related updates for the entity, including ground detection and velocity adjustments.
func (p *ThingPlayer) StageApply() {
	entity := p.GetEntity()
	wasGrounded := entity.IsOnGround()
	prevVz := entity.GetVz()
	p.ThingBase.StageApply()
	// Trigger: Atterraggio rilevato dal solver
	isGrounded := entity.IsOnGround()
	if !wasGrounded && isGrounded {
		// Inietta la velocità terminale reale calcolata dall'integratore per schiacciare la molla
		p.bobbing.InjectVerticalImpulse(prevVz)
	}
	// Fattore di allineamento per portare il 2.9 a ~60.0 fps o 120 fps...
	const dt = 0.016 //0.016 per 60fps
	//fmt.Printf("Vx: %f, Vy: %f, Speed: %f\n", p.entity.GetVx(), p.entity.GetVy(), p.speed)
	p.bobbing.Compute(dt, p.speed, entity.GetVx(), entity.GetVy())
}

// getEyeHeight computes the eye height of the player by considering their base height and ducking state.
func (p *ThingPlayer) getEyeHeight() float64 {
	entity := p.GetEntity()
	if p.ducking {
		return entity.GetDepth() * 0.25
	}
	return entity.GetDepth() * 0.80
}

// Throw creates and spawns a projectile at a position based on the player's orientation and camera position.
func (p *ThingPlayer) Throw(throwableIndex int, speed float64) {
	camX, camY, camZ := p.GetVisualPosition()
	entity := p.GetEntity()
	diameter := entity.GetWidth()
	spawnX := camX + (p.angleCos * diameter)
	spawnY := camY + (p.angleSin * diameter)
	spawnZ := camZ - (p.getEyeHeight() * 0.5)
	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	p.LaunchObject(throwableIndex, p.onCollision, p.onImpact, spawnPos, p.angle, -p.pitch, speed)
}

// Fire triggers a hitscan action from the player's position along a calculated direction vector.
func (p *ThingPlayer) Fire(id string) {
	// 1. Origine della vista (include Bobbing e Crouch)
	camX, camY, camZ := p.GetVisualPosition()
	// 2. Calcolo del vettore direzione (Radianti)
	// Usiamo lo angle invertito per la telecamera e lo scaliamo per il FOV
	pitchRad := -p.pitch
	dirX := p.angleCos * math.Cos(pitchRad)
	dirY := p.angleSin * math.Cos(pitchRad)
	dirZ := math.Sin(pitchRad)
	// 3. Punto di spawn fuori dalla hitbox del player
	// Usiamo il raggio dinamico per evitare l'auto-collisione nel BVH
	weaponForward := p.GetRadius() * 2.0
	weaponForce := 5000.0 // TODO CONFIG
	spawnX := camX + (p.angleCos * weaponForward)
	spawnY := camY + (p.angleSin * weaponForward)
	spawnZ := camZ - (p.getEyeHeight() * 0.5)
	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	p.FireHitscan(id, spawnPos, weaponForce, dirX, dirY, dirZ)
}
