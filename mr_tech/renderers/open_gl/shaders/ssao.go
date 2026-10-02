package shaders

import (
	"fmt"
	"math"
	rnd "math/rand"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// SSAOLoc represents an identifier for accessing SSAO shader uniform locations.
type SSAOLoc int

// ShaderSSAOLocGPosition represents the location of the G-position attribute in the SSAO shader.
// ShaderSSAOLocGNormal represents the location of the G-normal attribute in the SSAO shader.
// ShaderSSAOLocTexNoise represents the location of the texture noise attribute in the SSAO shader.
// ShaderSSAOLocSamples represents the location of the SSAO samples attribute in the shader.
// ShaderSSAOLocProjection represents the location of the projection matrix attribute in the SSAO shader.
// ShaderSSAOLocLast marks the end of the SSAOLoc constants.
const (
	SSAOLocPosition = SSAOLoc(iota)
	SSAOLocNormal
	SSAOLocTexNoise
	SSAOLocSamples
	SSAOLocProjection
	SSAOLocKernelSize
	SSAOLocRadius
	SSAOLocBias
	SSAOLocLast
)

// SSAO represents a shader implementation for Screen Space Ambient Occlusion (SSAO).
type SSAO struct {
	ctx              api.IContext
	prg              uint32
	table            [SSAOLocLast]int32
	noiseTex         uint32    // Texture di rumore 4x4
	kernel           []float32 // 64 campioni vec3
	noiseTextureSize int32
	bufferFbo        uint32
	positionDepth    uint32
	kernelSize       int32
	radius           float32
	bias             float32
	normal           uint32
	fbo              uint32
	colorBuffer      uint32
	blurTexture      uint32
	blurFbo          uint32
	rboDepth         uint32
	proj             [16]float32
	w                int32
	h                int32
}

// NewSSAO initializes and returns a new instance of SSAO with default values.
func NewSSAO(ctx api.IContext) *SSAO {
	return &SSAO{
		ctx:              ctx,
		prg:              0,
		kernelSize:       64,
		noiseTextureSize: 4 * 4,
		radius:           12.0,
		bias:             0.025,
		rboDepth:         0,
	}
}

// SetupSamplers configures the SSAO samplers for the shader, binding texture slots and initializing kernel samples.
func (s *SSAO) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	s.ctx.Uniform1i(s.GetUniform(SSAOLocPosition), 0)
	s.ctx.Uniform1i(s.GetUniform(SSAOLocNormal), 1)
	s.ctx.Uniform1i(s.GetUniform(SSAOLocTexNoise), 2)
	s.ctx.Uniform3fv(s.GetUniform(SSAOLocSamples), s.kernelSize, &s.kernel[0])
	s.ctx.Uniform1i(s.GetUniform(SSAOLocKernelSize), s.kernelSize)
	s.ctx.Uniform1f(s.GetUniform(SSAOLocRadius), s.radius)
	s.ctx.Uniform1f(s.GetUniform(SSAOLocBias), s.bias)
	return nil
}

// Init initializes the SSAO instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *SSAO) Init() error {
	return nil
}

// GetGBufferTextures returns the G-buffer textures: position-depth and normal as uint32 values.
func (s *SSAO) GetGBufferTextures() (uint32, uint32) {
	return s.positionDepth, s.normal
}

// GetSSAOResources returns the ID of the texture containing the SSAO noise pattern.
func (s *SSAO) GetSSAOResources() uint32 {
	return s.noiseTex
}

// GetSSAOBlurTexture returns the texture ID of the blurred SSAO texture used in the rendering pipeline.
func (s *SSAO) GetSSAOBlurTexture() uint32 {
	return s.blurTexture
}

// GetProgram returns the OpenGL program identifier associated with the SSAO instance.
func (s *SSAO) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the location of a uniform variable in the shader program by its identifier.
func (s *SSAO) GetUniform(id SSAOLoc) int32 {
	return s.table[id]
}

// Compile initializes and compiles the SSAO shader program, sets up buffers, and validates uniform locations.
func (s *SSAO) Compile(a IAssets) error {
	const vertId = "ssao.vert"
	const fragId = "ssao.frag"

	vertexSrc, fragmentSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}
	vertexShader, err := ShaderCompile(s.ctx, vertId, string(vertexSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(s.ctx, fragId, string(fragmentSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(s.ctx, "ssao", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[SSAOLocPosition] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_position\x00"))
	s.table[SSAOLocNormal] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_normal\x00"))
	s.table[SSAOLocTexNoise] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texNoise\x00"))
	s.table[SSAOLocSamples] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_samples\x00"))
	s.table[SSAOLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[SSAOLocKernelSize] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_kernelSize\x00"))
	s.table[SSAOLocRadius] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_radius\x00"))
	s.table[SSAOLocBias] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_bias\x00"))

	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in ssao: %d", idx)
		}
	}
	if err = s.createKernel(); err != nil {
		return err
	}

	return nil
}

// createKernel generates the kernel samples and noise texture required for performing SSAO calculations.
func (s *SSAO) createKernel() error {
	s.kernel = make([]float32, s.kernelSize*3)
	for i := int32(0); i < s.kernelSize; i++ {
		sample := [3]float32{
			(rnd.Float32() * 2.0) - 1.0,
			(rnd.Float32() * 2.0) - 1.0,
			rnd.Float32(), // Emisfero orientato verso Z+
		}
		// Normalizzazione
		mag := float32(math.Sqrt(float64(sample[0]*sample[0] + sample[1]*sample[1] + sample[2]*sample[2])))
		z := float32(i) / float32(s.kernelSize)
		scale := 0.1 + (z*z)*(1.0-0.1) // Lerp per concentrare i campioni vicino all'origine
		s.kernel[i*3] = (sample[0] / mag) * scale
		s.kernel[i*3+1] = (sample[1] / mag) * scale
		s.kernel[i*3+2] = (sample[2] / mag) * scale
	}

	// --- 5. SSAO: GENERAZIONE NOISE TEXTURE (4x4) ---
	noiseData := make([]float32, s.noiseTextureSize*3)
	for i := int32(0); i < s.noiseTextureSize; i++ {
		noiseData[i*3] = rnd.Float32()*2.0 - 1.0
		noiseData[i*3+1] = rnd.Float32()*2.0 - 1.0
		noiseData[i*3+2] = 0.0
	}

	s.ctx.GenTextures(1, &s.noiseTex)
	s.ctx.BindTexture(api.TEXTURE_2D, s.noiseTex)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGB32F, 4, 4, 0, api.RGB, api.FLOAT, s.ctx.Ptr(noiseData))
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.REPEAT)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.REPEAT)

	return nil
}

// Prepare initializes the framebuffer and clears buffers to set up for SSAO rendering.
func (s *SSAO) Prepare(fbw, fbh int32) {
	if fbw != s.w || fbh != s.h {
		s.allocate(fbw, fbh)
	}
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.bufferFbo)
	// Sfondo lontanissimo per evitare che il cielo occluda la geometria
	s.ctx.ClearColor(0.0, 0.0, -100000.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0) // Ripristina per eventuali pass successivi
}

// UpdateUniforms updates the shader's projection matrix uniform with the provided projection matrix.
func (s *SSAO) UpdateUniforms(view, proj [16]float32) {
	s.proj = proj
}

// Render performs the screen-space ambient occlusion rendering and applies a blur pass to smooth the results.
func (s *SSAO) Render(blurPgr, mainVAO, skyVAO, postFBO uint32, skyEnabled bool) {
	s.ctx.BindVertexArray(mainVAO)

	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.fbo)
	s.ctx.Clear(api.COLOR_BUFFER_BIT)

	s.ctx.UseProgram(s.GetProgram())

	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, s.positionDepth)
	s.ctx.ActiveTexture(api.TEXTURE1)
	s.ctx.BindTexture(api.TEXTURE_2D, s.normal)
	s.ctx.ActiveTexture(api.TEXTURE2)
	s.ctx.BindTexture(api.TEXTURE_2D, s.noiseTex)

	if skyEnabled {
		s.ctx.UniformMatrix4fv(s.GetUniform(SSAOLocProjection), 1, false, &s.proj[0])
		s.ctx.BindVertexArray(skyVAO)
		s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)
	}

	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.blurFbo)
	s.ctx.Clear(api.COLOR_BUFFER_BIT)

	if skyEnabled {
		s.ctx.UseProgram(blurPgr)
		s.ctx.ActiveTexture(api.TEXTURE0)
		s.ctx.BindTexture(api.TEXTURE_2D, s.colorBuffer)
		s.ctx.BindVertexArray(skyVAO)
		s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)
	}

	// 3. FORWARD MULTI-PASS
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, postFBO)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)
	s.ctx.BindVertexArray(mainVAO)

	// PASS A: BASE
	s.ctx.DepthFunc(api.LEQUAL)
	s.ctx.DepthMask(true)
}

// allocate initializes or reinitializes framebuffers, textures, and renderbuffers for the given width and height.
func (s *SSAO) allocate(width, height int32) {
	s.w = width
	s.h = height

	// Prevenzione memory leak: distruzione esplicita dei buffer precedenti
	if s.bufferFbo != 0 {
		s.ctx.DeleteFramebuffers(1, &s.bufferFbo)
		s.ctx.DeleteTextures(1, &s.positionDepth)
		s.ctx.DeleteTextures(1, &s.normal)
		s.ctx.DeleteRenderbuffers(1, &s.rboDepth)

		s.ctx.DeleteFramebuffers(1, &s.fbo)
		s.ctx.DeleteTextures(1, &s.colorBuffer)

		s.ctx.DeleteFramebuffers(1, &s.blurFbo)
		s.ctx.DeleteTextures(1, &s.blurTexture)
	}

	// 1. G-Buffer
	s.ctx.GenFramebuffers(1, &s.bufferFbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.bufferFbo)

	// Position + depth (RGBA16F per precisione spaziale)
	s.ctx.GenTextures(1, &s.positionDepth)
	s.ctx.BindTexture(api.TEXTURE_2D, s.positionDepth)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA16F, width, height, 0, api.RGBA, api.FLOAT, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.NEAREST)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.positionDepth, 0)

	// Normals
	s.ctx.GenTextures(1, &s.normal)
	s.ctx.BindTexture(api.TEXTURE_2D, s.normal)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RGBA16F, width, height, 0, api.RGBA, api.FLOAT, nil)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.NEAREST)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT1, api.TEXTURE_2D, s.normal, 0)

	// Aggiungi il depth Renderbuffer (ora salvato nella struct)
	s.ctx.GenRenderbuffers(1, &s.rboDepth)
	s.ctx.BindRenderbuffer(api.RENDERBUFFER, s.rboDepth)
	s.ctx.RenderbufferStorage(api.RENDERBUFFER, api.DEPTH_COMPONENT24, width, height)
	s.ctx.FramebufferRenderbuffer(api.FRAMEBUFFER, api.DEPTH_ATTACHMENT, api.RENDERBUFFER, s.rboDepth)

	attachments := []uint32{api.COLOR_ATTACHMENT0, api.COLOR_ATTACHMENT1}
	s.ctx.DrawBuffers(2, &attachments[0])

	// 2. SSAO FBO
	s.ctx.GenFramebuffers(1, &s.fbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.fbo)
	s.ctx.GenTextures(1, &s.colorBuffer)
	s.ctx.BindTexture(api.TEXTURE_2D, s.colorBuffer)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RED, width, height, 0, api.RED, api.FLOAT, nil)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.colorBuffer, 0)

	// 3. SSAO Blur FBO
	s.ctx.GenFramebuffers(1, &s.blurFbo)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.blurFbo)
	s.ctx.GenTextures(1, &s.blurTexture)
	s.ctx.BindTexture(api.TEXTURE_2D, s.blurTexture)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, api.RED, width, height, 0, api.RED, api.FLOAT, nil)
	s.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.COLOR_ATTACHMENT0, api.TEXTURE_2D, s.blurTexture, 0)

	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
}
