package model

// CompiledSector represents a data structure linking a Sector with its compiled polygons for optimized rendering workflows.
type CompiledSector struct {
	Sector           *Sector
	compiledPolygons *CompiledPolygons
}

// NewCompiledSector creates and returns a new instance of CompiledSector with initialized properties.
func NewCompiledSector() *CompiledSector {
	return &CompiledSector{
		Sector:           nil,
		compiledPolygons: NewCompiledPolygons(),
	}
}

// Setup initializes the compiledPolygons field by allocating and preparing the specified number of polygons.
func (cs *CompiledSector) Setup(count int) {
	cs.compiledPolygons.Setup(count)
}

// Bind associates the CompiledSector with a provided Sector and clears previously compiled polygons.
func (cs *CompiledSector) Bind(sector *Sector) {
	cs.Sector = sector
	cs.compiledPolygons.Clear()
}

// Clear resets the state of the CompiledSector by clearing its compiled polygon data.
func (cs *CompiledSector) Clear() {
	cs.compiledPolygons.Clear()
}

// Acquire creates or retrieves a CompiledPolygon, initializing it with Sector, neighbor, type, and geometric parameters.
func (cs *CompiledSector) Acquire(neighbor *Sector, kind int, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64) *CompiledPolygon {
	return cs.compiledPolygons.Acquire(cs.Sector, neighbor, kind, x1, x2, tz1, tz2, u0, u1)
}

// Get retrieves a slice of active CompiledPolygon instances associated with the CompiledSector.
func (cs *CompiledSector) Get() []*CompiledPolygon {
	return cs.compiledPolygons.Get()
}
