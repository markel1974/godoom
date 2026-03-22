package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// DepthLoc represents the location identifiers for shader uniform variables used in depth shaders.
type DepthLoc int

// DepthLocLightSpaceMatrix represents the shader location for the light space matrix in depth shaders.
// DepthLocTexture represents the shader location for the texture in depth shaders.
// DepthLocLast is a sentinel value indicating the last shader depth location.
const (
	DepthLocLightSpaceMatrix = DepthLoc(iota)
	DepthLocTexture
	DepthLocLast
)

// Depth is responsible for managing depth shaders and shadow map framebuffers for rendering depth-based effects.
type Depth struct {
	prg            uint32
	table          [DepthLocLast]int32
	shadowWidth    int32
	shadowHeight   int32
	roomShadowFbo  uint32
	roomShadowTex  uint32
	flashShadowFbo uint32
	flashShadowTex uint32

	// Aggiunte per cache locale
	roomMatrix  [16]float32
	flashMatrix [16]float32
	width       int32
	height      int32
}

// NewDepth initializes and returns a new instance of Depth with default uninitialized properties.
func NewDepth() *Depth {
	return &Depth{}
}

// Setup initializes the shadow map dimensions with default values or overrides them using the provided width and height.
func (s *Depth) Setup(width int32, height int32) {
	s.width = width
	s.height = height
	s.shadowWidth = 1024
	s.shadowHeight = 1024
}

// SetupSamplers initializes or configures the sampler bindings for the Depth program.
func (s *Depth) SetupSamplers() {
}

// GetProgram retrieves the OpenGL program ID associated with the Depth instance.
func (s *Depth) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location corresponding to the provided DepthLoc identifier from the uniform table.
func (s *Depth) GetUniform(id DepthLoc) int32 {
	return s.table[id]
}

// GetShadowTextures retrieves the texture IDs for the room and flashlight shadow maps, used for depth-based rendering.
func (s *Depth) GetShadowTextures() (uint32, uint32) {
	return s.roomShadowTex, s.flashShadowTex
}

// Compile initializes and compiles the shader program using vertex and fragment sources, and sets up uniform locations.
func (s *Depth) Compile(assets IAssets) error {
	vertexSrc, fragmentSrc, err := assets.ReadMulti("depth.vert", "depth.frag")

	s.roomShadowFbo, s.roomShadowTex = s.createDepthMap(s.shadowWidth, s.shadowHeight)
	s.flashShadowFbo, s.flashShadowTex = s.createDepthMap(s.shadowWidth, s.shadowHeight)

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
	s.table[DepthLocLightSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_lightSpaceMatrix\x00"))
	s.table[DepthLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))

	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}

// createDepthMap initializes a depth map with given width and height and returns the framebuffer and texture IDs.
func (s *Depth) createDepthMap(width, height int32) (uint32, uint32) {
	var fbo, tex uint32
	borderColor := []float32{1.0, 1.0, 1.0, 1.0}

	gl.GenFramebuffers(1, &fbo)
	gl.GenTextures(1, &tex)

	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT32F, width, height, 0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_MODE, gl.COMPARE_REF_TO_TEXTURE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_FUNC, gl.LEQUAL)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])

	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, tex, 0)
	gl.DrawBuffer(gl.NONE)
	gl.ReadBuffer(gl.NONE)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	return fbo, tex
}

// UpdateUniforms updates the uniform matrix values for room and flashlight space transformations for the shader.
func (s *Depth) UpdateUniforms(roomSpaceMatrix [16]float32, flashSpaceMatrix [16]float32) {
	s.roomMatrix = roomSpaceMatrix
	s.flashMatrix = flashSpaceMatrix
}

// Render performs the depth pre-pass for shadow mapping by rendering the scene to multiple framebuffers for shadows.
func (s *Depth) Render(renderScene func()) {
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.POLYGON_OFFSET_FILL)

	// FIX: Attiviamo il clamp della profondità.
	// Impedisce che la geometria sparisca dalla mappa delle ombre
	// quando la telecamera ci finisce letteralmente addosso.
	gl.Enable(gl.DEPTH_CLAMP)

	gl.Viewport(0, 0, s.shadowWidth, s.shadowHeight)

	// --- 1. OMBRE STANZA (Ortografica) ---
	gl.PolygonOffset(2.0, 4.0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.roomShadowFbo)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(s.GetProgram())
	gl.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.roomMatrix[0])
	gl.Uniform1i(s.GetUniform(DepthLocTexture), 0)
	renderScene()

	// --- 2. OMBRE TORCIA (Prospettica) ---
	gl.PolygonOffset(0.5, 1.0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.flashShadowFbo)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.flashMatrix[0])
	renderScene()

	// Ripristiniamo lo stato di default per non influenzare il resto del rendering
	gl.Disable(gl.DEPTH_CLAMP)
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.Viewport(0, 0, s.width, s.height)
}
