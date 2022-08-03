package pixels

import (
	"github.com/markel1974/godoom/pixels/executor"
	"math"
)

type GLPicture interface {
	IPictureColor
	Texture() *executor.Texture
	Update(p IPicture)
}

func NewGLPicture(p IPicture) GLPicture {
	bounds := p.Bounds()
	bx, by, bw, bh := intBounds(bounds)
	pixels := acquirePicture(p, bx, by, bw, bh, bounds.Min.X, bounds.Min.Y)
	var tex *executor.Texture

	executor.Thread.Call(func() {
		tex = executor.NewTexture(bw, bh, true, pixels)
	})

	gp := &glPicture{
		bounds: bounds,
		tex:    tex,
		pixels: pixels,
	}

	return gp
}

type glPicture struct {
	bounds Rect
	tex    *executor.Texture
	pixels []uint8
}

func (gp *glPicture) Bounds() Rect {
	return gp.bounds
}

func (gp *glPicture) Texture() *executor.Texture {
	return gp.tex
}

func (gp *glPicture) Color(at Vec) RGBA {
	if !gp.bounds.Contains(at) {
		return Alpha(0)
	}
	bx, by, bw, _ := intBounds(gp.bounds)
	x, y := int(at.X)-bx, int(at.Y)-by
	off := y*bw + x
	return RGBA{
		R: float64(gp.pixels[off*4+0]) / 255,
		G: float64(gp.pixels[off*4+1]) / 255,
		B: float64(gp.pixels[off*4+2]) / 255,
		A: float64(gp.pixels[off*4+3]) / 255,
	}
}

func (gp *glPicture) Update(p IPicture) {
	bounds := p.Bounds()
	bx, by, bw, bh := intBounds(bounds)
	pixels := acquirePicture(p, bx, by, bw, bh, bounds.Min.X, bounds.Min.Y)
	executor.Thread.Call(func() {
		gp.tex.Begin()
		gp.tex.SetPixels(bx, by, bw, bh, pixels)
		gp.tex.End()
	})
}

func acquirePicture(p IPicture, bx int, by int, bw int, bh int, bMinX float64, bMinY float64) []uint8 {

	/*
		else if pd, ok := p.(*pixel.PictureData); ok {
			pixels = make([]uint8, 4 * bw * bh)
			for y := 0; y < bh; y++ {
				for x := 0; x < bw; x++ {
					rgba := pd.Pix[y*pd.Stride+x]
					off := (y*bw + x) * 4
					pixels[off+0] = rgba.R
					pixels[off+1] = rgba.G
					pixels[off+2] = rgba.B
					pixels[off+3] = rgba.A
				}
			}
		}

	*/

	var pixels []uint8
	if pd, ok := p.(*PictureRGBA); ok {
		pixels = pd.Pixels()
	} else if p, ok := p.(IPictureColor); ok {
		pixels = make([]uint8, 4*bw*bh)

		for y := 0; y < bh; y++ {
			for x := 0; x < bw; x++ {
				at := MakeVec(
					math.Max(float64(bx+x), bMinX),
					math.Max(float64(by+y), bMinY),
				)
				color := p.Color(at)
				off := (y*bw + x) * 4
				pixels[off+0] = uint8(color.R * 255)
				pixels[off+1] = uint8(color.G * 255)
				pixels[off+2] = uint8(color.B * 255)
				pixels[off+3] = uint8(color.A * 255)
			}
		}
	} else {
		pixels = make([]uint8, 4*bw*bh)
	}
	return pixels
}
