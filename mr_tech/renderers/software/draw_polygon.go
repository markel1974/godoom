package software

import (
	"math"
	"sort"

	"github.com/markel1974/godoom/mr_tech/mathematic"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/pixels"
)

// DrawPolygon represents a structure for rendering polygons with texture, color, and perspective mapping features.
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

// NewDrawPolygon creates and initializes a new DrawPolygon instance with the given screen dimensions.
func NewDrawPolygon(maxW int, maxH int) *DrawPolygon {
	pHFov := (float64(maxH) / float64(maxW)) * model.HFov
	return &DrawPolygon{
		maxW:       maxW,
		maxH:       maxH,
		halfH:      float64(maxH) / 2,
		halfW:      float64(maxW) / 2,
		lastH:      maxH - 1,
		screenVFov: float64(maxH) * model.VFov,
		screenHFov: float64(maxW) * pHFov,
	}
}

// Setup initializes the DrawPolygon instance with the given surface, points, color, and calculates bounding box constraints.
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

// Verify checks if the polygon's bounding box intersects with the drawable area and verifies minimum dimensions.
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

// compileNodes identifies and sorts intersection points of polygon edges with a vertical line at the given x-coordinate.
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

// DrawTexture draws a textured polygon on the surface using perspective corrections and lighting adjustments.
func (dp *DrawPolygon) DrawTexture(texture *textures.Texture, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64, yRef float64, lightAmbient float64, lightArtificial float64) {
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
	texWidth, texHeight := texture.Size()

	for pixelX := dp.left; pixelX <= dp.right; pixelX++ {
		if nodeY := dp.compileNodes(pixelX); nodeY != nil {

			// Correzione prospettica 1D perfetta per X
			txtX := int((u0*((x2-float64(pixelX))*tz2) + u1*((float64(pixelX)-x1)*tz1)) / ((x2-float64(pixelX))*tz2 + (float64(pixelX)-x1)*tz1))

			safeTxtX := txtX % texWidth
			if safeTxtX < 0 {
				safeTxtX += texWidth
			}
			safeTxtX += texture.BeginX()

			light := 1.0
			if x2 != x1 {
				t := (float64(pixelX) - x1) / (x2 - x1)
				currentZ := 1.0 / (((1.0 - t) / tz1) + (t / tz2))
				light = dp.computeLight(currentZ, lightAmbient, lightArtificial)
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
					safeTxtY += texture.BeginY()

					r0, g0, b0 := ToRGB(texture.Get(safeTxtX, safeTxtY), light)
					dp.surface.SetRGBA(pixelX, pixelY, r0, g0, b0, 255)
				}
			}
		}
	}
}

// DrawPerspectiveTexture renders a textured polygon in perspective projection based on position, orientation, and lighting.
func (dp *DrawPolygon) DrawPerspectiveTexture(x float64, y float64, z float64, yaw float64, aSin float64, aCos float64, texture *textures.Texture, yMap float64, scaleFactor float64, lightAmbient float64, lightArtificial float64) {
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

	texWidth, texHeight := texture.Size()

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
					safeTxtX += texture.BeginX()
					safeTxtZ := mapZ % texHeight
					if safeTxtZ < 0 {
						safeTxtZ += texHeight
					}
					safeTxtZ += texture.BeginY()
					light := dp.computeLight(tz, lightAmbient, lightArtificial)
					red, green, blue := ToRGB(texture.Get(safeTxtZ, safeTxtX), light)
					dp.surface.SetRGBA(pixelX, pixelY, red, green, blue, 255)
				}
			}
		}
		// L'interpolazione lineare su X viene rimossa qui in quanto l'illuminazione è gestita spazialmente nel loop Y
	}
}

// DrawWireFrame renders the polygon as a wireframe. If filled is true, it draws a filled wireframe with the polygon color.
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

// DrawPoints applies a specified color and size to each point in the DrawPolygon's points list on the target surface.
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

// DrawRectangle draws a rectangle on the surface using the points stored in the DrawPolygon instance.
// It connects the first and second, first and fourth, second and third, and third and fourth points to form the rectangle.
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

// DrawLines draws lines connecting the points in the DrawPolygon instance based on the specified `contiguous` flag.
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

// drawLine draws a straight line on the specified surface using Bresenham's algorithm with color and lighting effects.
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

// computeLight calculates the light intensity at a specific depth (z) based on ambient and direct light distances.
// The function also handles attenuation and ensures the result is clamped between 0 and 1.
func (dp *DrawPolygon) computeLight(z float64, lightAmbient float64, lightArtificial float64) float64 {
	const visibilityMax = 10.0
	const visibility = visibilityMax - 5.0

	light := 1.0
	if lightArtificial > 0 {
		// OVERRIDE EMISSIVO
		// La luce del settore è sovrana. Se il settore è illuminato al massimo,
		light = lightArtificial
	} else {
		// LUCE AMBIENTALE
		// Se il settore è un normale passaggio senza luce propria (lightArtificial < 0),
		// subisce il decadimento atmosferico globale della camera.
		absZ := math.Abs(z)
		attenuation := absZ * visibility * lightAmbient
		light -= attenuation
	}

	if light < 0 {
		return 0
	}
	if light > 1 {
		return 1
	}
	return light
}
