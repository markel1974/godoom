package builder

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

func glToEngineVP(m [16]float32) [16]float32 {
	// Engine AABBs use Z-up (ex, ey, ez); OpenGL world positions use Y-up (ex, ez, -ey).
	return [16]float32{
		m[0], m[1], m[2], m[3],
		-m[8], -m[9], -m[10], -m[11],
		m[4], m[5], m[6], m[7],
		m[12], m[13], m[14], m[15],
	}
}

func mixU64(h, x uint64) uint64 {
	return geometry.MixU64(h, x)
}

func mixPtrHash(p unsafe.Pointer) uint64 {
	return geometry.MixUintptr(uintptr(p))
}

// facesEquivalent reports whether two face collections contain the same ordered faces, materials, and points.
func facesEquivalent(a, b *volumes.Faces) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil || a.Len() != b.Len() {
		return false
	}
	n := a.Len()
	for i := range n {
		fa := a.At(i)
		fb := b.At(i)
		if fa == fb {
			continue
		}
		if fa == nil || fb == nil || fa.GetMaterial() != fb.GetMaterial() {
			return false
		}
		ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := fa.GetPointsFlat()
		bx0, by0, bz0, bx1, by1, bz1, bx2, by2, bz2 := fb.GetPointsFlat()
		if ax0 != bx0 || ay0 != by0 || az0 != bz0 ||
			ax1 != bx1 || ay1 != by1 || az1 != bz1 ||
			ax2 != bx2 || ay2 != by2 || az2 != bz2 {
			return false
		}
	}
	return true
}
