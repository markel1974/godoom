package config

import "github.com/markel1974/godoom/mr_tech/model/geometry"

type Slope struct {
	Segment int     `json:"segment"`
	Angle   float64 `json:"angle"`
}

func NewSlope(segment int, angle float64) *Slope {
	return &Slope{
		Segment: segment,
		Angle:   angle,
	}
}

// Sector represents a Sector configuration in a level, including geometric, texture, and tag information.
type Sector struct {
	Id            string       `json:"id"`
	CeilY         float64      `json:"ceilY"`
	FloorY        float64      `json:"floorY"`
	Ceil          *Material    `json:"ceil"`
	Floor         *Material    `json:"floor"`
	Light         *Light       `json:"light"`
	Segments      []*Segment   `json:"segments"`
	Tag           string       `json:"tag"`
	SlopedCeiling geometry.XYZ `json:"slopedCeiling"`
	SlopedFloor   geometry.XYZ `json:"slopedFloor"`
	SlopeCeiling  *Slope       `json:"slopeCeiling"`
	SlopeFloor    *Slope       `json:"slopeFloor"`
}

// NewConfigSector creates a new Sector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string, lightIntensity float64, kind LightKind, falloff float64) *Sector {
	return &Sector{
		Id:    id,
		Ceil:  nil,
		Floor: nil,
		Light: NewConfigLight(geometry.XYZ{}, lightIntensity, kind, falloff),
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
