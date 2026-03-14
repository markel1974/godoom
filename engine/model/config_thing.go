package model

type ConfigThing struct {
	Id        string
	Position  XY
	Angle     float64
	Type      int
	Sector    string
	Animation *ConfigAnimation // Configurazione RAW dei frame
}

func NewConfigThing(id string, pos XY, angle float64, t int, sector string, anim *ConfigAnimation) *ConfigThing {
	return &ConfigThing{
		Id:        id,
		Position:  pos,
		Angle:     angle,
		Type:      t,
		Sector:    sector,
		Animation: anim,
	}
}
