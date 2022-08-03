package main

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

type CompiledPolygon struct {
	id       float64
	Sector   *Sector
	Neighbor *Sector
	kind     int
	light1   float64
	light2   float64
	zIndex   float64
	points   []XYZ
	pLen     int

	x1  float64
	x2  float64
	tz1 float64
	tz2 float64
	u0  float64
	u1  float64
}

func NewCompiledPolygon() *CompiledPolygon {
	return &CompiledPolygon{
		points: make([]XYZ, 32),
		pLen:   0,
	}
}

func (p *CompiledPolygon) Init(kind int) {
	p.kind = kind
	p.pLen = 0
	p.light1 = 0
	p.light2 = 0
	p.zIndex = 0
}

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

func (p *CompiledPolygon) Finalize() {
	p.zIndex /= float64(p.pLen)
	p.light1 /= float64(p.pLen) / 2
	p.light2 /= float64(p.pLen) / 2
}

type CompiledPolygons struct {
	data []*CompiledPolygon
	idx  int
}

func NewCompiledPolygons() *CompiledPolygons {
	cp := &CompiledPolygons{}
	return cp
}

func (cp *CompiledPolygons) Setup(count int) {
	cp.data = make([]*CompiledPolygon, count)
	for c := 0; c < len(cp.data); c++ {
		cp.data[c] = NewCompiledPolygon()
	}
}

func (cp *CompiledPolygons) Clear() {
	cp.idx = 0
}

func (cp *CompiledPolygons) Acquire(sector *Sector, neighbor *Sector, kind int, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64) *CompiledPolygon {
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

func (cp *CompiledPolygons) Get() []*CompiledPolygon {
	var t []*CompiledPolygon
	if cp.idx > 0 {
		t = cp.data[:cp.idx]
	}
	return t
}
