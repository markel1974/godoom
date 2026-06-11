package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// OcclusionBuffer represents a 2D depth buffer used for visibility testing within a 3D rendering pipeline.
type OcclusionBuffer struct {
	width  int
	height int
	depth  []float32 // Memorizza la distanza dalla telecamera (clipW)
	void   []float32
}

// NewOcclusionBuffer creates and returns a new OcclusionBuffer with the specified width and height, initializing its depth buffer.
func NewOcclusionBuffer(width, height int) *OcclusionBuffer {
	s := width * height
	ob := &OcclusionBuffer{
		width:  width,
		height: height,
		depth:  make([]float32, s),
	}
	ob.void = make([]float32, len(ob.depth))
	for i := range ob.void {
		ob.void[i] = math.MaxFloat32
	}
	ob.Clear()
	return ob
}

// Clear resets all depth values in the occlusion buffer to the maximum float32 value.
func (ob *OcclusionBuffer) Clear() {
	copy(ob.depth, ob.void)
}

// RasterizeTriangle rasterize a 3D triangle and updates the depth buffer for occlusion testing.
func (ob *OcclusionBuffer) RasterizeTriangle(p0, p1, p2 geometry.XYZ, mvp [16]float32) {
	pts := [3]geometry.XYZ{p0, p1, p2}
	var sx, sy, sw [3]float32

	for i, pt := range pts {
		x, y, z := float32(pt.X), float32(pt.Y), float32(pt.Z)
		clipX := x*mvp[0] + y*mvp[4] + z*mvp[8] + mvp[12]
		clipY := x*mvp[1] + y*mvp[5] + z*mvp[9] + mvp[13]
		clipW := x*mvp[3] + y*mvp[7] + z*mvp[11] + mvp[15]

		// Ignoriamo triangoli che attraversano la telecamera per sicurezza
		if clipW <= 0.01 {
			return
		}

		ndcX := clipX / clipW
		ndcY := clipY / clipW

		sx[i] = (ndcX + 1.0) * 0.5 * float32(ob.width)
		sy[i] = (-ndcY + 1.0) * 0.5 * float32(ob.height)
		sw[i] = clipW
	}

	minX := int(max(0, min(float64(sx[0]), min(float64(sx[1]), float64(sx[2])))))
	minY := int(max(0, min(float64(sy[0]), min(float64(sy[1]), float64(sy[2])))))
	maxX := int(min(float64(ob.width-1), max(float64(sx[0]), max(float64(sx[1]), float64(sx[2])))))
	maxY := int(min(float64(ob.height-1), max(float64(sy[0]), max(float64(sy[1]), float64(sy[2])))))

	// Troviamo la profondità PEGGIORATIVA (il punto più lontano del triangolo).
	// Garantisce che l'occlusione sia conservativa (non culliamo mai per sbaglio)
	maxW := sw[0]
	if sw[1] > maxW {
		maxW = sw[1]
	}
	if sw[2] > maxW {
		maxW = sw[2]
	}

	for y := minY; y <= maxY; y++ {
		fy := float32(y) + 0.5
		rowOffset := y * ob.width
		for x := minX; x <= maxX; x++ {
			fx := float32(x) + 0.5

			// Edge-Function (Coordinate Baricentriche): verifica se il pixel 2D è DENTRO il triangolo
			w01 := (sx[1]-sx[0])*(fy-sy[0]) - (sy[1]-sy[0])*(fx-sx[0])
			w12 := (sx[2]-sx[1])*(fy-sy[1]) - (sy[2]-sy[1])*(fx-sx[1])
			w20 := (sx[0]-sx[2])*(fy-sy[2]) - (sy[0]-sy[2])*(fx-sx[2])

			// Supporta triangoli CW e CCW
			isInside := (w01 >= 0 && w12 >= 0 && w20 >= 0) || (w01 <= 0 && w12 <= 0 && w20 <= 0)

			// Se è dentro il triangolo, aggiorna lo Z-Buffer solo se è più vicino (o se era vuoto)
			if isInside {
				idx := rowOffset + x
				if maxW < ob.depth[idx] {
					ob.depth[idx] = maxW
				}
			}
		}
	}
}

// IsAABBOccluded determines if a given axis-aligned bounding box (AABB) is occluded based on the occlusion buffer and view matrix.
func (ob *OcclusionBuffer) IsAABBOccluded(aabb *physics.AABB, mvp [16]float32) bool {
	mX, mY, mZ := aabb.GetMinX(), aabb.GetMinY(), aabb.GetMinZ()
	xX, xY, xZ := aabb.GetMaxX(), aabb.GetMaxY(), aabb.GetMaxZ()

	corners := [8][3]float32{
		{float32(mX), float32(mY), float32(mZ)}, {float32(xX), float32(mY), float32(mZ)}, {float32(mX), float32(xY), float32(mZ)}, {float32(xX), float32(xY), float32(mZ)},
		{float32(mX), float32(mY), float32(xZ)}, {float32(xX), float32(mY), float32(xZ)}, {float32(mX), float32(xY), float32(xZ)}, {float32(xX), float32(xY), float32(xZ)},
	}

	minScrX, minScrY := float32(math.MaxFloat32), float32(math.MaxFloat32)
	maxScrX, maxScrY := float32(-math.MaxFloat32), float32(-math.MaxFloat32)

	// Troviamo la profondità MIGLIORATIVA (il punto dell'AABB più vicino al giocatore)
	minW := float32(math.MaxFloat32)

	for _, pt := range corners {
		x, y, z := pt[0], pt[1], pt[2]
		clipX := x*mvp[0] + y*mvp[4] + z*mvp[8] + mvp[12]
		clipY := x*mvp[1] + y*mvp[5] + z*mvp[9] + mvp[13]
		clipW := x*mvp[3] + y*mvp[7] + z*mvp[11] + mvp[15]

		if clipW <= 0.01 {
			return false // Se tocca la telecamera, ovviamente lo consideriamo visibile
		}

		if clipW < minW {
			minW = clipW
		}

		ndcX := clipX / clipW
		ndcY := clipY / clipW

		scrX := (ndcX + 1.0) * 0.5 * float32(ob.width)
		scrY := (-ndcY + 1.0) * 0.5 * float32(ob.height)

		if scrX < minScrX {
			minScrX = scrX
		}
		if scrX > maxScrX {
			maxScrX = scrX
		}
		if scrY < minScrY {
			minScrY = scrY
		}
		if scrY > maxScrY {
			maxScrY = scrY
		}
	}

	minX := int(max(0, float64(minScrX)))
	minY := int(max(0, float64(minScrY)))
	maxX := int(min(float64(ob.width-1), float64(maxScrX)))
	maxY := int(min(float64(ob.height-1), float64(maxScrY)))

	if minX > maxX || minY > maxY {
		return true // Fuori dal frustum laterale
	}

	// Testiamo la profondità contro lo Z-Buffer
	for y := minY; y <= maxY; y++ {
		rowOffset := y * ob.width
		for x := minX; x <= maxX; x++ {
			// Se il punto dell'AABB più vicino alla telecamera (minW) è MINORE della
			// profondità del muro registrato in questo pixel, l'AABB sta DAVANTI al muro.
			// Quindi NON è occluso.
			if minW <= ob.depth[rowOffset+x] {
				return false
			}
		}
	}

	// Se per tutti i pixel la profondità del buffer era minore di minW, l'oggetto è murato vivo!
	return true
}
