package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type FlashlightLoc int

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
	FlashLocFlashBase
	FlashLocEnableShadows
	FlashLocShininessWall
	FlashLocShininessFloor
	FlashLocSpecBoostWall
	FlashLocSpecBoostFloor
	FlashLocBeamRatioFactor
	FlashLocVolumetricSteps
	FlashLocLast
)

type Flashlight struct {
	prg   uint32
	table [FlashLocLast]int32
}

func NewShaderFlashlight() *Flashlight {
	return &Flashlight{}
}

func (s *Flashlight) Setup(width, height int32) {}

func (s *Flashlight) SetupSamplers() {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(FlashLocTexture), 0)
	gl.Uniform1i(s.GetUniform(FlashLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(FlashLocFlashShadowMap), 4)
}

func (s *Flashlight) GetUniform(id FlashlightLoc) int32 {
	return s.table[id]
}

func (s *Flashlight) Compile(a IAssets) error {
	vSrc, fSrc, err := a.ReadMulti("main.vert", "flashlight.frag")
	if err != nil {
		return err
	}

	vSh, err := ShaderCompile(string(vSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(string(fSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vSh)
		return err
	}

	s.prg, err = ShaderCreateProgram(vSh, fSh)
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
	s.table[FlashLocFlashBase] = gl.GetUniformLocation(s.prg, gl.Str("u_flashBase\x00"))
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

func (s *Flashlight) Render(proj, view, invView, flashSpace [16]float32, flashDirY, flashFactor, flashOffsetX, flashOffsetY, fConeStart, fConeEnd, fBase float32, enableShadows, volSteps int32, screenW, screenH, shininessWall, shininessFloor, specBoostWall, specBoostFloor, beamRatio float32) {
	gl.UseProgram(s.prg)

	gl.UniformMatrix4fv(s.GetUniform(FlashLocProjection), 1, false, &proj[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocView), 1, false, &view[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocInvView), 1, false, &invView[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocFlashSpaceMatrix), 1, false, &flashSpace[0])

	gl.Uniform2f(s.GetUniform(FlashLocScreenResolution), screenW, screenH)
	gl.Uniform3f(s.GetUniform(FlashLocFlashDir), 0.0, flashDirY, -1.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashIntensityFactor), flashFactor)
	gl.Uniform3f(s.GetUniform(FlashLocFlashOffset), flashOffsetX, flashOffsetY, 0.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeStart), fConeStart)
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeEnd), fConeEnd)
	gl.Uniform1f(s.GetUniform(FlashLocFlashBase), fBase)
	gl.Uniform1i(s.GetUniform(FlashLocEnableShadows), enableShadows)

	gl.Uniform1f(s.GetUniform(FlashLocShininessWall), shininessWall)
	gl.Uniform1f(s.GetUniform(FlashLocShininessFloor), shininessFloor)
	gl.Uniform1f(s.GetUniform(FlashLocSpecBoostWall), specBoostWall)
	gl.Uniform1f(s.GetUniform(FlashLocSpecBoostFloor), specBoostFloor)
	gl.Uniform1f(s.GetUniform(FlashLocBeamRatioFactor), beamRatio)
	gl.Uniform1i(s.GetUniform(FlashLocVolumetricSteps), volSteps)
}
