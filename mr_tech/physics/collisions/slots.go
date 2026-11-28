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

// RotateRangeToEnd moves contiguous entries [start, end) to the back of [0, s.len)
// in O(len-start) time and O(1) space, preserving relative order of both blocks.
func (s *Slots) RotateRangeToEnd(start, end int) {
	if start < 0 || end <= start || end >= s.len {
		return
	}
	// Reverse [start, end)
	for l, r := start, end-1; l < r; l, r = l+1, r-1 {
		s.data[l], s.data[r] = s.data[r], s.data[l]
	}
	// Reverse [end, len)
	for l, r := end, s.len-1; l < r; l, r = l+1, r-1 {
		s.data[l], s.data[r] = s.data[r], s.data[l]
	}
	// Reverse [start, len)
	for l, r := start, s.len-1; l < r; l, r = l+1, r-1 {
		s.data[l], s.data[r] = s.data[r], s.data[l]
	}
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
