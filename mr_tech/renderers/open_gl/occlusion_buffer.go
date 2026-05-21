package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// OcclusionBuffer represents a low-resolution software rasterizer used for CPU-side early Z-culling.
// It maps the 3D frustum into a 2D tile mask to prevent rendering geometry hidden behind front walls.
type OcclusionBuffer struct {
	Width  int
	Height int
	Tiles  []bool // true means the tile is completely occluded by solid geometry
}

// NewOcclusionBuffer creates an initialized buffer for CPU culling. A resolution of 256x144 or 128x72 is recommended.
func NewOcclusionBuffer(width, height int) *OcclusionBuffer {
	return &OcclusionBuffer{
		Width:  width,
		Height: height,
		Tiles:  make([]bool, width*height),
	}
}

// Clear resets the occlusion mask. Must be called at the beginning of each frame.
func (ob *OcclusionBuffer) Clear() {
	for i := range ob.Tiles {
		ob.Tiles[i] = false
	}
}

// SetRect marks a 2D bounding box area as fully occluded in the buffer.
func (ob *OcclusionBuffer) SetRect(minX, minY, maxX, maxY int) {
	// Hardware clamping for safety against out-of-bounds array access
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > ob.Width-1 {
		maxX = ob.Width - 1
	}
	if maxY > ob.Height-1 {
		maxY = ob.Height - 1
	}

	for y := minY; y <= maxY; y++ {
		rowOffset := y * ob.Width
		for x := minX; x <= maxX; x++ {
			ob.Tiles[rowOffset+x] = true
		}
	}
}

// IsOccluded checks if a 2D bounding box is completely hidden by previously rendered geometry.
func (ob *OcclusionBuffer) IsOccluded(minX, minY, maxX, maxY int) bool {
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > ob.Width-1 {
		maxX = ob.Width - 1
	}
	if maxY > ob.Height-1 {
		maxY = ob.Height - 1
	}

	// If the clamped box is invalid or inverted, consider it occluded (out of screen bounds)
	if minX > maxX || minY > maxY {
		return true
	}

	for y := minY; y <= maxY; y++ {
		rowOffset := y * ob.Width
		for x := minX; x <= maxX; x++ {
			// If we find even one tile that is NOT occluded, the AABB is at least partially visible.
			if !ob.Tiles[rowOffset+x] {
				return false
			}
		}
	}
	// All tiles checked are true; the AABB is 100% occluded.
	return true
}

// ProjectAABB transforms the 8 corners of your physics.AABB using a 4x4 MVP matrix and returns the 2D bounding box on the screen.
func (ob *OcclusionBuffer) ProjectAABB(aabb *physics.AABB, mvp [16]float32) (minX, minY, maxX, maxY int, valid bool) {
	// Retrieve the bounds natively from your AABB implementation
	mX, mY, mZ := aabb.GetMinX(), aabb.GetMinY(), aabb.GetMinZ()
	xX, xY, xZ := aabb.GetMaxX(), aabb.GetMaxY(), aabb.GetMaxZ()

	corners := [8][3]float64{
		{mX, mY, mZ}, {xX, mY, mZ}, {mX, xY, mZ}, {xX, xY, mZ},
		{mX, mY, xZ}, {xX, mY, xZ}, {mX, xY, xZ}, {xX, xY, xZ},
	}

	minScrX, minScrY := float32(math.MaxFloat32), float32(math.MaxFloat32)
	maxScrX, maxScrY := float32(-math.MaxFloat32), float32(-math.MaxFloat32)
	validVertices := 0

	for _, pt := range corners {
		x := float32(pt[0])
		y := float32(pt[1])
		z := float32(pt[2])

		// Vector Matrix Multiplication (ViewProjection)
		clipW := x*mvp[3] + y*mvp[7] + z*mvp[11] + mvp[15]

		// If w <= 0, the vertex is behind the camera (near plane clipping)
		if clipW <= 0.001 {
			continue
		}

		clipX := x*mvp[0] + y*mvp[4] + z*mvp[8] + mvp[12]
		clipY := x*mvp[1] + y*mvp[5] + z*mvp[9] + mvp[13]

		// Perspective Divide (Clip Space -> NDC Space: -1.0 to 1.0)
		ndcX := clipX / clipW
		ndcY := clipY / clipW

		// Map NDC to the OcclusionBuffer tile resolution
		scrX := (ndcX + 1.0) * 0.5 * float32(ob.Width)
		scrY := (-ndcY + 1.0) * 0.5 * float32(ob.Height) // Inverted Y for 2D screen space

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

		validVertices++
	}

	// If no vertices survived the w-divide (entire box is behind the player)
	if validVertices == 0 {
		return 0, 0, 0, 0, false
	}

	return int(minScrX), int(minScrY), int(maxScrX), int(maxScrY), true
}
