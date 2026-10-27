package open_gl

import (
	"sort"

	"github.com/markel1974/godoom/mr_tech/model"
)

// VisibleVolumes manages a collection of visible 3D volumes, maintaining their order and proximity to a camera position.
type VisibleVolumes struct {
	volumes          []*model.Volume
	index            int
	camX, camY, camZ float64
}

// NewVisibleVols creates and initializes a new VisibleVolumes instance with the specified initial capacity for volumes.
func NewVisibleVols(initSize int) *VisibleVolumes {
	return &VisibleVolumes{
		volumes: make([]*model.Volume, initSize),
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
		vs.volumes = make([]*model.Volume, maxLen*2)
	}
}

// At returns the Volume at the specified index within the VisibleVolumes' internal volumes slice.
func (vs *VisibleVolumes) At(index int) *model.Volume {
	return vs.volumes[index]
}

// Add appends the given Volume to the VisibleVolumes list and increments the internal index.
func (vs *VisibleVolumes) Add(volume *model.Volume) {
	vs.volumes[vs.index] = volume
	vs.index++
}

// Sort organizes the `VisibleVolumes` in front-to-back order based on their distances to the camera coordinates.
func (vs *VisibleVolumes) Sort() {
	sort.Sort(vs)
}

// Len returns the number of volumes currently stored in the VisibleVolumes instance.
func (vs *VisibleVolumes) Len() int { return vs.index }

// Swap swaps the elements at indices i and j within the volumes slice of the VisibleVolumes struct.
func (vs *VisibleVolumes) Swap(i, j int) { vs.volumes[i], vs.volumes[j] = vs.volumes[j], vs.volumes[i] }

// Less compares two volumes based on their squared distance from the camera in a front-to-back sorting approach.
func (vs *VisibleVolumes) Less(i, j int) bool {
	// Sorting Front-to-Back (Distanza Quadra Pura)
	a := vs.volumes[i]
	b := vs.volumes[j]
	aX, aY, aZ := a.GetAABB().GetCentroid()
	bX, bY, bZ := b.GetAABB().GetCentroid()
	distA := (aX-vs.camX)*(aX-vs.camX) + (aY-vs.camY)*(aY-vs.camY) + (aZ-vs.camZ)*(aZ-vs.camZ)
	distB := (bX-vs.camX)*(bX-vs.camX) + (bY-vs.camY)*(bY-vs.camY) + (bZ-vs.camZ)*(bZ-vs.camZ)
	return distA < distB
}
