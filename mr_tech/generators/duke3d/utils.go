package duke3d

import "math"

// ToRadians converts a Build engine angle (0..2047) to radians.
func ToRadians(ang int16) float64 {
	// 2048 units = 2 * Pi radians (or 1024 units = Pi)
	return float64(ang) * (2.0 * math.Pi / 2048.0)
}

// ToDegrees converts a Build engine angle (0..2047) to standard degrees (0..360).
func ToDegrees(ang int16) float64 {
	// Build angle: 0 to 2047. In Build: 0=East (+X), 512=South (+Y).
	// With Y flipped, 0=East, 512=South (-Y), 1536=North (+Y).
	// In Cartesian degrees: (2048 - ang) * 360 / 2048.
	pAngle := (2048.0 - float64(ang)) * 360.0 / 2048.0
	for pAngle >= 360.0 {
		pAngle -= 360.0
	}
	for pAngle < 0.0 {
		pAngle += 360.0
	}
	return pAngle
}
