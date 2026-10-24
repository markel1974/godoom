package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// UVAlign defines UV alignment and transformation parameters for sector floors and ceilings.
type UVAlign struct {
	FixUV     bool        `json:"fixUV"`
	AlignWall bool        `json:"alignWall"`
	Start     geometry.XY `json:"start"`
	End       geometry.XY `json:"end"`
	SwapXY    bool        `json:"swapXY"`
	FlipX     bool        `json:"flipX"`
	FlipY     bool        `json:"flipY"`
	PanningX  float64     `json:"panningX"`
	PanningY  float64     `json:"panningY"`
}

// Sloped represents a geometric slope with X and Y gradients and a precomputed gradient value for calculations.
type Sloped struct {
	GX       float64 `json:"gX"`
	GY       float64 `json:"gY"`
	Gradient float64 `json:"gradient"`
}

// NewSloped initializes and returns a pointer to a new Sloped instance with default values.
func NewSloped() *Sloped {
	return &Sloped{
		GX:       0,
		GY:       0,
		Gradient: 0,
	}
}

// Sector represents a Sector configuration in a level, including geometric, texture, and tag information.
type Sector struct {
	Id            string     `json:"id"`
	CeilY         float64    `json:"ceilY"`
	FloorY        float64    `json:"floorY"`
	Ceil          *Material  `json:"ceil"`
	Floor         *Material  `json:"floor"`
	Light         *Light     `json:"light"`
	Segments      []*Segment `json:"segments"`
	Tag           string     `json:"tag"`
	SlopedFloor   *Sloped    `json:"slopedFloor"`
	SlopedCeiling *Sloped    `json:"slopedCeiling"`
	FloorAlign    *UVAlign   `json:"floorAlign,omitempty"`
	CeilAlign     *UVAlign   `json:"ceilAlign,omitempty"`
	DependsOn     []string   `json:"dependsOn,omitempty"`
}

// NewConfigSector creates a new Sector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string, lightIntensity float64, kind LightKind, falloff float64) *Sector {
	return &Sector{
		Id:            id,
		Ceil:          nil,
		Floor:         nil,
		Light:         NewConfigLight(geometry.XYZ{}, lightIntensity, kind, falloff),
		SlopedCeiling: NewSloped(),
		SlopedFloor:   NewSloped(),
	}
}

// Scale scales all the segments of the Sector by the given scale factor by applying it to their start and end points.
func (s *Sector) Scale(scale geometry.XYZ) {
	xy := geometry.XY{X: scale.X, Y: scale.Y}
	for _, seg := range s.Segments {
		if seg == nil {
			continue
		}
		seg.Start.Scale(xy)
		seg.End.Scale(xy)
	}
	if s.FloorAlign != nil {
		s.FloorAlign.Start.Scale(xy)
		s.FloorAlign.End.Scale(xy)
	}
	if s.CeilAlign != nil {
		s.CeilAlign.Start.Scale(xy)
		s.CeilAlign.End.Scale(xy)
	}
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

// AddDependency registers a sector ID that this sector depends on (must precede this sector in compilation order).
func (s *Sector) AddDependency(id string) {
	if len(id) == 0 || id == s.Id {
		return
	}
	for _, dep := range s.DependsOn {
		if dep == id {
			return
		}
	}
	s.DependsOn = append(s.DependsOn, id)
}
