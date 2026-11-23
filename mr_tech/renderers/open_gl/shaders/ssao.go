package shaders

import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

// SSAOLoc enumerates uniform locations for the depth-only screen-space ambient occlusion shader.
type SSAOLoc int

const (
	SSAOLocDepthTex = SSAOLoc(iota)
	SSAOLocProjParams
	SSAOLocTexelSize
	SSAOLocRadius
	SSAOLocBias
	SSAOLocLast
)

// ssaoBlurLoc enumerates uniform locations for the depth-aware bilateral blur shader.
type ssaoBlurLoc int

const (
	ssaoBlurLocInput = ssaoBlurLoc(iota)
	ssaoBlurLocDepthTex
	ssaoBlurLocTexelSize
	ssaoBlurLocProjParams
	ssaoBlurLocLast
)

// SSAO executes a half-resolution depth-only Horizon/SAO pass followed by a cross-bilateral
// edge-preserving blur between Z-Prepass and forward shading.
type SSAO struct {
	ctx        api.IContext
	prg        uint32
	table      [SSAOLocLast]int32
	blurPrg    uint32
	blurTable  [ssaoBlurLocLast]int32
	aoFbo      uint32
	aoTex      uint32
	blurFbo    uint32
	blurTex    uint32
	fallback   uint32
	quadVAO    uint32
	quadVBO    uint32
	quad       []float32
	quadPtr    unsafe.Pointer
	radius     float32
	bias       float32
	fullW      int32
	fullH      int32
	halfW      int32
	halfH      int32
	handler    *Handler
	mapMetrics *metrics.Map
}

// NewSSAO initializes and returns a new SSAO pass instance.
func NewSSAO() *SSAO {
	return &SSAO{}
}

// Setup connects the SSAO pass to Handler and camera projection metrics.
func (s *SSAO) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.mapMetrics = h.GetMapMetrics()
	s.radius = h.GetRadius()
	s.bias = h.GetBias()
	s.quad = []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	s.quadPtr = s.ctx.Ptr(s.quad)
	return nil
}

// Compile compiles both the depth-only AO pass (ssao.frag) and bilateral blur (ssao_blur.frag).
func (s *SSAO) Compile(a IAssets) error {
	const vertID = "post.vert"
	const aoFragID = "ssao.frag"
	const blurFragID = "ssao_blur.frag"

	vertSrc, aoFragSrc, err := a.ReadMulti(vertID, aoFragID)
	if err != nil {
		return err
	}
	vSh, err := ShaderCompile(s.ctx, vertID, string(vertSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(s.ctx, aoFragID, string(aoFragSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vSh)
		return err
	}
	s.prg, err = ShaderCreateProgram(s.ctx, "ssao", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[SSAOLocDepthTex] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_depthTex\x00"))
	s.table[SSAOLocProjParams] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projParams\x00"))
	s.table[SSAOLocTexelSize] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texelSize\x00"))
	s.table[SSAOLocRadius] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_radius\x00"))
	s.table[SSAOLocBias] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_bias\x00"))
	for idx, loc := range s.table {
		if loc < 0 {
			return fmt.Errorf("invalid uniform location in ssao: %d", idx)
		}
	}

	blurFragSrc, err := a.Read(blurFragID)
	if err != nil {
		return err
	}
	bvSh, err := ShaderCompile(s.ctx, vertID, string(vertSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	bfSh, err := ShaderCompile(s.ctx, blurFragID, string(blurFragSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(bvSh)
		return err
	}
	s.blurPrg, err = ShaderCreateProgram(s.ctx, "ssao_blur", bvSh, bfSh)
	if err != nil {
		return err
	}

	s.blurTable[ssaoBlurLocInput] = s.ctx.GetUniformLocation(s.blurPrg, s.ctx.Str("u_ssaoInput\x00"))
	s.blurTable[ssaoBlurLocDepthTex] = s.ctx.GetUniformLocation(s.blurPrg, s.ctx.Str("u_depthTex\x00"))
	s.blurTable[ssaoBlurLocTexelSize] = s.ctx.GetUniformLocation(s.blurPrg, s.ctx.Str("u_texelSize\x00"))
	s.blurTable[ssaoBlurLocProjParams] = s.ctx.GetUniformLocation(s.blurPrg, s.ctx.Str("u_projParams\x00"))
	for idx, loc := range s.blurTable {
		if loc < 0 {
			return fmt.Errorf("invalid uniform location in ssao_blur: %d", idx)
		}
	}
	return nil
}

// Init creates the full-screen quad VAO/VBO and 1x1 white fallback AO texture.
func (s *SSAO) Init() error {
	s.ensureFallbackAndQuad()
	return nil
}

// SetupSamplers binds fixed sampler unit indices and default parameters for SSAO programs.
func (s *SSAO) SetupSamplers() error {
	if s.prg != 0 {
		s.ctx.UseProgram(s.prg)
		s.ctx.Uniform1i(s.table[SSAOLocDepthTex], 0)
		s.ctx.Uniform1f(s.table[SSAOLocRadius], s.radius)
		s.ctx.Uniform1f(s.table[SSAOLocBias], s.bias)
	}
	if s.blurPrg != 0 {
		s.ctx.UseProgram(s.blurPrg)
		s.ctx.Uniform1i(s.blurTable[ssaoBlurLocInput], 0)
		s.ctx.Uniform1i(s.blurTable[ssaoBlurLocDepthTex], 1)
	}
	return nil
}

func (s *SSAO) ensureFallbackAndQuad() {
	if s.fallback == 0 && s.ctx != nil {
		white := []byte{255, 255, 255, 255}
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitSSAO)
		s.ctx.GenTextures(1, &s.fallback)
		s.ctx.BindTexture(api.TEXTURE_2D, s.fallback)
		s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA8, 1, 1, 0, api.RGBA, api.UNSIGNED_BYTE, s.ctx.Ptr(white))
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
		s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	if s.quadVAO == 0 && s.ctx != nil {
		if len(s.quad) == 0 {
			s.quad = []float32{-1, -1, 1, -1, -1, 1, 1, 1}
			s.quadPtr = s.ctx.Ptr(s.quad)
		}
		s.ctx.GenVertexArrays(1, &s.quadVAO)
		s.ctx.BindVertexArray(s.quadVAO)
		s.ctx.GenBuffers(1, &s.quadVBO)
		s.ctx.BindBuffer(api.ARRAY_BUFFER, s.quadVBO)
		s.ctx.BufferData(api.ARRAY_BUFFER, len(s.quad)*4, s.quadPtr, api.STATIC_DRAW)
		s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 0, nil)
		s.ctx.EnableVertexAttribArray(0)
		s.ctx.BindVertexArray(0)
	}
}

// IsReady reports whether the depth-only SSAO and blur programs are compiled and ready to execute.
func (s *SSAO) IsReady() bool {
	return s.prg != 0 && s.blurPrg != 0
}

// GetProgram returns the SSAO shader program ID.
func (s *SSAO) GetProgram() uint32 {
	return s.prg
}

// GetPositionDepthTexture returns 0 (depth-only forward SSAO uses no G-Buffer).
func (s *SSAO) GetPositionDepthTexture() uint32 { return 0 }

// GetNormalTexture returns 0 (depth-only forward SSAO uses no G-Buffer).
func (s *SSAO) GetNormalTexture() uint32 { return 0 }

// GetAlbedoTexture returns 0 (depth-only forward SSAO uses no G-Buffer).
func (s *SSAO) GetAlbedoTexture() uint32 { return 0 }

// GetEmissiveTexture returns 0 (depth-only forward SSAO uses no G-Buffer).
func (s *SSAO) GetEmissiveTexture() uint32 { return 0 }

// GetSSAOBlurTexture returns the blurred ambient occlusion texture ID (or a 1x1 white fallback if not yet rendered).
func (s *SSAO) GetSSAOBlurTexture() uint32 {
	if s.blurTex != 0 {
		return s.blurTex
	}
	return s.fallback
}

// GetFallbackTexture returns the 1x1 white fallback texture for safe unbound sampling when SSAO is disabled.
func (s *SSAO) GetFallbackTexture() uint32 {
	s.ensureFallbackAndQuad()
	return s.fallback
}

// RenderDepth computes half-resolution depth-only SSAO from depthTex and runs the cross-bilateral blur pass.
func (s *SSAO) RenderDepth(depthTex uint32, fullW, fullH int32) {
	if !s.IsReady() || depthTex == 0 || fullW <= 0 || fullH <= 0 {
		return
	}
	s.ensureFallbackAndQuad()
	if fullW != s.fullW || fullH != s.fullH || s.aoFbo == 0 || s.blurFbo == 0 {
		s.allocate(fullW, fullH)
	}

	proj := s.mapMetrics.GetProjMatrix()
	p0, p5, p10, p14 := proj[0], proj[5], proj[10], proj[14]
	if p0 == 0 {
		p0 = 1
	}
	if p5 == 0 {
		p5 = 1
	}
	invP0 := 1.0 / p0
	invP5 := 1.0 / p5

	s.ctx.Disable(api.DEPTH_TEST)
	s.ctx.Disable(api.CULL_FACE)
	s.ctx.Disable(api.BLEND)
	s.ctx.DepthMask(false)
	s.ctx.ColorMask(true, true, true, true)
	s.ctx.BindVertexArray(s.quadVAO)

	// Pass 1: Half-res horizon SAO from Z-Prepass depth.
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.aoFbo)
	s.ctx.Viewport(0, 0, s.halfW, s.halfH)
	s.ctx.UseProgram(s.prg)
	s.ctx.Uniform1i(s.table[SSAOLocDepthTex], 0)
	s.ctx.Uniform4f(s.table[SSAOLocProjParams], invP0, invP5, p10, p14)
	s.ctx.Uniform2f(s.table[SSAOLocTexelSize], 1.0/float32(s.fullW), 1.0/float32(s.fullH))
	s.ctx.Uniform1f(s.table[SSAOLocRadius], s.radius)
	s.ctx.Uniform1f(s.table[SSAOLocBias], s.bias)
	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, depthTex)
	s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)

	// Pass 2: Edge-preserving cross-bilateral blur.
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.blurFbo)
	s.ctx.Viewport(0, 0, s.halfW, s.halfH)
	s.ctx.UseProgram(s.blurPrg)
	s.ctx.Uniform1i(s.blurTable[ssaoBlurLocInput], 0)
	s.ctx.Uniform1i(s.blurTable[ssaoBlurLocDepthTex], 1)
	s.ctx.Uniform2f(s.blurTable[ssaoBlurLocTexelSize], 1.0/float32(s.halfW), 1.0/float32(s.halfH))
	s.ctx.Uniform4f(s.blurTable[ssaoBlurLocProjParams], invP0, invP5, p10, p14)
	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, s.aoTex)
	s.ctx.ActiveTexture(api.TEXTURE1)
	s.ctx.BindTexture(api.TEXTURE_2D, depthTex)
	s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)

	// Restore texture units and state cleanly for forward shading.
	s.ctx.ActiveTexture(api.TEXTURE1)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.BindVertexArray(0)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
	s.ctx.Viewport(0, 0, fullW, fullH)
}

func (s *SSAO) allocate(width, height int32) {
	s.fullW = width
	s.fullH = height
	s.halfW = max(width/2, 1)
	s.halfH = max(height/2, 1)

	if s.aoFbo != 0 {
		s.ctx.DeleteFramebuffers(1, &s.aoFbo)
		s.ctx.DeleteTextures(1, &s.aoTex)
		s.aoFbo = 0
		s.aoTex = 0
	}
	if s.blurFbo != 0 {
		s.ctx.DeleteFramebuffers(1, &s.blurFbo)
		s.ctx.DeleteTextures(1, &s.blurTex)
		s.blurFbo = 0
		s.blurTex = 0
	}

	s.ctx.ActiveTexture(api.TEXTURE0)

	s.ctx.GenFramebuffers(1, &s.aoFbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.aoFbo)
	s.ctx.GenTextures(1, &s.aoTex)
	s.ctx.BindTexture(api.TEXTURE_2D, s.aoTex)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA8, s.halfW, s.halfH, 0, api.RGBA, api.UNSIGNED_BYTE, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.aoTex, 0)
	if v := s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER); v != api.FRAMEBUFFER_COMPLETE {
		panic("ssao FBO not complete: " + strconv.Itoa(int(v)))
	}

	s.ctx.GenFramebuffers(1, &s.blurFbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.blurFbo)
	s.ctx.GenTextures(1, &s.blurTex)
	s.ctx.BindTexture(api.TEXTURE_2D, s.blurTex)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA8, s.halfW, s.halfH, 0, api.RGBA, api.UNSIGNED_BYTE, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.blurTex, 0)
	if v := s.ctx.CheckFramebufferStatus(api.FRAMEBUFFER); v != api.FRAMEBUFFER_COMPLETE {
		panic("ssao blur FBO not complete: " + strconv.Itoa(int(v)))
	}

	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
}
