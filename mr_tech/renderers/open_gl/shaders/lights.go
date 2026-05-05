package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
)

const (
	lightsDoubleBuffer = 2
)

// LightLoc represents an identifier for light-related uniform locations in a shader program.
type LightLoc int

// LightLocProjection specifies the light's projection matrix location.
// LightLocView specifies the light's view matrix location.
// LightLocInvView specifies the light's inverse view matrix location.
// LightLocRoomSpaceMatrix specifies the light's room-space matrix location.
// LightLocTexture specifies the light's texture resource location.
// LightLocNormalMap specifies the light's normal map resource location.
// LightLocRoomShadowMap specifies the light's room shadow map resource location.
// LightLocScreenResolution specifies the screen resolution location.
// LightLocAmbientLight specifies the ambient light intensity location.
// LightLocEnableShadows specifies whether shadows are enabled location.
// LightLocVolumetricSteps specifies the number of volumetric lighting steps location.
// LightLocBeamRatioFactor specifies the beam-to-light ratio factor location.
// LightLocNumLights specifies the number of active lights location.
// LightLocLast specifies the marker for the last LightLoc value.
const (
	LightLocProjection = LightLoc(iota)
	LightLocView
	LightLocInvView
	LightLocRoomSpaceMatrix
	LightLocTexture
	LightLocNormalMap
	LightLocRoomShadowMap
	LightLocScreenResolution
	LightLocAmbientLight
	LightLocEnableShadows
	LightLocVolumetricSteps
	LightLocBeamRatioFactor
	LightLocNumLights
	LightLocShininessWall
	LightLocShininessFloor
	LightLocSpecBoostWall
	LightLocSpecBoostFloor
	LightLocLast
)

// Lights represents a collection of light data and OpenGL resources for managing and rendering dynamic scene lighting.
type Lights struct {
	prg          uint32
	table        [LightLocLast]int32
	uboLights    [lightsDoubleBuffer]uint32
	frameIdx     int
	activeLights int32
	shadows      int32
	stride       int32
	cal          *model.Calibration
}

// NewLights initializes and returns a new instance of Lights with default settings.
func NewLights(stride int32, cal *model.Calibration) *Lights {
	return &Lights{
		cal:      cal,
		stride:   stride,
		frameIdx: 0,
	}
}

// EnableShadows enables or disables shadow rendering based on the provided boolean value.
func (s *Lights) EnableShadows(e bool) {
	if e {
		s.shadows = 1
	} else {
		s.shadows = 0
	}
}

// Init initializes the uniform buffer object (UBO) for storing light data with the specified stride size.
func (s *Lights) Init() error {
	size := 1024 * int(s.stride)

	gl.GenBuffers(lightsDoubleBuffer, &s.uboLights[0])

	for i := 0; i < lightsDoubleBuffer; i++ {
		gl.BindBuffer(gl.UNIFORM_BUFFER, s.uboLights[i])
		gl.BufferData(gl.UNIFORM_BUFFER, size, gl.Ptr(nil), gl.DYNAMIC_DRAW)
	}

	gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
	return nil
}

// SetupSamplers configures shader samplers for texture, normal map, and room shadow map locations.
func (s *Lights) SetupSamplers() error {
	gl.UseProgram(s.prg)
	diffuseUnits := []int32{0, 1, 2, 3}
	normalUnits := []int32{4, 5, 6, 7}
	gl.Uniform1iv(s.GetUniform(LightLocTexture), 4, &diffuseUnits[0]) // FlashLocTexture in flashlight.go
	gl.Uniform1iv(s.GetUniform(LightLocNormalMap), 4, &normalUnits[0])
	gl.Uniform1i(s.GetUniform(LightLocRoomShadowMap), 12) // gl.TEXTURE12 per Room, gl.TEXTURE13 per Flash

	return nil
}

// GetUniform retrieves the uniform location for a given LightLoc identifier from the internal table.
func (s *Lights) GetUniform(id LightLoc) int32 {
	return s.table[id]
}

// Compile compiles the shaders for the Lights object and initializes its uniform locations and shader program.
func (s *Lights) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "lights.frag"

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
	s.prg, err = ShaderCreateProgram("lights", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[LightLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[LightLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[LightLocInvView] = gl.GetUniformLocation(s.prg, gl.Str("u_invView\x00"))
	s.table[LightLocRoomSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_roomSpaceMatrix\x00"))
	s.table[LightLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[LightLocNormalMap] = gl.GetUniformLocation(s.prg, gl.Str("u_normalMap\x00"))
	s.table[LightLocRoomShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_roomShadowMap\x00"))
	s.table[LightLocScreenResolution] = gl.GetUniformLocation(s.prg, gl.Str("u_screenResolution\x00"))
	s.table[LightLocAmbientLight] = gl.GetUniformLocation(s.prg, gl.Str("u_ambient_light\x00"))
	s.table[LightLocEnableShadows] = gl.GetUniformLocation(s.prg, gl.Str("u_enableShadows\x00"))
	s.table[LightLocVolumetricSteps] = gl.GetUniformLocation(s.prg, gl.Str("u_volumetricSteps\x00"))
	s.table[LightLocBeamRatioFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_beamRatioFactor\x00"))
	s.table[LightLocNumLights] = gl.GetUniformLocation(s.prg, gl.Str("u_numLights\x00"))
	s.table[LightLocShininessWall] = gl.GetUniformLocation(s.prg, gl.Str("u_shininessWall\x00"))
	s.table[LightLocShininessFloor] = gl.GetUniformLocation(s.prg, gl.Str("u_shininessFloor\x00"))
	s.table[LightLocSpecBoostWall] = gl.GetUniformLocation(s.prg, gl.Str("u_specBoostWall\x00"))
	s.table[LightLocSpecBoostFloor] = gl.GetUniformLocation(s.prg, gl.Str("u_specBoostFloor\x00"))

	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("unused uniform location in lights: %d\n", idx)
		}
	}

	blockIndex := gl.GetUniformBlockIndex(s.prg, gl.Str("LightsBlock\x00"))
	if blockIndex != gl.INVALID_INDEX {
		gl.UniformBlockBinding(s.prg, blockIndex, 0)
	}

	return nil
}

// Prepare updates the uniform buffer object (UBO) with lighting data for rendering and sets the active lights count.
func (s *Lights) Prepare(frameLights []float32, numLights int32) {
	s.activeLights = numLights
	s.frameIdx = (s.frameIdx + 1) % lightsDoubleBuffer
	if numLights > 0 {
		gl.BindBuffer(gl.UNIFORM_BUFFER, s.uboLights[s.frameIdx])
		// Scrittura asincrona garantita sull'UBO inattivo
		gl.BufferSubData(gl.UNIFORM_BUFFER, 0, len(frameLights)*4, gl.Ptr(frameLights))
		gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
	}
}

// Render configures the shader program and draws geometry with lighting, shadows, and volumetric effects applied.
func (s *Lights) Render(renderGeometry func(), roomShadowTex uint32, view, proj, invView, roomSpace [16]float32, ambient float32, screenW, screenH float32) {
	const spotIntensity = 10.0
	const beamRatio = 0.05

	gl.UseProgram(s.prg)

	gl.UniformMatrix4fv(s.GetUniform(LightLocProjection), 1, false, &proj[0])
	gl.UniformMatrix4fv(s.GetUniform(LightLocView), 1, false, &view[0])
	gl.UniformMatrix4fv(s.GetUniform(LightLocInvView), 1, false, &invView[0])
	gl.UniformMatrix4fv(s.GetUniform(LightLocRoomSpaceMatrix), 1, false, &roomSpace[0])

	// Invio del contatore luci come uniform indipendente
	gl.Uniform1i(s.GetUniform(LightLocNumLights), s.activeLights)

	gl.Uniform2f(s.GetUniform(LightLocScreenResolution), screenW, screenH)
	gl.Uniform1f(s.GetUniform(LightLocAmbientLight), ambient)
	shadows := s.shadows
	volSteps := int32(s.cal.VolSteps)
	shadows = 0
	volSteps = 0
	gl.Uniform1i(s.GetUniform(LightLocEnableShadows), shadows)
	gl.Uniform1i(s.GetUniform(LightLocVolumetricSteps), volSteps)
	gl.Uniform1f(s.GetUniform(LightLocBeamRatioFactor), beamRatio)

	const shininessWall = 10.0
	const shininessFloor = 40.0
	const specBoostWall = 0.02
	const specBoostFloor = 0.05

	gl.Uniform1f(s.GetUniform(LightLocShininessWall), float32(shininessWall))
	gl.Uniform1f(s.GetUniform(LightLocShininessFloor), float32(shininessFloor))
	gl.Uniform1f(s.GetUniform(LightLocSpecBoostWall), float32(specBoostWall))
	gl.Uniform1f(s.GetUniform(LightLocSpecBoostFloor), float32(specBoostFloor))

	gl.BindBufferBase(gl.UNIFORM_BUFFER, 0, s.uboLights[s.frameIdx])

	if s.shadows != 0 {
		gl.ActiveTexture(gl.TEXTURE12)
		gl.BindTexture(gl.TEXTURE_2D, roomShadowTex)
	}

	renderGeometry()
}
