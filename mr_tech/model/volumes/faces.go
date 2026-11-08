package volumes

// Faces represents a collection of face entries.
type Faces struct {
	container []*Face
	count     int
}

// NewFaces creates a Faces collection with storage preallocated to initSize.
func NewFaces(initSize int, fill bool) *Faces {
	count := 0
	if fill {
		count = initSize
	}
	return &Faces{
		container: make([]*Face, initSize),
		count:     count,
	}
}

// Add appends face to the collection, growing the internal storage when needed.
func (f *Faces) Add(face *Face) {
	if f.count >= len(f.container) {
		newFaces := make([]*Face, f.count*2)
		copy(newFaces, f.container)
		f.container = newFaces
	}
	f.container[f.count] = face
	f.count++
}

// Len returns the number of faces currently stored.
func (f *Faces) Len() int {
	return f.count
}

// GetCentroid returns the average x, y, and z coordinates of all points across the faces.
// It returns zero coordinates when there are no points.
func (f *Faces) GetCentroid() (float64, float64, float64) {
	cx, cy, cz := 0.0, 0.0, 0.0
	count := 0
	for x := 0; x < len(f.container); x++ {
		face := f.container[x]
		cx, cy, cz, count = face.SumPoints(cx, cy, cz, count)
	}
	if count > 0 {
		return cx / float64(count), cy / float64(count), cz / float64(count)
	}
	return 0.0, 0.0, 0.0
}

// Set assigns face to the element at idx.
func (f *Faces) Set(idx int, face *Face) {
	f.container[idx] = face
}

// At returns the face stored at the specified index.
func (f *Faces) At(idx int) *Face {
	return f.container[idx]
}

// Clear resets the face count to zero, leaving allocated storage available for reuse.
func (f *Faces) Clear() {
	f.count = 0
}
