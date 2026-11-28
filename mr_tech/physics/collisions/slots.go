package collisions

// Slots stores a resizable collection of collision entries for spatial impact processing.
type Slots struct {
	data []*Entry
	len  int
}

// NewSlots creates an empty Slots collection with its initial storage sized for the default slot count.
func NewSlots() *Slots {
	return &Slots{
		data: make([]*Entry, totalSlots),
		len:  0,
	}
}

// Reset clears all slots without releasing allocated storage.
func (s *Slots) Reset() {
	s.len = 0
}

// Add appends entry to the active slot list, growing the backing storage if needed.
func (s *Slots) Add(entry *Entry) {
	if s.len >= len(s.data) {
		s.grow()
	}
	s.data[s.len] = entry
	s.len++
}

// HasBucket reports whether any slot contains a bucket of the specified type.
func (s *Slots) HasBucket(t BucketType) bool {
	for z := 0; z < s.len; z++ {
		if s.data[z].bucket == t {
			return true
		}
	}
	return false
}

// Len returns the number of entries currently stored in the slots.
func (s *Slots) Len() int {
	return s.len
}

// At returns the entry stored at index i.
func (s *Slots) At(i int) *Entry {
	return s.data[i]
}

// Decrease removes the entry at index i by replacing it with the last active entry.
func (s *Slots) Decrease(i int) {
	s.len--
	s.data[i] = s.data[s.len]
}

// RotateRangeToEnd moves the contiguous entries [start, end) to the back of
// the active range [0, s.len), preserving the relative order of both blocks.
// It performs an in-place left rotation by (end-start) positions over the
// absolute index range [start, s.len), using O(len-start) time and O(1) space.
func (s *Slots) RotateRangeToEnd(start, end int) {
	if end <= start || end >= s.len {
		return
	}
	n := s.len - start // entries to rotate
	d := end - start   // entries in the block moved to the back
	if d == n {
		return
	}
	for i, g := 0, gcd(d, n); i < g; i++ {
		tmp := s.data[start+i]
		j := i
		for {
			k := j + d
			if k >= n {
				k -= n
			}
			if k == i {
				s.data[start+j] = tmp
				break
			}
			s.data[start+j] = s.data[start+k]
			j = k
		}
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// HasImpactInelastic reports whether any inelastic impact has a normal opposing the direction beyond the threshold.
func (s *Slots) HasImpactInelastic(dirX, dirY, dirZ float64, wallBlockDotThreshold float64) bool {
	for i := 0; i < s.len; i++ {
		slot := s.data[i]
		if slot.iMode == ImpactInelastic {
			if dirX*slot.nX+dirY*slot.nY+dirZ*slot.nZ < wallBlockDotThreshold {
				return true
			}
		}
	}
	return false
}

// grow increases the backing storage capacity while preserving existing entries.
func (s *Slots) grow() {
	n := make([]*Entry, nextGrowCap(len(s.data), int(totalSlots)))
	copy(n, s.data)
	s.data = n
}
