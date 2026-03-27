package open_gl

// FrameLights gestisce il buffer contiguo delle luci per l'upload su UBO (std140).
type FrameLights struct {
	data []float32
	len  int
}

func NewFrameLights(capacity int) *FrameLights {
	return &FrameLights{
		// 4 float per luce: x, y, z, intensity (padding implicito per vec4)
		data: make([]float32, capacity*4),
	}
}

func (w *FrameLights) Reset()         { w.len = 0 }
func (w *FrameLights) Len() int       { return w.len }
func (w *FrameLights) Get() []float32 { return w.data[:w.len*4] }

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

func (w *FrameLights) Grow() {
	newSize := len(w.data) * 2
	if newSize == 0 {
		newSize = 128 * 4
	}
	newData := make([]float32, newSize)
	copy(newData, w.data)
	w.data = newData
}
