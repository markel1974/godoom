package shaders

import (
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// PostLoc represents positional constants used for defining specific locations or stages in a process.
type PostLoc int

// PostLocHDRBuffer represents the buffer location for high dynamic range processing.
// PostLocExposure represents the location for handling exposure settings.
// PostLocContrast represents the location for adjusting contrast settings.
// PostLocSaturation represents the location for managing saturation changes.
// PostLocBloomBlur represents the location for applying bloom blur effects.
// PostLocBloomIntensity represents the location for configuring bloom intensity.
// PostLocLast marks the end of the post-processing locations.
const (
	PostLocHDRBuffer = PostLoc(iota)
	PostLocExposure
	PostLocContrast
	PostLocSaturation
	PostLocBloomBlur
	PostLocBloomIntensity
	PostLocLast
)

// Post represents a structure used for managing post-processing effects and framebuffers in a rendering pipeline.
type Post struct {
	ctx   api.IContext
	prg   uint32
	table [PostLocLast]int32

	// FBO Standard per il Post-Processing
	fbo             uint32
	texColorBuffer  uint32
	texBrightBuffer uint32
	texDepthBuffer  uint32

	// Snapshot FBO created lazily only when MSAA is disabled and liquid refraction needs a copy.
	refractFbo         uint32
	refractColorBuffer uint32
	refractDepthBuffer uint32

	// FBO Multisampled per il rendering 3D
	msaaFbo       uint32
	rboColorMSAA  uint32
	rboBrightMSAA uint32
	rboDepthMSAA  uint32

	vao uint32
	vbo uint32

	exposure       float32
	contrast       float32
	saturation     float32
	bloomIntensity float32
	bloomBlur      int32

	handler        *Handler
	bloom          *Bloom
	attachments    []uint32
	attachmentsPtr *uint32

	w int32
	h int32

	// samples is the MSAA sample count of the scene FBO (0 = MSAA disabled).
	samples int32

	// Optional FXAA pass (runs on tone-mapped LDR buffer before presenting to FBO 0).
	fxaaEnabled  bool
	fxaaPrg      uint32
	fxaaTexLoc   int32
	fxaaTexelLoc int32
	ldrFbo       uint32
	texLdrBuffer uint32
}

// NewPost creates and returns a pointer to a new Post instance with default rendering configuration values.
func NewPost() *Post {
	p := &Post{
		attachments: []uint32{api.COLOR_ATTACHMENT0, api.COLOR_ATTACHMENT1},
	}
	p.attachmentsPtr = &p.attachments[0]
	return p
}

// Setup initializes the Post instance by assigning the rendering context from the provided Handler.
func (s *Post) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.bloom = h.GetBloom()
	s.exposure = h.GetExposure()
	s.contrast = h.GetContrast()
	s.saturation = h.GetSaturation()
	s.bloomIntensity = h.GetBloomIntensity()
	s.bloomBlur = h.GetBloomBlur()
	cal := h.GetCalibration()
	s.samples = int32(cal.MSAASamples)
	if s.samples < 0 {
		s.samples = 0
	}
	s.fxaaEnabled = cal.EnableFXAA
	return nil
}

// SetupSamplers initializes the samplers and VAO/VBO for rendering a screen quad in the post-processing pipeline.
func (s *Post) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	s.ctx.Uniform1i(s.table[PostLocHDRBuffer], 0)

	if s.fxaaEnabled {
		s.ctx.UseProgram(s.fxaaPrg)
		s.ctx.Uniform1i(s.fxaaTexLoc, 0)
	}

	s.ctx.GenVertexArrays(1, &s.vao)
	s.ctx.BindVertexArray(s.vao)
	s.ctx.GenBuffers(1, &s.vbo)
	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	s.ctx.BufferData(api.ARRAY_BUFFER, len(quad)*4, s.ctx.Ptr(quad), api.STATIC_DRAW)
	s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 0, nil)
	s.ctx.EnableVertexAttribArray(0)
	s.ctx.BindVertexArray(0)

	return nil
}

// Compile initializes and configures shaders, framebuffers, and textures required for post-processing operations.
func (s *Post) Compile(a IAssets) error {
	const vertId = "post.vert"
	const fragId = "post.frag"

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
	s.prg, err = ShaderCreateProgram(s.ctx, "post", vSh, fSh)
	if err != nil {
		return err
	}
	s.table[PostLocHDRBuffer] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_hdrBuffer\x00"))
	s.table[PostLocExposure] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_exposure\x00"))
	s.table[PostLocContrast] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_contrast\x00"))
	s.table[PostLocSaturation] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_saturation\x00"))
	s.table[PostLocBloomIntensity] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_bloomIntensity\x00"))
	s.table[PostLocBloomBlur] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_bloomBlur\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in post: %d", idx)
		}
	}

	if s.fxaaEnabled {
		if err := s.compileFXAA(a, vertId); err != nil {
			return err
		}
	}
	return nil
}

// compileFXAA compiles and links the FXAA post-process shader program.
func (s *Post) compileFXAA(a IAssets, vertId string) error {
	const fxaaFragId = "fxaa.frag"
	vSrc, fSrc, err := a.ReadMulti(vertId, fxaaFragId)
	if err != nil {
		return err
	}
	vSh, err := ShaderCompile(s.ctx, vertId, string(vSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(s.ctx, fxaaFragId, string(fSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vSh)
		return err
	}
	s.fxaaPrg, err = ShaderCreateProgram(s.ctx, "fxaa", vSh, fSh)
	if err != nil {
		return err
	}
	s.fxaaTexLoc = s.ctx.GetUniformLocation(s.fxaaPrg, s.ctx.Str("u_screenTex\x00"))
	s.fxaaTexelLoc = s.ctx.GetUniformLocation(s.fxaaPrg, s.ctx.Str("u_texelSize\x00"))
	if s.fxaaTexLoc < 0 || s.fxaaTexelLoc < 0 {
		return fmt.Errorf("invalid uniform location in fxaa: tex=%d texel=%d", s.fxaaTexLoc, s.fxaaTexelLoc)
	}
	return nil
}

// Init initializes the Post object and prepares it for usage, returning an error if initialization fails.
func (s *Post) Init() error {
	return nil
}

// EnsureAllocated guarantees that the scene MSAA, resolve, and optional LDR framebuffers exist at the active handler size.
func (s *Post) EnsureAllocated() {
	fbW, fbH := s.handler.GetSize()
	if fbW != s.w || fbH != s.h || s.GetFBO() == 0 {
		s.allocate(fbW, fbH)
	}
}

// Prepare prepares the post-processing pipeline by resolving the multisample antialiasing (MSAA) buffers to standard buffers.
func (s *Post) Prepare() {
	s.EnsureAllocated()
	if s.samples == 0 {
		return
	}
	// Physically resolve the multisampled FBO before 2D filters.
	// Depth is consumed only by the liquid pass (resolved earlier), so skip it here.
	s.ResolveMSAA(s.w, s.h, ResolveColor|ResolveBright)
}

// Render performs final post-processing, applying exposure, contrast, saturation, and bloom effects using two texture inputs.
// When FXAA is enabled, tone-mapping writes to an intermediate LDR texture and FXAA presents to the default framebuffer.
func (s *Post) Render(winW, winH int32) {
	bloomTex := s.bloom.GetBloomTexture()
	s.ctx.Disable(api.DEPTH_TEST)

	if s.fxaaEnabled && s.ldrFbo != 0 {
		s.ctx.Viewport(0, 0, s.w, s.h)
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.ldrFbo)
	} else {
		s.ctx.Viewport(0, 0, winW, winH)
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
	}

	s.ctx.UseProgram(s.prg)
	s.ctx.Uniform1f(s.table[PostLocExposure], s.exposure)
	s.ctx.Uniform1f(s.table[PostLocContrast], s.contrast)
	s.ctx.Uniform1f(s.table[PostLocSaturation], s.saturation)

	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, s.texColorBuffer)

	s.ctx.Uniform1f(s.table[PostLocBloomIntensity], s.bloomIntensity)
	s.ctx.Uniform1i(s.table[PostLocBloomBlur], s.bloomBlur)
	s.ctx.ActiveTexture(api.TEXTURE1)
	s.ctx.BindTexture(api.TEXTURE_2D, bloomTex)

	s.ctx.BindVertexArray(s.vao)
	s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)

	if s.fxaaEnabled && s.ldrFbo != 0 {
		s.ctx.Viewport(0, 0, winW, winH)
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
		s.ctx.UseProgram(s.fxaaPrg)
		s.ctx.Uniform2f(s.fxaaTexelLoc, 1.0/float32(s.w), 1.0/float32(s.h))
		s.ctx.ActiveTexture(api.TEXTURE0)
		s.ctx.BindTexture(api.TEXTURE_2D, s.texLdrBuffer)
		s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)
	}

	s.ctx.BindVertexArray(0)
	s.ctx.Enable(api.DEPTH_TEST)
}

// ResolveMask selects which MSAA attachments ResolveMSAA must blit to the resolve FBO.
type ResolveMask uint8

const (
	// ResolveColor resolves COLOR_ATTACHMENT0 (scene color, used by liquid refraction and final post).
	ResolveColor ResolveMask = 1 << iota
	// ResolveBright resolves COLOR_ATTACHMENT1 (brightness channel, used only by bloom).
	ResolveBright
	// ResolveDepth resolves the depth attachment (used only by liquid depth fog).
	ResolveDepth
)

// postDrawAttachments is the MRT layout of the MSAA FBO (kept global to avoid per-frame allocations).
var postDrawAttachments = [2]uint32{api.COLOR_ATTACHMENT0, api.COLOR_ATTACHMENT1}

// ResolveMSAA resolves the requested attachments of the multisample (MSAA) framebuffer
// to the standard framebuffer used for sampling, or snapshots the single-sampled scene FBO
// into the lazy refraction FBO when MSAA is disabled.
func (s *Post) ResolveMSAA(fbw, fbh int32, mask ResolveMask) {
	if mask == 0 {
		return
	}
	if s.samples == 0 {
		colorDepthMask := mask & (ResolveColor | ResolveDepth)
		if colorDepthMask == 0 {
			return
		}
		if s.refractFbo == 0 {
			s.allocateRefraction()
		}
		s.ctx.BindFramebuffer(api.READ_FRAMEBUFFER, s.fbo)
		s.ctx.BindFramebuffer(api.DRAW_FRAMEBUFFER, s.refractFbo)
		if colorDepthMask&ResolveColor != 0 {
			s.ctx.ReadBuffer(api.COLOR_ATTACHMENT0)
			s.ctx.DrawBuffer(api.COLOR_ATTACHMENT0)
			s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.COLOR_BUFFER_BIT, api.NEAREST)
		}
		if colorDepthMask&ResolveDepth != 0 {
			s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.DEPTH_BUFFER_BIT, api.NEAREST)
		}
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.fbo)
		s.ctx.DrawBuffers(2, &postDrawAttachments[0])
		s.ctx.ReadBuffer(api.COLOR_ATTACHMENT0)
		return
	}
	s.ctx.BindFramebuffer(api.READ_FRAMEBUFFER, s.msaaFbo)
	s.ctx.BindFramebuffer(api.DRAW_FRAMEBUFFER, s.fbo)

	// Blit Albedo Base
	if mask&ResolveColor != 0 {
		s.ctx.ReadBuffer(api.COLOR_ATTACHMENT0)
		s.ctx.DrawBuffer(api.COLOR_ATTACHMENT0)
		s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.COLOR_BUFFER_BIT, api.NEAREST)
	}

	// Blit Canale Bloom/Brightness
	if mask&ResolveBright != 0 {
		s.ctx.ReadBuffer(api.COLOR_ATTACHMENT1)
		s.ctx.DrawBuffer(api.COLOR_ATTACHMENT1)
		s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.COLOR_BUFFER_BIT, api.NEAREST)
	}

	// Blit Depth
	if mask&ResolveDepth != 0 {
		s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.DEPTH_BUFFER_BIT, api.NEAREST)
	}

	// Restore FBO state for subsequent frames
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.msaaFbo)
	s.ctx.DrawBuffers(2, &postDrawAttachments[0])
	s.ctx.ReadBuffer(api.COLOR_ATTACHMENT0)
}

// Allocate gestisce la creazione e il ridimensionamento lazy dei Framebuffer per il post-processing (MSAA + Resolve).
func (s *Post) allocate(width, height int32) {
	s.w = width
	s.h = height

	// Prevenzione memory leak: distruzione esplicita dei buffer precedenti
	if s.msaaFbo != 0 {
		s.ctx.DeleteFramebuffers(1, &s.msaaFbo)
		s.ctx.DeleteRenderbuffers(1, &s.rboColorMSAA)
		s.ctx.DeleteRenderbuffers(1, &s.rboBrightMSAA)
		s.ctx.DeleteRenderbuffers(1, &s.rboDepthMSAA)
		s.msaaFbo = 0
		s.rboColorMSAA = 0
		s.rboBrightMSAA = 0
		s.rboDepthMSAA = 0
	}
	if s.fbo != 0 {
		s.ctx.DeleteFramebuffers(1, &s.fbo)
		s.ctx.DeleteTextures(1, &s.texColorBuffer)
		s.ctx.DeleteTextures(1, &s.texBrightBuffer)
		s.ctx.DeleteTextures(1, &s.texDepthBuffer)
		s.fbo = 0
		s.texColorBuffer = 0
		s.texBrightBuffer = 0
		s.texDepthBuffer = 0
	}
	if s.refractFbo != 0 {
		s.ctx.DeleteFramebuffers(1, &s.refractFbo)
		s.ctx.DeleteTextures(1, &s.refractColorBuffer)
		s.ctx.DeleteTextures(1, &s.refractDepthBuffer)
		s.refractFbo = 0
		s.refractColorBuffer = 0
		s.refractDepthBuffer = 0
	}
	if s.ldrFbo != 0 {
		s.ctx.DeleteFramebuffers(1, &s.ldrFbo)
		s.ctx.DeleteTextures(1, &s.texLdrBuffer)
		s.ldrFbo = 0
		s.texLdrBuffer = 0
	}

	// --- 1. MSAA Scene FBO (allocated only when MSAA is active; otherwise render directly into s.fbo) ---
	if s.samples > 0 {
		s.ctx.GenFramebuffers(1, &s.msaaFbo)
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.msaaFbo)

		s.ctx.GenRenderbuffers(1, &s.rboColorMSAA)
		s.ctx.BindRenderbuffer(api.RENDERBUFFER, s.rboColorMSAA)
		s.renderbufferStorage(api.RGBA16F)
		s.ctx.FramebufferRenderbuffer(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.RENDERBUFFER, s.rboColorMSAA)

		s.ctx.GenRenderbuffers(1, &s.rboBrightMSAA)
		s.ctx.BindRenderbuffer(api.RENDERBUFFER, s.rboBrightMSAA)
		s.renderbufferStorage(api.RGBA16F)
		s.ctx.FramebufferRenderbuffer(api.FRAMEBUFFER, api.COLOR_ATTACHMENT1, api.RENDERBUFFER, s.rboBrightMSAA)

		s.ctx.DrawBuffers(2, s.attachmentsPtr)

		s.ctx.GenRenderbuffers(1, &s.rboDepthMSAA)
		s.ctx.BindRenderbuffer(api.RENDERBUFFER, s.rboDepthMSAA)
		s.renderbufferStorage(api.DEPTH_COMPONENT24)
		s.ctx.FramebufferRenderbuffer(api.FRAMEBUFFER, api.DEPTH_ATTACHMENT, api.RENDERBUFFER, s.rboDepthMSAA)

		if v := s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER); v != api.FRAMEBUFFER_COMPLETE {
			panic("post MSAA FBO not complete: " + strconv.Itoa(int(v)))
		}
	}

	// --- 2. RESOLVE FBO (Target Piatto per il Post-Processing, e target diretto quando MSAA == 0) ---
	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.GenFramebuffers(1, &s.fbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.fbo)

	s.ctx.GenTextures(1, &s.texColorBuffer)
	s.ctx.BindTexture(api.TEXTURE_2D, s.texColorBuffer)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA16F, s.w, s.h, 0, api.RGBA, api.FLOAT, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.texColorBuffer, 0)

	s.ctx.GenTextures(1, &s.texBrightBuffer)
	s.ctx.BindTexture(api.TEXTURE_2D, s.texBrightBuffer)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA16F, s.w, s.h, 0, api.RGBA, api.FLOAT, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT1, api.TEXTURE_2D, s.texBrightBuffer, 0)

	s.ctx.GenTextures(1, &s.texDepthBuffer)
	s.ctx.BindTexture(api.TEXTURE_2D, s.texDepthBuffer)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.DEPTH_COMPONENT24, s.w, s.h, 0, api.DEPTH_COMPONENT, api.UNSIGNED_INT, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.NEAREST)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.DEPTH_ATTACHMENT, api.TEXTURE_2D, s.texDepthBuffer, 0)

	s.ctx.DrawBuffers(2, s.attachmentsPtr)

	if v := s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER); v != api.FRAMEBUFFER_COMPLETE {
		panic("post Resolve FBO not complete: " + strconv.Itoa(int(v)))
	}

	if s.fxaaEnabled {
		s.allocateLDR()
	}

	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
}

// allocateLDR allocates the RGBA8 intermediate framebuffer used as input to the FXAA pass.
func (s *Post) allocateLDR() {
	s.ctx.GenFramebuffers(1, &s.ldrFbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.ldrFbo)

	s.ctx.GenTextures(1, &s.texLdrBuffer)
	s.ctx.BindTexture(api.TEXTURE_2D, s.texLdrBuffer)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA8, s.w, s.h, 0, api.RGBA, api.UNSIGNED_BYTE, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.texLdrBuffer, 0)

	if v := s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER); v != api.FRAMEBUFFER_COMPLETE {
		panic("post LDR FBO not complete: " + strconv.Itoa(int(v)))
	}
}

// renderbufferStorage allocates storage for the currently bound renderbuffer, multisampled only when MSAA is enabled.
func (s *Post) renderbufferStorage(internalFormat uint32) {
	if s.samples > 0 {
		s.ctx.RenderbufferStorageMultisample(api.RENDERBUFFER, s.samples, internalFormat, s.w, s.h)
		return
	}
	s.ctx.RenderbufferStorage(api.RENDERBUFFER, internalFormat, s.w, s.h)
}

// allocateRefraction lazily allocates the single-sampled refraction snapshot FBO used by the liquid pass when MSAA is disabled.
func (s *Post) allocateRefraction() {
	s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitRefraction)
	s.ctx.GenFramebuffers(1, &s.refractFbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.refractFbo)

	s.ctx.GenTextures(1, &s.refractColorBuffer)
	s.ctx.BindTexture(api.TEXTURE_2D, s.refractColorBuffer)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA16F, s.w, s.h, 0, api.RGBA, api.FLOAT, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.refractColorBuffer, 0)

	s.ctx.GenTextures(1, &s.refractDepthBuffer)
	s.ctx.BindTexture(api.TEXTURE_2D, s.refractDepthBuffer)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.DEPTH_COMPONENT24, s.w, s.h, 0, api.DEPTH_COMPONENT, api.UNSIGNED_INT, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.DEPTH_ATTACHMENT, api.TEXTURE_2D, s.refractDepthBuffer, 0)

	if v := s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER); v != api.FRAMEBUFFER_COMPLETE {
		panic("post Refraction FBO not complete: " + strconv.Itoa(int(v)))
	}

	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.fbo)
}

// GetSamples returns the MSAA sample count of the scene framebuffer (0 = MSAA disabled).
func (s *Post) GetSamples() int32 {
	return s.samples
}

// GetColorBuffer returns the resolved color texture ID
func (s *Post) GetColorBuffer() uint32 {
	return s.texColorBuffer
}

// GetDepthBuffer returns the resolved depth texture ID
func (s *Post) GetDepthBuffer() uint32 {
	return s.texDepthBuffer
}

// GetRefractionColorBuffer returns the scene color texture safe for liquid sampling without FBO feedback loops.
func (s *Post) GetRefractionColorBuffer() uint32 {
	if s.samples == 0 && s.refractColorBuffer != 0 {
		return s.refractColorBuffer
	}
	return s.texColorBuffer
}

// GetRefractionDepthBuffer returns the scene depth texture safe for liquid sampling without FBO feedback loops.
func (s *Post) GetRefractionDepthBuffer() uint32 {
	if s.samples == 0 && s.refractDepthBuffer != 0 {
		return s.refractDepthBuffer
	}
	return s.texDepthBuffer
}

// GetBrightBuffer returns the texture buffer ID assigned for bloom and brightness post-processing effects.
func (s *Post) GetBrightBuffer() uint32 {
	return s.texBrightBuffer
}

// GetFBO returns the active 3D scene framebuffer object (single-sampled resolve FBO directly when MSAA is disabled).
func (s *Post) GetFBO() uint32 {
	if s.samples == 0 {
		return s.fbo
	}
	return s.msaaFbo
}
