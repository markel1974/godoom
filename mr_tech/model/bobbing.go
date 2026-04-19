package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Bobbing represents a procedural motion system for simulating walking and vertical head movement effects.
type Bobbing struct {
	maxAmplitudeX float64
	maxAmplitudeY float64
	idleDrift     float64
	strideLength  float64
	speedLerp     float64
	ampLerp       float64
	smoothedSpeed float64
	bobX          float64
	bobY          float64
	phase         float64
	ampX          float64
	ampY          float64
	impactMax     float64
	impactScale   float64
	idleAmp       float64
	springTension float64
	springDamping float64
	// --- Jump/Fall Bob (Procedural Spring) ---
	jumpBobOffset   float64 // L'offset verticale reale applicato alla telecamera
	jumpBobVelocity float64 // La velocità accumulata della molla
	swayScale       float64
}

// NewBobbing initializes and returns a new Bobbing instance with the given parameters for motion and amplitude behavior.
func NewBobbing(cfg *config.Bobbing) *Bobbing {
	return &Bobbing{
		swayScale:     cfg.SwayScale,
		maxAmplitudeX: cfg.MaxAmplitudeX,
		maxAmplitudeY: cfg.MaxAmplitudeY,
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

	targetAmpX := p.idleAmp + (ratio * (p.maxAmplitudeX - p.idleAmp))
	p.ampX = (p.ampX * (1.0 - p.ampLerp)) + (targetAmpX * p.ampLerp)

	targetAmpY := p.idleAmp + (ratio * (p.maxAmplitudeY - p.idleAmp))
	p.ampY = (p.ampY * (1.0 - p.ampLerp)) + (targetAmpY * p.ampLerp)

	// --- LA MATEMATICA DEL VIEW-BOBBING (Lissajous Curve) ---
	// Orizzontale: Spostamento del peso (Coseno della fase base)
	p.bobX = math.Cos(p.phase*0.5) * p.ampX
	// Verticale: Rimbalzo ad ogni passo (Valore assoluto del seno, o Seno con fase doppia)
	// Usiamo Abs(Sin) perché crea quell'impatto "duro" del tallone a terra
	p.bobY = math.Abs(math.Sin(p.phase)) * p.ampY

	// --- Jump/Fall Bob (Procedural Spring) ---
	p.jumpBobVelocity -= p.jumpBobOffset * p.springTension
	p.jumpBobVelocity *= p.springDamping
	p.jumpBobOffset += p.jumpBobVelocity
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

func (p *Bobbing) GetSway() (float64, float64) {
	x := (p.bobX * 1.1) * p.swayScale
	y := (p.bobY * 1.2) * p.swayScale
	//fmt.Printf("DEBUG BOB -> amp: %f | maxAmp: %f | idleAmp: %f\n", p.amp, p.maxAmplitude, p.idleAmp)
	return x, y
}
