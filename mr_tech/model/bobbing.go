package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Bobbing represents a procedural motion system for simulating walking and vertical head movement effects.
type Bobbing struct {
	maxAmplitude  float64
	idleDrift     float64
	strideLength  float64
	speedLerp     float64
	ampLerp       float64
	smoothedSpeed float64
	bob           float64
	phase         float64
	amp           float64

	impactMax   float64
	impactScale float64
	idleAmp     float64

	springTension float64
	springDamping float64

	// --- Jump/Fall Bob (Procedural Spring) ---
	jumpBobOffset   float64 // L'offset verticale reale applicato alla telecamera
	jumpBobVelocity float64 // La velocità accumulata della molla
}

// NewBobbing initializes and returns a new Bobbing instance with the given parameters for motion and amplitude behavior.
func NewBobbing(cfg *config.Bobbing) *Bobbing {
	return &Bobbing{
		maxAmplitude:  cfg.MaxAmplitude,
		idleDrift:     cfg.IdleDrift,
		strideLength:  cfg.StrideLength,
		speedLerp:     cfg.SpeedLerp,
		ampLerp:       cfg.AmpLerp,
		impactMax:     cfg.ImpactMax,
		impactScale:   cfg.ImpactScale,
		idleAmp:       cfg.IdleAmp,
		springTension: cfg.SpringTension,
		springDamping: cfg.SpringDamping,
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

// Compute updates the procedural bobbing motion based on the given 2D velocity components (v2x, v2y).
func (p *Bobbing) Compute(maxSpeed, v2x, v2y float64) {
	rawSpeed := math.Sqrt(v2x*v2x + v2y*v2y)
	if rawSpeed > maxSpeed {
		maxSpeed = (maxSpeed * 0.95) + (rawSpeed * 0.05)
	}
	p.smoothedSpeed = (p.smoothedSpeed * (1.0 - p.speedLerp)) + (rawSpeed * p.speedLerp)
	p.phase += p.idleDrift + (p.smoothedSpeed * p.strideLength)
	ratio := 0.0
	if maxSpeed > 0 {
		ratio = p.smoothedSpeed / maxSpeed
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	// Impediamo all'ampiezza di scendere a zero. Da fermo (ratio=0), l'ampiezza
	// sarà 0.3 (respiro corto). In corsa (ratio=1), scala fluidamente fino a maxAmplitude.
	targetAmp := p.idleAmp + (ratio * (p.maxAmplitude - p.idleAmp))
	p.amp = (p.amp * (1.0 - p.ampLerp)) + (targetAmp * p.ampLerp)
	p.bob = math.Sin(p.phase) * p.amp
	// --- Jump/Fall Bob (Procedural Spring) ---
	p.jumpBobVelocity -= p.jumpBobOffset * p.springTension
	p.jumpBobVelocity *= p.springDamping
	p.jumpBobOffset += p.jumpBobVelocity
}

// GetBob returns the current vertical bobbing displacement as a float64 value.
func (p *Bobbing) GetBob() float64 { return p.bob }

// GetPhase returns the current phase of the bobbing motion as a float64.
func (p *Bobbing) GetPhase() float64 { return p.phase }

// GetJumpBob returns the current vertical offset applied to the camera due to procedural spring-based jumping effects.
func (p *Bobbing) GetJumpBob() float64 { return p.jumpBobOffset }
