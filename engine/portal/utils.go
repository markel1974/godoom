package portal

import "math/bits"

// NextPowerOf2 calculates the next power of 2 greater than or equal to the given integer n.
func NextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << (64 - bits.LeadingZeros64(uint64(n-1)))
}
