package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
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
	prg   uint32
	table [PostLocLast]int32

	// FBO Standard per il Post-Processing
	fbo             uint32
	texColorBuffer  uint32
	texBrightBuffer uint32

	// FBO Multisampled per il rendering 3D
	msaaFbo             uint32
	texColorBufferMSAA  uint32
	texBrightBufferMSAA uint32
	rboDepthMSAA        uint32

	width  int32
	height int32
	vao    uint32
	vbo    uint32

	exposure       float32
	contrast       float32
	saturation     float32
	bloomIntensity float32
	bloomBlur      int32
}

// NewPost creates and returns a pointer to a new Post instance with default rendering configuration values.
func NewPost() *Post {
	return &Post{
		exposure:       0.1,
		contrast:       1.05,
		saturation:     1.0,
		bloomIntensity: 0.05,
		bloomBlur:      1.0,
	}
}

// Setup configures the dimensions for the post-processing system by setting the width and height properties.
func (s *Post) Setup(width, height int32) error {
	s.width, s.height = width, height
	return nil
}

// GetBrightBuffer returns the texture buffer ID assigned for bloom and brightness post-processing effects.
func (s *Post) GetBrightBuffer() uint32 {
	return s.texBrightBuffer
}

// GetFBO returns the ID of the multisampled framebuffer object (FBO) used for 3D rendering in the post-processing pipeline.
func (s *Post) GetFBO() uint32 {
	return s.msaaFbo
}

// SetupSamplers initializes the samplers and VAO/VBO for rendering a screen quad in the post-processing pipeline.
func (s *Post) SetupSamplers() error {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.table[PostLocHDRBuffer], 0)

	gl.GenVertexArrays(1, &s.vao)
	gl.BindVertexArray(s.vao)
	gl.GenBuffers(1, &s.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, nil)
	gl.EnableVertexAttribArray(0)

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

	vSh, err := ShaderCompile(vertId, string(vSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(fragId, string(fSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram("post", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[PostLocHDRBuffer] = gl.GetUniformLocation(s.prg, gl.Str("u_hdrBuffer\x00"))
	s.table[PostLocExposure] = gl.GetUniformLocation(s.prg, gl.Str("u_exposure\x00"))
	s.table[PostLocContrast] = gl.GetUniformLocation(s.prg, gl.Str("u_contrast\x00"))
	s.table[PostLocSaturation] = gl.GetUniformLocation(s.prg, gl.Str("u_saturation\x00"))
	s.table[PostLocBloomIntensity] = gl.GetUniformLocation(s.prg, gl.Str("u_bloomIntensity\x00"))
	s.table[PostLocBloomBlur] = gl.GetUniformLocation(s.prg, gl.Str("u_bloomBlur\x00"))

	// --- 1. MSAA FBO (Target Principale 4x Anti-Aliasing) ---
	gl.GenFramebuffers(1, &s.msaaFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.msaaFbo)

	gl.GenTextures(1, &s.texColorBufferMSAA)
	gl.BindTexture(gl.TEXTURE_2D_MULTISAMPLE, s.texColorBufferMSAA)
	gl.TexImage2DMultisample(gl.TEXTURE_2D_MULTISAMPLE, 4, gl.RGBA16F, s.width, s.height, true)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D_MULTISAMPLE, s.texColorBufferMSAA, 0)

	gl.GenTextures(1, &s.texBrightBufferMSAA)
	gl.BindTexture(gl.TEXTURE_2D_MULTISAMPLE, s.texBrightBufferMSAA)
	gl.TexImage2DMultisample(gl.TEXTURE_2D_MULTISAMPLE, 4, gl.RGBA16F, s.width, s.height, true)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT1, gl.TEXTURE_2D_MULTISAMPLE, s.texBrightBufferMSAA, 0)

	attachments := []uint32{gl.COLOR_ATTACHMENT0, gl.COLOR_ATTACHMENT1}
	gl.DrawBuffers(2, &attachments[0])

	gl.GenRenderbuffers(1, &s.rboDepthMSAA)
	gl.BindRenderbuffer(gl.RENDERBUFFER, s.rboDepthMSAA)
	gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, 4, gl.DEPTH_COMPONENT24, s.width, s.height)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, s.rboDepthMSAA)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("post MSAA FBO not complete")
	}

	// --- 2. RESOLVE FBO (Target Piatto per il Post-Processing) ---
	gl.GenFramebuffers(1, &s.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.fbo)

	gl.GenTextures(1, &s.texColorBuffer)
	gl.BindTexture(gl.TEXTURE_2D, s.texColorBuffer)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, s.width, s.height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.texColorBuffer, 0)

	gl.GenTextures(1, &s.texBrightBuffer)
	gl.BindTexture(gl.TEXTURE_2D, s.texBrightBuffer)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, s.width, s.height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT1, gl.TEXTURE_2D, s.texBrightBuffer, 0)

	gl.DrawBuffers(2, &attachments[0])

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("post Resolve FBO not complete")
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return nil
}

// Init initializes the Post object and prepares it for usage, returning an error if initialization fails.
func (s *Post) Init() error {
	return nil
}

// Prepare prepares the post-processing pipeline by resolving the multisample anti-aliasing (MSAA) buffers to standard buffers.
func (s *Post) Prepare() {
	// Physically resolve the multisampled FBO before 2D filters
	s.resolveMSAA()
}

// resolveMSAA resolves a multisample anti-aliasing (MSAA) framebuffer to a standard framebuffer for post-processing.
func (s *Post) resolveMSAA() {
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, s.msaaFbo)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, s.fbo)

	// Blit Albedo Base
	gl.ReadBuffer(gl.COLOR_ATTACHMENT0)
	gl.DrawBuffer(gl.COLOR_ATTACHMENT0)
	gl.BlitFramebuffer(0, 0, s.width, s.height, 0, 0, s.width, s.height, gl.COLOR_BUFFER_BIT, gl.NEAREST)

	// Blit Canale Bloom/Brightness
	gl.ReadBuffer(gl.COLOR_ATTACHMENT1)
	gl.DrawBuffer(gl.COLOR_ATTACHMENT1)
	gl.BlitFramebuffer(0, 0, s.width, s.height, 0, 0, s.width, s.height, gl.COLOR_BUFFER_BIT, gl.NEAREST)

	// Restore FBO state for subsequent frames
	attachments := []uint32{gl.COLOR_ATTACHMENT0, gl.COLOR_ATTACHMENT1}
	gl.DrawBuffers(2, &attachments[0])
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

// Render performs final post-processing, applying exposure, contrast, saturation, and bloom effects using two texture inputs.
func (s *Post) Render(bloomTex uint32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Disable(gl.DEPTH_TEST)

	gl.UseProgram(s.prg)
	gl.Uniform1f(s.table[PostLocExposure], s.exposure)
	gl.Uniform1f(s.table[PostLocContrast], s.contrast)
	gl.Uniform1f(s.table[PostLocSaturation], s.saturation)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.texColorBuffer)

	gl.Uniform1f(s.table[PostLocBloomIntensity], s.bloomIntensity)
	gl.Uniform1i(s.table[PostLocBloomBlur], s.bloomBlur)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, bloomTex)

	gl.BindVertexArray(s.vao)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.Enable(gl.DEPTH_TEST)
}
