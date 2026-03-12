package portal

// RingQueue represents a circular queue of QueueItem objects with head and tail indices for tracking elements.
type RingQueue struct {
	items   []*QueueItem
	headIdx int
	tailIdx int
}

// NewRingQueue initializes and returns a new RingQueue with the specified size, preallocating all QueueItem elements.
func NewRingQueue(size int) *RingQueue {
	q := &RingQueue{
		headIdx: 0,
		tailIdx: 0,
		items:   make([]*QueueItem, size),
	}
	for x := 0; x < len(q.items); x++ {
		q.items[x] = NewQueueItem()
	}
	return q
}

// Reset resets the queue by resetting the head and tail indices to their initial positions.
func (q *RingQueue) Reset() {
	q.headIdx = 0
	q.tailIdx = 0
}

// Len returns the number of items currently stored in the queue.
func (q *RingQueue) Len() int {
	return len(q.items)
}

// GetHead retrieves the queue item at the current head index and updates the head index to the next position in the queue.
func (q *RingQueue) GetHead() *QueueItem {
	qi := q.items[q.headIdx]
	q.headIdx = (q.headIdx + 1) % len(q.items)
	return qi
}

// GetTail retrieves the next item from the tail of the queue and increments the tail index cyclically.
func (q *RingQueue) GetTail() *QueueItem {
	qi := q.items[q.tailIdx]
	q.tailIdx = (q.tailIdx + 1) % len(q.items)
	return qi
}

// IsEmpty checks if the queue is empty by comparing the head and tail indices and returns true if they are equal.
func (q *RingQueue) IsEmpty() bool {
	return q.headIdx == q.tailIdx
}

// IsFull checks if the queue is full by verifying if incrementing headIdx wraps around to tailIdx.
func (q *RingQueue) IsFull() bool {
	return (q.headIdx+1)%len(q.items) == q.tailIdx
}
