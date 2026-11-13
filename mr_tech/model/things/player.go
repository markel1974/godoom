package things

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

// Player represents a controllable entity with movement, physics, and gameplay-related properties.
type Player struct {
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
	volumes        IVolumeHandler
	things         IThingHandler
	cfg            *config.Player
	*Base
}

// NewPlayer creates and initializes a new Player entity using the provided configuration, world, and things.
// It ensures the player is placed in a valid sector and properly configures position, angle, and other properties.
// Returns the initialized Player or an error if the player's sector is not found or configuration fails.
func NewPlayer(volumes IVolumeHandler, things IThingHandler, c *config.Player, debug bool) *Player {
	player := &Player{
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
	return player
}

// Setup initializes the Player instance and prepares it for operation, returning an error if initialization fails.
func (pr *Player) Setup() error {
	location, _ := pr.volumes.QueryPoint(pr.cfg.Position.X, pr.cfg.Position.Y, pr.cfg.Position.Z)
	if location == nil {
		return fmt.Errorf("can't find 3d player location at X: %f Y: %f Z: %f\n", pr.cfg.Position.X, pr.cfg.Position.Y, pr.cfg.Position.Z)
	}
	pr.Base = NewBase(pr, pr.things, pr.cfg.Thing, location)
	pr.cfg.Thing.OnBind(pr)
	entity := pr.GetEntity()
	entity.MoveTo(pr.cfg.Position.X, pr.cfg.Position.Y, pr.cfg.Position.Z)
	pr.SetAngle(pr.cfg.Angle)

	//pr.things.SetPlayer(pr)
	return nil
}

// IsActive determines whether the Player instance is currently active, returning true if active, otherwise false.
func (pr *Player) IsActive() bool {
	return true
}

// IsVisible determines whether the Player instance is currently visible.
func (pr *Player) IsVisible() bool {
	return pr.visible
}

// IsEqual reports whether other is the same Player instance.
func (pr *Player) IsEqual(other volumes.IThing) bool {
	if v, ok := other.(*Player); ok && pr == v {
		return true
	}
	return false
}

// Jump attempts to make the enemy leap with the given horizontal movement and vertical scaling, reporting success.
func (pr *Player) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return pr.Base.Jump(pr, leapX, leapY, zFactor)
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (pr *Player) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
	pr.Base.FireHitscan(pr, id, pos, force, dirX, dirY, dirZ)
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (pr *Player) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
	pr.things.CreateThrowable(pr, throwableIndex, onCollision, onImpact, pr.location, pos, angle, pitch, speed)
}

// PostMessage sends the provided ThingEvent to the Player's inbox channel for processing.
func (pr *Player) PostMessage(ec *volumes.ThingEvent) {
	pr.inbox <- ec
}

// StagePrepare prepares the stage for the Player by invoking the StagePrepare method of its Base.
func (pr *Player) StagePrepare(session uint64) {
	pr.GetEntity().StopAngular()
	pr.Base.StagePrepare(session)
}

// StartLoop initializes and starts a concurrent processing loop for handling incoming events and player state updates.
func (pr *Player) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-pr.inbox:
				switch evt.GetKind() {
				case volumes.StageThinking:
					pr.StageThinking(evt.GetCoords())
				case volumes.StageCompute:
					//p.StageCompute()
				case volumes.StageResolve:
					pr.StageResolve()
				case volumes.StageApply:
					pr.StageApply()
				}
				evt.Done()
			case <-pr.done:
				return
			}
		}
	}()
}

// GetLight retrieves the light object associated with the Player.
func (pr *Player) GetLight() *lights.Light {
	return nil
}

// AddAngle adjusts the player's current angle by the specified value in radians, updating related trigonometric properties.
func (pr *Player) AddAngle(angle float64) {
	pr.SetAngle(pr.angle + angle)
}

// SetAngle updates the player's orientation angle and recalculates the sine and cosine of the angle.
func (pr *Player) SetAngle(angle float64) {
	pr.Base.SetAngle(angle)
	pr.angleSin = math.Sin(pr.angle)
	pr.angleCos = math.Cos(pr.angle)
}

// GetAngleFull returns the sine and cosine of the current angle of the Player as two float64 values.
func (pr *Player) GetAngleFull() (float64, float64) {
	return pr.angleSin, pr.angleCos
}

// SetPitch adjusts the player's pitch angle within the allowed range, affecting vertical view orientation.
func (pr *Player) SetPitch(y float64) {
	pr.pitchState = geometry.ClampF(pr.pitchState-(y*pr.pitchSens), pr.pitchMin, pr.pitchMax)
	pr.pitch = pr.pitchState
}

// SetPitchOptions configures the minimum, maximum, and sensitivity values for pitch adjustment.
func (pr *Player) SetPitchOptions(min, max, sens float64) {
	pr.pitchMin = min
	pr.pitchMax = max
	pr.pitchSens = sens
}

// Move applies a directional impulse to the player based on input flags (up, down, left, right) and a given impulse magnitude.
func (pr *Player) Move(impulse float64, up, down, left, right bool) {
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

// SetJump applies an upward force to make the Player jump.
func (pr *Player) SetJump(multi bool) {
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
func (pr *Player) SetDucking() {
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

// GetFlash retrieves the flash instance associated with the Player.
func (pr *Player) GetFlash() *Flash {
	return pr.flash
}

// GetBob retrieves the current bobbing values of the Player as three float64 components (x, y, z).
func (pr *Player) GetBob() (float64, float64, float64) {
	return pr.bobbing.Get()
}

// GetSway retrieves the flashlight's current state as two float64 values.
func (pr *Player) GetSway() (float64, float64, float64) {
	return pr.bobbing.GetSway()
}

func (pr *Player) GetTilt() float64 {
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
func (pr *Player) GetVisualPosition() (float64, float64, float64) {
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
func (pr *Player) GetLightIntensity() float64 {
	return pr.lightIntensity
}

// SetLightIntensity sets the light intensity level for the Player instance.
func (pr *Player) SetLightIntensity(lightIntensity float64) {
	pr.lightIntensity = lightIntensity
}

// GetPitch returns the current pitch angle of the Player as a float64 value.
func (pr *Player) GetPitch() float64 {
	return pr.pitch
}

// HasThinking determines if the Player instance has a thinking function assigned. Returns true if it is set.
func (pr *Player) HasThinking() bool {
	return true
}

// StageThinking updates the player's internal state based on the provided x, y, and z coordinates.
func (pr *Player) StageThinking(playerX float64, playerY float64, playerZ float64) {
	//
}

// StageApply processes the physics-related updates for the entity, including ground detection and velocity adjustments.
func (pr *Player) StageApply() {
	entity := pr.GetEntity()
	wasGrounded := entity.IsOnGround()
	prevVz := entity.GetVz()
	pr.Base.StageApply()
	// Trigger: Atterraggio rilevato dal solver
	isGrounded := entity.IsOnGround()
	if !wasGrounded && isGrounded {
		// Inietta la velocità terminale reale calcolata dall'integratore per schiacciare la molla
		pr.bobbing.InjectVerticalImpulse(prevVz)
	}
	pr.bobbing.Compute(entity.GetDt(), pr.speed, entity.GetVx(), entity.GetVy())
}

// getEyeHeight computes the eye height of the player by considering their base height and ducking state.
func (pr *Player) getEyeHeight() float64 {
	entity := pr.GetEntity()
	if pr.ducking {
		return entity.GetDepth() * 0.25
	}
	return entity.GetDepth() * 0.80
}

// Throw creates and spawns a projectile at a position based on the player's orientation and camera position.
func (pr *Player) Throw(throwableIndex int, speed float64) {
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
func (pr *Player) Fire(id string) {
	// Origine della vista (camera eye, include Bobbing e Crouch)
	camX, camY, camZ := pr.GetVisualPosition()
	// Calcolo del vettore direzione (Radianti)
	// Usiamo lo angle invertito per la telecamera e lo scaliamo per il FOV
	pitchRad := -pr.pitch
	dirX := pr.angleCos * math.Cos(pitchRad)
	dirY := pr.angleSin * math.Cos(pitchRad)
	dirZ := math.Sin(pitchRad)

	const weaponForce = 50000.0 * 10.0 // TODO CONFIG
	// Il raggio hitscan parte direttamente dal punto di vista (camera eye) per collimare
	// perfettamente con il mirino a schermo. Il self-culling è gestito nativamente in Base.FireHitscan.
	spawnPos := geometry.XYZ{X: camX, Y: camY, Z: camZ}
	pr.FireHitscan(id, spawnPos, weaponForce, dirX, dirY, dirZ)
}
