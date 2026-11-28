package collisions

import "math"

// BucketType represents a categorized type for spatial partitioning or collision detection in 3D environments.
type BucketType int

// BucketWallWest represents the west (-X) wall bucket type.
// BucketWallEast represents the east (+X) wall bucket type.
// BucketWallNorth represents the north (-Y) wall bucket type.
// BucketWallSouth represents the south (+Y) wall bucket type.
// BucketCeiling represents the ceiling (-Z) bucket type.
// BucketFloor represents the floor (+Z) bucket type.
const (
	BucketWallWest  = BucketType(0) // -X
	BucketWallEast  = BucketType(1) // +X
	BucketWallNorth = BucketType(2) // -Y
	BucketWallSouth = BucketType(3) // +Y
	BucketCeiling   = BucketType(4) // -Z
	BucketFloor     = BucketType(5) // +Z
	BucketCount     = BucketType(6)
)

// Normal2BucketType returns the bucket matching the dominant signed axis of the given normal vector.
func Normal2BucketType(nX, nY, nZ float64) BucketType {
	nAbsX, nAbsY, nAbsZ := math.Abs(nX), math.Abs(nY), math.Abs(nZ)
	if nAbsX > nAbsY && nAbsX > nAbsZ {
		if nX < 0 {
			return BucketWallWest
		}
		return BucketWallEast
	}
	if nAbsY > nAbsZ {
		if nY < 0 {
			return BucketWallNorth
		}
		return BucketWallSouth
	}
	if nZ > 0 {
		return BucketFloor
	}
	return BucketCeiling
}

// String returns the string representation of the BucketType value.
func (p BucketType) String() string {
	switch p {
	case BucketWallWest:
		return "BucketWallWest"
	case BucketWallEast:
		return "BucketWallEast"
	case BucketWallNorth:
		return "BucketWallNorth"
	case BucketWallSouth:
		return "BucketWallSouth"
	case BucketCeiling:
		return "BucketCeiling"
	case BucketFloor:
		return "BucketFloor"
	default:
		return "BucketUnknown"
	}
}

// IsWall determines if the BucketType value represents a wall (west, east, north, or south).
func (p BucketType) IsWall() bool {
	return p == BucketWallWest || p == BucketWallEast || p == BucketWallNorth || p == BucketWallSouth
}

// Opposite returns the opposite directional BucketType.
func (p BucketType) Opposite() BucketType {
	switch p {
	case BucketWallWest:
		return BucketWallEast
	case BucketWallEast:
		return BucketWallWest
	case BucketWallNorth:
		return BucketWallSouth
	case BucketWallSouth:
		return BucketWallNorth
	case BucketCeiling:
		return BucketFloor
	case BucketFloor:
		return BucketCeiling
	default:
		return p
	}
}
