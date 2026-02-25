package portal

import "github.com/markel1974/godoom/engine/model"

// VisibilityData holds information about visibility boundaries defined by a start (begin) and end (end) position in 2D space.
type VisibilityData struct {
	id    int
	begin model.XY
	end   model.XY
}

// NewVisibilityData creates a new instance of VisibilityData with default values.
func NewVisibilityData() *VisibilityData {
	return &VisibilityData{}
}

// Visibility represents a 2D data structure used to manage visibility data within a specified width and height.
type Visibility struct {
	id   int
	w    int
	h    int
	data [][]*VisibilityData
}

// NewVisibility initializes a Visibility object with the specified width and height, allocating necessary data structures.
func NewVisibility(w int, h int) *Visibility {
	data := make([][]*VisibilityData, w)
	for x := 0; x < w; x++ {
		data[x] = make([]*VisibilityData, h)
		for y := 0; y < h; y++ {
			data[x][y] = NewVisibilityData()
		}
	}
	return &Visibility{
		id:   0,
		w:    w,
		h:    h,
		data: data,
	}
}

// Update increments the internal visibility ID to track changes in visibility data.
func (w *Visibility) Update() {
	w.id++
}

// GetId returns the unique identifier associated with the current state of the Visibility instance.
func (w *Visibility) GetId() int {
	return w.id
}

// All returns the entire 2D slice of VisibilityData for the current Visibility instance.
func (w *Visibility) All() [][]*VisibilityData {
	return w.data
}

// Add updates visibility data for the given coordinates with begin and end points if the coordinates are valid.
func (w *Visibility) Add(x int, y int, begin model.XY, end model.XY) {
	w.set(x, y, begin, end)
}

// set updates the visibility data at the specified coordinates if they are valid.
func (w *Visibility) set(x int, y int, begin model.XY, end model.XY) {
	if !w.valid(x, y) {
		return
	}
	d := w.data[x][y]
	d.id = w.id
	d.begin = begin
	d.end = end
}

// Get retrieves the `begin` and `end` coordinates for the specified (x, y) cell if valid and updated with the current ID.
func (w *Visibility) Get(x int, y int) (model.XY, model.XY, bool) {
	if w.valid(x, y) {
		d := w.data[x][y]
		if d.id == w.id {
			return d.begin, d.end, true
		}
	}
	return model.XY{}, model.XY{}, false
}

// Has determines if the specified coordinates (x, y) in the Visibility grid are marked with the current visibility id.
func (w *Visibility) Has(x int, y int) bool {
	if w.valid(x, y) {
		d := w.data[x][y]
		return d.id == w.id
	}
	return false
}

// valid checks if the given x and y coordinates are within the bounds of the Visibility grid dimensions.
func (w *Visibility) valid(x int, y int) bool {
	if x >= 0 && x < w.w && y >= 0 && y < w.h {
		return true
	}
	return false
}
