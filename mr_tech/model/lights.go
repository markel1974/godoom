package model

import "github.com/markel1974/godoom/mr_tech/physics"

// Lights represents a collection of Light instances managed within a spatial partitioning structure (AABBTree).
type Lights struct {
	container []*Light
	tree      *physics.AABBTree
}

// NewLights initializes and returns a new Lights object with an empty container and a new AABBTree with a capacity of 1024.
func NewLights() *Lights {
	l := &Lights{
		container: make([]*Light, 0),
		tree:      physics.NewAABBTree(1024),
	}
	return l
}

// AddLights adds multiple Light objects to the Lights collection and inserts them into the spatial partitioning tree.
func (l *Lights) AddLights(e []*Light) {
	for _, light := range e {
		l.AddLight(light)
	}
}

// AddLight adds a single light to the container and inserts it into the AABB tree for spatial management.
func (l *Lights) AddLight(e *Light) {
	l.container = append(l.container, e)
	l.tree.InsertObject(e)
}

// Get retrieves all Light objects stored in the Lights container.
func (l *Lights) Get() []*Light {
	return l.container
}

// QueryFrustum performs a spatial query using the specified frustum and invokes the callback for each intersected object.
func (l *Lights) QueryFrustum(front *physics.Frustum, callback func(object physics.IAABB) bool) {
	l.tree.QueryFrustum(front, callback)
}

// QueryMultiFrustum performs a query on the spatial tree using two frustums and invokes a callback for each intersected object.
func (l *Lights) QueryMultiFrustum(rear *physics.Frustum, front *physics.Frustum, callback func(object physics.IAABB) bool) {
	l.tree.QueryMultiFrustum(rear, front, callback)
}
