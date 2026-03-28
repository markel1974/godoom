package open_gl

// FrameLights is a structure for managing and storing positional light data with intensity in a 3D scene.
type FrameLights struct {
	data []float32
	len  int
}

// NewFrameLights initializes and returns a new FrameLights instance with a given capacity for storing light data.
func NewFrameLights(capacity int) *FrameLights {
	return &FrameLights{
		// 4 float per luce: x, y, z, intensity (padding implicito per vec4)
		data: make([]float32, capacity*4),
	}
}

// Reset clears all data from the FrameLights instance by resetting its length to zero.
func (w *FrameLights) Reset() { w.len = 0 }

// Len returns the current number of lights stored in the FrameLights structure.
func (w *FrameLights) Len() int { return w.len }

// Get retrieves a slice of float32 representing the current frame lights data scaled by 4 elements per light.
func (w *FrameLights) Get() []float32 {
	return w.data[:w.len*4]
}

// Add appends a light source with position (x, y, z) and intensity to the FrameLights, growing the storage if needed.
func (w *FrameLights) Add(x, y, z, intensity float32) {
	if w.len*4+4 > len(w.data) {
		w.Grow()
	}
	idx := w.len * 4
	w.data[idx] = x
	w.data[idx+1] = y
	w.data[idx+2] = z
	w.data[idx+3] = intensity
	w.len++
}

// Grow doubles the capacity of the internal data slice or initializes it if unallocated.
func (w *FrameLights) Grow() {
	newSize := len(w.data) * 2
	if newSize == 0 {
		newSize = 128 * 4
	}
	newData := make([]float32, newSize)
	copy(newData, w.data)
	w.data = newData
}
