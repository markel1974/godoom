package shaders

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
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

	factor           float32
	offsetX          float32
	offsetY          float32
	enableShadows    bool
	enableShadowsInt int32
}

func NewShaderFlashlight() *Flashlight {
	f := &Flashlight{
		factor:           30.0,
		offsetX:          0.0,
		offsetY:          0.0,
		enableShadows:    false,
		enableShadowsInt: 0,
	}
	f.EnableFlash(false)
	return f
}

func (s *Flashlight) GetFactor() float32 {
	return s.factor
}

func (s *Flashlight) GetOffsetX() float32 {
	return s.offsetX
}

func (s *Flashlight) GetOffsetY() float32 {
	return s.offsetY
}

func (s *Flashlight) IncreaseFlashFactor() {
	s.factor++
}

// DecreaseFlashFactor reduces the factor value by 1, ensuring it does not drop below 0.
func (s *Flashlight) DecreaseFlashFactor() {
	if s.factor > 0 {
		s.factor--
	}
}

func (s *Flashlight) EnableFlash(e bool) {
	s.enableShadows = e
	if s.enableShadows {
		s.offsetX, s.offsetY = 0.1, -0.05
		s.enableShadowsInt = 1
	} else {
		s.offsetX, s.offsetY = 0.0, 0.0
		s.enableShadowsInt = 0
	}
}

func (s *Flashlight) Setup(width, height int32) {

}

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

func (s *Flashlight) Render(view, proj, invView, flashSpace [16]float32, pitchShear float32, fSwayX, fSwayY float32, screenW, screenH float32) {
	const shininessWall = 128.0
	const shininessFloor = 64.0
	const specBoostWall = 0.05
	const specBoostFloor = 0.1
	const beamRatio = 0.05
	const fBase = 0.9
	const volSteps = 32

	fConeStart := float32(math.Cos(fovFlashDeg/2.0*math.Pi/180.0)) + 0.01
	fConeEnd := float32(math.Cos(fovFlashDeg / 2.0 * 0.6 * math.Pi / 180.0))
	flashDirY := pitchShear / (2.0 * float32(model.VFov))

	gl.UseProgram(s.prg)

	gl.UniformMatrix4fv(s.GetUniform(FlashLocProjection), 1, false, &proj[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocView), 1, false, &view[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocInvView), 1, false, &invView[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocFlashSpaceMatrix), 1, false, &flashSpace[0])

	gl.Uniform2f(s.GetUniform(FlashLocScreenResolution), screenW, screenH)
	gl.Uniform3f(s.GetUniform(FlashLocFlashDir), 0.0, flashDirY, -1.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashIntensityFactor), s.factor)
	gl.Uniform3f(s.GetUniform(FlashLocFlashOffset), fSwayX, fSwayY, 0.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeStart), fConeStart)
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeEnd), fConeEnd)
	gl.Uniform1f(s.GetUniform(FlashLocFlashBase), fBase)
	gl.Uniform1i(s.GetUniform(FlashLocEnableShadows), s.enableShadowsInt)

	gl.Uniform1f(s.GetUniform(FlashLocShininessWall), shininessWall)
	gl.Uniform1f(s.GetUniform(FlashLocShininessFloor), shininessFloor)
	gl.Uniform1f(s.GetUniform(FlashLocSpecBoostWall), specBoostWall)
	gl.Uniform1f(s.GetUniform(FlashLocSpecBoostFloor), specBoostFloor)
	gl.Uniform1f(s.GetUniform(FlashLocBeamRatioFactor), beamRatio)
	gl.Uniform1i(s.GetUniform(FlashLocVolumetricSteps), volSteps)
}
