package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
)

// MainLoc represents an identifier for uniform variables in the Main shader program.
type MainLoc int

// MainLocView represents the location for the view matrix in the shader.
// MainLocProj represents the location for the projection matrix in the shader.
// MainLocAmbientLight represents the location for the ambient light in the shader.
// MainLocProjection represents the location for the projection data in the shader.
// MainLocScreenResolution represents the location for screen resolution in the shader.
// MainLocFlashDir represents the location for the flashlight direction in the shader.
// MainLocFlashIntensityFactor represents the location for the flashlight intensity factor in the shader.
// MainLocFlashConeStart represents the location for the flashlight cone start in the shader.
// MainLocFlashConeEnd represents the location for the flashlight cone end in the shader.
// MainLocTexture represents the location for the texture in the shader.
// MainLocNormalMap represents the location for the normal map in the shader.
// MainLocSSAO represents the location for the screen-space ambient occlusion in the shader.
// MainLocRoomSpaceMatrix represents the location for the room space matrix in the shader.
// MainLocFlashSpaceMatrix represents the location for the flashlight space matrix in the shader.
// MainLocRoomShadowMap represents the location for the room shadow map in the shader.
// MainLocFlashShadowMap represents the location for the flashlight shadow map in the shader.
// MainLocLast represents the end of the MainLoc enumerations.
const (
	MainLocView = MainLoc(iota)
	MainLocProj
	MainLocAmbientLight
	MainLocProjection
	MainLocScreenResolution
	MainLocFlashDir
	MainLocFlashIntensityFactor
	MainLocFlashConeStart
	MainLocFlashConeEnd
	MainLocFlashBase
	MainLocTexture
	MainLocNormalMap
	MainLocSSAO // Allineato al tuo codice
	MainLocRoomSpaceMatrix
	MainLocFlashSpaceMatrix
	MainLocRoomShadowMap
	MainLocFlashShadowMap
	MainLocEnableShadows
	MainLocFlashOffset
	MainLocEmissiveMap
	MainLocEmissiveIntensity
	MainLocShininessWall // Nuove locazioni
	MainLocShininessFloor
	MainLocSpecBoostWall
	MainLocSpecBoostFloor
	MainLocBeamRatioFactor
	MainLocAoFactor
	MainLocRoomSpotIntensityFactor
	MainLocVolumetricSteps
	MainLocInvView
	MainLocLast
)

// Main represents the main OpenGL shader and associated resources for rendering.
type Main struct {
	prg         uint32
	table       [MainLocLast]int32
	mainVao     uint32
	mainVbo     uint32
	width       int32
	height      int32
	flashFactor float32

	// Cache Uniformi
	view                    [16]float32
	proj                    [16]float32
	roomSpaceMatrix         [16]float32
	flashSpaceMatrix        [16]float32
	ambientLight            float32
	flashDirY               float32
	enableShadows           int32
	flashOffsetX            float32
	flashOffsetY            float32
	flashConeStart          float32
	flashConeEnd            float32
	flashBase               float32
	shininessWall           float32
	shininessFloor          float32
	specBoostWall           float32
	specBoostFloor          float32
	emissiveIntensity       float32
	beamRatioFactor         float32
	aoFactor                float32
	roomSpotIntensityFactor float32
	volumetricSteps         int32
	invView                 [16]float32
}

// NewMain initializes and returns a new instance of Main with default parameters.
func NewMain() *Main {
	return &Main{
		prg:                     0,
		flashFactor:             0.0,
		flashConeStart:          0.60,
		flashConeEnd:            0.90,
		flashBase:               0.9,
		enableShadows:           0,
		shininessWall:           128.0,
		shininessFloor:          64.0,
		specBoostWall:           0.05,
		specBoostFloor:          0.1,
		emissiveIntensity:       4.0,
		beamRatioFactor:         0.05,
		aoFactor:                0.8,
		roomSpotIntensityFactor: 1.2,
		volumetricSteps:         32,
	}
}

// Init initializes the main VAO and VBO for the shader, and allocates buffer data for dynamic drawing.
func (s *Main) Init() {
	const vboMaxFloats = 1024 * 1024 * 4

	gl.GenVertexArrays(1, &s.mainVao)
	gl.BindVertexArray(s.mainVao)
	gl.GenBuffers(1, &s.mainVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, vboMaxFloats*4, nil, gl.DYNAMIC_DRAW)
}

// SetupSamplers configures the shader program's texture samplers with specified uniform locations and texture units.
func (s *Main) SetupSamplers() {
	gl.UseProgram(s.prg)

	gl.Uniform1i(s.GetUniform(MainLocTexture), 0)
	gl.Uniform1i(s.GetUniform(MainLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(MainLocSSAO), 2)
	gl.Uniform1i(s.GetUniform(MainLocTexture), 0)
	gl.Uniform1i(s.GetUniform(MainLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(MainLocRoomShadowMap), 3)
	gl.Uniform1i(s.GetUniform(MainLocFlashShadowMap), 4)
	gl.Uniform1i(s.GetUniform(MainLocEmissiveMap), 5)
}

// Setup initializes the width and height properties of the Main instance based on the provided dimensions.
func (s *Main) Setup(width int32, height int32) {
	s.width = width
	s.height = height
}

func (s *Main) SetMaterialParams(shWall, shFloor, sbWall, sbFloor float32) {
	s.shininessWall = shWall
	s.shininessFloor = shFloor
	s.specBoostWall = sbWall
	s.specBoostFloor = sbFloor
}

// GetProgram returns the OpenGL program ID associated with the Main instance.
func (s *Main) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location for the given MainLoc identifier from the predefined lookup table.
func (s *Main) GetUniform(id MainLoc) int32 {
	return s.table[id]
}

// GetVao returns the ID of the main Vertex Array Object (VAO) associated with the Main instance.
func (s *Main) GetVao() uint32 {
	return s.mainVao
}

// Compile initializes, compiles, and links the shaders required for the Main program using provided assets.
func (s *Main) Compile(a IAssets) error {
	vertexSrc, fragmentSrc, err := a.ReadMulti("main.vert", "main.frag")
	if err != nil {
		return err
	}
	vertexShader, err := ShaderCompile(string(vertexSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(string(fragmentSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[MainLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[MainLocProj] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[MainLocAmbientLight] = gl.GetUniformLocation(s.prg, gl.Str("u_ambient_light\x00"))
	s.table[MainLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[MainLocScreenResolution] = gl.GetUniformLocation(s.prg, gl.Str("u_screenResolution\x00"))
	s.table[MainLocFlashDir] = gl.GetUniformLocation(s.prg, gl.Str("u_flashDir\x00"))
	s.table[MainLocFlashIntensityFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_flashIntensityFactor\x00"))
	s.table[MainLocFlashConeStart] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeStart\x00"))
	s.table[MainLocFlashConeEnd] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeEnd\x00"))
	s.table[MainLocFlashOffset] = gl.GetUniformLocation(s.prg, gl.Str("u_flashOffset\x00"))
	s.table[MainLocFlashBase] = gl.GetUniformLocation(s.prg, gl.Str("u_flashBase\x00"))
	s.table[MainLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[MainLocNormalMap] = gl.GetUniformLocation(s.prg, gl.Str("u_normalMap\x00"))
	s.table[MainLocSSAO] = gl.GetUniformLocation(s.prg, gl.Str("u_ssao\x00"))
	s.table[MainLocRoomSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_roomSpaceMatrix\x00"))
	s.table[MainLocFlashSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_flashSpaceMatrix\x00"))
	s.table[MainLocRoomShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_roomShadowMap\x00"))
	s.table[MainLocFlashShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_flashShadowMap\x00"))
	s.table[MainLocEnableShadows] = gl.GetUniformLocation(s.prg, gl.Str("u_enableShadows\x00"))
	s.table[MainLocEmissiveMap] = gl.GetUniformLocation(s.prg, gl.Str("u_emissiveMap\x00"))
	s.table[MainLocEmissiveIntensity] = gl.GetUniformLocation(s.prg, gl.Str("u_emissiveIntensity\x00"))
	s.table[MainLocShininessWall] = gl.GetUniformLocation(s.prg, gl.Str("u_shininessWall\x00"))
	s.table[MainLocShininessFloor] = gl.GetUniformLocation(s.prg, gl.Str("u_shininessFloor\x00"))
	s.table[MainLocSpecBoostWall] = gl.GetUniformLocation(s.prg, gl.Str("u_specBoostWall\x00"))
	s.table[MainLocSpecBoostFloor] = gl.GetUniformLocation(s.prg, gl.Str("u_specBoostFloor\x00"))
	s.table[MainLocBeamRatioFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_beamRatioFactor\x00"))
	s.table[MainLocAoFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_aoFactor\x00"))
	s.table[MainLocRoomSpotIntensityFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_roomSpotIntensityFactor\x00"))
	s.table[MainLocVolumetricSteps] = gl.GetUniformLocation(s.prg, gl.Str("u_volumetricSteps\x00"))
	s.table[MainLocInvView] = gl.GetUniformLocation(s.prg, gl.Str("u_invView\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}

// Prepare uploads vertex data from the given FrameVertices to the GPU buffer for rendering.
func (s *Main) Prepare(fv []float32, l int) {
	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, l*4, nil, gl.STREAM_DRAW)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, l*4, gl.Ptr(fv))
}

// UpdateUniforms updates the shader's uniform variables based on the view matrix and framebuffer dimensions.
func (s *Main) UpdateUniforms(vi *model.ViewMatrix, roomSpaceMatrix, flashSpaceMatrix [16]float32, flashFactor float32, enableShadows bool, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32) {
	const near, far = float32(1.0), float32(4096.0)
	aspect := float32(s.width) / float32(s.height)
	scaleX := float32(-(2.0 / float64(aspect)) * model.HFov)
	scaleY := float32(2.0 * model.VFov)
	pitchShear := float32(-vi.GetYaw())

	sinA, cosA := vi.GetAngle()
	fX, fZ := float32(cosA), float32(-sinA)
	rX, rZ := float32(sinA), float32(cosA)
	viX, viY := vi.GetXY()
	viZ := vi.GetZ()
	ex, ey, ez := float32(viX), float32(viZ), float32(-viY)

	tx := -(rX*ex + rZ*ez)
	ty := -ey
	tz := fX*ex + fZ*ez

	s.flashDirY = pitchShear / scaleY
	s.flashFactor = flashFactor
	s.flashOffsetX = flashOffsetX
	s.flashOffsetY = flashOffsetY
	s.ambientLight = float32(vi.GetLightIntensity())
	s.roomSpaceMatrix = roomSpaceMatrix
	s.flashSpaceMatrix = flashSpaceMatrix
	if enableShadows {
		s.enableShadows = 1
	} else {
		s.enableShadows = 0
	}
	s.proj = [16]float32{
		scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, pitchShear, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}
	s.view = [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}
	if inv, ok := Inverse4x4(s.view); ok {
		s.invView = inv
	}
	return s.proj, s.view
}

// Render sets up and executes the main rendering pipeline using provided transformation matrices and texture IDs.
func (s *Main) Render(roomShadowTex, flashShadowTex, ssaoBlurTex uint32) {
	//gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.UseProgram(s.GetProgram())

	// Upload differito e isolato
	gl.UniformMatrix4fv(s.GetUniform(MainLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(MainLocProjection), 1, false, &s.proj[0])
	gl.Uniform1f(s.GetUniform(MainLocAmbientLight), s.ambientLight)
	gl.Uniform2f(s.GetUniform(MainLocScreenResolution), float32(s.width), float32(s.height))
	gl.Uniform3f(s.GetUniform(MainLocFlashDir), 0.0, s.flashDirY, -1.0)
	gl.Uniform1f(s.GetUniform(MainLocFlashIntensityFactor), s.flashFactor)
	gl.Uniform1f(s.GetUniform(MainLocFlashConeStart), s.flashConeStart)
	gl.Uniform1f(s.GetUniform(MainLocFlashConeEnd), s.flashConeEnd)
	gl.Uniform1f(s.GetUniform(MainLocFlashBase), s.flashBase)
	gl.Uniform1i(s.GetUniform(MainLocEnableShadows), s.enableShadows)
	gl.Uniform3f(s.GetUniform(MainLocFlashOffset), s.flashOffsetX, s.flashOffsetY, 0.0)
	gl.Uniform1f(s.GetUniform(MainLocShininessWall), s.shininessWall)
	gl.Uniform1f(s.GetUniform(MainLocShininessFloor), s.shininessFloor)
	gl.Uniform1f(s.GetUniform(MainLocSpecBoostWall), s.specBoostWall)
	gl.Uniform1f(s.GetUniform(MainLocSpecBoostFloor), s.specBoostFloor)
	gl.UniformMatrix4fv(s.GetUniform(MainLocRoomSpaceMatrix), 1, false, &s.roomSpaceMatrix[0])
	gl.UniformMatrix4fv(s.GetUniform(MainLocFlashSpaceMatrix), 1, false, &s.flashSpaceMatrix[0])
	gl.Uniform1f(s.GetUniform(MainLocEmissiveIntensity), s.emissiveIntensity)
	gl.Uniform1f(s.GetUniform(MainLocBeamRatioFactor), s.beamRatioFactor)
	gl.Uniform1f(s.GetUniform(MainLocAoFactor), s.aoFactor)
	gl.Uniform1f(s.GetUniform(MainLocRoomSpotIntensityFactor), s.roomSpotIntensityFactor)
	gl.Uniform1i(s.GetUniform(MainLocVolumetricSteps), s.volumetricSteps)
	gl.UniformMatrix4fv(s.GetUniform(MainLocInvView), 1, false, &s.invView[0])

	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
	gl.BindVertexArray(s.mainVao)

	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, ssaoBlurTex)
	if s.enableShadows != 0 {
		gl.ActiveTexture(gl.TEXTURE3)
		gl.BindTexture(gl.TEXTURE_2D, roomShadowTex)
		gl.ActiveTexture(gl.TEXTURE4)
		gl.BindTexture(gl.TEXTURE_2D, flashShadowTex)
	}
}
