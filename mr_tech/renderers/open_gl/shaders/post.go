package shaders

import (
	"fmt"

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

	w int32
	h int32
}

// NewPost creates and returns a pointer to a new Post instance with default rendering configuration values.
func NewPost(ctx api.IContext) *Post {
	return &Post{
		ctx:            ctx,
		exposure:       0.1,
		contrast:       1.05,
		saturation:     1.0,
		bloomIntensity: 0.05,
		bloomBlur:      1.0,
	}
}

// GetBrightBuffer returns the texture buffer ID assigned for bloom and brightness post-processing effects.
// GetColorBuffer returns the resolved color texture ID
func (s *Post) GetColorBuffer() uint32 {
	return s.texColorBuffer
}

// GetDepthBuffer returns the resolved depth texture ID
func (s *Post) GetDepthBuffer() uint32 {
	return s.texDepthBuffer
}

func (s *Post) GetBrightBuffer() uint32 {
	return s.texBrightBuffer
}

// GetFBO returns the ID of the multisampled framebuffer object (FBO) used for 3D rendering in the post-processing pipeline.
func (s *Post) GetFBO() uint32 {
	return s.msaaFbo
}

// SetupSamplers initializes the samplers and VAO/VBO for rendering a screen quad in the post-processing pipeline.
func (s *Post) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	s.ctx.Uniform1i(s.table[PostLocHDRBuffer], 0)

	s.ctx.GenVertexArrays(1, &s.vao)
	s.ctx.BindVertexArray(s.vao)
	s.ctx.GenBuffers(1, &s.vbo)
	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	s.ctx.BufferData(api.ARRAY_BUFFER, len(quad)*4, s.ctx.Ptr(quad), api.STATIC_DRAW)
	s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 0, nil)
	s.ctx.EnableVertexAttribArray(0)

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
	return nil
}

// Init initializes the Post object and prepares it for usage, returning an error if initialization fails.
func (s *Post) Init() error {
	return nil
}

// Prepare prepares the post-processing pipeline by resolving the multisample anti-aliasing (MSAA) buffers to standard buffers.
func (s *Post) Prepare(fbw, fbh int32) {
	if fbw != s.w || fbh != s.h {
		s.allocate(fbw, fbh)
	}
	// Physically resolve the multisampled FBO before 2D filters
	s.ResolveMSAA(fbw, fbh)
}

// Render performs final post-processing, applying exposure, contrast, saturation, and bloom effects using two texture inputs.
func (s *Post) Render(bloomTex uint32, fbW, fbH int32) {
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
	s.ctx.Disable(api.DEPTH_TEST)

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

	s.ctx.Enable(api.DEPTH_TEST)
}

// resolveMSAA resolves a multisample anti-aliasing (MSAA) framebuffer to a standard framebuffer for post-processing.
func (s *Post) ResolveMSAA(fbw, fbh int32) {
	s.ctx.BindFramebuffer(api.READ_FRAMEBUFFER, s.msaaFbo)
	s.ctx.BindFramebuffer(api.DRAW_FRAMEBUFFER, s.fbo)

	// Blit Albedo Base
	s.ctx.ReadBuffer(api.COLOR_ATTACHMENT0)
	s.ctx.DrawBuffer(api.COLOR_ATTACHMENT0)
	s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.COLOR_BUFFER_BIT, api.NEAREST)

	// Blit Canale Bloom/Brightness
	s.ctx.ReadBuffer(api.COLOR_ATTACHMENT1)
	s.ctx.DrawBuffer(api.COLOR_ATTACHMENT1)
	s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.COLOR_BUFFER_BIT, api.NEAREST)

	// Blit Depth
	s.ctx.BlitFramebuffer(0, 0, fbw, fbh, 0, 0, fbw, fbh, api.DEPTH_BUFFER_BIT, api.NEAREST)

	// Restore FBO state for subsequent frames
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.msaaFbo)
	attachments := []uint32{api.COLOR_ATTACHMENT0, api.COLOR_ATTACHMENT1}
	s.ctx.DrawBuffers(2, &attachments[0])
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

		s.ctx.DeleteFramebuffers(1, &s.fbo)
		s.ctx.DeleteTextures(1, &s.texColorBuffer)
		s.ctx.DeleteTextures(1, &s.texBrightBuffer)
		s.ctx.DeleteTextures(1, &s.texDepthBuffer)
	}

	// --- 1. MSAA FBO (Target Principale 4x Anti-Aliasing) ---
	s.ctx.GenFramebuffers(1, &s.msaaFbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.msaaFbo)

	s.ctx.GenRenderbuffers(1, &s.rboColorMSAA)
	s.ctx.BindRenderbuffer(api.RENDERBUFFER, s.rboColorMSAA)
	s.ctx.RenderbufferStorageMultisample(api.RENDERBUFFER, 4, api.RGBA16F, s.w, s.h)
	s.ctx.FramebufferRenderbuffer(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.RENDERBUFFER, s.rboColorMSAA)

	s.ctx.GenRenderbuffers(1, &s.rboBrightMSAA)
	s.ctx.BindRenderbuffer(api.RENDERBUFFER, s.rboBrightMSAA)
	s.ctx.RenderbufferStorageMultisample(api.RENDERBUFFER, 4, api.RGBA16F, s.w, s.h)
	s.ctx.FramebufferRenderbuffer(api.FRAMEBUFFER, api.COLOR_ATTACHMENT1, api.RENDERBUFFER, s.rboBrightMSAA)

	attachments := []uint32{api.COLOR_ATTACHMENT0, api.COLOR_ATTACHMENT1}
	s.ctx.DrawBuffers(2, &attachments[0])

	s.ctx.GenRenderbuffers(1, &s.rboDepthMSAA)
	s.ctx.BindRenderbuffer(api.RENDERBUFFER, s.rboDepthMSAA)
	s.ctx.RenderbufferStorageMultisample(api.RENDERBUFFER, 4, api.DEPTH_COMPONENT24, s.w, s.h)
	s.ctx.FramebufferRenderbuffer(api.FRAMEBUFFER, api.DEPTH_ATTACHMENT, api.RENDERBUFFER, s.rboDepthMSAA)

	if s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER) != api.FRAMEBUFFER_COMPLETE {
		panic("post MSAA FBO not complete: " + string(s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER)))
	}

	// --- 2. RESOLVE FBO (Target Piatto per il Post-Processing) ---
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

	s.ctx.DrawBuffers(2, &attachments[0])

	if s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER) != api.FRAMEBUFFER_COMPLETE {
		panic("post Resolve FBO not complete: " + string(s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER)))
	}

	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
}
