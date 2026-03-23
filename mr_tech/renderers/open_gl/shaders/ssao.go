package shaders

import (
	"fmt"
	"math"
	rnd "math/rand"

	"github.com/go-gl/gl/v3.3-core/gl"
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
	SSAOLocGNormal
	SSAOLocTexNoise
	SSAOLocSamples
	SSAOLocProjection
	SSAOLocLast
)

// SSAO represents a shader implementation for Screen Space Ambient Occlusion (SSAO).
type SSAO struct {
	prg             uint32
	table           [SSAOLocLast]int32
	width           int32
	height          int32
	ssaoNoiseTex    uint32    // Texture di rumore 4x4
	ssaoKernel      []float32 // 64 campioni vec3
	bufferFbo       uint32
	positionDepth   uint32
	normal          uint32
	ssaoFbo         uint32
	ssaoColorBuffer uint32
	ssaoBlurTexture uint32
	ssaoBlurFbo     uint32
	proj            [16]float32
}

// NewSSAO initializes and returns a new instance of SSAO with default values.
func NewSSAO() *SSAO {
	return &SSAO{
		prg: 0,
	}
}

// Setup initializes the SSAO instance with the specified width and height, updating internal dimensions.
func (s *SSAO) Setup(width int32, height int32) {
	s.width = width
	s.height = height
}

// SetupSamplers configures the SSAO samplers for the shader, binding texture slots and initializing kernel samples.
func (s *SSAO) SetupSamplers() {
	// Setup SSAO Samplers
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(SSAOLocPosition), 0)
	gl.Uniform1i(s.GetUniform(SSAOLocGNormal), 1)
	gl.Uniform1i(s.GetUniform(SSAOLocTexNoise), 2)

	gl.Uniform3fv(s.GetUniform(SSAOLocSamples), 64, &s.ssaoKernel[0])
}

// GetGBufferTextures returns the G-buffer textures: position-depth and normal as uint32 values.
func (s *SSAO) GetGBufferTextures() (uint32, uint32) {
	return s.positionDepth, s.normal
}

// GetSSAOResources returns the ID of the texture containing the SSAO noise pattern.
func (s *SSAO) GetSSAOResources() uint32 {
	return s.ssaoNoiseTex
}

// GetSSAOBlurTexture returns the texture ID of the blurred SSAO texture used in the rendering pipeline.
func (s *SSAO) GetSSAOBlurTexture() uint32 {
	return s.ssaoBlurTexture
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
	if s.width == 0 || s.height == 0 {
		return fmt.Errorf("invalid shader dimensions: width=%d, height=%d", s.width, s.height)
	}
	vertexSrc, fragmentSrc, err := a.ReadMulti("ssao.vert", "ssao.frag")
	if err != nil {
		return err
	}
	if err = s.createBuffers(s.width, s.height); err != nil {
		return err
	}
	vertexShader, err := ShaderCompile(string(vertexSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(string(fragmentSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[SSAOLocPosition] = gl.GetUniformLocation(s.prg, gl.Str("u_position\x00"))
	s.table[SSAOLocGNormal] = gl.GetUniformLocation(s.prg, gl.Str("u_normal\x00"))
	s.table[SSAOLocTexNoise] = gl.GetUniformLocation(s.prg, gl.Str("u_texNoise\x00"))
	s.table[SSAOLocSamples] = gl.GetUniformLocation(s.prg, gl.Str("u_samples\x00"))
	s.table[SSAOLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	if err = s.createKernel(); err != nil {
		return err
	}

	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(SSAOLocPosition), 0)
	gl.Uniform1i(s.GetUniform(SSAOLocGNormal), 1)
	gl.Uniform1i(s.GetUniform(SSAOLocTexNoise), 2)

	return nil
}

// createBuffers initializes and configures framebuffer objects and textures required for SSAO rendering.
func (s *SSAO) createBuffers(width int32, height int32) error {
	// 1. G-Buffer
	gl.GenFramebuffers(1, &s.bufferFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.bufferFbo)

	// Position + Depth (RGBA16F per precisione spaziale)
	gl.GenTextures(1, &s.positionDepth)
	gl.BindTexture(gl.TEXTURE_2D, s.positionDepth)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.positionDepth, 0)

	// Normals
	gl.GenTextures(1, &s.normal)
	gl.BindTexture(gl.TEXTURE_2D, s.normal)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT1, gl.TEXTURE_2D, s.normal, 0)

	// Aggiungi il Depth Renderbuffer
	var rboDepth uint32
	gl.GenRenderbuffers(1, &rboDepth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, rboDepth)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, width, height)
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, rboDepth)

	attachments := []uint32{gl.COLOR_ATTACHMENT0, gl.COLOR_ATTACHMENT1}
	gl.DrawBuffers(2, &attachments[0])

	// 2. SSAO FBO
	gl.GenFramebuffers(1, &s.ssaoFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.ssaoFbo)
	gl.GenTextures(1, &s.ssaoColorBuffer)
	gl.BindTexture(gl.TEXTURE_2D, s.ssaoColorBuffer)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, width, height, 0, gl.RED, gl.FLOAT, nil)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.ssaoColorBuffer, 0)

	// 3. SSAO Blur FBO
	gl.GenFramebuffers(1, &s.ssaoBlurFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.ssaoBlurFbo)
	gl.GenTextures(1, &s.ssaoBlurTexture)
	gl.BindTexture(gl.TEXTURE_2D, s.ssaoBlurTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, width, height, 0, gl.RED, gl.FLOAT, nil)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.ssaoBlurTexture, 0)

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	return nil
}

// createKernel generates a 64-sample SSAO kernel and initializes a 4x4 noise texture used in ambient occlusion rendering.
func (s *SSAO) createKernel() error {
	// --- 4. SSAO: GENERAZIONE KERNEL (64 campioni) ---
	s.ssaoKernel = make([]float32, 64*3)
	for i := 0; i < 64; i++ {
		sample := [3]float32{
			rnd.Float32()*2.0 - 1.0,
			rnd.Float32()*2.0 - 1.0,
			rnd.Float32(), // Emisfero orientato verso Z+
		}
		// Normalizzazione
		mag := float32(math.Sqrt(float64(sample[0]*sample[0] + sample[1]*sample[1] + sample[2]*sample[2])))
		z := float32(i) / 64.0
		scale := 0.1 + (z*z)*(1.0-0.1) // Lerp per concentrare i campioni vicino all'origine
		s.ssaoKernel[i*3] = (sample[0] / mag) * scale
		s.ssaoKernel[i*3+1] = (sample[1] / mag) * scale
		s.ssaoKernel[i*3+2] = (sample[2] / mag) * scale
	}

	// --- 5. SSAO: GENERAZIONE NOISE TEXTURE (4x4) ---
	noiseData := make([]float32, 16*3)
	for i := 0; i < 16; i++ {
		noiseData[i*3] = rnd.Float32()*2.0 - 1.0
		noiseData[i*3+1] = rnd.Float32()*2.0 - 1.0
		noiseData[i*3+2] = 0.0
	}

	gl.GenTextures(1, &s.ssaoNoiseTex)
	gl.BindTexture(gl.TEXTURE_2D, s.ssaoNoiseTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB32F, 4, 4, 0, gl.RGB, gl.FLOAT, gl.Ptr(noiseData))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	return nil
}

// Prepare initializes the framebuffer and clears buffers to set up for SSAO rendering.
func (s *SSAO) Prepare() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.bufferFbo)
	// Sfondo lontanissimo per evitare che il cielo occluda la geometria
	gl.ClearColor(0.0, 0.0, -100000.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0) // Ripristina per eventuali pass successivi
}

// UpdateUniforms updates the shader's projection matrix uniform with the provided projection matrix.
func (s *SSAO) UpdateUniforms(view, proj [16]float32) {
	s.proj = proj
}

// Render performs the screen-space ambient occlusion rendering and applies a blur pass to smooth the results.
func (s *SSAO) Render(drawScreenQuad func(), blurPgr uint32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.ssaoFbo)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.UseProgram(s.GetProgram())

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.positionDepth)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, s.normal)
	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, s.ssaoNoiseTex)

	gl.UniformMatrix4fv(s.GetUniform(SSAOLocProjection), 1, false, &s.proj[0])
	drawScreenQuad()

	gl.BindFramebuffer(gl.FRAMEBUFFER, s.ssaoBlurFbo)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.UseProgram(blurPgr)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.ssaoColorBuffer)
	drawScreenQuad()
}
