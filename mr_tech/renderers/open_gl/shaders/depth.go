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

// DepthMap represents a structure for managing depth framebuffers and textures for shadow mapping and depth rendering.
type DepthMap struct {
	fbo    uint32
	tex    uint32
	matrix [16]float32
}

func NewDepthMap() *DepthMap {
	return &DepthMap{
		fbo: 0,
		tex: 0,
	}
}

// Update initializes and configures the framebuffer and texture for depth rendering with the given dimensions.
func (d *DepthMap) Update(width, height int32) {
	d.Shutdown()
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
	d.fbo = fbo
	d.tex = tex
}

// SetMatrix assigns a 4x4 transformation matrix to the DepthMap instance.
func (d *DepthMap) SetMatrix(matrix [16]float32) {
	d.matrix = matrix
}

// Shutdown releases OpenGL resources associated with the framebuffer and texture of the DepthMap.
func (d *DepthMap) Shutdown() {
	if d.fbo != 0 {
		gl.DeleteFramebuffers(1, &d.fbo)
		return
	}
	if d.tex != 0 {
		gl.DeleteTextures(1, &d.tex)
	}
}

// Depth is responsible for managing depth shaders and shadow map framebuffers for rendering depth-based effects.
type Depth struct {
	prg              uint32
	table            [DepthLocLast]int32
	sWidth           int32
	sHeight          int32
	roomMap          *DepthMap
	flashMap         *DepthMap
	shadowLightsMap  []*DepthMap
	viewMatrix       [16]float32
	shadows          bool
	metrics          *MapMetrics
	shadowLightCount uint32
}

// NewDepth initializes and returns a new instance of Depth with default uninitialized properties.
func NewDepth(m *MapMetrics, shadowLightMax int) *Depth {
	d := &Depth{
		metrics:          m,
		roomMap:          NewDepthMap(),
		flashMap:         NewDepthMap(),
		shadowLightCount: 0,
	}
	for i := 0; i < shadowLightMax; i++ {
		d.shadowLightsMap = append(d.shadowLightsMap, NewDepthMap())
	}
	return d
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
	return s.roomMap.tex
}

// GetFlashShadowTextures retrieves the OpenGL texture ID associated with the flashlight's shadow map.
func (s *Depth) GetFlashShadowTextures() uint32 {
	return s.flashMap.tex
}

// GetShadowLightTextures retrieves the texture ID for a specific dynamic shadow light by its index. Returns 0 if index is out of range.
func (s *Depth) GetShadowLightTextures(idx uint32) uint32 {
	if idx >= s.shadowLightCount {
		return 0
	}
	return s.shadowLightsMap[idx].tex
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
func (s *Depth) UpdateUniforms(roomSpaceMatrix [16]float32, flashSpaceMatrix [16]float32, viewMatrix [16]float32, dynaLight [][16]float32) {
	s.viewMatrix = viewMatrix
	s.roomMap.SetMatrix(roomSpaceMatrix)
	s.flashMap.SetMatrix(flashSpaceMatrix)
	s.shadowLightCount = uint32(len(dynaLight))
	if s.shadowLightCount >= uint32(len(s.shadowLightsMap)) {
		s.shadowLightCount = uint32(len(s.shadowLightsMap)) - 1
	}
	for x := uint32(0); x < s.shadowLightCount; x++ {
		s.shadowLightsMap[x].SetMatrix(dynaLight[x])
	}
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
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.roomMap.fbo)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(s.GetProgram())
	// Invia la View Matrix del Player per i calcoli del Billboard degli Sprite
	gl.UniformMatrix4fv(s.GetUniform(DepthLocView), 1, false, &s.viewMatrix[0])

	//ROOM
	gl.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.roomMap.matrix[0])
	gl.Uniform1i(s.GetUniform(DepthLocTexture), 0)
	renderScene()

	// --- 2. OMBRE TORCIA (Prospettica) ---
	gl.PolygonOffset(0.5, 1.0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.flashMap.fbo)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.flashMap.matrix[0])
	renderScene()

	for x := 0; x < int(s.shadowLightCount); x++ {
		gl.PolygonOffset(0.5, 1.0)
		gl.BindFramebuffer(gl.FRAMEBUFFER, s.shadowLightsMap[x].fbo)
		gl.Clear(gl.DEPTH_BUFFER_BIT)
		gl.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.shadowLightsMap[x].matrix[0])
		renderScene()
	}

	// Ripristiniamo lo stato di default per non influenzare il resto del rendering
	gl.Disable(gl.DEPTH_CLAMP)
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.Viewport(0, 0, fbw, fbh)
}

// allocate configures the internal shadow map dimensions and updates the associated depth maps for rendering.
func (s *Depth) allocate(width, height int32) {
	s.sWidth = width
	s.sHeight = height
	// roomShadowFbo e roomShadowTex gestiscono le ombre delle luci ambientali.
	s.roomMap.Update(s.sWidth, s.sHeight)
	// flashShadowFbo e flashShadowTex gestiscono l'ombra della torcia (Flashlight).
	s.flashMap.Update(s.sWidth, s.sHeight)
	for _, dm := range s.shadowLightsMap {
		dm.Update(s.sWidth, s.sHeight)
	}
}
