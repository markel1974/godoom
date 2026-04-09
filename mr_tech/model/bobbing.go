package model

import "math"

// Bobbing represents procedural bobbing behavior, such as movement oscillations for entities or objects.
type Bobbing struct {
	maxSpeed      float64 // La tua nuova velocità di crociera a regime
	maxAmplitude  float64 // Escursione massima desiderata per l'arma/cam (es. 0.9)
	idleDrift     float64 // Frequenza di rientro da fermo (es. 0.03)
	strideLength  float64 // Moltiplicatore per la falcata (es. 0.015)
	speedLerp     float64 // Inerzia procedurale della velocità (es. 0.15)
	ampLerp       float64 // Smorzamento dell'ampiezza (es. 0.10)
	smoothedSpeed float64
	bob           float64
	phase         float64
	amp           float64
}

// NewBobbing initializes and returns a new Bobbing instance with the specified parameters for motion dynamics.
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

// Compute updates the bobbing effect based on the input velocity components and configured parameters.
func (p *Bobbing) Compute(v2x float64, v2y float64) {
	rawSpeed := math.Sqrt(v2x*v2x + v2y*v2y)
	if rawSpeed > p.maxSpeed && rawSpeed < 5.0 {
		// Invece di uno snap brutale, lo facciamo salire morbidamente
		// per evitare micro-scatti nell'ampiezza durante la ricalibrazione
		p.maxSpeed = (p.maxSpeed * 0.95) + (rawSpeed * 0.05)
	}
	// Low-Pass filter configurabile
	p.smoothedSpeed = (p.smoothedSpeed * (1.0 - p.speedLerp)) + (rawSpeed * p.speedLerp)
	// Oscillatore continuo
	p.phase += p.idleDrift + (p.smoothedSpeed * p.strideLength)
	// Ampiezza dinamica basata sulla costante MaxSpeed
	targetAmp := (p.smoothedSpeed / p.maxSpeed) * p.maxAmplitude
	if targetAmp > p.maxAmplitude {
		targetAmp = p.maxAmplitude
	}
	p.amp = (p.amp * (1.0 - p.ampLerp)) + (targetAmp * p.ampLerp)
	p.bob = math.Sin(p.phase) * p.amp
}

// GetBob retrieves the current bobbing value calculated based on speed, phase, and amplitude.
func (p *Bobbing) GetBob() float64 {
	return p.bob
}

// GetPhase retrieves the current phase value of the bobbing calculation.
func (p *Bobbing) GetPhase() float64 {
	return p.phase
}
