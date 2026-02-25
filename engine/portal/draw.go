package portal

import (
	"math"
	"sort"

	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
)

const fullLightDistance = 0.0039 // 1 / distance == 1 / 255

func toRGB(rgb int, light float64) (r uint8, g uint8, b uint8) {
	fr := float64(uint8((rgb>>16)&255)) * light
	fg := float64(uint8((rgb>>8)&255)) * light
	fb := float64(uint8(rgb&255)) * light
	return uint8(fr), uint8(fg), uint8(fb)
}

type DrawPolygon struct {
	maxW       int
	maxH       int
	halfH      float64
	halfW      float64
	lastH      int
	screenVFov float64
	screenHFov float64
	nodeY      [1000]int

	surface    *pixels.PictureRGBA
	color      int
	points     []model.XYZ
	pointsLen  int
	top        int
	bottom     int
	left       int
	right      int
	lightStart float64
	lightStep  float64
}

func NewDrawPolygon(maxW int, maxH int) *DrawPolygon {
	pHFov := (float64(maxH) / float64(maxW)) * hFov
	return &DrawPolygon{
		maxW:       maxW,
		maxH:       maxH,
		halfH:      float64(maxH) / 2,
		halfW:      float64(maxW) / 2,
		lastH:      maxH - 1,
		screenVFov: float64(maxH) * vFov,
		screenHFov: float64(maxW) * pHFov,
	}
}

func (dp *DrawPolygon) Setup(surface *pixels.PictureRGBA, points1 []model.XYZ, pointsLen1 int, color int, lightStart float64, lightStop float64) {
	dp.color = color
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

	dp.lightStart = lightStart
	if lightStart > lightStop {
		dp.lightStep = -(lightStart - lightStop) / float64(dp.right-dp.left) //left
	} else {
		dp.lightStep = (lightStop - lightStart) / float64(dp.right-dp.left) //right
	}
}

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

func (dp *DrawPolygon) compileNodes(pixelX int) []int {
	nodes := 0
	j := dp.pointsLen - 1
	for i := 0; i < dp.pointsLen; i++ {
		piX := int(dp.points[i].X)
		piY := int(dp.points[i].Y)
		pjX := int(dp.points[j].X)
		pjY := int(dp.points[j].Y)
		if piX < pixelX && pjX >= pixelX || pjX < pixelX && piX >= pixelX {
			val := piY + int(((float64(pixelX-piX))/float64(pjX-piX))*float64(pjY-piY))
			dp.nodeY[nodes] = val
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

func (dp *DrawPolygon) DrawTexture(texture *textures.Texture, x1 float64, x2 float64, tz1 float64, tz2 float64, u0 float64, u1 float64, yRef float64) {
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

	for pixelX := dp.left; pixelX <= dp.right; pixelX++ {
		if nodeY := dp.compileNodes(pixelX); nodeY != nil {
			txtX := int((u0*((x2-float64(pixelX))*tz2) + u1*((float64(pixelX)-x1)*tz1)) / ((x2-float64(pixelX))*tz2 + (float64(pixelX)-x1)*tz1))
			//TODO BUG VALORE NEGATIVO IN txtX!!!!!
			if txtX < 0 {
				txtX = mathematic.Abs(txtX)
			}

			for i := 0; i < len(nodeY); i += 2 {
				y1 := nodeY[i]
				y2 := nodeY[i+1]
				//div := (float64(y2) * ty2Scale) - (float64(y1) * ty1Scale)
				div := ((float64(y2)) - (float64(y1))) * yRef
				if (y1 < 0 && y2 < 0) || (y1 >= dp.maxH && y2 >= dp.maxH) {
					continue
				}
				//cY1 := clamp(y1, -1, dp.maxH)
				//cY2 := clamp(y2, -1, dp.maxH)
				cY1 := mathematic.Clamp(y1, 0, dp.lastH)
				cY2 := mathematic.Clamp(y2, 0, dp.lastH)

				//txtY := int(((float64(cY1))-(float64(y1)))*(float64(TextureEnd-TextureBegin))/div + TextureBegin)
				//r0, g0, b0 := toRGB(texture.Get(txtX, txtY), lightStart)

				for pixelY := cY1; pixelY <= cY2; pixelY++ {
					//txtY := int(float64(pixelY - y1) * float64(TextureEnd - TextureBegin) / float64(y2 - y1) + TextureBegin)
					txtY := int(((float64(pixelY))-(float64(y1)))*(float64(textures.TextureEnd-textures.TextureBegin))/div + textures.TextureBegin)
					//TODO REMOVE
					dp.lightStart = 1.0

					r0, g0, b0 := toRGB(texture.Get(uint(txtX), uint(txtY)), dp.lightStart)
					dp.surface.SetRGBA(pixelX, pixelY, r0, g0, b0, 255)
				}
			}
		}
		dp.lightStart += dp.lightStep
	}
}

func (dp *DrawPolygon) DrawPerspectiveTexture(x float64, y float64, z float64, yaw float64, aSin float64, aCos float64, texture *textures.Texture, yMap float64) {
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
	const textureZoom = 256
	p1 := (yMap - z) * dp.screenVFov
	p2 := yaw * dp.screenVFov

	for pixelX := dp.left; pixelX <= dp.right; pixelX++ {
		if nodeY := dp.compileNodes(pixelX); nodeY != nil {
			for i := 0; i < len(nodeY); i += 2 {
				y1 := nodeY[i]
				y2 := nodeY[i+1]
				if (y1 < 0 && y2 < 0) || (y1 >= dp.maxH && y2 >= dp.maxH) {
					continue
				}
				cY1 := mathematic.Clamp(y1, 0, dp.lastH)
				cY2 := mathematic.Clamp(y2, 0, dp.lastH)
				p3 := (dp.halfW - float64(pixelX)) / dp.screenHFov
				for pixelY := cY1; pixelY <= cY2; pixelY++ {
					tz := p1 / ((dp.halfH - float64(pixelY)) - p2)
					tx := tz * p3
					txtX := (((tz * aCos) + (tx * aSin)) + x) * textureZoom
					txtZ := (((tz * aSin) - (tx * aCos)) + y) * textureZoom
					//TODO REMOVE
					dp.lightStart = 1.0

					red, green, blue := toRGB(texture.Get(uint(txtZ), uint(txtX)), dp.lightStart)
					dp.surface.SetRGBA(pixelX, pixelY, red, green, blue, 255)
				}
			}
		}
		dp.lightStart += dp.lightStep
	}
}

func (dp *DrawPolygon) DrawWireFrame(filled bool) {
	if dp.surface == nil {
		return
	}
	if !dp.Verify() {
		return
	}
	for pixelX := dp.left; pixelX <= dp.right; pixelX++ {
		if nodeY := dp.compileNodes(pixelX); nodeY != nil {
			//TODO REMOVE
			dp.lightStart = 1.0

			r0, g0, b0 := toRGB(dp.color, dp.lightStart)
			r1, g1, b1 := toRGB(dp.color, dp.lightStart/2)
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
		dp.lightStart += dp.lightStep
	}
}

func (dp *DrawPolygon) DrawPoints(size int) {
	if dp.surface == nil {
		return
	}
	r0, g0, b0 := toRGB(dp.color, dp.lightStart)
	for _, k := range dp.points {
		dp.surface.SetRGBASize(int(k.X), int(k.Y), r0, g0, b0, 255, size)
	}
}

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

func (dp *DrawPolygon) drawLine(x1 float64, y1 float64, x2 float64, y2 float64) {
	// Bresenham's line algorithm
	r0, g0, b0 := toRGB(dp.color, dp.lightStart)
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
