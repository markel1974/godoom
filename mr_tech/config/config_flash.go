package config

type Flash struct {
	FovDeg          float64 `json:"flashFovDeg"`
	ShadowFovFactor float64 `json:"flashShadowFovDeg"`
	ZNear           float64 `json:"flashZNear"`
	ZFar            float64 `json:"flashZFar"`
	Intensity       float64 `json:"flashIntensity"`
	Falloff         float64 `json:"flashFalloff"`
	OffsetX         float64 `json:"flashOffsetX"`
	OffsetY         float64 `json:"flashOffsetY"`
}

func NewFlash() *Flash {
	return &Flash{}
}
