package vertices_element_3d

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// Composer links a volume with its collision solver for composed 3D vertex processing.
type Composer struct {
	volume *volumes.Volume
	solver *collisions.Solver
}

// NewComposer creates a composed 3D vertices entry for volume using its collision solver.
func NewComposer(volume *volumes.Volume) *Composer {
	return &Composer{
		solver: volume.GetSolver(),
		volume: volume,
	}
}

// Setup initializes the collision solver with the provided object.
func (vm *Composer) Setup(object aabb.IObject) error {
	return vm.solver.Setup(object)
}

// QueryOverlaps searches the active volume for objects whose AABB overlaps object's AABB, invoking callback for each.
func (vm *Composer) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	vm.volume.QueryOverlaps(object, callback)
}

// GetSolver returns the collision solver associated with the composed 3D vertices entry.
func (vm *Composer) GetSolver() *collisions.Solver {
	return vm.solver
}

// GetFaces returns the face collection for the current volume.
func (vm *Composer) GetFaces() *volumes.Faces {
	return vm.volume.GetFaces()
}

// SetVolume assigns the active volume used by the composed entry.
func (vm *Composer) SetVolume(activeVolume *volumes.Volume) {
	vm.volume = activeVolume
}

// Deform applies an impulse to the volume around impact within radius and reports whether it changed.
func (vm *Composer) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	if vm.volume != nil {
		return vm.volume.Deform(impact, impulse, radius)
	}
	return false
}

// DeformDent applies an inward dent at impact along the opposite normal, scaled by depth within radius.
func (vm *Composer) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	impulse := geometry.XYZ{
		X: -normal.X * depth,
		Y: -normal.Y * depth,
		Z: -normal.Z * depth,
	}
	return vm.Deform(impact, impulse, radius)
}
