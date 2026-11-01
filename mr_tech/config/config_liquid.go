package config

type LiquidType int

const (
	LiquidTypeWater LiquidType = iota // Acqua limpida e trasparente (default)
	LiquidTypeSea                     // Mare oceanico (con onde ampie e schiuma su creste e riva)
	LiquidTypeLava                    // Lava incandescente opaca con vene di magma
	LiquidTypeSlime                   // Slime / Nukage acido radioattivo torbido
	LiquidTypeBlood                   // Sangue denso e viscoso
	LiquidTypeFog                     // Nebbia volumetrica atmosferica
	LiquidTypeLatest
)

type Liquid struct {
	Type LiquidType `json:"type"`
}

func NewLiquid(kind LiquidType) *Liquid {
	return &Liquid{
		Type: kind,
	}
}
