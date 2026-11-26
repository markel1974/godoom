package geometry

import "math"

const HashOffset64 uint64 = 14695981039346656037
const hashPrime64 uint64 = 1099511628211

// F32BitsStable returns the IEEE-754 bit pattern of v, normalizing -0 to +0.
func F32BitsStable(v float32) uint64 {
	if v == 0 {
		return 0
	}
	return uint64(math.Float32bits(v))
}

// MixUintptr hashes a pointer address into a 64-bit mixing state.
func MixUintptr(p uintptr) uint64 {
	return MixU64(14695981039346656037, uint64(p))
}

// MixU64 mixes x into running 64-bit hash h using a SplitMix64 finalizer.
func MixU64(h, x uint64) uint64 {
	z := h ^ x
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31
	return z
}

func MixU32(h uint64, v uint32) uint64 {
	h ^= uint64(v)
	h *= hashPrime64
	return h
}

func MixF32(h uint64, v float32) uint64 {
	return MixU32(h, math.Float32bits(v))
}

func MixStr(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= hashPrime64
	}
	h ^= 0xff
	h *= hashPrime64
	return h
}
