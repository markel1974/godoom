package model

import "github.com/markel1974/godoom/mr_tech/textures"

// CompiledSector represents a preprocessed rendering-ready sector containing compiled polygons and tick information.
type CompiledSector struct {
	Sector           *Sector
	compiledPolygons *CompiledPolygons
}

// NewCompiledSector creates and returns a pointer to a new initialized CompiledSector instance.
func NewCompiledSector() *CompiledSector {
	return &CompiledSector{
		Sector:           nil,
		compiledPolygons: NewCompiledPolygons(),
	}
}

// Setup initializes the compiledPolygons of the CompiledSector with the specified number of empty CompiledPolygon objects.
func (cs *CompiledSector) Setup() {
	cs.compiledPolygons.Setup()
}

// Bind associates the CompiledSector with a given Sector and resets its compiledPolygons collection.
func (cs *CompiledSector) Bind(sector *Sector) {
	cs.Sector = sector
	cs.compiledPolygons.Clear()
}

// Clear resets the compiled polygons in the CompiledSector by delegating the operation to the compiledPolygons instance.
func (cs *CompiledSector) Clear() {
	cs.compiledPolygons.Clear()
}

// Acquire returns a compiled polygon by reusing or creating it from the provided neighbor, textures, coordinates, and type.
func (cs *CompiledSector) Acquire(neighbor *Sector, kind int, c, f, t *textures.Animation, x1, x2, tx1, tx2, tz1, tz2, u0, u1 float64) *CompiledPolygon {
	return cs.compiledPolygons.Acquire(cs.Sector, neighbor, kind, c, f, t, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
}

// Get retrieves the collection of currently active CompiledPolygon instances from the CompiledSector.
func (cs *CompiledSector) Get() []*CompiledPolygon {
	return cs.compiledPolygons.Get()
}
