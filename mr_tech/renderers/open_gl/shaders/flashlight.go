package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
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
	FlashLocLast
)

// ShadowLight represents a flashlight shader utility for rendering with advanced lighting and shadow effects.
type ShadowLight struct {
	prg        uint32
	table      [FlashLocLast]int32
	factor     float32
	shadows    bool
	shadowsInt int32
	metrics    *MapMetrics
	cal        *model.Calibration
}

// NewShaderShadowLight creates and returns a new instance of ShadowLight with default values and shadows disabled.
func NewShaderShadowLight(metrics *MapMetrics, cal *model.Calibration) *ShadowLight {
	f := &ShadowLight{
		metrics:    metrics,
		cal:        cal,
		factor:     float32(cal.FlashFactor),
		shadows:    false,
		shadowsInt: 0,
	}
	f.EnableShadows(false)
	return f
}

// GetFactor returns the current intensity factor of the flashlight.
func (s *ShadowLight) GetFactor() float32 {
	return s.factor
}

// IncreaseFlashFactor increments the flashlight's intensity factor by increasing the `factor` field by 1.
func (s *ShadowLight) IncreaseFlashFactor() {
	s.factor++
}

// DecreaseFlashFactor reduces the flashlight's intensity factor by 1, ensuring it does not drop below 0.
func (s *ShadowLight) DecreaseFlashFactor() {
	if s.factor > 0 {
		s.factor--
	}
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
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(FlashLocTexture), 0)
	gl.Uniform1i(s.GetUniform(FlashLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(FlashLocFlashShadowMap), 4)
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

	vSh, err := ShaderCompile(vertId, string(vSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(fragId, string(fSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram("flashLight", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[FlashLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[FlashLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[FlashLocInvView] = gl.GetUniformLocation(s.prg, gl.Str("u_invView\x00"))
	s.table[FlashLocFlashSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_flashSpaceMatrix\x00"))
	s.table[FlashLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[FlashLocNormalMap] = gl.GetUniformLocation(s.prg, gl.Str("u_normalMap\x00"))
	s.table[FlashLocFlashShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_flashShadowMap\x00"))
	s.table[FlashLocScreenResolution] = gl.GetUniformLocation(s.prg, gl.Str("u_screenResolution\x00"))
	s.table[FlashLocFlashDir] = gl.GetUniformLocation(s.prg, gl.Str("u_flashDir\x00"))
	s.table[FlashLocFlashIntensityFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_flashIntensityFactor\x00"))
	s.table[FlashLocFlashOffset] = gl.GetUniformLocation(s.prg, gl.Str("u_flashOffset\x00"))
	s.table[FlashLocFlashConeStart] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeStart\x00"))
	s.table[FlashLocFlashConeEnd] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeEnd\x00"))
	s.table[FlashLocFalloff] = gl.GetUniformLocation(s.prg, gl.Str("u_flashFalloff\x00"))
	s.table[FlashLocEnableShadows] = gl.GetUniformLocation(s.prg, gl.Str("u_enableShadows\x00"))
	s.table[FlashLocShininessWall] = gl.GetUniformLocation(s.prg, gl.Str("u_shininessWall\x00"))
	s.table[FlashLocShininessFloor] = gl.GetUniformLocation(s.prg, gl.Str("u_shininessFloor\x00"))
	s.table[FlashLocSpecBoostWall] = gl.GetUniformLocation(s.prg, gl.Str("u_specBoostWall\x00"))
	s.table[FlashLocSpecBoostFloor] = gl.GetUniformLocation(s.prg, gl.Str("u_specBoostFloor\x00"))
	s.table[FlashLocBeamRatioFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_beamRatioFactor\x00"))
	s.table[FlashLocVolumetricSteps] = gl.GetUniformLocation(s.prg, gl.Str("u_volumetricSteps\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("unused uniform location in flashlight: %d\n", idx)
		}
	}
	return nil
}

// Render applies flashlight rendering techniques, configuring shader uniforms and invoking provided geometry rendering logic.
func (s *ShadowLight) Render(renderGeometry func(), flashShadowTex uint32, view, proj, invView, flashSpace [16]float32, fSwayX, fSwayY, fSwaySensitivity float32, screenW, screenH float32) {
	if s.factor <= 0 {
		return
	}
	// Calcola la direzione perturbata nello spazio di vista
	targetDirX := float32(s.cal.FlashOffsetX) - (fSwayX * fSwaySensitivity)
	targetDirY := float32(s.cal.FlashOffsetY) + (fSwayY * fSwaySensitivity)

	gl.UseProgram(s.prg)

	gl.UniformMatrix4fv(s.GetUniform(FlashLocProjection), 1, false, &proj[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocView), 1, false, &view[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocInvView), 1, false, &invView[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocFlashSpaceMatrix), 1, false, &flashSpace[0])
	gl.Uniform2f(s.GetUniform(FlashLocScreenResolution), screenW, screenH)
	gl.Uniform3f(s.GetUniform(FlashLocFlashDir), targetDirX, targetDirY, -1.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashIntensityFactor), s.factor)
	gl.Uniform3f(s.GetUniform(FlashLocFlashOffset), fSwayX, fSwayY, 0.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeStart), s.metrics.GetFlashConeStart())
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeEnd), s.metrics.GetFlashConeEnd())
	gl.Uniform1i(s.GetUniform(FlashLocEnableShadows), s.shadowsInt)

	gl.Uniform1f(s.GetUniform(FlashLocShininessWall), float32(s.cal.ShininessWall))
	gl.Uniform1f(s.GetUniform(FlashLocShininessFloor), float32(s.cal.ShininessFloor))
	gl.Uniform1f(s.GetUniform(FlashLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	gl.Uniform1f(s.GetUniform(FlashLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))
	gl.Uniform1f(s.GetUniform(FlashLocBeamRatioFactor), float32(s.cal.BeamRatio))
	gl.Uniform1i(s.GetUniform(FlashLocVolumetricSteps), int32(s.cal.VolSteps))
	gl.Uniform1f(s.GetUniform(FlashLocFalloff), float32(s.cal.FlashFalloff))

	if s.shadows {
		gl.ActiveTexture(gl.TEXTURE4)
		gl.BindTexture(gl.TEXTURE_2D, flashShadowTex)
	}

	renderGeometry()
}
