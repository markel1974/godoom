package shaders

import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

const bloomMipCount = 5

// BloomLoc enumerates shader uniform locations for the Jimenez mip-pyramid bloom shader.
type BloomLoc int

const (
	BloomLocSrcTex = BloomLoc(iota)
	BloomLocTexelSize
	BloomLocMode
	BloomLocKaris
	BloomLocFilterRadius
	BloomLocUpsampleWeight
	BloomLocLast
)

type bloomMip struct {
	fbo uint32
	tex uint32
	w   int32
	h   int32
}

// Bloom manages a 5-level HDR mip-pyramid bloom pipeline using Karis-averaged
// 13-tap downsampling and additive 9-tap tent upsampling.
type Bloom struct {
	ctx     api.IContext
	prg     uint32
	table   [BloomLocLast]int32
	mips    [bloomMipCount]bloomMip
	vao     uint32
	vbo     uint32
	w       int32
	h       int32
	handler *Handler
	post    *Post
	quad    []float32
	quadPtr unsafe.Pointer
}

// NewBloom creates and returns a new Bloom instance.
func NewBloom() *Bloom {
	return &Bloom{}
}

// Setup binds the Bloom instance to Handler and Post resources.
func (s *Bloom) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.post = h.GetPost()

	s.quad = []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	s.quadPtr = s.ctx.Ptr(s.quad)
	return nil
}

// GetBloomTexture returns the final composited mip-0 bloom texture.
func (s *Bloom) GetBloomTexture() uint32 {
	return s.mips[0].tex
}

// Init initializes non-shader Bloom state.
func (s *Bloom) Init() error {
	return nil
}

// SetupSamplers initializes the full-screen quad VAO/VBO and binds the source sampler to unit 0.
func (s *Bloom) SetupSamplers() error {
	if s.prg != 0 {
		s.ctx.UseProgram(s.prg)
		s.ctx.Uniform1i(s.table[BloomLocSrcTex], 0)
	}

	s.ctx.GenVertexArrays(1, &s.vao)
	s.ctx.BindVertexArray(s.vao)
	s.ctx.GenBuffers(1, &s.vbo)
	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.vbo)

	s.ctx.BufferData(api.ARRAY_BUFFER, len(s.quad)*4, s.quadPtr, api.STATIC_DRAW)
	s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 0, nil)
	s.ctx.EnableVertexAttribArray(0)
	s.ctx.BindVertexArray(0)
	return nil
}

// Compile compiles and links the pyramidal bloom shader program and resolves uniform locations.
func (s *Bloom) Compile(a IAssets) error {
	const vertId = "post.vert"
	const fragId = "bloom.frag"
	vSrc, fSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}

	vSh, err := ShaderCompile(s.ctx, vertId, string(vSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(s.ctx, fragId, string(fSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram(s.ctx, "bloom", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[BloomLocSrcTex] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_srcTex\x00"))
	s.table[BloomLocTexelSize] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texelSize\x00"))
	s.table[BloomLocMode] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_mode\x00"))
	s.table[BloomLocKaris] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_karis\x00"))
	s.table[BloomLocFilterRadius] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_filterRadius\x00"))
	s.table[BloomLocUpsampleWeight] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_upsampleWeight\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in bloom: %d", idx)
		}
	}
	return nil
}

// Render executes the 5-level Jimenez downsample + tent upsample bloom chain.
func (s *Bloom) Render() {
	fbW, fbH := s.handler.GetSize()
	brightTex := s.post.GetBrightBuffer()

	if fbW != s.w || fbH != s.h || s.mips[0].fbo == 0 {
		s.allocate(fbW, fbH)
	}

	if s.handler.GetBloomIntensity() <= 0.001 {
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.mips[0].fbo)
		s.ctx.Viewport(0, 0, s.mips[0].w, s.mips[0].h)
		s.ctx.ClearColor(0.0, 0.0, 0.0, 0.0)
		s.ctx.Clear(api.COLOR_BUFFER_BIT)
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
		s.ctx.Viewport(0, 0, fbW, fbH)
		return
	}

	const filterRadius = float32(1.25)
	const upsampleWeight = float32(0.55)

	s.ctx.UseProgram(s.prg)
	s.ctx.Uniform1i(s.table[BloomLocSrcTex], 0)
	s.ctx.Uniform1f(s.table[BloomLocFilterRadius], filterRadius)
	s.ctx.Uniform1f(s.table[BloomLocUpsampleWeight], upsampleWeight)
	s.ctx.BindVertexArray(s.vao)
	s.ctx.Disable(api.DEPTH_TEST)
	s.ctx.Disable(api.CULL_FACE)
	s.ctx.Disable(api.BLEND)
	s.ctx.ActiveTexture(api.TEXTURE0)

	// 1. Progressive 13-tap downsample pyramid (mip 0 uses Karis weighting).
	s.ctx.Uniform1i(s.table[BloomLocMode], 0)
	srcTex := brightTex
	srcW, srcH := fbW, fbH
	for i := 0; i < bloomMipCount; i++ {
		mip := &s.mips[i]
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, mip.fbo)
		s.ctx.Viewport(0, 0, mip.w, mip.h)
		karis := int32(0)
		if i == 0 {
			karis = 1
		}
		s.ctx.Uniform1i(s.table[BloomLocKaris], karis)
		s.ctx.Uniform2f(s.table[BloomLocTexelSize], 1.0/float32(max(srcW, 1)), 1.0/float32(max(srcH, 1)))
		s.ctx.BindTexture(api.TEXTURE_2D, srcTex)
		s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)

		srcTex = mip.tex
		srcW, srcH = mip.w, mip.h
	}

	// 2. Additive 9-tap tent upsample from coarsest mip back down to mip 0.
	s.ctx.Enable(api.BLEND)
	s.ctx.BlendEquation(api.FUNC_ADD)
	s.ctx.BlendFunc(api.ONE, api.ONE)
	s.ctx.Uniform1i(s.table[BloomLocMode], 1)
	s.ctx.Uniform1i(s.table[BloomLocKaris], 0)

	for i := bloomMipCount - 1; i > 0; i-- {
		srcMip := &s.mips[i]
		dstMip := &s.mips[i-1]
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, dstMip.fbo)
		s.ctx.Viewport(0, 0, dstMip.w, dstMip.h)
		s.ctx.Uniform2f(s.table[BloomLocTexelSize], 1.0/float32(max(srcMip.w, 1)), 1.0/float32(max(srcMip.h, 1)))
		s.ctx.BindTexture(api.TEXTURE_2D, srcMip.tex)
		s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)
	}

	s.ctx.Disable(api.BLEND)
	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
	s.ctx.BindVertexArray(0)
	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.Enable(api.DEPTH_TEST)
}

// allocate creates or resizes the 5-level HDR bloom mip pyramid.
func (s *Bloom) allocate(width, height int32) {
	s.w = width
	s.h = height

	for i := 0; i < bloomMipCount; i++ {
		if s.mips[i].fbo != 0 {
			s.ctx.DeleteFramebuffers(1, &s.mips[i].fbo)
			s.mips[i].fbo = 0
		}
		if s.mips[i].tex != 0 {
			s.ctx.DeleteTextures(1, &s.mips[i].tex)
			s.mips[i].tex = 0
		}
	}

	mipW := max(width/2, 1)
	mipH := max(height/2, 1)

	s.ctx.ActiveTexture(api.TEXTURE0)
	for i := 0; i < bloomMipCount; i++ {
		var fbo, tex uint32
		s.ctx.GenFramebuffers(1, &fbo)
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, fbo)
		s.ctx.GenTextures(1, &tex)
		s.ctx.BindTexture(api.TEXTURE_2D, tex)

		s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA16F, mipW, mipH, 0, api.RGBA, api.FLOAT, nil)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
		s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, tex, 0)

		if v := s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER); v != api.FRAMEBUFFER_COMPLETE {
			panic("bloom mip FBO not complete: " + strconv.Itoa(int(v)))
		}

		s.mips[i] = bloomMip{fbo: fbo, tex: tex, w: mipW, h: mipH}
		mipW = max(mipW/2, 1)
		mipH = max(mipH/2, 1)
	}

	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
}
