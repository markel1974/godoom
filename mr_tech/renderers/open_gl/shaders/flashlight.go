package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// FlashlightLoc represents a uniform location identifier used in the ShadowLight's shader program.
type FlashlightLoc int

// FlashLocProjection specifies the location of the projection matrix for the flashlight.
// FlashLocView specifies the location of the view matrix for the flashlight.
// FlashLocInvView specifies the location of the inverse view matrix for the flashlight.
// FlashLocFlashSpaceMatrix specifies the location of the flashlight space matrix.
// FlashLocTexture specifies the location of the flashlight texture.
// FlashLocNormalMap specifies the location of the flashlight normal map.
// FlashLocFlashShadowMap specifies the location of the flashlight shadow map.
// FlashLocScreenResolution specifies the location of the screen resolution data.
// FlashLocFlashDir specifies the location of the flashlight direction data.
// FlashLocFlashIntensityFactor specifies the location of the flashlight intensity factor.
// FlashLocFlashOffset specifies the location of the flashlight offset data.
// FlashLocFlashConeStart specifies the location of the flashlight cone start parameter.
// FlashLocFlashConeEnd specifies the location of the flashlight cone end parameter.
// FlashLocFlashBase specifies the location of the base position of the flashlight.
// FlashLocEnableShadows specifies the location of the flashlight shadow enable flag.
// FlashLocShininessWall specifies the location of the wall shininess factor for the flashlight effect.
// FlashLocShininessFloor specifies the location of the floor shininess factor for the flashlight effect.
// FlashLocSpecBoostWall specifies the location of the wall specular boost factor for the flashlight effect.
// FlashLocSpecBoostFloor specifies the location of the floor specular boost factor for the flashlight effect.
// FlashLocBeamRatioFactor specifies the location of the flashlight beam ratio factor.
// FlashLocVolumetricSteps specifies the location of the number of volumetric rendering steps for the flashlight.
// FlashLocLast represents the last flashlight location, marking the end of the enumeration.
const (
	FlashLocProjection = FlashlightLoc(iota)
	FlashLocView
	FlashLocInvView
	FlashLocFlashSpaceMatrix
	FlashLocTexture
	FlashLocNormalMap
	FlashLocFlashShadowMap
	FlashLocScreenResolution
	FlashLocFlashDir
	FlashLocFlashIntensityFactor
	FlashLocFlashOffset
	FlashLocFlashConeStart
	FlashLocFlashConeEnd
	FlashLocFalloff
	FlashLocEnableShadows
	FlashLocShininessWall
	FlashLocShininessFloor
	FlashLocSpecBoostWall
	FlashLocSpecBoostFloor
	FlashLocBeamRatioFactor
	FlashLocVolumetricSteps
	FlashLocIsAbsolute
	FlashLocDebugLights
	FlashLocLast
)

// ShadowLight represents a flashlight shader utility for rendering with advanced lighting and shadow effects.
type ShadowLight struct {
	ctx         api.IContext
	prg         uint32
	table       [FlashLocLast]int32
	shadows     bool
	shadowsInt  int32
	cal         *model.Calibration
	debugLights int
}

// NewShaderShadowLight creates and returns a new instance of ShadowLight with default values and shadows disabled.
func NewShaderShadowLight(ctx api.IContext, cal *model.Calibration) *ShadowLight {
	f := &ShadowLight{
		ctx:         ctx,
		cal:         cal,
		shadows:     false,
		shadowsInt:  0,
		debugLights: 0,
	}
	f.EnableShadows(false)
	return f
}

// EnableShadows toggles shadow rendering for the flashlight and updates related shadow parameters.
func (s *ShadowLight) EnableShadows(e bool) {
	s.shadows = e
	if s.shadows {
		s.shadowsInt = 1
	} else {
		s.shadowsInt = 0
	}
}

// HasShadow checks if the flashlight has shadows enabled and returns true if shadows are active.
func (s *ShadowLight) HasShadow() bool {
	return s.shadows
}

// SetupSamplers configures the shader program with uniform texture bindings for standard, normal, and shadow maps.
func (s *ShadowLight) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{0, 1, 2, 3}
	normalUnits := []int32{4, 5, 6, 7}
	s.ctx.Uniform1iv(s.GetUniform(FlashLocTexture), 4, &diffuseUnits[0])
	s.ctx.Uniform1iv(s.GetUniform(FlashLocNormalMap), 4, &normalUnits[0])
	s.ctx.Uniform1i(s.GetUniform(FlashLocFlashShadowMap), 13)
	return nil
}

// Init initializes the Depth instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *ShadowLight) Init() error {
	return nil
}

// GetUniform retrieves the uniform location associated with the given FlashlightLoc ID from the internal table.
func (s *ShadowLight) GetUniform(id FlashlightLoc) int32 {
	return s.table[id]
}

// Compile builds and links the shader program for the flashlight, resolving uniforms and handling shader errors.
func (s *ShadowLight) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "flashlight.frag"

	vSrc, fSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}

	vSh, err := ShaderCompile(s.ctx, vertId, string(vSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(s.ctx, fragId, string(fSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram(s.ctx, "flashLight", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[FlashLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[FlashLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[FlashLocInvView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_invView\x00"))
	s.table[FlashLocFlashSpaceMatrix] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashSpaceMatrix\x00"))
	s.table[FlashLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))
	s.table[FlashLocNormalMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_normalMap\x00"))
	s.table[FlashLocFlashShadowMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashShadowMap\x00"))
	s.table[FlashLocScreenResolution] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_screenResolution\x00"))
	s.table[FlashLocFlashDir] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashDir\x00"))
	s.table[FlashLocFlashIntensityFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashIntensityFactor\x00"))
	s.table[FlashLocFlashOffset] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashOffset\x00"))
	s.table[FlashLocFlashConeStart] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashConeStart\x00"))
	s.table[FlashLocFlashConeEnd] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashConeEnd\x00"))
	s.table[FlashLocFalloff] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashFalloff\x00"))
	s.table[FlashLocEnableShadows] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_enableShadows\x00"))
	s.table[FlashLocShininessWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessWall\x00"))
	s.table[FlashLocShininessFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessFloor\x00"))
	s.table[FlashLocSpecBoostWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostWall\x00"))
	s.table[FlashLocSpecBoostFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostFloor\x00"))
	s.table[FlashLocBeamRatioFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_beamRatioFactor\x00"))
	s.table[FlashLocVolumetricSteps] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_volumetricSteps\x00"))
	s.table[FlashLocIsAbsolute] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_isAbsolute\x00"))
	s.table[FlashLocDebugLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_debugLights\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("unused uniform location in flashlight: %d\n", idx)
		}
	}
	return nil
}

// Render applies flashlight rendering techniques, configuring shader uniforms and invoking provided geometry rendering logic.
func (s *ShadowLight) Render(renderGeometry func(), vao uint32, shadowTex uint32, view, proj, invView, lightSpace *float32, isAbsolute int32, posViewX, posViewY, posViewZ, dirViewX, dirViewY, dirViewZ, intensity, falloff, coneStart, coneEnd, screenW, screenH float32) {
	if intensity <= 0 {
		return
	}

	s.ctx.UseProgram(s.prg)
	s.ctx.BindVertexArray(vao)

	s.ctx.UniformMatrix4fv(s.GetUniform(FlashLocProjection), 1, false, proj)
	s.ctx.UniformMatrix4fv(s.GetUniform(FlashLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniform(FlashLocInvView), 1, false, invView)
	s.ctx.UniformMatrix4fv(s.GetUniform(FlashLocFlashSpaceMatrix), 1, false, lightSpace)

	s.ctx.Uniform2f(s.GetUniform(FlashLocScreenResolution), screenW, screenH)

	// Passaggio diretto dei vettori in View-Space
	s.ctx.Uniform3f(s.GetUniform(FlashLocFlashDir), dirViewX, dirViewY, dirViewZ)
	s.ctx.Uniform3f(s.GetUniform(FlashLocFlashOffset), posViewX, posViewY, posViewZ)

	s.ctx.Uniform1f(s.GetUniform(FlashLocFlashIntensityFactor), intensity)
	s.ctx.Uniform1f(s.GetUniform(FlashLocFalloff), falloff)

	s.ctx.Uniform1f(s.GetUniform(FlashLocFlashConeStart), coneStart)
	s.ctx.Uniform1f(s.GetUniform(FlashLocFlashConeEnd), coneEnd)
	s.ctx.Uniform1i(s.GetUniform(FlashLocEnableShadows), s.shadowsInt)

	s.ctx.Uniform1f(s.GetUniform(FlashLocShininessWall), float32(s.cal.ShininessWall))
	s.ctx.Uniform1f(s.GetUniform(FlashLocShininessFloor), float32(s.cal.ShininessFloor))
	s.ctx.Uniform1f(s.GetUniform(FlashLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	s.ctx.Uniform1f(s.GetUniform(FlashLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))
	s.ctx.Uniform1f(s.GetUniform(FlashLocBeamRatioFactor), float32(s.cal.BeamRatio))
	s.ctx.Uniform1i(s.GetUniform(FlashLocVolumetricSteps), int32(s.cal.VolSteps))
	s.ctx.Uniform1i(s.GetUniform(FlashLocIsAbsolute), isAbsolute)
	s.ctx.Uniform1i(s.GetUniform(FlashLocDebugLights), int32(s.debugLights))
	if s.shadows {
		s.ctx.ActiveTexture(api.TEXTURE13)
		s.ctx.BindTexture(api.TEXTURE_2D, shadowTex)
	}

	renderGeometry()
}
