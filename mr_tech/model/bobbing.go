package model

import "math"

// Bobbing represents a procedural motion system for simulating walking and vertical head movement effects.
type Bobbing struct {
	maxSpeed      float64
	maxAmplitude  float64
	idleDrift     float64
	strideLength  float64
	speedLerp     float64
	ampLerp       float64
	smoothedSpeed float64
	bob           float64
	phase         float64
	amp           float64

	// --- Jump/Fall Bob (Procedural Spring) ---
	jumpBobOffset   float64 // L'offset verticale reale applicato alla telecamera
	jumpBobVelocity float64 // La velocità accumulata della molla
}

// NewBobbing initializes and returns a new Bobbing instance with the given parameters for motion and amplitude behavior.
func NewBobbing(maxSpeed, maxAmplitude, idleDrift, strideLength, speedLerp, ampLerp float64) *Bobbing {
	return &Bobbing{
		maxSpeed:     maxSpeed,
		maxAmplitude: maxAmplitude,
		idleDrift:    idleDrift,
		strideLength: strideLength,
		speedLerp:    speedLerp,
		ampLerp:      ampLerp,
	}
}

// InjectVerticalImpulse applies a vertical impulse to the jump bob effect, clamping extreme values to avoid excessive motion.
func (p *Bobbing) InjectVerticalImpulse(vz float64) {
	// Cappiamo l'impulso per evitare che cadute estreme portino la telecamera nel petto
	const maxImpact = 20.0
	if vz < -maxImpact {
		vz = -maxImpact
	} else if vz > maxImpact {
		vz = maxImpact
	}

	// Moltiplicatore di scala: converte la velocità fisica (Vz) in forza visiva.
	// Se la Vz di caduta è -10.0, l'impulso sarà -0.5
	const impactScale = 0.05
	p.jumpBobVelocity += vz * impactScale
}

// Compute updates the procedural bobbing motion based on the given 2D velocity components (v2x, v2y).
func (p *Bobbing) Compute(v2x float64, v2y float64) {
	rawSpeed := math.Sqrt(v2x*v2x + v2y*v2y)
	if rawSpeed > p.maxSpeed {
		p.maxSpeed = (p.maxSpeed * 0.95) + (rawSpeed * 0.05)
	}
	p.smoothedSpeed = (p.smoothedSpeed * (1.0 - p.speedLerp)) + (rawSpeed * p.speedLerp)
	p.phase += p.idleDrift + (p.smoothedSpeed * p.strideLength)
	ratio := 0.0
	if p.maxSpeed > 0 {
		ratio = p.smoothedSpeed / p.maxSpeed
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	// Impediamo all'ampiezza di scendere a zero. Da fermo (ratio=0), l'ampiezza
	// sarà 0.3 (respiro corto). In corsa (ratio=1), scala fluidamente fino a maxAmplitude.
	const idleAmp = 0.3
	targetAmp := idleAmp + (ratio * (p.maxAmplitude - idleAmp))
	p.amp = (p.amp * (1.0 - p.ampLerp)) + (targetAmp * p.ampLerp)
	p.bob = math.Sin(p.phase) * p.amp
	// --- Jump/Fall Bob (Procedural Spring) ---
	const springTension = 0.15
	const springDamping = 0.75
	p.jumpBobVelocity -= p.jumpBobOffset * springTension
	p.jumpBobVelocity *= springDamping
	p.jumpBobOffset += p.jumpBobVelocity
}

// GetBob returns the current vertical bobbing displacement as a float64 value.
func (p *Bobbing) GetBob() float64 { return p.bob }

// GetPhase returns the current phase of the bobbing motion as a float64.
func (p *Bobbing) GetPhase() float64 { return p.phase }

// GetJumpBob returns the current vertical offset applied to the camera due to procedural spring-based jumping effects.
func (p *Bobbing) GetJumpBob() float64 { return p.jumpBobOffset }
