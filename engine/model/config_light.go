package model

type LightKind int

const (
	LightKindNone LightKind = iota
	LightKindSpot
	LightKindDiffuse
	LightKindDirectional
	LightKindParticle
)

// ConfigLight defines a light configuration with intensity and type attributes.
type ConfigLight struct {
	Intensity float64   `json:"Intensity"`
	Kind      LightKind `json:"kind"`
}

// NewConfigLight creates a new instance of ConfigLight with default values for Intensity and Kind.
func NewConfigLight() *ConfigLight {
	return &ConfigLight{
		Intensity: 0.0,
		Kind:      LightKindNone,
	}
}

// Clone creates a deep copy of the ConfigLight instance, duplicating its fields into a new object.
func (cl *ConfigLight) Clone() *ConfigLight {
	out := NewConfigLight()
	out.Intensity = cl.Intensity
	out.Kind = cl.Kind
	return out
}
