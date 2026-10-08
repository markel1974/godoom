package metrics

import "math"

// Spotlights represents a structure used to handle spotlight space matrices and their pointer for shadow mapping operations.
type Spotlights struct {
	proj     [16]float32
	projPtr  *float32
	view     [16]float32
	viewPtr  *float32
	space    [16]float32
	spacePtr *float32
}

// NewSpotlight creates and initializes a new Spotlights instance with a pointer to the first element of its array.
func NewSpotlight() *Spotlights {
	s := &Spotlights{}
	s.projPtr = &s.proj[0]
	s.viewPtr = &s.view[0]
	s.spacePtr = &s.space[0]
	return s
}

// GetSpotLightSpacePtr returns a pointer to the spotLightSpace matrix used for spotlight transformation calculations.
func (m *Spotlights) GetSpotLightSpacePtr() *float32 {
	return m.spacePtr
}

// CreateSpotLightSpace computes a spotlight's light space matrix based on its position, direction, FOV, near, and far planes.
func (m *Spotlights) CreateSpotLightSpace(posX, posY, posZ, dirX, dirY, dirZ float32, fovDeg, near, far float32) {
	// Projection Matrix (Perspective)
	// For a shadow map, aspect ratio is strictly 1.0 (it's square)
	fovRad := fovDeg * math.Pi / 180.0
	f := float32(1.0 / math.Tan(float64(fovRad)/2.0))
	m.proj[0], m.proj[1], m.proj[2], m.proj[3] = f, 0, 0, 0
	m.proj[4], m.proj[5], m.proj[6], m.proj[7] = 0, f, 0, 0
	m.proj[8], m.proj[9], m.proj[10], m.proj[11] = 0, 0, (far+near)/(near-far), -1.0
	m.proj[12], m.proj[13], m.proj[14], m.proj[15] = 0, 0, (2.0*far*near)/(near-far), 0
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
	m.view[0], m.view[1], m.view[2], m.view[3] = rrX, uuX, -ffX, 0
	m.view[4], m.view[5], m.view[6], m.view[7] = rrY, uuY, -ffY, 0
	m.view[8], m.view[9], m.view[10], m.view[11] = rrZ, uuZ, -ffZ, 0
	m.view[12], m.view[13], m.view[14], m.view[15] = tX, tY, tZ, 1
	// Final Light Space (Proj * View)
	MatrixMultiply4x4Ptr(m.spacePtr, m.projPtr, m.viewPtr)
}
