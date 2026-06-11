package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// Sector represents a Sector configuration in a level, including geometric, texture, and tag information.
type Sector struct {
	Id                    string     `json:"id"`
	CeilY                 float64    `json:"ceilY"`
	FloorY                float64    `json:"floorY"`
	Ceil                  *Material  `json:"ceil"`
	Floor                 *Material  `json:"floor"`
	Light                 *Light     `json:"light"`
	Segments              []*Segment `json:"segments"`
	Tag                   string     `json:"tag"`
	SlopedCeilingGradient float64    `json:"slopedCeilingGradient"`
	SlopedFloorGradient   float64    `json:"slopedFloorGradient"`
	//SlopedCeiling        geometry.XYZ `json:"slopedCeiling"`
	//SlopedFloor          geometry.XYZ `json:"slopedFloor"`

}

// NewConfigSector creates a new Sector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string, lightIntensity float64, kind LightKind, falloff float64) *Sector {
	return &Sector{
		Id:                    id,
		Ceil:                  nil,
		Floor:                 nil,
		Light:                 NewConfigLight(geometry.XYZ{}, lightIntensity, kind, falloff),
		SlopedCeilingGradient: 0,
		SlopedFloorGradient:   0,
	}
}

// Scale scales all the segments of the Sector by the given scale factor by applying it to their start and end points.
func (s *Sector) Scale(scale geometry.XYZ) {
	xy := geometry.XY{X: scale.X, Y: scale.Y}
	for _, seg := range s.Segments {
		seg.Start.Scale(xy)
		seg.End.Scale(xy)
	}

	// Scala l'equazione del piano inclinati (Floor)

	/*
		// 1. Z è un'intercetta (altezza all'origine), quindi VA SCALATA.
		s.SlopedFloor.Z *= scale.Z
		// 2. X e Y sono GRADIENTI.
		// Se la scala è uniforme (es. 0.01 su X e 0.01 su Z), il rapporto è 1.
		// I gradienti si scalano solo per il rapporto di deformazione degli assi!
		s.SlopedFloor.X *= scale.Z / scale.X
		s.SlopedFloor.Y *= scale.Z / scale.Y

		s.SlopedCeiling.Z *= scale.Z
		s.SlopedCeiling.X *= scale.Z / scale.X
		s.SlopedCeiling.Y *= scale.Z / scale.Y

	*/
}

// IsCCW determines if the sector's segments form a counter-clockwise loop by calculating the signed area of the polygon.
func (s *Sector) IsCCW() bool {
	area := 0.0
	for _, segment := range s.Segments {
		// Fallback di sicurezza se la mappa ha indici corrotti
		v1 := segment.Start
		v2 := segment.End
		// Formula di calcolo dell'area per curve chiuse: (X2 - X1) * (Y2 + Y1)
		area += (v2.X - v1.X) * (v2.Y + v1.Y)
	}
	// Nello Screen-Space (Y che cresce verso il basso come in Outlaws/Doom):
	// Area < 0 significa CCW (Antiorario)
	// Area > 0 significa CW (Orario)
	// se Y-up cartesiano puro, invertire operatore (> 0).*
	return area < 0
}
