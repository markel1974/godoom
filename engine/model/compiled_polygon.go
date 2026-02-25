package model

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
	Points   []XYZ
	Sector   *Sector
	Neighbor *Sector
	Kind     int
	id       float64
	Light1   float64
	Light2   float64
	zIndex   float64
	PLen     int
	X1       float64
	X2       float64
	Tz1      float64
	Tz2      float64
	U0       float64
	U1       float64
}

// NewCompiledPolygon creates and initializes a new instance of CompiledPolygon with default values and preallocated Points.
func NewCompiledPolygon() *CompiledPolygon {
	return &CompiledPolygon{
		Points: make([]XYZ, 32),
		PLen:   0,
	}
}

// Init initializes the CompiledPolygon with the specified Kind and resets its properties.
func (p *CompiledPolygon) Init(kind int) {
	p.Kind = kind
	p.PLen = 0
	p.Light1 = 0
	p.Light2 = 0
	p.zIndex = 0
}

// Triangle defines a triangle by setting three Points, their respective coordinates, and light intensities.
func (p *CompiledPolygon) Triangle(x1 float64, y1 float64, y2 float64, z1 float64, l1 float64, x2 float64, y3 float64, z2 float64, l2 float64) {
	p.Points[0].X = x1
	p.Points[0].Y = y1
	p.Points[0].Z = z1

	p.Points[1].X = x2
	p.Points[1].Y = y2
	p.Points[1].Z = z2

	p.Points[2].X = x2
	p.Points[2].Y = y3
	p.Points[2].Z = z2

	p.Light1 = l1
	p.Light2 = l2
	p.PLen = 4
	p.zIndex = (p.Points[0].Z + p.Points[1].Z + p.Points[2].Z + p.Points[3].Z) / float64(p.PLen)
}

// Rect defines a rectangular polygon by setting its four corner Points, lighting values, and calculates its z-index.
func (p *CompiledPolygon) Rect(x1 float64, y1 float64, y2 float64, z1 float64, l1 float64, x3 float64, y3 float64, y4 float64, z2 float64, l2 float64) {
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
	p.Light1 = l1
	p.Light2 = l2
	p.PLen = 4
	p.zIndex = (p.Points[0].Z + p.Points[1].Z + p.Points[2].Z + p.Points[3].Z) / float64(p.PLen)
}

// AddPoint adds two Points with their coordinates and light values to the polygon and updates its zIndex and light properties.
func (p *CompiledPolygon) AddPoint(x1 float64, y1 float64, z1 float64, l1 float64, x2 float64, y2 float64, z2 float64, l2 float64) {
	p.zIndex += z1
	p.zIndex += z2

	p.Light1 += l1
	p.Light2 += l2

	p.Points[p.PLen].X = x1
	p.Points[p.PLen].Y = y1
	p.Points[p.PLen].Z = z1
	p.PLen++
	p.Points[p.PLen].X = x2
	p.Points[p.PLen].Y = y2
	p.Points[p.PLen].Z = z2
	p.PLen++
}

// Finalize recalculates and normalizes zIndex, Light1, and Light2 by dividing them with respect to PLen.
func (p *CompiledPolygon) Finalize() {
	p.zIndex /= float64(p.PLen)
	p.Light1 /= float64(p.PLen) / 2
	p.Light2 /= float64(p.PLen) / 2
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
func (cp *CompiledPolygons) Acquire(sector *Sector, neighbor *Sector, kind int, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64) *CompiledPolygon {
	p := cp.data[cp.idx]
	p.id = float64(cp.idx)
	cp.idx++

	p.Sector = sector
	p.Neighbor = neighbor
	p.X1 = x1
	p.X2 = x2
	p.Tz1 = tz1
	p.Tz2 = tz2
	p.U0 = u0
	p.U1 = u1
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
