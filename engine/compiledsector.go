package main

type CompiledSector struct {
	sector           *Sector
	compiledPolygons *CompiledPolygons
}

func NewCompiledSector() *CompiledSector {
	return &CompiledSector{
		sector:           nil,
		compiledPolygons: NewCompiledPolygons(),
	}
}

func (cs *CompiledSector) Setup(count int) {
	cs.compiledPolygons.Setup(count)
}

func (cs *CompiledSector) Bind(sector *Sector) {
	cs.sector = sector
	cs.compiledPolygons.Clear()
}

func (cs *CompiledSector) Clear() {
	cs.compiledPolygons.Clear()
}

func (cs *CompiledSector) Acquire(neighbor *Sector, kind int, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64) *CompiledPolygon {
	return cs.compiledPolygons.Acquire(cs.sector, neighbor, kind, x1, x2, tz1, tz2, u0, u1)
}

func (cs *CompiledSector) Get() []*CompiledPolygon {
	return cs.compiledPolygons.Get()
}
