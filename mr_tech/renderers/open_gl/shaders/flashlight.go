package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// FlashlightLoc is an enumerated type used as a key for accessing uniform variable locations in a Flashlight shader program.
type FlashlightLoc int

// FlashLocGAlbedoSpec represents the location of global albedo and specular texture in the flashlight shader.
// FlashLocGNormalEmiss represents the location of global normal and emissive texture in the flashlight shader.
// FlashLocGPositionDepth represents the location for global position and depth data in the flashlight shader.
// FlashLocFlashShadowMap represents the location of the flashlight's shadow map in the flashlight shader.
// FlashLocView represents the view matrix location in the flashlight shader.
// FlashLocInvView represents the inverse view matrix location in the flashlight shader.
// FlashLocFlashSpaceMatrix represents the flashlight-space transformation matrix location in the shader.
// FlashLocFlashDir represents the direction vector of the flashlight in the flashlight shader.
// FlashLocFlashIntensityFactor represents the intensity factor of the flashlight in the shader.
// FlashLocFlashOffset represents the offset used for the flashlight in the flashlight shader.
// FlashLocFlashConeStart represents the starting angle of the flashlight cone in the shader.
// FlashLocFlashConeEnd represents the endpoint angle of the flashlight cone in the shader.
// FlashLocFlashBase represents the base location of the flashlight in the flashlight shader.
// FlashLocVolumetricSteps represents the number of volumetric steps used in the flashlight shader.
// FlashLocLast marks the last valid flashlight shader location.
const (
	FlashLocGAlbedoSpec = FlashlightLoc(iota)
	FlashLocGNormalEmiss
	FlashLocGPositionDepth
	FlashLocFlashShadowMap
	FlashLocView
	FlashLocInvView
	FlashLocFlashSpaceMatrix
	FlashLocFlashDir
	FlashLocFlashIntensityFactor
	FlashLocFlashOffset
	FlashLocFlashConeStart
	FlashLocFlashConeEnd
	FlashLocFlashBase
	FlashLocVolumetricSteps
	FlashLocLast
)

// Flashlight represents a shader structure for handling flashlight effects in rendering.
type Flashlight struct {
	prg   uint32
	table [FlashLocLast]int32
}

// NewShaderFlashlight creates and returns a new instance of the Flashlight shader object.
func NewShaderFlashlight() *Flashlight {
	return &Flashlight{}
}

// Setup configures the Flashlight instance with the specified width and height values.
func (s *Flashlight) Setup(width, height int32) {}

// SetupSamplers configures the texture samplers by binding uniform locations to predefined texture units.
func (s *Flashlight) SetupSamplers() {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(FlashLocGAlbedoSpec), 0)
	gl.Uniform1i(s.GetUniform(FlashLocGNormalEmiss), 1)
	gl.Uniform1i(s.GetUniform(FlashLocGPositionDepth), 2)
	gl.Uniform1i(s.GetUniform(FlashLocFlashShadowMap), 3)
}

// GetUniform retrieves the OpenGL uniform location for the given FlashlightLoc identifier.
func (s *Flashlight) GetUniform(id FlashlightLoc) int32 {
	return s.table[id]
}

// Compile initializes and compiles the shaders, creates a shader program, and sets up uniform locations for the flashlight.
func (s *Flashlight) Compile(a IAssets) error {
	vSrc, fSrc, err := a.ReadMulti("post.vert", "flashlight.frag")
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

	// Binding locators (omessi per brevità, mapping 1:1 con la const FlashlightLoc)
	s.table[FlashLocGAlbedoSpec] = gl.GetUniformLocation(s.prg, gl.Str("gAlbedoSpec\x00"))
	s.table[FlashLocGNormalEmiss] = gl.GetUniformLocation(s.prg, gl.Str("gNormalEmiss\x00"))
	s.table[FlashLocGPositionDepth] = gl.GetUniformLocation(s.prg, gl.Str("gPositionDepth\x00"))
	s.table[FlashLocFlashShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_flashShadowMap\x00"))
	s.table[FlashLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[FlashLocInvView] = gl.GetUniformLocation(s.prg, gl.Str("u_invView\x00"))
	s.table[FlashLocFlashSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_flashSpaceMatrix\x00"))
	s.table[FlashLocFlashDir] = gl.GetUniformLocation(s.prg, gl.Str("u_flashDir\x00"))
	s.table[FlashLocFlashIntensityFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_flashIntensityFactor\x00"))
	s.table[FlashLocFlashOffset] = gl.GetUniformLocation(s.prg, gl.Str("u_flashOffset\x00"))
	s.table[FlashLocFlashConeStart] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeStart\x00"))
	s.table[FlashLocFlashConeEnd] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeEnd\x00"))
	s.table[FlashLocFlashBase] = gl.GetUniformLocation(s.prg, gl.Str("u_flashBase\x00"))
	s.table[FlashLocVolumetricSteps] = gl.GetUniformLocation(s.prg, gl.Str("u_volumetricSteps\x00"))

	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in ShaderFlashlight: %d", v)
		}
	}

	return nil
}

// Render executes the rendering process for the flashlight, applying transformations and configuring OpenGL states.
func (s *Flashlight) Render(drawScreenQuad func(), view, invView, flashSpace [16]float32, config *Main) {
	gl.UseProgram(s.prg)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DepthMask(false)
	gl.Disable(gl.DEPTH_TEST)

	gl.UniformMatrix4fv(s.GetUniform(FlashLocView), 1, false, &view[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocInvView), 1, false, &invView[0])
	gl.UniformMatrix4fv(s.GetUniform(FlashLocFlashSpaceMatrix), 1, false, &flashSpace[0])

	// Passaggio dei parametri torcia (prelevati dal config o dal player)
	gl.Uniform3f(s.GetUniform(FlashLocFlashDir), 0.0, config.flashDirY, -1.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashIntensityFactor), config.flashFactor)
	gl.Uniform3f(s.GetUniform(FlashLocFlashOffset), config.flashOffsetX, config.flashOffsetY, 0.0)
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeStart), config.flashConeStart)
	gl.Uniform1f(s.GetUniform(FlashLocFlashConeEnd), config.flashConeEnd)
	gl.Uniform1f(s.GetUniform(FlashLocFlashBase), config.flashBase)
	gl.Uniform1i(s.GetUniform(FlashLocVolumetricSteps), config.volumetricSteps)

	drawScreenQuad()

	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
}
