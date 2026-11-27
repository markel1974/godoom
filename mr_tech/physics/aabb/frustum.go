package aabb

import "math"

const MaxPlanes = 6

// Frustum represents a view frustum in 3D space, defined by six contiguous planes.
type Frustum struct {
	Planes    [MaxPlanes]*Plane
	bMinX     float64
	bMinY     float64
	bMinZ     float64
	bMaxX     float64
	bMaxY     float64
	bMaxZ     float64
	hasBounds bool
}

// NewFrustum creates and returns a pointer to a new Frustum instance.
func NewFrustum() *Frustum {
	t := &Frustum{}
	for i := range t.Planes {
		t.Planes[i] = NewPlane()
	}
	return t
}

// DisableNearPlane replaces the near plane with an always-pass plane (used for orthogonal shadow depth clamping).
func (f *Frustum) DisableNearPlane() {
	ptr := f.Planes[4]
	ptr.NormalX = 0
	ptr.NormalY = 0
	ptr.NormalZ = 0
	ptr.D = 1
	f.hasBounds = false
}

// Rebuild updates the MaxPlanes frustum planes using the given 4x4 column-major view-projection matrix.
func (f *Frustum) Rebuild(vp [16]float32) *Frustum {
	// Left Plane (w + x)
	f.Planes[0].Update(float64(vp[3]+vp[0]), float64(vp[7]+vp[4]), float64(vp[11]+vp[8]), float64(vp[15]+vp[12]))
	// Right Plane (w - x)
	f.Planes[1].Update(float64(vp[3]-vp[0]), float64(vp[7]-vp[4]), float64(vp[11]-vp[8]), float64(vp[15]-vp[12]))
	// Bottom Plane (w + y)
	f.Planes[2].Update(float64(vp[3]+vp[1]), float64(vp[7]+vp[5]), float64(vp[11]+vp[9]), float64(vp[15]+vp[13]))
	// Top Plane (w - y)
	f.Planes[3].Update(float64(vp[3]-vp[1]), float64(vp[7]-vp[5]), float64(vp[11]-vp[9]), float64(vp[15]-vp[13]))
	// Near Plane (w + z) -> OpenGL clip depth [-1, 1]
	f.Planes[4].Update(float64(vp[3]+vp[2]), float64(vp[7]+vp[6]), float64(vp[11]+vp[10]), float64(vp[15]+vp[14]))
	// Far Plane (w - z)
	f.Planes[5].Update(float64(vp[3]-vp[2]), float64(vp[7]-vp[6]), float64(vp[11]-vp[10]), float64(vp[15]-vp[14]))
	f.computeBounds()
	return f
}

func (f *Frustum) computeBounds() {
	f.hasBounds = false
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	for lr := range 2 {
		pLR := f.Planes[lr]
		for bt := 2; bt < 4; bt++ {
			pBT := f.Planes[bt]
			for nf := 4; nf < MaxPlanes; nf++ {
				x, y, z, ok := intersectThreePlanes(pLR, pBT, f.Planes[nf])
				if !ok {
					return
				}
				for i := range MaxPlanes {
					pl := f.Planes[i]
					if pl.NormalX*x+pl.NormalY*y+pl.NormalZ*z+pl.D < -1e-2 {
						return
					}
				}
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
				if z < minZ {
					minZ = z
				}
				if z > maxZ {
					maxZ = z
				}
			}
		}
	}
	const pad = 1e-2
	f.bMinX = minX - pad
	f.bMinY = minY - pad
	f.bMinZ = minZ - pad
	f.bMaxX = maxX + pad
	f.bMaxY = maxY + pad
	f.bMaxZ = maxZ + pad
	f.hasBounds = true
}

func intersectThreePlanes(p1, p2, p3 *Plane) (float64, float64, float64, bool) {
	c23x := p2.NormalY*p3.NormalZ - p2.NormalZ*p3.NormalY
	c23y := p2.NormalZ*p3.NormalX - p2.NormalX*p3.NormalZ
	c23z := p2.NormalX*p3.NormalY - p2.NormalY*p3.NormalX
	det := p1.NormalX*c23x + p1.NormalY*c23y + p1.NormalZ*c23z
	if math.Abs(det) < 1e-8 {
		return 0, 0, 0, false
	}
	c31x := p3.NormalY*p1.NormalZ - p3.NormalZ*p1.NormalY
	c31y := p3.NormalZ*p1.NormalX - p3.NormalX*p1.NormalZ
	c31z := p3.NormalX*p1.NormalY - p3.NormalY*p1.NormalX
	c12x := p1.NormalY*p2.NormalZ - p1.NormalZ*p2.NormalY
	c12y := p1.NormalZ*p2.NormalX - p1.NormalX*p2.NormalZ
	c12z := p1.NormalX*p2.NormalY - p1.NormalY*p2.NormalX

	invDet := 1.0 / det
	x := (-p1.D*c23x - p2.D*c31x - p3.D*c12x) * invDet
	y := (-p1.D*c23y - p2.D*c31y - p3.D*c12y) * invDet
	z := (-p1.D*c23z - p2.D*c31z - p3.D*c12z) * invDet
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		return 0, 0, 0, false
	}
	return x, y, z, true
}

// Frustums holds a collection of view frustums for 3D visibility and culling operations.
type Frustums struct {
	Container []*Frustum
}

// NewFrustums creates a Frustums collection containing count initialized Frustum instances.
func NewFrustums(count int) *Frustums {
	f := &Frustums{
		Container: make([]*Frustum, count),
	}
	for i := range f.Container {
		f.Container[i] = NewFrustum()
	}
	return f
}
