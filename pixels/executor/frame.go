package executor

import (
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type Frame struct {
	fb  binder
	rf  binder
	df  binder
	tex *Texture
}

func NewFrame(width, height int, smooth bool) *Frame {
	f := &Frame{
		fb: binder{
			restoreLoc: gl.FRAMEBUFFER_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindFramebuffer(gl.FRAMEBUFFER, obj)
			},
		},
		rf: binder{
			restoreLoc: gl.READ_FRAMEBUFFER_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindFramebuffer(gl.READ_FRAMEBUFFER, obj)
			},
		},
		df: binder{
			restoreLoc: gl.DRAW_FRAMEBUFFER_BINDING,
			bindFunc: func(obj uint32) {
				gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, obj)
			},
		},
		tex: NewTexture(width, height, smooth, make([]uint8, width*height*4)),
	}

	gl.GenFramebuffers(1, &f.fb.obj)

	f.fb.bind()
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, f.tex.tex.obj, 0)
	f.fb.restore()

	runtime.SetFinalizer(f, (*Frame).delete)

	return f
}

func (f *Frame) delete() {
	Thread.Post(func() {
		gl.DeleteFramebuffers(1, &f.fb.obj)
	})
}

func (f *Frame) ID() uint32 {
	return f.fb.obj
}

func (f *Frame) Begin() {
	f.fb.bind()
}

func (f *Frame) End() {
	f.fb.restore()
}

func (f *Frame) Blit(dst *Frame, sx0, sy0, sx1, sy1, dx0, dy0, dx1, dy1 int) {
	f.rf.obj = f.fb.obj
	if dst != nil {
		f.df.obj = dst.fb.obj
	} else {
		f.df.obj = 0
	}
	f.rf.bind()
	f.df.bind()

	filter := gl.NEAREST
	if f.tex.smooth {
		filter = gl.LINEAR
	}

	gl.BlitFramebuffer(
		int32(sx0), int32(sy0), int32(sx1), int32(sy1),
		int32(dx0), int32(dy0), int32(dx1), int32(dy1),
		gl.COLOR_BUFFER_BIT, uint32(filter),
	)

	f.rf.restore()
	f.df.restore()
}

func (f *Frame) Texture() *Texture {
	return f.tex
}
