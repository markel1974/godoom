package pixels

import (
	"github.com/markel1974/godoom/pixels/executor"
)

type GLFrame struct {
	frame  *executor.Frame
	bounds Rect
	pixels []uint8
	dirty  bool
}

func NewGLFrame(bounds Rect) *GLFrame {
	gf := new(GLFrame)
	gf.SetBounds(bounds)
	return gf
}

func (gf *GLFrame) SetBounds(bounds Rect) {
	if bounds == gf.Bounds() {
		return
	}

	executor.Thread.Call(func() {
		oldF := gf.frame

		_, _, w, h := intBounds(bounds)
		if w <= 0 {
			w = 1
		}
		if h <= 0 {
			h = 1
		}
		gf.frame = executor.NewFrame(w, h, true)

		if oldF != nil {
			ox, oy, ow, oh := intBounds(bounds)
			oldF.Blit(
				gf.frame,
				ox, oy, ox+ow, oy+oh,
				ox, oy, ox+ow, oy+oh,
			)
		}
	})

	gf.bounds = bounds
	gf.pixels = nil
	gf.dirty = true
}

func (gf *GLFrame) Bounds() Rect {
	return gf.bounds
}

func (gf *GLFrame) Color(at Vec) RGBA {
	if gf.dirty {
		executor.Thread.Call(func() {
			tex := gf.frame.Texture()
			tex.Begin()
			gf.pixels = tex.Pixels(0, 0, tex.Width(), tex.Height())
			tex.End()
		})
		gf.dirty = false
	}
	if !gf.bounds.Contains(at) {
		return Alpha(0)
	}
	bx, by, bw, _ := intBounds(gf.bounds)
	x, y := int(at.X)-bx, int(at.Y)-by
	off := y*bw + x
	return RGBA{
		R: float64(gf.pixels[off*4+0]) / 255,
		G: float64(gf.pixels[off*4+1]) / 255,
		B: float64(gf.pixels[off*4+2]) / 255,
		A: float64(gf.pixels[off*4+3]) / 255,
	}
}

func (gf *GLFrame) Frame() *executor.Frame {
	return gf.frame
}

func (gf *GLFrame) Texture() *executor.Texture {
	return gf.frame.Texture()
}

func (gf *GLFrame) Dirty() {
	gf.dirty = true
}
