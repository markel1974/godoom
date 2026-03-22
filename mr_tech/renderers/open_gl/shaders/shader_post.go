package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// ShaderPostLoc represents a numerical identifier for specific shader post-processing locations.
type ShaderPostLoc int

// ShaderPostLocHDRBuffer represents the location for the HDR buffer in the shader pipeline.
// ShaderPostLocExposure represents the location for exposure adjustment in the shader pipeline.
// ShaderPostLocContrast represents the location for contrast adjustment in the shader pipeline.
// ShaderPostLocSaturation represents the location for saturation adjustment in the shader pipeline.
// ShaderPostLocLast represents the sentinel value for the last shader post-processing location.
const (
	ShaderPostLocHDRBuffer = ShaderPostLoc(iota)
	ShaderPostLocExposure
	ShaderPostLocContrast
	ShaderPostLocSaturation
	ShaderPostLocLast
)

// ShaderPost represents a post-processing shader structure with framebuffer, textures, and various configurable parameters.
type ShaderPost struct {
	prg            uint32
	table          [ShaderPostLocLast]int32
	fbo            uint32
	texColorBuffer uint32
	rboDepth       uint32
	width          int32
	height         int32
	vao            uint32
	vbo            uint32

	Exposure   float32
	Contrast   float32
	Saturation float32
}

// NewShaderPost initializes and returns a pointer to a ShaderPost with predefined Exposure, Contrast, and Saturation values.
func NewShaderPost() *ShaderPost {
	return &ShaderPost{
		Exposure:   1.2,
		Contrast:   1.05,
		Saturation: 1.1,
	}
}

// Setup initializes the width and height properties of the ShaderPost instance.
func (s *ShaderPost) Setup(width, height int32) {
	s.width, s.height = width, height
}

// SetupSamplers initializes OpenGL vertex arrays, buffers, and sets up the shader samplers for post-processing rendering.
func (s *ShaderPost) SetupSamplers() {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.table[ShaderPostLocHDRBuffer], 0)

	gl.GenVertexArrays(1, &s.vao)
	gl.BindVertexArray(s.vao)
	gl.GenBuffers(1, &s.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	gl.BufferData(gl.ARRAY_BUFFER, len(quad)*4, gl.Ptr(quad), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, nil)
	gl.EnableVertexAttribArray(0)
}

// Compile initializes the shader program, framebuffers, and texture buffers required for the post-processing effect.
func (s *ShaderPost) Compile(a IAssets) error {
	vSrc, fSrc, err := a.ReadMulti("post.vert", "post.frag")
	if err != nil {
		return err
	}

	vSh, err := ShaderCompile(string(vSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(string(fSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram(vSh, fSh)
	if err != nil {
		return err
	}

	s.table[ShaderPostLocHDRBuffer] = gl.GetUniformLocation(s.prg, gl.Str("u_hdrBuffer\x00"))
	s.table[ShaderPostLocExposure] = gl.GetUniformLocation(s.prg, gl.Str("u_exposure\x00"))
	s.table[ShaderPostLocContrast] = gl.GetUniformLocation(s.prg, gl.Str("u_contrast\x00"))
	s.table[ShaderPostLocSaturation] = gl.GetUniformLocation(s.prg, gl.Str("u_saturation\x00"))

	gl.GenFramebuffers(1, &s.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.fbo)

	gl.GenTextures(1, &s.texColorBuffer)
	gl.BindTexture(gl.TEXTURE_2D, s.texColorBuffer)
	// Formato HDR 16-bit lineare
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, s.width, s.height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.texColorBuffer, 0)

	gl.GenRenderbuffers(1, &s.rboDepth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, s.rboDepth)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, s.width, s.height)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, s.rboDepth)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("post FBO not complete")
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return nil
}

// GetFBO retrieves the framebuffer object (FBO) ID associated with the ShaderPost instance.
func (s *ShaderPost) GetFBO() uint32 { return s.fbo }

// Render handles the post-processing rendering pipeline, including clearing buffers and applying shader effects.
func (s *ShaderPost) Render() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	// Il Clear è rimosso. Il quad sovrascrive il 100% dei pixel.
	gl.Disable(gl.DEPTH_TEST)

	gl.UseProgram(s.prg)
	gl.Uniform1f(s.table[ShaderPostLocExposure], s.Exposure)
	gl.Uniform1f(s.table[ShaderPostLocContrast], s.Contrast)
	gl.Uniform1f(s.table[ShaderPostLocSaturation], s.Saturation)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.texColorBuffer)

	gl.BindVertexArray(s.vao)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.Enable(gl.DEPTH_TEST)
}
