package renderers

import (
	"math"
	"sort"

	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
)

// DrawPolygon represents a drawable polygon with specific properties such as size, surface, color, and points for rendering.
type DrawPolygon struct {
	maxW       int
	maxH       int
	halfH      float64
	halfW      float64
	lastH      int
	screenVFov float64
	screenHFov float64
	nodeY      [1000]int

	surface   *pixels.PictureRGBA
	Color     int
	points    []model.XYZ
	pointsLen int
	top       int
	bottom    int
	left      int
	right     int
}

// NewDrawPolygon initializes and returns a new instance of DrawPolygon with specified screen width and height.
func NewDrawPolygon(maxW int, maxH int) *DrawPolygon {
	pHFov := (float64(maxH) / float64(maxW)) * HFov
	return &DrawPolygon{
		maxW:       maxW,
		maxH:       maxH,
		halfH:      float64(maxH) / 2,
		halfW:      float64(maxW) / 2,
		lastH:      maxH - 1,
		screenVFov: float64(maxH) * VFov,
		screenHFov: float64(maxW) * pHFov,
	}
}

// Setup initializes the DrawPolygon with surface, points, points length, and color, and calculates bounding box values.
func (dp *DrawPolygon) Setup(surface *pixels.PictureRGBA, points1 []model.XYZ, pointsLen1 int, color int) {
	dp.Color = color
	dp.surface = surface
	dp.points = points1
	dp.pointsLen = pointsLen1
	dp.top = int(dp.points[0].Y)
	dp.bottom = int(dp.points[0].Y)
	dp.left = int(dp.points[0].X)
	dp.right = int(dp.points[0].X)
	for x := 1; x < dp.pointsLen; x++ {
		if int(dp.points[x].X) < dp.left {
			dp.left = int(dp.points[x].X)
		}
		if int(dp.points[x].X) > dp.right {
			dp.right = int(dp.points[x].X)
		}
		if int(dp.points[x].Y) < dp.top {
			dp.top = int(dp.points[x].Y)
		}
		if int(dp.points[x].Y) > dp.bottom {
			dp.bottom = int(dp.points[x].Y)
		}
	}
	dp.left = mathematic.Clamp(dp.left, 0, dp.maxW-1)
	dp.right = mathematic.Clamp(dp.right, 0, dp.maxW-1)
	dp.top = mathematic.Clamp(dp.top, 0, dp.maxH-1)
	dp.bottom = mathematic.Clamp(dp.bottom, 0, dp.maxH-1)
}

// Verify verifies if the polygon's bounding box is within the defined screen bounds and has valid dimensions.
func (dp *DrawPolygon) Verify() bool {
	if !mathematic.IntersectBox(dp.left, dp.top, dp.right, dp.bottom, 0, 0, dp.maxW, dp.maxH) {
		return false
	}
	if dp.right-dp.left < 1 {
		return false
	}
	if dp.bottom-dp.top < 1 {
		return false
	}
	return true
}

// compileNodes processes the polygon's edges to compute intersection points on a vertical line at pixelX.
func (dp *DrawPolygon) compileNodes(pixelX int) []int {
	nodes := 0
	j := dp.pointsLen - 1
	px := float64(pixelX)

	for i := 0; i < dp.pointsLen; i++ {
		piX := dp.points[i].X
		piY := dp.points[i].Y
		pjX := dp.points[j].X
		pjY := dp.points[j].Y

		if (piX < px && pjX >= px) || (pjX < px && piX >= px) {
			// Calcolo in float64 puro per preservare l'edge esatto
			intersectY := piY + ((px-piX)/(pjX-piX))*(pjY-piY)

			// Arrotondamento (evita il troncamento asimmetrico verso lo zero di int())
			dp.nodeY[nodes] = int(math.Round(intersectY))
			nodes++
		}
		j = i
	}

	var nodeY []int
	switch nodes {
	case 0, 1:
	case 2:
		nodeY = dp.nodeY[:nodes]
		if nodeY[0] > nodeY[1] {
			nodeY[0], nodeY[1] = mathematic.Swap(nodeY[0], nodeY[1])
		}
	default:
		nodeY = dp.nodeY[:nodes]
		sort.Ints(nodeY)
	}
	return nodeY
}

// DrawTexture renders a texture onto a polygon using perspective-correct texture mapping and lighting adjustments.
func (dp *DrawPolygon) DrawTexture(texture *textures.Texture, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64, yRef float64, lightDistance float64) {
	if dp.surface == nil {
		return
	}
	if texture == nil {
		dp.DrawWireFrame(true)
		return
	}
	if !dp.Verify() {
		return
	}

	texWidth := textures.TextureEnd - textures.TextureBegin
	texHeight := textures.TextureEnd - textures.TextureBegin
	if texWidth <= 0 {
		texWidth = 64
	}
	if texHeight <= 0 {
		texHeight = 64
	}

	for pixelX := dp.left; pixelX <= dp.right; pixelX++ {
		if nodeY := dp.compileNodes(pixelX); nodeY != nil {

			// Correzione prospettica 1D perfetta per X
			txtX := int((u0*((x2-float64(pixelX))*tz2) + u1*((float64(pixelX)-x1)*tz1)) / ((x2-float64(pixelX))*tz2 + (float64(pixelX)-x1)*tz1))

			safeTxtX := txtX % texWidth
			if safeTxtX < 0 {
				safeTxtX += texWidth
			}
			safeTxtX += textures.TextureBegin

			light := 1.0
			if x2 != x1 {
				t := (float64(pixelX) - x1) / (x2 - x1)
				currentZ := 1.0 / (((1.0 - t) / tz1) + (t / tz2))
				light = computeLight(currentZ, lightDistance)
			}

			for i := 0; i < len(nodeY); i += 2 {
				y1 := nodeY[i]
				y2 := nodeY[i+1]

				div := float64(y2 - y1)
				if div <= 0 {
					continue
				}

				if (y1 < 0 && y2 < 0) || (y1 >= dp.maxH && y2 >= dp.maxH) {
					continue
				}

				cY1 := mathematic.Clamp(y1, 0, dp.lastH)
				cY2 := mathematic.Clamp(y2, 0, dp.lastH)

				// Overlap di 1 pixel per sigillare i gap sub-pixel
				if cY2 < dp.lastH {
					cY2++
				}

				for pixelY := cY1; pixelY <= cY2; pixelY++ {
					rawTxtY := int(float64(pixelY-y1) * yRef / div)

					safeTxtY := rawTxtY % texHeight
					if safeTxtY < 0 {
						safeTxtY += texHeight
					}
					safeTxtY += textures.TextureBegin

					r0, g0, b0 := ToRGB(texture.Get(uint(safeTxtX), uint(safeTxtY)), light)
					dp.surface.SetRGBA(pixelX, pixelY, r0, g0, b0, 255)
				}
			}
		}
	}
}

// DrawPerspectiveTexture maps a texture onto a polygon in perspective-correct space, handling lighting and scaling factors.
func (dp *DrawPolygon) DrawPerspectiveTexture(x float64, y float64, z float64, yaw float64, aSin float64, aCos float64, texture *textures.Texture, yMap float64, scaleFactor float64, lightDistance float64) {
	if dp.surface == nil {
		return
	}
	if texture == nil {
		dp.DrawWireFrame(true)
		return
	}
	if !dp.Verify() {
		return
	}

	p1 := (yMap - z) * dp.screenVFov
	p2 := yaw * dp.screenVFov

	texWidth := textures.TextureEnd - textures.TextureBegin
	texHeight := textures.TextureEnd - textures.TextureBegin
	if texWidth <= 0 {
		texWidth = 64
	}
	if texHeight <= 0 {
		texHeight = 64
	}

	for pixelX := dp.left; pixelX <= dp.right; pixelX++ {
		if nodeY := dp.compileNodes(pixelX); nodeY != nil {

			p3 := (dp.halfW - float64(pixelX)) / dp.screenHFov

			for i := 0; i < len(nodeY); i += 2 {
				y1 := nodeY[i]
				y2 := nodeY[i+1]
				if (y1 < 0 && y2 < 0) || (y1 >= dp.maxH && y2 >= dp.maxH) {
					continue
				}
				cY1 := mathematic.Clamp(y1, 0, dp.lastH)
				cY2 := mathematic.Clamp(y2, 0, dp.lastH)

				// Overlap di 1 pixel per sigillare i gap sub-pixel
				if cY2 < dp.lastH {
					cY2++
				}

				for pixelY := cY1; pixelY <= cY2; pixelY++ {
					denom := (dp.halfH - float64(pixelY)) - p2
					if denom == 0 {
						denom = 0.0001
					}

					tz := p1 / denom
					tx := tz * p3

					mapX := int((((tz * aCos) + (tx * aSin)) + x) * scaleFactor)
					mapZ := int((((tz * aSin) - (tx * aCos)) + y) * scaleFactor)

					safeTxtX := mapX % texWidth
					if safeTxtX < 0 {
						safeTxtX += texWidth
					}
					safeTxtX += textures.TextureBegin

					safeTxtZ := mapZ % texHeight
					if safeTxtZ < 0 {
						safeTxtZ += texHeight
					}
					safeTxtZ += textures.TextureBegin
					light := computeLight(tz, lightDistance)
					red, green, blue := ToRGB(texture.Get(uint(safeTxtZ), uint(safeTxtX)), light)
					dp.surface.SetRGBA(pixelX, pixelY, red, green, blue, 255)
				}
			}
		}
		// L'interpolazione lineare su X viene rimossa qui in quanto l'illuminazione è gestita spazialmente nel loop Y
	}
}

// DrawWireFrame renders the outline of a polygon as a wireframe, optionally filling it based on the filled parameter.
func (dp *DrawPolygon) DrawWireFrame(filled bool) {
	if dp.surface == nil {
		return
	}
	if !dp.Verify() {
		return
	}
	for pixelX := dp.left; pixelX <= dp.right; pixelX++ {
		if nodeY := dp.compileNodes(pixelX); nodeY != nil {
			lightStart := 1.0

			r0, g0, b0 := ToRGB(dp.Color, lightStart)
			r1, g1, b1 := ToRGB(dp.Color, lightStart/2)
			for i := 0; i < len(nodeY); i += 2 {
				y1 := nodeY[i]
				y2 := nodeY[i+1]
				if (y1 < 0 && y2 < 0) || (y1 >= dp.maxH && y2 >= dp.maxH) {
					continue
				}
				cY1 := mathematic.Clamp(y1, -1, dp.maxH)
				cY2 := mathematic.Clamp(y2, -1, dp.maxH)
				if filled {
					for pixelY := cY1; pixelY <= cY2; pixelY++ {
						dp.surface.SetRGBA(pixelX, pixelY, r0, g0, b0, 255)
					}
				}
				dp.surface.SetRGBA(pixelX, cY1, r1, g1, b1, 255)
				dp.surface.SetRGBA(pixelX, cY2, r1, g1, b1, 255)
			}
		}
		//dp.lightStart += dp.lightStep
	}
}

// DrawPoints renders all points in the polygon to the surface with a specified size and the current color.
func (dp *DrawPolygon) DrawPoints(size int) {
	if dp.surface == nil {
		return
	}
	lightStart := 1.0
	r0, g0, b0 := ToRGB(dp.Color, lightStart)
	for _, k := range dp.points {
		dp.surface.SetRGBASize(int(k.X), int(k.Y), r0, g0, b0, 255, size)
	}
}

// DrawRectangle draws a rectangle using the first four points in the polygon's points list if they are available.
func (dp *DrawPolygon) DrawRectangle() {
	if dp.surface == nil {
		return
	}
	if len(dp.points) >= 4 {
		dp.drawLine(dp.points[0].X, dp.points[0].Y, dp.points[1].X, dp.points[1].Y)
		dp.drawLine(dp.points[0].X, dp.points[0].Y, dp.points[3].X, dp.points[3].Y)
		dp.drawLine(dp.points[1].X, dp.points[1].Y, dp.points[2].X, dp.points[2].Y)
		dp.drawLine(dp.points[2].X, dp.points[2].Y, dp.points[3].X, dp.points[3].Y)
	}
}

// DrawLines connects points in the polygon using lines. If contiguous is true, lines are drawn between consecutive points.
func (dp *DrawPolygon) DrawLines(contiguous bool) {
	if dp.surface == nil {
		return
	}
	interval := 2
	if contiguous {
		interval = 1
	}

	for c := 0; c < len(dp.points)-1; c += interval {
		from := dp.points[c]
		to := dp.points[c+1]
		dp.drawLine(from.X, from.Y, to.X, to.Y)
	}
}

// drawLine renders a line between two points (x1, y1) and (x2, y2) using the Bresenham's line algorithm.
func (dp *DrawPolygon) drawLine(x1 float64, y1 float64, x2 float64, y2 float64) {
	// Bresenham's line algorithm
	lightStart := 1.0
	r0, g0, b0 := ToRGB(dp.Color, lightStart)
	steep := math.Abs(y2-y1) > math.Abs(x2-x1)
	if steep {
		x1, y1 = mathematic.SwapF(x1, y1)
		x2, y2 = mathematic.SwapF(x2, y2)
	}

	if x1 > x2 {
		x1, x2 = mathematic.SwapF(x1, x2)
		y1, y2 = mathematic.SwapF(y1, y2)
	}
	dx := x2 - x1
	dy := math.Abs(y2 - y1)
	errorDx := dx / 2.0
	var yStep int
	if y1 < y2 {
		yStep = 1
	} else {
		yStep = -1
	}
	y := int(y1)

	maxX := int(x2)
	//maxX = clamp(maxX, 0, maxW)

	for x := int(x1); x <= maxX; x++ {
		if y >= 0 {
			if steep {
				dp.surface.SetRGBA(y, x, r0, g0, b0, 255)
			} else {
				dp.surface.SetRGBA(x, y, r0, g0, b0, 255)
			}
		}
		errorDx -= dy
		if errorDx < 0 {
			y += yStep
			//if y > maxH {
			//	break
			//}
			errorDx += dx
		}
	}
}

// computeLight calculates the light intensity based on the absolute depth value z and the given lightDistance.
func computeLight(z float64, lightDistance float64) float64 {
	const visibilityMax = 10.0
	const visibility = visibilityMax - 7.0
	// Depth-shading per-pixel basato su Z assoluto (allineato alla scala calcolata in portal.go)
	light := 1.0 - (math.Abs(z) * visibility * lightDistance)
	if light < 0 {
		light = 0
	} else if light > 1 {
		light = 1
	}
	return light
}
