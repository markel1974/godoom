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
	DepthLocView
	DepthLocLast
)

// Depth is responsible for managing depth shaders and shadow map framebuffers for rendering depth-based effects.
type Depth struct {
	prg            uint32
	table          [DepthLocLast]int32
	sWidth         int32
	sHeight        int32
	roomShadowFbo  uint32
	roomShadowTex  uint32
	flashShadowFbo uint32
	flashShadowTex uint32
	roomMatrix     [16]float32
	flashMatrix    [16]float32
	viewMatrix     [16]float32
	shadows        bool
	metrics        *MapMetrics
}

// NewDepth initializes and returns a new instance of Depth with default uninitialized properties.
func NewDepth(m *MapMetrics) *Depth {
	return &Depth{
		metrics: m,
	}
}

// SetupSamplers initializes or configures the sampler bindings for the Depth program.
func (s *Depth) SetupSamplers() error {
	return nil
}

// Init initializes the Depth instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *Depth) Init() error {
	return nil
}

// EnableShadows toggles shadow rendering by setting the internal shadows flag to the provided boolean value.
func (s *Depth) EnableShadows(e bool) {
	s.shadows = e
}

// GetProgram retrieves the OpenGL program ID associated with the Depth instance.
func (s *Depth) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location corresponding to the provided DepthLoc identifier from the uniform table.
func (s *Depth) GetUniform(id DepthLoc) int32 {
	return s.table[id]
}

// GetRoomShadowTextures retrieves the texture ID associated with the room shadow map.
func (s *Depth) GetRoomShadowTextures() uint32 {
	return s.roomShadowTex
}

// GetFlashShadowTextures retrieves the OpenGL texture ID associated with the flashlight's shadow map.
func (s *Depth) GetFlashShadowTextures() uint32 {
	return s.flashShadowTex
}

// Compile initializes and compiles the shader program using vertex and fragment sources, and sets up uniform locations.
func (s *Depth) Compile(assets IAssets) error {
	const vertId = "depth.vert"
	const fragId = "depth.frag"
	vertexSrc, fragmentSrc, err := assets.ReadMulti(vertId, fragId)

	//s.roomShadowFbo, s.roomShadowTex = s.createDepthMap(s.shadowWidth, s.shadowHeight)
	//s.flashShadowFbo, s.flashShadowTex = s.createDepthMap(s.shadowWidth, s.shadowHeight)

	vertexShader, err := ShaderCompile(vertId, string(vertexSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(fragId, string(fragmentSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram("depth", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[DepthLocLightSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_lightSpaceMatrix\x00"))
	s.table[DepthLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[DepthLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))

	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in depth: %d", idx)
		}
	}
	return nil
}

// UpdateUniforms updates the uniform matrix values for room and flashlight space transformations for the shader.
func (s *Depth) UpdateUniforms(roomSpaceMatrix [16]float32, flashSpaceMatrix [16]float32, viewMatrix [16]float32) {
	s.roomMatrix = roomSpaceMatrix
	s.flashMatrix = flashSpaceMatrix
	s.viewMatrix = viewMatrix
}

// Render performs the depth pre-pass for shadow mapping by rendering the scene to multiple framebuffers for shadows.
func (s *Depth) Render(renderScene func(), mainVao uint32, fbw, fbh int32) {
	if !s.shadows {
		return
	}

	sWidth, sHeight := s.metrics.GetShadowSize()
	if sWidth != s.sWidth || sHeight != s.sHeight {
		s.allocate(sWidth, sHeight)
	}

	gl.BindVertexArray(mainVao)

	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.POLYGON_OFFSET_FILL)

	// FIX: Attiviamo il clamp della profondità.
	// Impedisce che la geometria sparisca dalla mappa delle ombre
	// quando la telecamera ci finisce letteralmente addosso.
	gl.Enable(gl.DEPTH_CLAMP)

	gl.Viewport(0, 0, sWidth, sHeight)

	// --- 1. OMBRE STANZA (Ortografica) ---
	gl.PolygonOffset(2.0, 4.0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.roomShadowFbo)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(s.GetProgram())
	// Invia la View Matrix del Player per i calcoli del Billboard degli Sprite
	gl.UniformMatrix4fv(s.GetUniform(DepthLocView), 1, false, &s.viewMatrix[0]) // <-- Aggiunto
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
	gl.Viewport(0, 0, fbw, fbh)
}

func (s *Depth) allocate(width, height int32) {
	s.sWidth = width
	s.sHeight = height
	// 1. PULIZIA PREVENTIVA (Prevenzione Memory Leak)
	if s.roomShadowFbo != 0 {
		gl.DeleteFramebuffers(1, &s.roomShadowFbo)
		gl.DeleteTextures(1, &s.roomShadowTex)
	}
	if s.flashShadowFbo != 0 {
		gl.DeleteFramebuffers(1, &s.flashShadowFbo)
		gl.DeleteTextures(1, &s.flashShadowTex)
	}

	// 2. CREAZIONE DELLE NUOVE RISORSE
	// roomShadowFbo e roomShadowTex gestiscono le ombre delle luci ambientali.
	s.roomShadowFbo, s.roomShadowTex = s.createDepthMap(s.sWidth, s.sHeight)
	// flashShadowFbo e flashShadowTex gestiscono l'ombra della torcia (Flashlight).
	s.flashShadowFbo, s.flashShadowTex = s.createDepthMap(s.sWidth, s.sHeight)
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
