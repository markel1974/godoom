package model

type ConfigLight struct {
	Intensity float64 `json:"Intensity"`
	Kind      string  `json:"kind"`
}

func NewConfigLight() *ConfigLight {
	return &ConfigLight{
		Intensity: 0.0,
		Kind:      "none",
	}
}

func (cl *ConfigLight) Clone() *ConfigLight {
	out := NewConfigLight()
	out.Intensity = cl.Intensity
	out.Kind = cl.Kind
	return out
}
