package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/lights"
)

// ThingPlayer represents a controllable entity with movement, physics, and gameplay-related properties.
type ThingPlayer struct {
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
	visible        bool
	debug          bool
	volumes        *Volumes
	things         *Things
	cfg            *config.Player
	*ThingBase
}

// NewThingPlayer creates and initializes a new ThingPlayer entity using the provided configuration, world, and things.
// It ensures the player is placed in a valid sector and properly configures position, angle, and other properties.
// Returns the initialized ThingPlayer or an error if the player's sector is not found or configuration fails.
func NewThingPlayer(volumes *Volumes, things *Things, c *config.Player, debug bool) *ThingPlayer {
	thing := &ThingPlayer{
		volumes:        volumes,
		things:         things,
		cfg:            c,
		pitch:          0,
		pitchState:     0,
		bobbing:        NewBobbing(c.Bobbing),
		lightIntensity: 0.0039,
		debug:          debug,
		flash:          NewFlash(c.Flash),
		pitchMin:       -5.0,
		pitchMax:       5.0,
		pitchSens:      0.05,
		visible:        c.IsVisible,
	}
	return thing
}

// Setup initializes the ThingPlayer instance and prepares it for operation, returning an error if initialization fails.
func (pr *ThingPlayer) Setup() error {
	location, _ := pr.volumes.QueryPoint(pr.cfg.Position.X, pr.cfg.Position.Y, pr.cfg.Position.Z)
	if location == nil {
		return fmt.Errorf("can't find 3d player location at X: %f Y: %f Z: %f\n", pr.cfg.Position.X, pr.cfg.Position.Y, pr.cfg.Position.Z)
	}
	pr.ThingBase = NewThingBase(pr, pr.things, pr.cfg.Thing, location)
	entity := pr.GetEntity()
	entity.SetOnGround(false)
	entity.MoveTo(pr.cfg.Position.X, pr.cfg.Position.Y, pr.cfg.Position.Z)
	pr.SetAngle(pr.cfg.Angle)

	pr.things.SetPlayer(pr)
	return nil
}

// IsActive determines whether the ThingPlayer instance is currently active, returning true if active, otherwise false.
func (pr *ThingPlayer) IsActive() bool {
	return true
}

// IsVisible determines whether the ThingPlayer instance is currently visible.
func (pr *ThingPlayer) IsVisible() bool {
	return pr.visible
}

// PostMessage sends the provided ThingEvent to the ThingPlayer's inbox channel for processing.
func (pr *ThingPlayer) PostMessage(ec *ThingEvent) {
	pr.inbox <- ec
}

// StagePrepare prepares the stage for the ThingPlayer by invoking the StagePrepare method of its ThingBase.
func (pr *ThingPlayer) StagePrepare() bool {
	_ = pr.ThingBase.StagePrepare()
	return true
}

// StartLoop initializes and starts a concurrent processing loop for handling incoming events and player state updates.
func (pr *ThingPlayer) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-pr.inbox:
				switch evt.GetKind() {
				case StageThinking:
					pr.StageThinking(evt.GetCoords())
				case StageCompute:
					//p.StageCompute()
				case StageResolve:
					pr.StageResolve(evt.GetSolverIndex(), evt.GetSolverJitter())
				case StageApply:
					pr.StageApply(evt.GetSolverJitter())
				}
				evt.Done()
			case <-pr.done:
				return
			}
		}
	}()
}

// GetLight retrieves the light object associated with the ThingPlayer.
func (pr *ThingPlayer) GetLight() *lights.Light {
	return nil
}

// AddAngle adjusts the player's current angle by the specified value in radians, updating related trigonometric properties.
func (pr *ThingPlayer) AddAngle(angle float64) {
	pr.SetAngle(pr.angle + angle)
}

// SetAngle updates the player's orientation angle and recalculates the sine and cosine of the angle.
func (pr *ThingPlayer) SetAngle(angle float64) {
	pr.angle = angle
	pr.angleSin = math.Sin(pr.angle)
	pr.angleCos = math.Cos(pr.angle)
}

// GetAngleFull returns the sine and cosine of the current angle of the ThingPlayer as two float64 values.
func (pr *ThingPlayer) GetAngleFull() (float64, float64) {
	return pr.angleSin, pr.angleCos
}

// SetPitch adjusts the player's pitch angle within the allowed range, affecting vertical view orientation.
func (pr *ThingPlayer) SetPitch(y float64) {
	pr.pitchState = geometry.ClampF(pr.pitchState-(y*pr.pitchSens), pr.pitchMin, pr.pitchMax)
	pr.pitch = pr.pitchState
}

// SetPitchOptions configures the minimum, maximum, and sensitivity values for pitch adjustment.
func (pr *ThingPlayer) SetPitchOptions(min, max, sens float64) {
	pr.pitchMin = min
	pr.pitchMax = max
	pr.pitchSens = sens
}

// Move applies a directional impulse to the player based on input flags (up, down, left, right) and a given impulse magnitude.
func (pr *ThingPlayer) Move(impulse float64, up, down, left, right bool) {
	if !up && !down && !left && !right {
		return
	}
	var fx, fy float64
	if up {
		fx += pr.angleCos
		fy += pr.angleSin
	}
	if down {
		fx -= pr.angleCos
		fy -= pr.angleSin
	}
	if left {
		fx += pr.angleSin
		fy -= pr.angleCos
	}
	if right {
		fx -= pr.angleSin
		fy += pr.angleCos
	}
	if mag := math.Sqrt(fx*fx + fy*fy); mag > 0 {
		// Vettore direzione puro (Normalizzato)
		dirX := fx / mag
		dirY := fy / mag
		pr.MoveTowards(dirX, dirY, pr.speed*impulse, pr.speed)
	}
}

// SetJump applies an upward force to make the ThingPlayer jump.
func (pr *ThingPlayer) SetJump(multi bool) {
	factor := 1.0
	if multi {
		factor = 0.2
		pr.GetEntity().SetOnGround(true)
	}
	if pr.Jump(0, 0, factor) {
		pr.bobbing.InjectVerticalImpulse(1.5)
	}
}

// SetDucking toggles the player's ducking state between true and false.
func (pr *ThingPlayer) SetDucking() {
	entity := pr.GetEntity()
	pr.ducking = !pr.ducking
	if pr.ducking {
		entity.SetSize(entity.GetWidth(), entity.GetHeight(), entity.GetDepth()*0.5)
		//p.pos.Z -= p.entity.GetDepth() * 0.25
	} else {
		//p.pos.Z += p.entity.GetDepth() * 0.25
		entity.SetSize(entity.GetWidth(), entity.GetHeight(), entity.GetDepth()*2.0)
	}
}

// GetFlash retrieves the flash instance associated with the ThingPlayer.
func (pr *ThingPlayer) GetFlash() *Flash {
	return pr.flash
}

// GetBob retrieves the current bobbing values of the ThingPlayer as three float64 components (x, y, z).
func (pr *ThingPlayer) GetBob() (float64, float64, float64) {
	return pr.bobbing.Get()
}

// GetSway retrieves the flashlight's current state as two float64 values.
func (pr *ThingPlayer) GetSway() (float64, float64, float64) {
	return pr.bobbing.GetSway()
}

func (pr *ThingPlayer) GetTilt() float64 {
	//if !p.entity.IsOnGround() {
	//	return 0.0
	//}

	entity := pr.GetEntity()
	rawTilt := pr.bobbing.GetTilt()
	// 3. Calcoliamo la velocità reale sul piano
	vx := entity.GetVx()
	vy := entity.GetVy()
	currentSpeed := math.Sqrt(vx*vx + vy*vy)
	// 4. Creiamo la maschera (Ratio)
	// p.speed è la tua velocità massima (es. 60.0).
	// Vogliamo che il tilt sia al 100% già quando siamo a mezza velocità
	ratio := currentSpeed / (pr.speed * 0.5)
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
func (pr *ThingPlayer) GetVisualPosition() (float64, float64, float64) {
	visualX, visualY, visualZ := pr.GetEntity().GetBottomCenter() //p.pos.X, p.pos.Y, p.pos.Z
	angleSin, angleCos := pr.GetAngleFull()
	bobX, bobY, _ := pr.GetBob()
	visualZ += pr.getEyeHeight() + bobY + pr.bobbing.GetJump()
	rightX := angleSin
	rightY := -angleCos
	visualX += bobX * rightX
	visualY += bobX * rightY
	return visualX, visualY, visualZ
}

// GetLightIntensity retrieves the current light intensity value associated with the player.
func (pr *ThingPlayer) GetLightIntensity() float64 {
	return pr.lightIntensity
}

// SetLightIntensity sets the light intensity level for the ThingPlayer instance.
func (pr *ThingPlayer) SetLightIntensity(lightIntensity float64) {
	pr.lightIntensity = lightIntensity
}

// GetPitch returns the current pitch angle of the ThingPlayer as a float64 value.
func (pr *ThingPlayer) GetPitch() float64 {
	return pr.pitch
}

// HasThinking determines if the ThingPlayer instance has a thinking function assigned. Returns true if it is set.
func (pr *ThingPlayer) HasThinking() bool {
	return true
}

// StageThinking updates the player's internal state based on the provided x, y, and z coordinates.
func (pr *ThingPlayer) StageThinking(playerX float64, playerY float64, playerZ float64) {
	//
}

// StageApply processes the physics-related updates for the entity, including ground detection and velocity adjustments.
func (pr *ThingPlayer) StageApply(solverJitter float64) {
	entity := pr.GetEntity()
	wasGrounded := entity.IsOnGround()
	prevVz := entity.GetVz()
	pr.ThingBase.StageApply(solverJitter)
	// Trigger: Atterraggio rilevato dal solver
	isGrounded := entity.IsOnGround()
	if !wasGrounded && isGrounded {
		// Inietta la velocità terminale reale calcolata dall'integratore per schiacciare la molla
		pr.bobbing.InjectVerticalImpulse(prevVz)
	}
	pr.bobbing.Compute(entity.GetDt(), pr.speed, entity.GetVx(), entity.GetVy())
}

// getEyeHeight computes the eye height of the player by considering their base height and ducking state.
func (pr *ThingPlayer) getEyeHeight() float64 {
	entity := pr.GetEntity()
	if pr.ducking {
		return entity.GetDepth() * 0.25
	}
	return entity.GetDepth() * 0.80
}

// Throw creates and spawns a projectile at a position based on the player's orientation and camera position.
func (pr *ThingPlayer) Throw(throwableIndex int, speed float64) {
	camX, camY, camZ := pr.GetVisualPosition()
	entity := pr.GetEntity()
	diameter := entity.GetWidth()
	spawnX := camX + (pr.angleCos * diameter)
	spawnY := camY + (pr.angleSin * diameter)
	spawnZ := camZ - (pr.getEyeHeight() * 0.5)
	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	pr.LaunchObject(throwableIndex, pr.onCollision, pr.onImpact, spawnPos, pr.angle, -pr.pitch, speed)
}

// Fire triggers a hitscan action from the player's position along a calculated direction vector.
func (pr *ThingPlayer) Fire(id string) {
	// Origine della vista (include Bobbing e Crouch)
	camX, camY, camZ := pr.GetVisualPosition()
	// Calcolo del vettore direzione (Radianti)
	// Usiamo lo angle invertito per la telecamera e lo scaliamo per il FOV
	pitchRad := -pr.pitch
	dirX := pr.angleCos * math.Cos(pitchRad)
	dirY := pr.angleSin * math.Cos(pitchRad)
	dirZ := math.Sin(pitchRad)
	// Punto di spawn fuori dalla hitbox del player
	// Usiamo il raggio dinamico per evitare l'auto-collisione nel BVH
	weaponForward := pr.GetEntity().GetWidth()
	const weaponForce = 50000.0 * 10.0 // TODO CONFIG
	spawnX := camX + (pr.angleCos * weaponForward)
	spawnY := camY + (pr.angleSin * weaponForward)
	spawnZ := camZ - (pr.getEyeHeight() * 0.5)
	spawnPos := geometry.XYZ{X: spawnX, Y: spawnY, Z: spawnZ}
	pr.FireHitscan(id, spawnPos, weaponForce, dirX, dirY, dirZ)
}
