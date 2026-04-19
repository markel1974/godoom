package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Bobbing represents a procedural motion system for simulating walking and vertical head movement effects.
type Bobbing struct {
	maxAmplitudeX   float64
	maxAmplitudeY   float64
	idleDrift       float64
	strideLength    float64
	speedLerp       float64
	ampLerp         float64
	smoothedSpeed   float64
	bobX            float64
	bobY            float64
	phase           float64
	ampX            float64
	ampY            float64
	impactMax       float64
	impactScale     float64
	idleAmpX        float64
	idleAmpY        float64
	springTension   float64
	springDamping   float64
	jumpBobOffset   float64 // L'offset verticale reale applicato alla telecamera
	jumpBobVelocity float64 // La velocità accumulata della molla
	swayScale       float64
	swayOffsetX     float64
	swayOffsetY     float64
	swayMultiplierX float64
	swayMultiplierY float64
}

// NewBobbing initializes and returns a new Bobbing instance with the given parameters for motion and amplitude behavior.
func NewBobbing(cfg *config.Bobbing) *Bobbing {
	return &Bobbing{
		swayMultiplierX: cfg.SwayMultiplierX,
		swayMultiplierY: cfg.SwayMultiplierY,
		swayOffsetX:     cfg.SwayOffsetX,
		swayOffsetY:     cfg.SwayOffsetY,
		swayScale:       cfg.SwayScale,
		maxAmplitudeX:   cfg.MaxAmplitudeX,
		maxAmplitudeY:   cfg.MaxAmplitudeY,
		idleDrift:       cfg.IdleDrift,
		strideLength:    cfg.StrideLength,
		speedLerp:       cfg.SpeedLerp,
		ampLerp:         cfg.AmpLerp,
		impactMax:       cfg.ImpactMax,
		impactScale:     cfg.ImpactScale,
		idleAmpX:        cfg.IdleAmpX,
		idleAmpY:        cfg.IdleAmpY,
		springTension:   cfg.SpringTension,
		springDamping:   cfg.SpringDamping,
	}
}

// InjectVerticalImpulse applies a vertical impulse to the jump bob effect, clamping extreme values to avoid excessive motion.
func (p *Bobbing) InjectVerticalImpulse(vz float64) {
	// Cappiamo l'impulso per evitare che cadute estreme portino la telecamera nel petto
	if vz < -p.impactMax {
		vz = -p.impactMax
	} else if vz > p.impactMax {
		vz = p.impactMax
	}
	// Moltiplicatore di scala: converte la velocità fisica (Vz) in forza visiva.
	// Se la Vz di caduta è -10.0, l'impulso sarà -0.5
	p.jumpBobVelocity += vz * p.impactScale
}

// Compute updates the bobbing motion based on delta time, speed, and velocity components, smoothing transitions and physics effects.
func (p *Bobbing) Compute(dt, maxSpeed, v2x, v2y float64) {
	frameMaxSpeed := maxSpeed * dt * 3.0
	rawSpeed := math.Sqrt(v2x*v2x + v2y*v2y)
	if rawSpeed > frameMaxSpeed {
		frameMaxSpeed = (frameMaxSpeed * 0.95) + (rawSpeed * 0.05)
	}
	timeLerpSpeed := math.Min(p.speedLerp*dt*60.0, 1.0)
	timeLerpAmp := math.Min(p.ampLerp*dt*60.0, 1.0)
	// Time-corrected lerp protetto
	p.smoothedSpeed = p.smoothedSpeed + (rawSpeed-p.smoothedSpeed)*timeLerpSpeed
	// La fase deve accumularsi in base al tempo, non ai frame
	p.phase += (p.idleDrift + (p.smoothedSpeed * p.strideLength)) * dt * 60.0
	ratio := 0.0
	if frameMaxSpeed > 0 {
		ratio = p.smoothedSpeed / frameMaxSpeed
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	// Lerp delle ampiezze protetto
	targetAmpX := p.idleAmpX + (ratio * (p.maxAmplitudeX - p.idleAmpX))
	p.ampX = p.ampX + (targetAmpX-p.ampX)*timeLerpAmp
	targetAmpY := p.idleAmpY + (ratio * (p.maxAmplitudeY - p.idleAmpY))
	p.ampY = p.ampY + (targetAmpY-p.ampY)*timeLerpAmp
	p.bobX = math.Cos(p.phase*0.5) * p.ampX
	p.bobY = math.Abs(math.Sin(p.phase)) * p.ampY
	// Integrazione di Eulero scalata nel tempo per la molla
	accel := -(p.jumpBobOffset * p.springTension)
	p.jumpBobVelocity += accel * dt * 60.0
	// Damping applicato come esponenziale negativo per essere framerate-independent
	p.jumpBobVelocity *= math.Pow(p.springDamping, dt*60.0)
	p.jumpBobOffset += p.jumpBobVelocity * dt * 60.0
}

// Get returns the current horizontal bob offset, vertical bob offset, and phase of the procedural bobbing motion.
func (p *Bobbing) Get() (float64, float64, float64) {
	return p.bobX, p.bobY, p.phase
}

// GetX returns the current horizontal bob offset as a float64 value.
func (p *Bobbing) GetX() float64 {
	return p.bobX
}

// GetY returns the current vertical component of the procedural bobbing motion.
func (p *Bobbing) GetY() float64 {
	return p.bobY
}

// Phase returns the current phase of the procedural bobbing motion.
func (p *Bobbing) Phase() float64 {
	return p.phase
}

// GetJump returns the current vertical offset from the jump or fall bob effect.
func (p *Bobbing) GetJump() float64 { return p.jumpBobOffset }

// GetSway calculates and returns the horizontal and vertical sway offsets as adjusted by bobbing motion and sway scale.
func (p *Bobbing) GetSway() (float64, float64) {
	x := p.swayOffsetX + ((p.bobX * p.swayMultiplierX) * p.swayScale)
	y := p.swayOffsetY - ((p.bobY * p.swayMultiplierY) * p.swayScale)
	//fmt.Printf("DEBUG BOB -> amp: %f | maxAmp: %f | idleAmp: %f\n", p.amp, p.maxAmplitude, p.idleAmp)
	return x, y
}
