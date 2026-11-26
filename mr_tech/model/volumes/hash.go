package volumes

import "math"

const hashOffset64 uint64 = 14695981039346656037
const hashPrime64 uint64 = 1099511628211

func HashMixU32(h uint64, v uint32) uint64 {
	h ^= uint64(v)
	h *= hashPrime64
	return h
}

func HashMixF32(h uint64, v float32) uint64 {
	return HashMixU32(h, math.Float32bits(v))
}

func HashMixStr(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= hashPrime64
	}
	h ^= 0xff
	h *= hashPrime64
	return h
}
