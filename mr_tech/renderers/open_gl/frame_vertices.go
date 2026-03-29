package open_gl

import "fmt"

// FrameVertices is a structure for managing and storing vertex data in a flat array format for rendering purposes.
type FrameVertices struct {
	vertices []float32
	len      int32
	stride   int32
}

// NewFrameVertices creates and returns a pointer to a new FrameVertices instance with a preallocated vertex buffer of size s.
func NewFrameVertices(s int) *FrameVertices {
	const vertexAlignment = 8
	return &FrameVertices{
		vertices: make([]float32, s),
		len:      0,
		stride:   vertexAlignment,
	}
}

// Reset resets the FrameVertices by setting its length to zero, effectively clearing the existing vertex data.
func (w *FrameVertices) Reset() {
	w.len = 0
}

// Len returns the current number of elements in the FrameVertices instance.
func (w *FrameVertices) Len() int32 {
	return w.len
}

// Count calculates the number of vertices in the structure based on its length and stride properties.
func (w *FrameVertices) Count() int32 {
	return w.len / w.stride
}

// Stride returns the number of elements that make up a single vertex in the frame's vertex buffer.
func (w *FrameVertices) Stride() int32 {
	return w.stride
}

// AddVertex adds a new vertex to the vertex buffer with position, texture coordinates, and normals.
func (w *FrameVertices) AddVertex(x, y, z, u, v, nX, nY, nZ float32) {
	if int(w.len+(w.stride+w.stride)) > len(w.vertices) {
		w.Grow()
	}

	idx := w.len
	w.vertices[idx] = x
	w.vertices[idx+1] = y
	w.vertices[idx+2] = z
	w.vertices[idx+3] = u
	w.vertices[idx+4] = v
	w.vertices[idx+5] = nX
	w.vertices[idx+6] = nY
	w.vertices[idx+7] = nZ
	w.len += w.stride
}

// Get retrieves a slice of float32 containing the current vertices up to the length of the FrameVertices instance.
func (w *FrameVertices) Get() []float32 {
	//w.CheckDuplicatedTriangles()
	return w.vertices[:w.len]
}

// Grow expands the internal vertex buffer to accommodate additional vertices by doubling its size or initializing it if empty.
func (w *FrameVertices) Grow() {
	newSize := len(w.vertices) * 2
	if newSize == 0 {
		newSize = 128 * int(w.stride)
	}
	newVertices := make([]float32, newSize)
	copy(newVertices, w.vertices)
	w.vertices = newVertices
}

// CheckDuplicatedTriangles detects and reports any duplicate triangles in the vertex buffer using centroid-based hashing.
func (w *FrameVertices) CheckDuplicatedTriangles() {
	floatsPerTri := w.stride * 3
	seen := make(map[string]int)
	duplicates := 0
	// Itera su ogni triangolo inserito nell'array flat
	for i := int32(0); i+floatsPerTri <= w.len; i += floatsPerTri {
		// Estrai le (x, y, z) dei 3 vertici
		x1, y1, z1 := w.vertices[i], w.vertices[i+1], w.vertices[i+2]
		x2, y2, z2 := w.vertices[i+w.stride], w.vertices[i+w.stride+1], w.vertices[i+w.stride+2]
		x3, y3, z3 := w.vertices[i+w.stride*2], w.vertices[i+w.stride*2+1], w.vertices[i+w.stride*2+2]
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
	}
}
