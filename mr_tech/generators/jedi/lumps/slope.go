package lumps

import (
	"math"
)

// Slope represents a sloped surface defined by a sector index, pitch, and wall index.
type Slope struct {
	SectorIndex int
	Pitch       int
	WallIndex   int
}

// NewSlope creates and returns a pointer to a Slope structure initialized with the given sector index, pitch, and wall index.
func NewSlope(sectorIndex, pitch, wallIndex int) *Slope {
	return &Slope{
		SectorIndex: sectorIndex,
		Pitch:       pitch,
		WallIndex:   wallIndex,
	}
}

// GetGradient calculates and returns the angle of the slope in radians using the pitch value.
func (s *Slope) GetGradient() float64 {
	degrees := float64(s.Pitch) / 100.0
	radians := degrees * (math.Pi / 180.0)
	gradient := math.Tan(radians)
	if math.IsNaN(gradient) || math.IsInf(gradient, 0) {
		return 0 //TODO return false
	}
	return gradient //TODO return true
}

/*
// ComputeSlopePlane calculates the slope plane of a given wall within a sector, returning its normal as an XYZ vector.
func (s *Slope) ComputeSlopePlane(l *Level, baseZ float64) (geometry.XYZ, error) {
	if s.SectorIndex < 0 || s.SectorIndex >= len(l.Sectors) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid SectorId: %d", s.SectorIndex)
	}
	pivotSector := l.Sectors[s.SectorIndex]

	if s.WallIndex < 0 || s.WallIndex >= len(pivotSector.Walls) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid WallId: %d", s.WallIndex)
	}
	wall := pivotSector.Walls[s.WallIndex]

	if wall.LeftVertex < 0 || wall.LeftVertex >= len(pivotSector.Vertices) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid LeftVertex index")
	}
	if wall.RightVertex < 0 || wall.RightVertex >= len(pivotSector.Vertices) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid RightVertex index")
	}
	v1 := pivotSector.Vertices[wall.LeftVertex]
	v2 := pivotSector.Vertices[wall.RightVertex]

	dx := v2.X - v1.X
	dy := v2.Y - v1.Y

	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("pivot wall has zero length")
	}
	var nX, nY float64
	if pivotSector.IsCCW() {
		nX, nY = -dy/length, dx/length
	} else {
		nX, nY = dy/length, -dx/length
	}
	degrees := float64(s.Pitch) / 100.0
	gradient := math.Tan(degrees * math.Pi / 180.0)
	slopeX := nX * gradient
	slopeY := nY * gradient
	slopeZ := baseZ - (slopeX * v1.X) - (slopeY * v1.Y)
	return geometry.XYZ{X: slopeX, Y: slopeY, Z: slopeZ}, nil
}
*/
