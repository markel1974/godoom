package open_gl

import (
	"fmt"
	"math"
	rnd "math/rand"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// ShaderSSAOLoc is an enumerated type representing uniform locations used in the ShaderSSAO rendering pipeline.
type ShaderSSAOLoc int

// ShaderSSAOLocGPosition represents the location for the G-position in the SSAO shader.
// ShaderSSAOLocGNormal represents the location for the G-normal in the SSAO shader.
// ShaderSSAOLocTexNoise represents the location for the texture noise in the SSAO shader.
// ShaderSSAOLocSamples represents the location for the SSAO sample kernel.
// ShaderSSAOLocProjection represents the location for the SSAO projection matrix.
// ShaderSSAOLocLast marks the end of the SSAO shader locations.
const (
	ShaderSSAOLocGPosition = ShaderSSAOLoc(iota)
	ShaderSSAOLocGNormal
	ShaderSSAOLocTexNoise
	ShaderSSAOLocSamples
	ShaderSSAOLocProjection
	ShaderSSAOLocLast
)

// ShaderSSAO represents a data structure for managing an SSAO shader and its associated resources.
type ShaderSSAO struct {
	prg             uint32
	table           [ShaderSSAOLocLast]int32
	width           int32
	height          int32
	ssaoNoiseTex    uint32    // Texture di rumore 4x4
	ssaoKernel      []float32 // 64 campioni vec3
	gBufferFbo      uint32
	gPositionDepth  uint32
	gNormal         uint32
	ssaoFbo         uint32
	ssaoColorBuffer uint32
	ssaoBlurTexture uint32
	ssaoBlurFbo     uint32
}

// NewShaderSSAO creates and returns a new instance of ShaderSSAO with default uninitialized properties.
func NewShaderSSAO() *ShaderSSAO {
	return &ShaderSSAO{
		prg: 0,
	}
}

func (s *ShaderSSAO) Setup(width int32, height int32) {
	s.width = width
	s.height = height
}

func (s *ShaderSSAO) SetupSamplers() {
	// Setup SSAO Samplers
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(ShaderSSAOLocGPosition), 0)
	gl.Uniform1i(s.GetUniform(ShaderSSAOLocGNormal), 1)
	gl.Uniform1i(s.GetUniform(ShaderSSAOLocTexNoise), 2)
}

// GetGBufferTextures returns the texture IDs for the position-depth and normal buffers of the G-Buffer.
func (s *ShaderSSAO) GetGBufferTextures() (uint32, uint32) {
	return s.gPositionDepth, s.gNormal
}

// GetSSAOResources retrieves the SSAO noise texture and kernel data used for screen space ambient occlusion calculations.
func (s *ShaderSSAO) GetSSAOResources() (uint32, []float32) {
	return s.ssaoNoiseTex, s.ssaoKernel
}

// GetSSAOBlurTexture retrieves the texture ID of the blurred SSAO (Screen Space Ambient Occlusion) buffer.
func (s *ShaderSSAO) GetSSAOBlurTexture() uint32 {
	return s.ssaoBlurTexture
}

// GetProgram returns the OpenGL program ID associated with the ShaderSSAO instance.
func (s *ShaderSSAO) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the location of a specified uniform variable from the shader's uniform table.
func (s *ShaderSSAO) GetUniform(id ShaderSSAOLoc) int32 {
	return s.table[id]
}

// Compile initializes the ShaderSSAO instance by compiling and linking shaders and retrieving uniform locations.
func (s *ShaderSSAO) Compile(vertexSrc string, fragmentSrc string) error {
	if s.width == 0 || s.height == 0 {
		return fmt.Errorf("invalid shader dimensions: width=%d, height=%d", s.width, s.height)
	}

	if err := s.setupSSAOBuffers(s.width, s.height); err != nil {
		return err
	}
	vertexShader, err := ShaderCompile(vertexSrc, gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(fragmentSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[ShaderSSAOLocGPosition] = gl.GetUniformLocation(s.prg, gl.Str("gPosition\x00"))
	s.table[ShaderSSAOLocGNormal] = gl.GetUniformLocation(s.prg, gl.Str("gNormal\x00"))
	s.table[ShaderSSAOLocTexNoise] = gl.GetUniformLocation(s.prg, gl.Str("texNoise\x00"))
	s.table[ShaderSSAOLocSamples] = gl.GetUniformLocation(s.prg, gl.Str("samples\x00"))
	s.table[ShaderSSAOLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("projection\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	if err = s.createKernel(); err != nil {
		return err
	}

	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(ShaderSSAOLocGPosition), 0)
	gl.Uniform1i(s.GetUniform(ShaderSSAOLocGNormal), 1)
	gl.Uniform1i(s.GetUniform(ShaderSSAOLocTexNoise), 2)

	return nil
}

func (s *ShaderSSAO) setupSSAOBuffers(width int32, height int32) error {
	// 1. G-Buffer
	gl.GenFramebuffers(1, &s.gBufferFbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.gBufferFbo)

	// Position + Depth (RGBA16F per precisione spaziale)
	gl.GenTextures(1, &s.gPositionDepth)
	gl.BindTexture(gl.TEXTURE_2D, s.gPositionDepth)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, s.gPositionDepth, 0)

	// Normals
	gl.GenTextures(1, &s.gNormal)
	gl.BindTexture(gl.TEXTURE_2D, s.gNormal)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT1, gl.TEXTURE_2D, s.gNormal, 0)

	// Aggiungi il Depth Renderbuffer
	var rboDepth uint32
	gl.GenRenderbuffers(1, &rboDepth)
	gl.BindRenderbuffer(gl.RENDERBUFFER, rboDepth)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT, width, height)
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

func (s *ShaderSSAO) createKernel() error {
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
