package executor

import (
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type Texture struct {
	tex           binder
	width, height int
	smooth        bool
}

func NewTexture(width, height int, smooth bool, pixels []uint8) *Texture {
	tex := &Texture{
		tex: binder{
			restoreLoc: gl.TEXTURE_BINDING_2D,
			bindFunc: func(obj uint32) {
				gl.BindTexture(gl.TEXTURE_2D, obj)
			},
		},
		width:  width,
		height: height,
	}

	gl.GenTextures(1, &tex.tex.obj)

	tex.Begin()

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))

	//borderColor := mgl32.Vec4{0, 0, 0, 0}
	//gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

	tex.SetSmooth(smooth)

	runtime.SetFinalizer(tex, (*Texture).delete)

	tex.End()

	return tex
}

func (t *Texture) delete() {
	Thread.Post(func() {
		gl.DeleteTextures(1, &t.tex.obj)
	})
}

func (t *Texture) ID() uint32 {
	return t.tex.obj
}

func (t *Texture) Width() int {
	return t.width
}

func (t *Texture) Height() int {
	return t.height
}

func (t *Texture) SetPixels(x, y, w, h int, pixels []uint8) {
	if len(pixels) != w*h*4 {
		return
		//panic("set pixels: wrong number of pixels")
	}
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(x), int32(y), int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
}

func (t *Texture) Pixels(x, y, w, h int) []uint8 {
	pixels := make([]uint8, t.width*t.height*4)
	gl.GetTexImage(gl.TEXTURE_2D, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	subPixels := make([]uint8, w*h*4)
	for i := 0; i < h; i++ {
		row := pixels[(i+y)*t.width*4+x*4 : (i+y)*t.width*4+(x+w)*4]
		subRow := subPixels[i*w*4 : (i+1)*w*4]
		copy(subRow, row)
	}
	return subPixels
}

func (t *Texture) SetSmooth(smooth bool) {
	t.smooth = smooth
	if smooth {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	} else {
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	}
}

func (t *Texture) Smooth() bool {
	return t.smooth
}

func (t *Texture) Begin() {
	t.tex.bind()
}

func (t *Texture) End() {
	t.tex.restore()
}
