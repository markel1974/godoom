package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// ShadowLightLoc represents an integer-based enumeration for shader uniform locations related to shadow and light properties.
type ShadowLightLoc int

// ShadowLightLocProjection represents the projection location for shadow lighting.
// ShadowLightLocView represents the view location for shadow lighting.
// ShadowLightLocInvView represents the inverse view location for shadow lighting.
// ShadowLightLocFlashSpaceMatrix represents the flash space matrix location for shadow lighting.
// ShadowLightLocTexture represents the texture location for shadow lighting.
// ShadowLightLocNormalMap represents the normal map location for shadow lighting.
// ShadowLightLocFlashShadowMap represents the flash shadow map location for shadow lighting.
// ShadowLightLocScreenResolution represents the screen resolution location for shadow lighting.
// ShadowLightLocFlashDir represents the flash direction location for shadow lighting.
// ShadowLightLocFlashIntensityFactor represents the intensity factor for flash lighting.
// ShadowLightLocFlashOffset represents the offset location for flash lighting.
// ShadowLightLocFlashConeStart represents the start of the flash cone for shadow lighting calculations.
// ShadowLightLocFlashConeEnd represents the end of the flash cone for shadow lighting calculations.
// ShadowLightLocFalloff represents the falloff factor for shadow lighting.
// ShadowLightLocEnableShadows represents whether shadows are enabled for shadow lighting.
// ShadowLightLocShininessWall represents shininess for wall calculations in shadow lighting.
// ShadowLightLocShininessFloor represents shininess for floor calculations in shadow lighting.
// ShadowLightLocSpecBoostWall represents the specular boost for walls in shadow lighting.
// ShadowLightLocSpecBoostFloor represents the specular boost for floors in shadow lighting.
// ShadowLightLocBeamRatioFactor represents the beam ratio factor for volumetric lighting effects.
// ShadowLightLocVolumetricSteps represents the number of steps for volumetric lighting calculations.
// ShadowLightLocIsAbsolute determines if absolute positioning is used for shadow lighting.
// ShadowLightLocDebugLights enables debugging for lights within the shadow lighting system.
// ShadowLightLocLast is the last enum value for the shadow lighting locations.
const (
	ShadowLightLocProjection = ShadowLightLoc(iota)
	ShadowLightLocView
	ShadowLightLocInvView
	ShadowLightLocFlashSpaceMatrix
	ShadowLightLocTexture
	ShadowLightLocNormalMap
	ShadowLightLocFlashShadowMap
	ShadowLightLocScreenResolution
	ShadowLightLocFlashDir
	ShadowLightLocFlashIntensityFactor
	ShadowLightLocFlashOffset
	ShadowLightLocFlashConeStart
	ShadowLightLocFlashConeEnd
	ShadowLightLocFalloff
	ShadowLightLocEnableShadows
	ShadowLightLocShininessWall
	ShadowLightLocShininessFloor
	ShadowLightLocSpecBoostWall
	ShadowLightLocSpecBoostFloor
	ShadowLightLocBeamRatioFactor
	ShadowLightLocVolumetricSteps
	ShadowLightLocIsAbsolute
	ShadowLightLocDebugLights
	ShadowLightLocLast
)

// ShadowLight represents a structure for managing lighting and shadow parameters in a 3D rendering context.
type ShadowLight struct {
	ctx         api.IContext
	prg         uint32
	table       [ShadowLightLocLast]int32
	shadows     bool
	shadowsInt  int32
	cal         *model.Calibration
	debugLights int
}

// NewShadowLight creates and returns a new instance of ShadowLight with the provided context and calibration settings.
func NewShadowLight(ctx api.IContext, cal *model.Calibration) *ShadowLight {
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

// EnableShadows sets the shadow rendering state by enabling or disabling shadows and updates the internal shadow flag.
func (s *ShadowLight) EnableShadows(e bool) {
	s.shadows = e
	if s.shadows {
		s.shadowsInt = 1
	} else {
		s.shadowsInt = 0
	}
}

// HasShadow returns true if the shadow light has shadows enabled, otherwise false.
func (s *ShadowLight) HasShadow() bool {
	return s.shadows
}

// SetupSamplers configures texture samplers for the shadow light, mapping specific texture units to shader uniforms.
func (s *ShadowLight) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{0, 1, 2, 3}
	normalUnits := []int32{4, 5, 6, 7}
	s.ctx.Uniform1iv(s.GetUniform(ShadowLightLocTexture), 4, &diffuseUnits[0])
	s.ctx.Uniform1iv(s.GetUniform(ShadowLightLocNormalMap), 4, &normalUnits[0])
	s.ctx.Uniform1i(s.GetUniform(ShadowLightLocFlashShadowMap), 13)
	return nil
}

// Init initializes the ShadowLight instance and prepares it for use, returning an error if initialization fails.
func (s *ShadowLight) Init() error {
	return nil
}

// GetUniform returns the uniform location corresponding to the provided ShadowLightLoc identifier.
func (s *ShadowLight) GetUniform(id ShadowLightLoc) int32 {
	return s.table[id]
}

// Compile initializes and compiles the shaders and shader program for the ShadowLight, and retrieves uniform locations.
func (s *ShadowLight) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "shadow_light.frag"

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

	s.table[ShadowLightLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[ShadowLightLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[ShadowLightLocInvView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_invView\x00"))
	s.table[ShadowLightLocFlashSpaceMatrix] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashSpaceMatrix\x00"))
	s.table[ShadowLightLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))
	s.table[ShadowLightLocNormalMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_normalMap\x00"))
	s.table[ShadowLightLocFlashShadowMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashShadowMap\x00"))
	s.table[ShadowLightLocScreenResolution] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_screenResolution\x00"))
	s.table[ShadowLightLocFlashDir] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashDir\x00"))
	s.table[ShadowLightLocFlashIntensityFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashIntensityFactor\x00"))
	s.table[ShadowLightLocFlashOffset] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashOffset\x00"))
	s.table[ShadowLightLocFlashConeStart] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashConeStart\x00"))
	s.table[ShadowLightLocFlashConeEnd] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashConeEnd\x00"))
	s.table[ShadowLightLocFalloff] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashFalloff\x00"))
	s.table[ShadowLightLocEnableShadows] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_enableShadows\x00"))
	s.table[ShadowLightLocShininessWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessWall\x00"))
	s.table[ShadowLightLocShininessFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessFloor\x00"))
	s.table[ShadowLightLocSpecBoostWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostWall\x00"))
	s.table[ShadowLightLocSpecBoostFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostFloor\x00"))
	s.table[ShadowLightLocBeamRatioFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_beamRatioFactor\x00"))
	s.table[ShadowLightLocVolumetricSteps] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_volumetricSteps\x00"))
	s.table[ShadowLightLocIsAbsolute] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_isAbsolute\x00"))
	s.table[ShadowLightLocDebugLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_debugLights\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("unused uniform location in flashlight: %d\n", idx)
		}
	}
	return nil
}

// Render configures the ShadowLight uniforms and executes the provided renderGeometry function.
func (s *ShadowLight) Render(renderGeometry func(), vao uint32, shadowTex uint32, view, proj, invView, lightSpace *float32, isAbsolute int32, posViewX, posViewY, posViewZ, dirViewX, dirViewY, dirViewZ, intensity, falloff, coneStart, coneEnd, screenW, screenH float32) {
	if intensity <= 0 {
		return
	}

	s.ctx.UseProgram(s.prg)
	s.ctx.BindVertexArray(vao)

	s.ctx.UniformMatrix4fv(s.GetUniform(ShadowLightLocProjection), 1, false, proj)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShadowLightLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShadowLightLocInvView), 1, false, invView)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShadowLightLocFlashSpaceMatrix), 1, false, lightSpace)

	s.ctx.Uniform2f(s.GetUniform(ShadowLightLocScreenResolution), screenW, screenH)

	// Passaggio diretto dei vettori in View-Space
	s.ctx.Uniform3f(s.GetUniform(ShadowLightLocFlashDir), dirViewX, dirViewY, dirViewZ)
	s.ctx.Uniform3f(s.GetUniform(ShadowLightLocFlashOffset), posViewX, posViewY, posViewZ)

	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocFlashIntensityFactor), intensity)
	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocFalloff), falloff)

	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocFlashConeStart), coneStart)
	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocFlashConeEnd), coneEnd)
	s.ctx.Uniform1i(s.GetUniform(ShadowLightLocEnableShadows), s.shadowsInt)

	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocShininessWall), float32(s.cal.ShininessWall))
	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocShininessFloor), float32(s.cal.ShininessFloor))
	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))
	s.ctx.Uniform1f(s.GetUniform(ShadowLightLocBeamRatioFactor), float32(s.cal.BeamRatio))
	s.ctx.Uniform1i(s.GetUniform(ShadowLightLocVolumetricSteps), int32(s.cal.VolSteps))
	s.ctx.Uniform1i(s.GetUniform(ShadowLightLocIsAbsolute), isAbsolute)
	s.ctx.Uniform1i(s.GetUniform(ShadowLightLocDebugLights), int32(s.debugLights))
	if s.shadows {
		s.ctx.ActiveTexture(api.TEXTURE13)
		s.ctx.BindTexture(api.TEXTURE_2D, shadowTex)
	}

	renderGeometry()
}
