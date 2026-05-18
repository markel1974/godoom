package portal

import "github.com/markel1974/godoom/mr_tech/model"

// LinearBatch is a structure managing a slice of QueueItem pointers for batching and updating operations in a 3D environment.
type LinearBatch struct {
	items []QueueItem
}

// NewLinearBatch initializes and returns a new LinearBatch with a specified number of pre-allocated QueueItem instances.
func NewLinearBatch(size int) *LinearBatch {
	q := &LinearBatch{
		items: make([]QueueItem, size),
	}
	return q
}

// Len returns the number of items currently stored in the LinearBatch.
func (q *LinearBatch) Len() int {
	return len(q.items)
}

// Items return the slice of QueueItem pointers managed by the LinearBatch.
func (q *LinearBatch) Items() []QueueItem {
	return q.items
}

// Grow doubles the size of the internal queue, initializing new elements to default values.
func (q *LinearBatch) Grow() {
	oldLen := len(q.items)
	newLen := oldLen * 2
	newQueue := make([]QueueItem, newLen)
	copy(newQueue, q.items)
	q.items = newQueue
}

// UpdateItem updates the specified QueueItem's coordinates and associates it with a neighbor sector at the given index.
func (q *LinearBatch) UpdateItem(neighbor *model.Sector, outIdx int, qi *QueueItem) int {
	return q.Update(neighbor, outIdx, qi.x1, qi.x2, qi.y1t, qi.y2t, qi.y1b, qi.y2b)
}

// Update incrementally updates a QueueItem with new sector data and coordinate boundaries, growing capacity if necessary.
func (q *LinearBatch) Update(neighbor *model.Sector, outIdx int, x1, x2, y1t, y2t, y1b, y2b float64) int {
	if outIdx >= len(q.items) {
		q.Grow()
	}
	target := &q.items[outIdx]
	target.Update(neighbor, x1, x2, y1t, y2t, y1b, y2b)
	outIdx++
	return outIdx
}
