package buffers

// PtrPool provides reusable pointers to values of type T, growing as needed and resetting without reallocating.
type PtrPool[T any] struct {
	Items []*T
	Len   int
}

// Init prepares the pool with capacity for at least initCap items and resets its length.
func (p *PtrPool[T]) Init(initCap int) {
	if initCap <= len(p.Items) {
		p.Len = 0
		return
	}
	p.grow(initCap)
	p.Len = 0
}

// Reset marks all pooled pointers as available without releasing allocated storage.
func (p *PtrPool[T]) Reset() {
	p.Len = 0
}

// Acquire returns the next pooled pointer, growing the pool when all existing items are in use.
func (p *PtrPool[T]) Acquire() *T {
	if p.Len >= len(p.Items) {
		p.grow(0)
	}
	item := p.Items[p.Len]
	p.Len++
	return item
}

// grow expands the pool to at least minCap, preserving existing items and allocating pointers for new slots.
func (p *PtrPool[T]) grow(minCap int) {
	oldLen := len(p.Items)
	newLen := max(oldLen*2, max(minCap, 16))
	grown := make([]*T, newLen)
	copy(grown, p.Items)
	for i := oldLen; i < newLen; i++ {
		grown[i] = new(T)
	}
	p.Items = grown
}
