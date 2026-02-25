package portal

import "github.com/markel1974/godoom/engine/model"

type VisibilityData struct {
	id    int
	begin model.XY
	end   model.XY
}

func NewVisibilityData() *VisibilityData {
	return &VisibilityData{}
}

type Visibility struct {
	id   int
	w    int
	h    int
	data [][]*VisibilityData
}

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

func (w *Visibility) Update() {
	w.id++
}

func (w *Visibility) GetId() int {
	return w.id
}

func (w *Visibility) All() [][]*VisibilityData {
	return w.data
}

func (w *Visibility) Add(x int, y int, begin model.XY, end model.XY) {
	w.set(x, y, begin, end)
}

func (w *Visibility) set(x int, y int, begin model.XY, end model.XY) {
	if !w.valid(x, y) {
		return
	}
	d := w.data[x][y]
	d.id = w.id
	d.begin = begin
	d.end = end
}

func (w *Visibility) Get(x int, y int) (model.XY, model.XY, bool) {
	if w.valid(x, y) {
		d := w.data[x][y]
		if d.id == w.id {
			return d.begin, d.end, true
		}
	}
	return model.XY{}, model.XY{}, false
}

func (w *Visibility) Has(x int, y int) bool {
	if w.valid(x, y) {
		d := w.data[x][y]
		return d.id == w.id
	}
	return false
}

func (w *Visibility) valid(x int, y int) bool {
	if x >= 0 && x < w.w && y >= 0 && y < w.h {
		return true
	}
	return false
}
