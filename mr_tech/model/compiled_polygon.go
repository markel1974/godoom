package model

import "github.com/markel1974/godoom/mr_tech/textures"

// IdFloor represents the hexadecimal color value for the floor.
// IdCeil represents the hexadecimal color value for the ceiling.
// IdUpper represents the hexadecimal color value for the upper structure.
// IdLower represents the hexadecimal color value for the lower structure.
// IdWall represents the hexadecimal color value for the wall.
// IdFloorTest represents the hexadecimal color value for floor testing purposes.
// IdCeilTest represents the hexadecimal color value for ceiling testing purposes.
const (
	IdFloor = 0xA9A9A9
	IdCeil  = 0xF5F5F5
	IdUpper = 0xFFFF00
	IdLower = 0x00FFFF
	IdWall  = 0xBBBBBB

	IdFloorTest = 0xFF00FF
	IdCeilTest  = 0xAA00AA
)

// CompiledPolygon represents a polygon with precompiled data for rendering or simulation purposes.
type CompiledPolygon struct {
	Points         []XYZ
	Sector         *Sector
	Neighbor       *Sector
	Kind           int
	id             float64
	PLen           int
	X1             float64
	X2             float64
	Tx1            float64
	Tx2            float64
	Tz1            float64
	Tz2            float64
	U0             float64
	U1             float64
	Animation      *textures.Animation
	AnimationCeil  *textures.Animation
	AnimationFloor *textures.Animation
}

// NewCompiledPolygon creates and returns a pointer to an empty CompiledPolygon with preallocated points.
func NewCompiledPolygon() *CompiledPolygon {
	return &CompiledPolygon{
		Points: make([]XYZ, 32),
		PLen:   0,
	}
}

// Init initializes the CompiledPolygon with the specified kind and resets the PLen field to 0.
func (p *CompiledPolygon) Init(kind int) {
	p.Kind = kind
	p.PLen = 0
}

// Triangle sets the vertices of a triangular polygon based on the provided 3D coordinates and updates the point length.
func (p *CompiledPolygon) Triangle(x1 float64, y1 float64, y2 float64, z1 float64, x2 float64, y3 float64, z2 float64) {
	p.Points[0].X = x1
	p.Points[0].Y = y1
	p.Points[0].Z = z1

	p.Points[1].X = x2
	p.Points[1].Y = y2
	p.Points[1].Z = z2

	p.Points[2].X = x2
	p.Points[2].Y = y3
	p.Points[2].Z = z2

	p.PLen = 4
}

// Rect sets the points of the polygon to form a rectangle based on the given coordinates and updates the point length to 4.
func (p *CompiledPolygon) Rect(x1 float64, y1 float64, y2 float64, z1 float64, x3 float64, y3 float64, y4 float64, z2 float64) {
	p.Points[0].X = x1
	p.Points[0].Y = y1
	p.Points[0].Z = z1
	p.Points[3].X = x1
	p.Points[3].Y = y2
	p.Points[3].Z = z1
	p.Points[1].X = x3
	p.Points[1].Y = y3
	p.Points[1].Z = z2
	p.Points[2].X = x3
	p.Points[2].Y = y4
	p.Points[2].Z = z2
	p.PLen = 4
}

// AddPoint adds two consecutive points to the polygon and increments the point length counter accordingly.
func (p *CompiledPolygon) AddPoint(x1 float64, y1 float64, z1 float64, x2 float64, y2 float64, z2 float64) {
	p.Points[p.PLen].X = x1
	p.Points[p.PLen].Y = y1
	p.Points[p.PLen].Z = z1
	p.PLen++
	p.Points[p.PLen].X = x2
	p.Points[p.PLen].Y = y2
	p.Points[p.PLen].Z = z2
	p.PLen++
}

// Finalize prepares the CompiledPolygon for use by applying necessary final calculations or adjustments.
func (p *CompiledPolygon) Finalize() {
}

// CompiledPolygons is a collection of preallocated and reusable CompiledPolygon instances for rendering or processing.
// It maintains an internal index to manage allocation and retrieval of polygons efficiently.
type CompiledPolygons struct {
	data        []*CompiledPolygon
	idx         int
	initialSize int
}

// NewCompiledPolygons creates and returns a pointer to a new initialized CompiledPolygons instance.
func NewCompiledPolygons() *CompiledPolygons {
	cp := &CompiledPolygons{}
	return cp
}

// Setup initializes the data slice with the specified number of new CompiledPolygon instances.
func (cp *CompiledPolygons) Setup() {
	cp.Grow()
}

// Clear resets the internal index of CompiledPolygons to 0, effectively marking all compiled polygons as unused.
func (cp *CompiledPolygons) Clear() {
	cp.idx = 0
}

// Acquire assigns properties to a `CompiledPolygon` from the pool and returns the initialized instance.
func (cp *CompiledPolygons) Acquire(sector *Sector, neighbor *Sector, kind int, c, f, t *textures.Animation, x1, x2, tx1, tx2, tz1, tz2, u0, u1 float64) *CompiledPolygon {
	if cp.idx >= len(cp.data) {
		cp.Grow()
	}

	p := cp.data[cp.idx]
	p.id = float64(cp.idx)
	cp.idx++

	p.Sector = sector
	p.Neighbor = neighbor
	p.X1 = x1
	p.X2 = x2
	p.Tx1 = tx1
	p.Tx2 = tx2
	p.Tz1 = tz1
	p.Tz2 = tz2
	p.U0 = u0
	p.U1 = u1
	p.Animation = t
	p.AnimationCeil = c
	p.AnimationFloor = f
	p.Init(kind)

	return p
}

// Get retrieves the active subset of CompiledPolygon instances from the pre-allocated collection.
func (cp *CompiledPolygons) Get() []*CompiledPolygon {
	var t []*CompiledPolygon
	if cp.idx > 0 {
		t = cp.data[:cp.idx]
	}
	return t
}

func (cp *CompiledPolygons) Grow() {
	oldSize := len(cp.data)
	newSize := 16
	if oldSize == 0 {
		cp.data = make([]*CompiledPolygon, newSize)
	} else {
		newSize = oldSize * 2
		newData := make([]*CompiledPolygon, newSize)
		copy(newData, cp.data)
		cp.data = newData
	}
	for cs := oldSize; cs < newSize; cs++ {
		cp.data[cs] = NewCompiledPolygon()
	}
}
