package lights

import (
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Lights represents a collection of Light instances managed within a spatial partitioning structure (AABBTree).
type Lights struct {
	container []*Light
	tree      *aabb.Tree
}

// NewLights initializes and returns a new Lights object with an empty container and a new AABBTree with a capacity of 1024.
func NewLights(e []*Light) *Lights {
	l := &Lights{
		container: make([]*Light, len(e)),
		tree:      aabb.NewTree(1024, 4.0),
	}
	for idx, light := range e {
		l.container[idx] = light
		l.tree.InsertObject(light)
	}
	return l
}

// Setup initializes the Lights collection and prepares the spatial partitioning structure (AABBTree) for usage.
func (l *Lights) Setup() error {
	return nil
}

// Len returns the number of Light objects contained in the Lights collection.
func (l *Lights) Len() int {
	return len(l.container)
}

/*
// AddLight adds a single light to the container and inserts it into the AABB tree for spatial management.
func (l *Lights) AddLight(e *Light) {
	l.container = append(l.container, e)
	l.tree.InsertObject(e)
}
*/

// Get retrieves all Light objects stored in the Lights container.
func (l *Lights) Get() []*Light {
	return l.container
}

// QueryFrustum performs a spatial query using the specified frustum and invokes the callback for each intersected object.
func (l *Lights) QueryFrustum(front *aabb.Frustum, callback func(object aabb.IObject) bool) {
	l.tree.QueryFrustum(front, callback)
}

// QueryMultiFrustum performs a query on the spatial tree using two frustums and invokes a callback for each intersected object.
func (l *Lights) QueryMultiFrustum(rear *aabb.Frustum, front *aabb.Frustum, callback func(object aabb.IObject) bool) {
	l.tree.QueryMultiFrustum(rear, front, callback)
}
