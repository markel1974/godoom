package atlas

import "github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"

// Element manages a backing depth framebuffer and texture for one shadow atlas buffer.
type Element struct {
	ctx    api.IContext
	fbo    uint32
	tex    uint32
	width  int32
	height int32
	gen    uint64
}

// NewElement creates and returns a new instance of Element with default uninitialized properties.
func NewElement(ctx api.IContext) *Element {
	return &Element{
		ctx: ctx,
	}
}

// GetGen returns the generation counter for this atlas data.
func (d *Element) GetGen() uint64 {
	return d.gen
}

// GetFBO returns the framebuffer object handle for the shadow atlas depth buffer.
func (d *Element) GetFBO() uint32 {
	return d.fbo
}

// GetTex returns the texture handle for the atlas depth texture.
func (d *Element) GetTex() uint32 {
	return d.tex
}

// Update initializes and configures the framebuffer and texture for depth rendering with the given dimensions.
func (d *Element) Update(width, height int32) {
	if d.ctx == nil || width <= 0 || height <= 0 {
		return
	}
	if d.fbo != 0 && d.width == width && d.height == height {
		return
	}
	nextGen := d.gen + 1
	d.Shutdown()
	d.gen = nextGen
	d.width = width
	d.height = height
	var fbo, tex uint32
	borderColor := [4]float32{1.0, 1.0, 1.0, 1.0}
	d.ctx.GenFramebuffers(1, &fbo)
	d.ctx.GenTextures(1, &tex)
	d.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
	d.ctx.BindTexture(api.TEXTURE_2D, tex)
	d.ctx.TexImage2D(api.TEXTURE_2D, 0, api.DEPTH_COMPONENT32F, width, height, 0, api.DEPTH_COMPONENT, api.FLOAT, nil)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_COMPARE_MODE, api.COMPARE_REF_TO_TEXTURE)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_COMPARE_FUNC, api.LEQUAL)

	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_BORDER)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_BORDER)
	d.ctx.TexParameterfv(api.TEXTURE_2D, api.TEXTURE_BORDER_COLOR, &borderColor[0])

	d.ctx.BindFramebuffer(api.FRAMEBUFFER, fbo)
	d.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.DEPTH_ATTACHMENT, api.TEXTURE_2D, tex, 0)
	d.ctx.DrawBuffer(api.NONE)
	d.ctx.ReadBuffer(api.NONE)
	d.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
	d.ctx.BindTexture(api.TEXTURE_2D, 0)
	d.fbo = fbo
	d.tex = tex
}

// Shutdown releases OpenGL resources associated with the framebuffer and texture of the Element.
func (d *Element) Shutdown() {
	if d.ctx != nil {
		if d.fbo != 0 {
			d.ctx.DeleteFramebuffers(1, &d.fbo)
		}
		if d.tex != 0 {
			d.ctx.DeleteTextures(1, &d.tex)
		}
	}
	d.fbo = 0
	d.tex = 0
	d.width = 0
	d.height = 0
	d.gen = 0
}
