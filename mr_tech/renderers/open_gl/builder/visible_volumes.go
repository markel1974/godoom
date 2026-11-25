package builder

import (
	"sort"

	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

// VisibleVolumes manages a collection of visible 3D volumes, maintaining their order and proximity to a camera position.
type VisibleVolumes struct {
	volumes          []*volumes.Volume
	dists            []float64
	index            int
	camX, camY, camZ float64
}

// NewVisibleVols creates and initializes a new VisibleVolumes instance with the specified initial capacity for volumes.
func NewVisibleVols(initSize int) *VisibleVolumes {
	return &VisibleVolumes{
		volumes: make([]*volumes.Volume, initSize),
		dists:   make([]float64, initSize),
		index:   0,
		camX:    0,
		camY:    0,
		camZ:    0,
	}
}

// Reset reinitializes the VisibleVolumes instance with a new maximum length and updates the camera position.
func (vs *VisibleVolumes) Reset(maxLen int, camX, camY, camZ float64) {
	vs.index = 0
	vs.camX = camX
	vs.camY = camY
	vs.camZ = camZ
	if maxLen >= len(vs.volumes) {
		newCap := maxLen * 2
		if newCap < 16 {
			newCap = 16
		}
		vs.volumes = make([]*volumes.Volume, newCap)
		vs.dists = make([]float64, newCap)
	}
}

// RawSlice returns the active slice of collected volumes in raw query order.
func (vs *VisibleVolumes) RawSlice() []*volumes.Volume {
	return vs.volumes[:vs.index]
}

// At returns the Volume at the specified index within the VisibleVolumes' internal volumes slice.
func (vs *VisibleVolumes) At(index int) *volumes.Volume {
	return vs.volumes[index]
}

// Add appends the given Volume to the VisibleVolumes list and increments the internal index.
func (vs *VisibleVolumes) Add(volume *volumes.Volume) {
	if vs.index >= len(vs.volumes) {
		newCap := len(vs.volumes) * 2
		if newCap < 16 {
			newCap = 16
		}
		newVols := make([]*volumes.Volume, newCap)
		copy(newVols, vs.volumes)
		vs.volumes = newVols
		newDists := make([]float64, newCap)
		copy(newDists, vs.dists)
		vs.dists = newDists
	}
	vs.volumes[vs.index] = volume
	vs.index++
}

// Sort organizes the `VisibleVolumes` in front-to-back order based on their distances to the camera coordinates.
func (vs *VisibleVolumes) Sort() {
	for i := 0; i < vs.index; i++ {
		aX, aY, aZ := vs.volumes[i].GetAABB().GetCentroid()
		dx := aX - vs.camX
		dy := aY - vs.camY
		dz := aZ - vs.camZ
		vs.dists[i] = dx*dx + dy*dy + dz*dz
	}
	sort.Sort(vs)
}

// Len returns the number of volumes currently stored in the VisibleVolumes instance.
func (vs *VisibleVolumes) Len() int { return vs.index }

// Swap swaps the elements at indices i and j within the volumes slice of the VisibleVolumes struct.
func (vs *VisibleVolumes) Swap(i, j int) {
	vs.volumes[i], vs.volumes[j] = vs.volumes[j], vs.volumes[i]
	vs.dists[i], vs.dists[j] = vs.dists[j], vs.dists[i]
}

// Less compares two volumes based on their precomputed squared distance from the camera in a front-to-back order.
func (vs *VisibleVolumes) Less(i, j int) bool {
	return vs.dists[i] < vs.dists[j]
}
