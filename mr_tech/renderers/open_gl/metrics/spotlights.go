package metrics

import "math"

// Spotlights represents a structure used for managing spotlight transformation matrices in rendering contexts.
type Spotlights struct {
	spotLightSpace    [16]float32
	spotLightSpacePtr *float32
}

// NewSpotlight initializes a new instance of Spotlights with a default transformation matrix pointer.
func NewSpotlight() *Spotlights {
	s := &Spotlights{
		spotLightSpacePtr: nil,
	}
	s.spotLightSpacePtr = &s.spotLightSpace[0]
	return s
}

// CreateSpotLightSpace generates a 4x4 transformation matrix for a spotlight's view and projection in shadow mapping.
func (m *Spotlights) CreateSpotLightSpace(posX, posY, posZ, dirX, dirY, dirZ float32, fovDeg, near, far float32) {
	// Projection Matrix (Perspective)
	// For a shadow map, aspect ratio is strictly 1.0 (it's square)
	fovRad := fovDeg * math.Pi / 180.0
	f := float32(1.0 / math.Tan(float64(fovRad)/2.0))
	proj := [16]float32{
		f, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), -1.0,
		0, 0, (2.0 * far * near) / (near - far), 0,
	}
	// View Matrix (LookAt)
	ffX, ffY, ffZ := normalize(dirX, dirY, dirZ)
	// Standard UP vector (Y-up in OpenGL)
	upX, upY, upZ := float32(0.0), float32(1.0), float32(0.0)
	// Anti-Gimbal-Lock safety: if the spotlight points straight up or down (floor/ceiling)
	// the cross product would fail. Use -Z as alternative UP.
	if math.Abs(float64(ffY)) > 0.999 {
		upX, upY, upZ = 0.0, 0.0, -1.0
	}
	// R = Right, U = Recalculated Up
	rrX, rrY, rrZ := normalize(crossProduct(ffX, ffY, ffZ, upX, upY, upZ))
	uuX, uuY, uuZ := crossProduct(rrX, rrY, rrZ, ffX, ffY, ffZ) // Already normalized
	// Negative translation (dot product between inverted axes and position)
	tX := -dotProduct(rrX, rrY, rrZ, posX, posY, posZ)
	tY := -dotProduct(uuX, uuY, uuZ, posX, posY, posZ)
	tZ := dotProduct(ffX, ffY, ffZ, posX, posY, posZ)
	view := [16]float32{
		rrX, uuX, -ffX, 0,
		rrY, uuY, -ffY, 0,
		rrZ, uuZ, -ffZ, 0,
		tX, tY, tZ, 1,
	}
	// Final Light Space (Proj * View)
	spotLightSpace := MatrixMultiply4x4(proj, view)
	copy(m.spotLightSpace[:], spotLightSpace[:])
}

func (m *Spotlights) GetSpotLightSpacePtr() *float32 {
	return m.spotLightSpacePtr
}
