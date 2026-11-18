package config

type Bobbing struct {
	SwayScale       float64 `json:"swayScale"`
	SwaySensitivity float64 `json:"swaySensitivity"`
	SwayMultiplierX float64 `json:"swayMultiplierX"`
	SwayMultiplierY float64 `json:"swayMultiplierY"`
	SwayOffsetX     float64 `json:"swayOffsetX"`
	SwayOffsetY     float64 `json:"swayOffsetY"`
	MaxAmplitudeX   float64 `json:"maxAmplitudeX"`
	MaxAmplitudeY   float64 `json:"maxAmplitudeY"`
	IdleDrift       float64 `json:"idleDrift"`
	StrideLength    float64 `json:"strideLength"`
	SpeedLerp       float64 `json:"speedLerp"`
	AmpLerp         float64 `json:"ampLerp"`
	ImpactMax       float64 `json:"impactMax"`
	ImpactScale     float64 `json:"impactScale"`
	IdleAmpX        float64 `json:"idleAmpX"`
	IdleAmpY        float64 `json:"idleAmpY"`
	SpringTension   float64 `json:"springTension"`
	SpringDamping   float64 `json:"springDamping"`
	TiltAmp         float64 `json:"tiltAmp"`
}

func NewBobbing() *Bobbing {
	return &Bobbing{}
}
