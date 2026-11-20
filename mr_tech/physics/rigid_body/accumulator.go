package rigid_body

// Accumulator stores accumulated collision impulses and restitution target state for iterative rigid-body resolution.
type Accumulator struct {
	normal        float64
	tx            float64
	ty            float64
	tz            float64
	restTarget    float64
	hasRestTarget bool
}

// NewAccumulator creates and returns a new Accumulator with zeroed state.
func NewAccumulator() *Accumulator {
	return &Accumulator{}
}

// Reset clears all accumulated impulse and restitution target state.
func (s *Accumulator) Reset() {
	*s = Accumulator{}
}

// GetNormal returns the accumulated normal impulse.
func (s *Accumulator) GetNormal() float64 {
	return s.normal
}

// GetRestTarget returns the stored rest target value.
func (s *Accumulator) GetRestTarget() float64 {
	return s.restTarget
}

// HasRestTarget reports whether a restitution target velocity has been initialized for the current contact step.
func (s *Accumulator) HasRestTarget() bool {
	return s.hasRestTarget
}

// Get returns the accumulated normal impulse and tangential impulse components.
func (s *Accumulator) Get() (float64, float64, float64, float64) {
	return s.normal, s.tx, s.ty, s.tz
}

// Set updates the accumulated normal and tangential impulse components.
func (s *Accumulator) Set(normal, tx, ty, tz float64) {
	s.normal, s.tx, s.ty, s.tz = normal, tx, ty, tz
}

// SetRestTarget updates the restitution target velocity and marks it as initialized.
func (s *Accumulator) SetRestTarget(restTarget float64) {
	s.restTarget = restTarget
	s.hasRestTarget = true
}

// ClearRestTarget clears the restitution target velocity and resets its initialization flag.
func (s *Accumulator) ClearRestTarget() {
	s.restTarget = 0.0
	s.hasRestTarget = false
}

// ScaleWarm scales the accumulated normal and tangential impulses by factor and clears per-step restitution state.
func (s *Accumulator) ScaleWarm(factor float64) {
	s.normal *= factor
	s.tx *= factor
	s.ty *= factor
	s.tz *= factor
	s.restTarget = 0.0
	s.hasRestTarget = false
}
