package open_gl_legacy

import "fmt"

const vertexAlignment = 12

// FrameVertices represents a structure for storing 3D vertex data, including positions, UV coordinates, and light values.
type FrameVertices struct {
	vertices []float32
	len      int
}

// NewFrameVertices creates and returns a pointer to a new FrameVertices instance with an initial capacity for vertices.
func NewFrameVertices(s int) *FrameVertices {
	return &FrameVertices{
		vertices: make([]float32, s),
		len:      0,
	}
}

// Reset clears all vertices from the FrameVertices, preparing it for reuse without allocating new memory.
func (w *FrameVertices) Reset() {
	w.len = 0
}

// Len returns the number of vertices currently stored in the FrameVertices struct.
func (w *FrameVertices) Len() int {
	return w.len
}

// Alignment returns the alignment value of the vertices in the FrameVertices struct as an int32.
func (w *FrameVertices) Alignment() int32 {
	return vertexAlignment
}

// AddVertex appends a new vertex defined by position (x, y, z), texture coordinates (u, v), and lighting intensity.
// AddVertex appends a new vertex defined by position (x, y, z), texture coordinates (u, v), and lighting intensity.
func (w *FrameVertices) AddVertex(x, y, z, u, v, light, lcX, lcY, lcZ, nX, nY, nZ float32) {
	if w.len+vertexAlignment > len(w.vertices) {
		w.Grow()
	}

	idx := w.len
	w.vertices[idx] = x
	w.vertices[idx+1] = y
	w.vertices[idx+2] = z
	w.vertices[idx+3] = u
	w.vertices[idx+4] = v
	w.vertices[idx+5] = light
	w.vertices[idx+6] = lcX
	w.vertices[idx+7] = lcY
	w.vertices[idx+8] = lcZ
	w.vertices[idx+9] = nX
	w.vertices[idx+10] = nY
	w.vertices[idx+11] = nZ
	w.len += vertexAlignment
}

// Get returns the slice of float32 vertices stored in the FrameVertices instance.
func (w *FrameVertices) Get() []float32 {
	w.CheckDuplicatedTriangles()
	return w.vertices[:w.len]
}

// Grow increases the capacity of the vertices slice to accommodate additional vertex data.
func (w *FrameVertices) Grow() {
	newSize := len(w.vertices) * 2
	if newSize == 0 {
		newSize = vertexAlignment * 128
	}
	newVertices := make([]float32, newSize)
	copy(newVertices, w.vertices)
	w.vertices = newVertices
}

func (w *FrameVertices) CheckDuplicatedTriangles() {
	const stride = 12 // x, y, z, u, v, light, lcX, lcY, lcZ, nX, nY, nZ
	const floatsPerTri = stride * 3

	seen := make(map[string]int)
	duplicates := 0

	// Itera su ogni triangolo inserito nell'array flat
	for i := 0; i+floatsPerTri <= w.len; i += floatsPerTri {

		// Estrai le (x, y, z) dei 3 vertici
		x1, y1, z1 := w.vertices[i], w.vertices[i+1], w.vertices[i+2]
		x2, y2, z2 := w.vertices[i+stride], w.vertices[i+stride+1], w.vertices[i+stride+2]
		x3, y3, z3 := w.vertices[i+stride*2], w.vertices[i+stride*2+1], w.vertices[i+stride*2+2]

		// Calcola il baricentro per identificazione spaziale
		cX := (x1 + x2 + x3) / 3.0
		cY := (y1 + y2 + y3) / 3.0
		cZ := (z1 + z2 + z3) / 3.0

		// Chiave di hash con 3 decimali (assorbe l'imprecisione del float32 IEEE 754)
		key := fmt.Sprintf("%.3f_%.3f_%.3f", cX, cY, cZ)

		if count, exists := seen[key]; exists {
			fmt.Printf("OVERDRAW RILEVATO: Triangolo al centroide [%s] sottomesso %d volte\n", key, count+1)
			duplicates++
			seen[key]++
		} else {
			seen[key] = 1
		}
	}

	if duplicates > 0 {
		fmt.Printf("CRITICO: Rilevati %d triangoli duplicati nel VBO di questo frame!\n", duplicates)
	} else {
		fmt.Println("TOPOLOGIA PULITA: Nessun triangolo sovrapposto rilevato.")
	}
}
