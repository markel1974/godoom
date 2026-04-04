package model

import "github.com/markel1974/godoom/mr_tech/textures"

// CompiledVolume represents a preprocessed rendering-ready sector containing compiled polygons and tick information.
type CompiledVolume struct {
	Volume           *Volume
	compiledPolygons *CompiledPolygons
}

// NewCompiledSector creates and returns a pointer to a new initialized CompiledVolume instance.
func NewCompiledSector() *CompiledVolume {
	return &CompiledVolume{
		Volume:           nil,
		compiledPolygons: NewCompiledPolygons(),
	}
}

// Setup initializes the compiledPolygons of the CompiledVolume with the specified number of empty CompiledPolygon objects.
func (cs *CompiledVolume) Setup() {
	cs.compiledPolygons.Setup()
}

// Bind associates the CompiledVolume with a given Sector and resets its compiledPolygons collection.
func (cs *CompiledVolume) Bind(volume *Volume) {
	cs.Volume = volume
	cs.compiledPolygons.Clear()
}

// Clear resets the compiled polygons in the CompiledVolume by delegating the operation to the compiledPolygons instance.
func (cs *CompiledVolume) Clear() {
	cs.compiledPolygons.Clear()
}

// Acquire returns a compiled polygon by reusing or creating it from the provided neighbor, textures, coordinates, and type.
func (cs *CompiledVolume) Acquire(neighbor *Volume, kind int, c, f, t *textures.Animation, x1, x2, tx1, tx2, tz1, tz2, u0, u1 float64) *CompiledPolygon {
	return cs.compiledPolygons.Acquire(cs.Volume, neighbor, kind, c, f, t, x1, x2, tx1, tx2, tz1, tz2, u0, u1)
}

// Get retrieves the collection of currently active CompiledPolygon instances from the CompiledVolume.
func (cs *CompiledVolume) Get() []*CompiledPolygon {
	return cs.compiledPolygons.Get()
}
