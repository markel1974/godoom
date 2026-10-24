package duke3d

import "math"

func ToRadians(ang int16) float64 {
	// In Build, ang: 0..2047, where 2048 is 360 degrees.
	return float64(ang) * 2.0 * math.Pi / 2048.0
}
