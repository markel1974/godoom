package open_gl

// FrameLights stores data for multiple lights in a frame, including their positions, colors, directions, and other attributes.
type FrameLights struct {
	data   []float32
	count  int
	stride int32
}

// NewFrameLights initializes and returns a pointer to a new FrameLights instance with a specified maximum light capacity.
func NewFrameLights(maxLights int) *FrameLights {
	const stride = 16
	return &FrameLights{
		data:   make([]float32, 0, maxLights*stride),
		stride: stride,
	}
}

// Reset clears all stored data and resets the count in the FrameLights structure, preparing it for reuse without reallocation.
func (f *FrameLights) Reset() {
	f.data = f.data[:0]
	f.count = 0
}

// Len returns the number of lights currently stored in the FrameLights structure.
func (f *FrameLights) Len() int {
	return f.count
}

// Get returns the slice of float32 values representing the frame light data stored in the FrameLights instance.
func (f *FrameLights) Get() []float32 {
	return f.data
}

// Stride returns the stride value, which represents the spacing between consecutive elements in the data array.
func (f *FrameLights) Stride() int32 {
	return f.stride
}

// Add appends a light's properties to the internal buffer and increments the light count.
func (f *FrameLights) Add(
	posX, posY, posZ, lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff, pad1, pad2 float32,
) {
	if f.count*int(f.stride+f.stride) > len(f.data) {
		f.Grow()
	}
	f.data = append(f.data,
		posX, posY, posZ, lightType,
		colR, colG, colB, intensity,
		dirX, dirY, dirZ, falloff,
		cutOff, outerCutOff, pad1, pad2,
	)
	f.count++
}

// Grow doubles the capacity of the internal data slice or initializes it if unallocated.
func (f *FrameLights) Grow() {
	newSize := len(f.data) * 2
	if newSize == 0 {
		newSize = 128 * int(f.stride)
	}
	newData := make([]float32, newSize)
	copy(newData, f.data)
	f.data = newData
}
