package portal

import "github.com/markel1974/godoom/engine/model"

// IdEmpty represents an empty identifier with a value of 0.
// IdFloor represents the identifier for a floor with a hexadecimal value of 0xA9A9A9.
// IdCeil represents the identifier for a ceiling with a hexadecimal value of 0xF5F5F5.
// IdUpper represents the identifier for an upper boundary with a hexadecimal value of 0xFFFF00.
// IdLower represents the identifier for a lower boundary with a hexadecimal value of 0x00FFFF.
// IdWall represents the identifier for a wall with a hexadecimal value of 0xBBBBBB.
// IdFloorTest represents the identifier for testing floors with a hexadecimal value of 0xFF00FF.
// IdCeilTest represents the identifier for testing ceilings with a hexadecimal value of 0xAA00AA.
const (
	IdEmpty = 0
	IdFloor = 0xA9A9A9
	IdCeil  = 0xF5F5F5
	IdUpper = 0xFFFF00
	IdLower = 0x00FFFF
	IdWall  = 0xBBBBBB

	IdFloorTest = 0xFF00FF
	IdCeilTest  = 0xAA00AA
)

// CompiledPolygon represents a preprocessed polygon with metadata for rendering, lighting, and spatial organization.
type CompiledPolygon struct {
	id       float64
	Sector   *model.Sector
	Neighbor *model.Sector
	kind     int
	light1   float64
	light2   float64
	zIndex   float64
	points   []model.XYZ
	pLen     int

	x1  float64
	x2  float64
	tz1 float64
	tz2 float64
	u0  float64
	u1  float64
}

// NewCompiledPolygon creates and initializes a new instance of CompiledPolygon with default values and preallocated points.
func NewCompiledPolygon() *CompiledPolygon {
	return &CompiledPolygon{
		points: make([]model.XYZ, 32),
		pLen:   0,
	}
}

// Init initializes the CompiledPolygon with the specified kind and resets its properties.
func (p *CompiledPolygon) Init(kind int) {
	p.kind = kind
	p.pLen = 0
	p.light1 = 0
	p.light2 = 0
	p.zIndex = 0
}

// Triangle defines a triangle by setting three points, their respective coordinates, and light intensities.
func (p *CompiledPolygon) Triangle(x1 float64, y1 float64, y2 float64, z1 float64, l1 float64, x2 float64, y3 float64, z2 float64, l2 float64) {
	p.points[0].X = x1
	p.points[0].Y = y1
	p.points[0].Z = z1

	p.points[1].X = x2
	p.points[1].Y = y2
	p.points[1].Z = z2

	p.points[2].X = x2
	p.points[2].Y = y3
	p.points[2].Z = z2

	p.light1 = l1
	p.light2 = l2
	p.pLen = 4
	p.zIndex = (p.points[0].Z + p.points[1].Z + p.points[2].Z + p.points[3].Z) / float64(p.pLen)
}

// Rect defines a rectangular polygon by setting its four corner points, lighting values, and calculates its z-index.
func (p *CompiledPolygon) Rect(x1 float64, y1 float64, y2 float64, z1 float64, l1 float64, x3 float64, y3 float64, y4 float64, z2 float64, l2 float64) {
	p.points[0].X = x1
	p.points[0].Y = y1
	p.points[0].Z = z1
	p.points[3].X = x1
	p.points[3].Y = y2
	p.points[3].Z = z1
	p.points[1].X = x3
	p.points[1].Y = y3
	p.points[1].Z = z2
	p.points[2].X = x3
	p.points[2].Y = y4
	p.points[2].Z = z2
	p.light1 = l1
	p.light2 = l2
	p.pLen = 4
	p.zIndex = (p.points[0].Z + p.points[1].Z + p.points[2].Z + p.points[3].Z) / float64(p.pLen)
}

// AddPoint adds two points with their coordinates and light values to the polygon and updates its zIndex and light properties.
func (p *CompiledPolygon) AddPoint(x1 float64, y1 float64, z1 float64, l1 float64, x2 float64, y2 float64, z2 float64, l2 float64) {
	p.zIndex += z1
	p.zIndex += z2

	p.light1 += l1
	p.light2 += l2

	p.points[p.pLen].X = x1
	p.points[p.pLen].Y = y1
	p.points[p.pLen].Z = z1
	p.pLen++
	p.points[p.pLen].X = x2
	p.points[p.pLen].Y = y2
	p.points[p.pLen].Z = z2
	p.pLen++
}

// Finalize recalculates and normalizes zIndex, light1, and light2 by dividing them with respect to pLen.
func (p *CompiledPolygon) Finalize() {
	p.zIndex /= float64(p.pLen)
	p.light1 /= float64(p.pLen) / 2
	p.light2 /= float64(p.pLen) / 2
}

// CompiledPolygons is a collection of CompiledPolygon objects used for efficient geometric operations and management.
type CompiledPolygons struct {
	data []*CompiledPolygon
	idx  int
}

// NewCompiledPolygons initializes and returns a new instance of CompiledPolygons.
func NewCompiledPolygons() *CompiledPolygons {
	cp := &CompiledPolygons{}
	return cp
}

// Setup initializes the CompiledPolygons instance by creating and storing a specified number of empty CompiledPolygon objects.
func (cp *CompiledPolygons) Setup(count int) {
	cp.data = make([]*CompiledPolygon, count)
	for c := 0; c < len(cp.data); c++ {
		cp.data[c] = NewCompiledPolygon()
	}
}

// Clear resets the internal index to 0, effectively clearing the managed collection without deallocating memory.
func (cp *CompiledPolygons) Clear() {
	cp.idx = 0
}

// Acquire creates or reinitializes a CompiledPolygon with the given parameters, associating it with sectors and coordinates.
func (cp *CompiledPolygons) Acquire(sector *model.Sector, neighbor *model.Sector, kind int, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64) *CompiledPolygon {
	p := cp.data[cp.idx]
	p.id = float64(cp.idx)
	cp.idx++

	p.Sector = sector
	p.Neighbor = neighbor
	p.x1 = x1
	p.x2 = x2
	p.tz1 = tz1
	p.tz2 = tz2
	p.u0 = u0
	p.u1 = u1
	p.Init(kind)

	return p
}

// Get retrieves a slice of currently active CompiledPolygon instances from the CompiledPolygons structure.
func (cp *CompiledPolygons) Get() []*CompiledPolygon {
	var t []*CompiledPolygon
	if cp.idx > 0 {
		t = cp.data[:cp.idx]
	}
	return t
}
