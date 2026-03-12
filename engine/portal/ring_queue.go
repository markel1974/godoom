package portal

import "math/bits"

func nextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << (64 - bits.LeadingZeros64(uint64(n-1)))
}

// RingQueue is a circular queue implementation that utilizes a fixed-size buffer for efficient enqueue and dequeue operations.
type RingQueue struct {
	items   []QueueItem
	headIdx int
	tailIdx int
	mask    int
}

// NewRingQueue creates a new RingQueue with a fixed size and initializes its internal storage and mask value.
func NewRingQueue(size int) *RingQueue {
	size = nextPowerOf2(size)
	return &RingQueue{
		items: make([]QueueItem, size),
		mask:  size - 1,
	}
}

// Reset clears the queue by resetting both headIdx and tailIdx to 0, effectively emptying the RingQueue.
func (q *RingQueue) Reset() {
	q.headIdx = 0
	q.tailIdx = 0
}

// Len returns the current number of items in the RingQueue.
func (q *RingQueue) Len() int {
	return len(q.items)
}

// GetHead retrieves the current head item from the queue and advances the head index, wrapping around if necessary.
func (q *RingQueue) GetHead() *QueueItem {
	qi := &q.items[q.headIdx]
	q.headIdx = (q.headIdx + 1) & q.mask // 1 ciclo di clock
	return qi
}

// GetTail retrieves the current tail item from the queue and advances the tail index using a wrap-around mechanism.
func (q *RingQueue) GetTail() *QueueItem {
	qi := &q.items[q.tailIdx]
	q.tailIdx = (q.tailIdx + 1) & q.mask // 1 ciclo di clock
	return qi
}

// IsFull checks if the RingQueue has reached its capacity, indicating no more items can be enqueued without removal.
func (q *RingQueue) IsFull() bool {
	return ((q.headIdx + 1) & q.mask) == q.tailIdx
}

// IsEmpty checks if the RingQueue is empty by comparing the head and tail indices. Returns true if empty, false otherwise.
func (q *RingQueue) IsEmpty() bool {
	return q.headIdx == q.tailIdx
}
