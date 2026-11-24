package vertices_multi_sprite

import (
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// Composer represents an entity composed of multiple sprites linked to a physics entity and volume system.
type Composer struct {
	volume *volumes.Volume
	solver *collisions.Solver
}

// NewComposer creates a new VerticeMultiSpriteEntity instance using the provided Volume.
func NewComposer(volume *volumes.Volume) *Composer {
	return &Composer{
		solver: volume.GetSolver(),
		volume: volume,
	}
}

// Setup initializes the composed multi-sprite volume's collision solver with object.
func (vm *Composer) Setup(object aabb.IObject) error {
	return vm.solver.Setup(object)
}

// QueryOverlaps checks for overlaps between the entity's volume and the given object, invoking the callback for each match.
func (vm *Composer) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	vm.volume.QueryOverlaps(object, callback)
}

// GetSolver returns the collision solver associated with this composed multi-sprite entity.
func (vm *Composer) GetSolver() *collisions.Solver {
	return vm.solver
}

// GetFaces retrieves the collection of faces associated with the entity's current volume.
func (vm *Composer) GetFaces() *volumes.Faces {
	return vm.volume.GetFaces()
}

// GetVolume returns the currently active volume of the entity.
func (vm *Composer) GetVolume() *volumes.Volume {
	return vm.volume
}

// SetVolume updates the current volume of the entity.
func (vm *Composer) SetVolume(activeVolume *volumes.Volume) {
	if activeVolume == vm.volume {
		return
	}
	vm.volume = activeVolume
}
