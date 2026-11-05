package model

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
