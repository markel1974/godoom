package collisions

const bucketCacheSlots = 4

type bucketCacheEntry struct {
	from       uint64
	penetrable int8
	depth      int8
}

// BucketCache caches bucket penetrability results by source entity and traversal depth.
type BucketCache struct {
	entries [bucketCacheSlots]bucketCacheEntry
	count   uint8
	next    uint8
}

// NewBucketCache creates and returns a reset BucketCache.
func NewBucketCache() *BucketCache {
	return &BucketCache{}
}

// Reset clears all cached penetrability entries in O(1).
func (b *BucketCache) Reset() {
	b.count = 0
	b.next = 0
}

// ClearPenetrable invalidates cached penetrability results in O(1).
func (b *BucketCache) ClearPenetrable() {
	b.count = 0
	b.next = 0
}

// Store records a penetrability result for the source entity and depth, preserving the tightest depth bound per source.
func (b *BucketCache) Store(from uint64, depth int, res bool) {
	d := int8(depth)
	p := int8(-1)
	if res {
		p = 1
	}
	n := int(b.count)
	for i := 0; i < n; i++ {
		e := &b.entries[i]
		if e.from == from {
			if e.penetrable != p || (res && d < e.depth) || (!res && d > e.depth) {
				e.penetrable = p
				e.depth = d
			}
			return
		}
	}
	if b.count < bucketCacheSlots {
		b.entries[b.count] = bucketCacheEntry{
			from:       from,
			penetrable: p,
			depth:      d,
		}
		b.count++
		return
	}
	idx := b.next
	b.next = (idx + 1) & (bucketCacheSlots - 1)
	b.entries[idx] = bucketCacheEntry{
		from:       from,
		penetrable: p,
		depth:      d,
	}
}

// Check reports whether a penetrability result is cached for from at depth.
// It returns cached as true with the cached value, or false if no cached result applies.
func (b *BucketCache) Check(from uint64, depth int) (cached bool, val bool) {
	n := int(b.count)
	d := int8(depth)
	for i := 0; i < n; i++ {
		e := &b.entries[i]
		if e.from != from {
			continue
		}
		if e.penetrable == 1 && d >= e.depth {
			return true, true
		}
		if e.penetrable == -1 && d <= e.depth {
			return true, false
		}
	}
	return false, false
}
