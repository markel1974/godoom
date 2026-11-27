package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ShaderSkyLoc represents the location index of shader uniforms specific to the Sky renderer.
type ShaderSkyLoc int

// ShaderSkyLocProjection represents the projection matrix location for the sky shader.
// ShaderSkyLocView represents the view matrix location for the sky shader.
// ShaderSkyLocSky represents the sky texture location for the sky shader.
// ShaderSkyLocLast represents the total count of sky shader locations.
const (
	ShaderSkyLocProjection = ShaderSkyLoc(iota)
	ShaderSkyLocView
	ShaderSkyLocSky
	ShaderSkyLocSkyLayer
	ShaderSkyLocScrollU
	ShaderSkyLocScrollV
	ShaderSkyLocHasTex
	ShaderSkyLocSunEnabled
	ShaderSkyLocSunDirWorld
	ShaderSkyLocSunColorIntensity
	ShaderSkyLocSunSkyColor
	ShaderSkyLocFogEnabled
	ShaderSkyLocFogColor
	ShaderSkyLocFogSunColor
	ShaderSkyLocFogScatter
	ShaderSkyLocLast
)

// Sky manages rendering resources and shader state for drawing the sky and related atmospheric effects.
type Sky struct {
	ctx           api.IContext
	handler       *Handler
	tex           *vramtex.Textures
	prg           uint32
	table         [ShaderSkyLocLast]int32
	skyVAO        uint32
	skyVBO        uint32
	mapMetrics    *metrics.Map
	sunCSM        *metrics.SunCSM
	atmosphereFog *metrics.AtmosphereFog
}

// NewSky creates and returns a new instance of Sky with default uninitialized properties.
func NewSky() *Sky {
	s := &Sky{
		prg: 0,
	}
	for i := range s.table {
		s.table[i] = -1
	}
	return s
}

// Setup initializes the Sky instance by assigning the rendering context from the provided Handler.
func (s *Sky) Setup(h *Handler) error {
	s.handler = h
	s.tex = h.GetTextures()
	s.ctx = h.GetContext()
	s.mapMetrics = h.GetMapMetrics()
	s.sunCSM = h.GetSunCSM()
	s.atmosphereFog = h.GetAtmosphereFog()
	return nil
}

// SetupSamplers initializes the sky vertex array and buffer objects and prepares the sampler for rendering sky elements.
func (s *Sky) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)

	skyUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2}
	s.ctx.Uniform1iv(s.GetUniform(ShaderSkyLocSky), 3, &skyUnits[0])

	s.ctx.GenVertexArrays(1, &s.skyVAO)
	s.ctx.BindVertexArray(s.skyVAO)
	s.ctx.GenBuffers(1, &s.skyVBO)
	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.skyVBO)
	skyQuadVertices := []float32{-1.0, -1.0, 1.0, -1.0, -1.0, 1.0, 1.0, 1.0}
	s.ctx.BufferData(api.ARRAY_BUFFER, len(skyQuadVertices)*4, s.ctx.Ptr(skyQuadVertices), api.STATIC_DRAW)
	s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 2*4, s.ctx.PtrOffset(0))
	s.ctx.EnableVertexAttribArray(0)
	// Restore default state
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	return nil
}

// Init initializes the Sky instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *Sky) Init() error {
	return nil
}

// GetProgram returns the OpenGL program identifier associated with the Sky instance.
func (s *Sky) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the location of a shader uniform variable by its ID.
func (s *Sky) GetUniform(id ShaderSkyLoc) int32 {
	return s.table[id]
}

// GetVAO retrieves the vertex array object (VAO) associated with the Sky instance.
func (s *Sky) GetVAO() uint32 {
	return s.skyVAO
}

// Compile compiles the shader program for rendering the sky using provided vertex and fragment shaders from assets.
func (s *Sky) Compile(a IAssets) error {
	const vertId = "sky.vert"
	const fragId = "sky.frag"

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
	s.prg, err = ShaderCreateProgram(s.ctx, "sky", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[ShaderSkyLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[ShaderSkyLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[ShaderSkyLocSky] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_sky\x00"))
	s.table[ShaderSkyLocSkyLayer] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_skyLayer\x00"))
	s.table[ShaderSkyLocScrollU] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_scrollU\x00"))
	s.table[ShaderSkyLocScrollV] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_scrollV\x00"))
	s.table[ShaderSkyLocHasTex] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_hasSkyTex\x00"))
	s.table[ShaderSkyLocSunEnabled] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_sunEnabled\x00"))
	s.table[ShaderSkyLocSunDirWorld] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_sunDirWorld\x00"))
	s.table[ShaderSkyLocSunColorIntensity] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_sunColorIntensity\x00"))
	s.table[ShaderSkyLocSunSkyColor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_sunSkyColor\x00"))
	s.table[ShaderSkyLocFogEnabled] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_fogEnabled\x00"))
	s.table[ShaderSkyLocFogColor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_fogColor\x00"))
	s.table[ShaderSkyLocFogSunColor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_fogSunColor\x00"))
	s.table[ShaderSkyLocFogScatter] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_fogScatter\x00"))
	for idx := range int(ShaderSkyLocHasTex) {
		if s.table[idx] < 0 {
			return fmt.Errorf("invalid uniform location in sky: %d", idx)
		}
	}
	return nil
}

func (s *Sky) Render(skyMat *textures.Material) {
	if skyMat != nil {
		if skyTex := skyMat.GetCurrentFrame(); skyTex != nil {
			skyTexId := skyTex.GetIdentifier()
			if skyTexId < 0 {
				skyTexId, _ = s.tex.Get(skyTex)
				skyTex.SetIdentifier(skyTexId)
			}
			u, v := float32(skyMat.GetU()), float32(skyMat.GetV())
			tick := float32(textures.TickF())
			//s.Render(skyTexId, u*tick, v*tick)
			s.renderInternal(true, skyTexId, u*tick, v*tick)
		} else if s.handler.IsSunEnabled() || s.handler.IsHorizonFogEnabled() {
			s.renderInternal(false, 0, 0, 0)
		}
	} else if s.handler.IsSunEnabled() || s.handler.IsHorizonFogEnabled() {
		s.renderInternal(false, 0, 0, 0)
	}
}

/*
// Render handles the rendering of the textured skybox with optional horizon fog and sun aureole blending.
func (s *Sky) Render(skyLayer float32, u, v float32) {
	s.renderInternal(true, skyLayer, u, v)
}

// RenderProcedural draws the procedural outdoor sky gradient, sun disc, and horizon haze without a sky texture.
func (s *Sky) RenderProcedural() {
	s.renderInternal(false, 0, 0, 0)
}

*/

func (s *Sky) renderInternal(hasTex bool, skyLayer float32, u, v float32) {
	viewPtr := s.mapMetrics.GetViewPtr()
	projPtr := s.mapMetrics.GetProjPtr()

	s.ctx.UseProgram(s.GetProgram())

	s.ctx.DepthFunc(api.LEQUAL)
	s.ctx.DepthMask(false)

	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderSkyLocProjection), 1, false, projPtr)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderSkyLocView), 1, false, viewPtr)

	s.ctx.BindVertexArray(s.skyVAO)

	if loc := s.GetUniform(ShaderSkyLocHasTex); loc >= 0 {
		if hasTex {
			s.ctx.Uniform1i(loc, 1)
		} else {
			s.ctx.Uniform1i(loc, 0)
		}
	}
	s.ctx.Uniform1f(s.GetUniform(ShaderSkyLocSkyLayer), skyLayer)
	s.ctx.Uniform1f(s.GetUniform(ShaderSkyLocScrollU), u)
	s.ctx.Uniform1f(s.GetUniform(ShaderSkyLocScrollV), v)

	if s.sunCSM != nil && s.sunCSM.IsEnabled() {
		if loc := s.GetUniform(ShaderSkyLocSunEnabled); loc >= 0 {
			s.ctx.Uniform1i(loc, 1)
		}
		if loc := s.GetUniform(ShaderSkyLocSunDirWorld); loc >= 0 {
			dx, dy, dz := s.sunCSM.GetDirectionGL()
			s.ctx.Uniform3f(loc, dx, dy, dz)
		}
		if loc := s.GetUniform(ShaderSkyLocSunColorIntensity); loc >= 0 {
			sr, sg, sb, si := s.sunCSM.GetColorIntensity()
			s.ctx.Uniform4f(loc, sr, sg, sb, si)
		}
		if loc := s.GetUniform(ShaderSkyLocSunSkyColor); loc >= 0 {
			skyR, skyG, skyB, _, _, _ := s.sunCSM.GetSkyAmbient()
			s.ctx.Uniform3f(loc, skyR, skyG, skyB)
		}
	} else {
		if loc := s.GetUniform(ShaderSkyLocSunEnabled); loc >= 0 {
			s.ctx.Uniform1i(loc, 0)
		}
		if loc := s.GetUniform(ShaderSkyLocSunSkyColor); loc >= 0 {
			s.ctx.Uniform3f(loc, 0.42, 0.62, 0.92)
		}
	}

	if s.atmosphereFog != nil {
		fogEn, fogCol, fogSunCol, fogParams, _ := s.atmosphereFog.GetShaderParams()
		if loc := s.GetUniform(ShaderSkyLocFogEnabled); loc >= 0 {
			s.ctx.Uniform1i(loc, fogEn)
		}
		if loc := s.GetUniform(ShaderSkyLocFogColor); loc >= 0 {
			s.ctx.Uniform3f(loc, fogCol[0], fogCol[1], fogCol[2])
		}
		if loc := s.GetUniform(ShaderSkyLocFogSunColor); loc >= 0 {
			s.ctx.Uniform3f(loc, fogSunCol[0], fogSunCol[1], fogSunCol[2])
		}
		if loc := s.GetUniform(ShaderSkyLocFogScatter); loc >= 0 {
			s.ctx.Uniform1f(loc, fogParams[3])
		}
	} else if loc := s.GetUniform(ShaderSkyLocFogEnabled); loc >= 0 {
		s.ctx.Uniform1i(loc, 0)
	}

	s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)

	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
}
