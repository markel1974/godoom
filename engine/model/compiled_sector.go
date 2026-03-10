package model

import "github.com/markel1974/godoom/engine/textures"

// CompiledSector represents a preprocessed rendering-ready sector containing compiled polygons and tick information.
type CompiledSector struct {
	Sector           *Sector
	compiledPolygons *CompiledPolygons
	tick             uint
}

// NewCompiledSector creates and returns a pointer to a new initialized CompiledSector instance.
func NewCompiledSector() *CompiledSector {
	return &CompiledSector{
		tick:             0,
		Sector:           nil,
		compiledPolygons: NewCompiledPolygons(),
	}
}

// Setup initializes the compiledPolygons of the CompiledSector with the specified number of empty CompiledPolygon objects.
func (cs *CompiledSector) Setup(count int) {
	cs.compiledPolygons.Setup(count)
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
func (cs *CompiledSector) Acquire(neighbor *Sector, kind int, c, f, t *textures.Texture, x1, x2, tx1, tx2, tz1, tz2, u0, u1 float64) *CompiledPolygon {
	return cs.compiledPolygons.Acquire(cs.Sector, neighbor, kind, c, f, t, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
}

// Prepare increments the internal tick counter of the CompiledSector instance.
func (cs *CompiledSector) Prepare() {
	cs.tick++
}

// Tick returns the current tick count of the CompiledSector instance.
func (cs *CompiledSector) Tick() uint {
	const tickInterval = 64
	return cs.tick / tickInterval
}

// Get retrieves the collection of currently active CompiledPolygon instances from the CompiledSector.
func (cs *CompiledSector) Get() []*CompiledPolygon {
	return cs.compiledPolygons.Get()
}
