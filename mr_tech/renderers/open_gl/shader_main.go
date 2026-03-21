package open_gl

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
)

// ShaderMainLoc represents an identifier for uniform variables in the ShaderMain shader program.
type ShaderMainLoc int

// ShaderMainLocView represents the location for the view matrix in the shader.
// ShaderMainLocProj represents the location for the projection matrix in the shader.
// ShaderMainLocAmbientLight represents the location for the ambient light in the shader.
// ShaderMainLocProjection represents the location for the projection data in the shader.
// ShaderMainLocScreenResolution represents the location for screen resolution in the shader.
// ShaderMainLocFlashDir represents the location for the flashlight direction in the shader.
// ShaderMainLocFlashIntensityFactor represents the location for the flashlight intensity factor in the shader.
// ShaderMainLocFlashConeStart represents the location for the flashlight cone start in the shader.
// ShaderMainLocFlashConeEnd represents the location for the flashlight cone end in the shader.
// ShaderMainLocTexture represents the location for the texture in the shader.
// ShaderMainLocNormalMap represents the location for the normal map in the shader.
// ShaderMainLocSSAO represents the location for the screen-space ambient occlusion in the shader.
// ShaderMainLocRoomSpaceMatrix represents the location for the room space matrix in the shader.
// ShaderMainLocFlashSpaceMatrix represents the location for the flashlight space matrix in the shader.
// ShaderMainLocRoomShadowMap represents the location for the room shadow map in the shader.
// ShaderMainLocFlashShadowMap represents the location for the flashlight shadow map in the shader.
// ShaderMainLocLast represents the end of the ShaderMainLoc enumerations.
const (
	ShaderMainLocView = ShaderMainLoc(iota)
	ShaderMainLocProj
	ShaderMainLocAmbientLight
	ShaderMainLocProjection
	ShaderMainLocScreenResolution
	ShaderMainLocFlashDir
	ShaderMainLocFlashIntensityFactor
	ShaderMainLocFlashConeStart
	ShaderMainLocFlashConeEnd
	ShaderMainLocTexture
	ShaderMainLocNormalMap
	ShaderMainLocSSAO // Allineato al tuo codice
	ShaderMainLocRoomSpaceMatrix
	ShaderMainLocFlashSpaceMatrix
	ShaderMainLocRoomShadowMap
	ShaderMainLocFlashShadowMap
	ShaderMainLocEnableShadows
	ShaderMainLocLast
)

// ShaderMain represents the main OpenGL shader and associated resources for rendering.
type ShaderMain struct {
	prg         uint32
	table       [ShaderMainLocLast]int32
	mainVao     uint32
	mainVbo     uint32
	width       int32
	height      int32
	flashFactor float32

	// Cache Uniformi
	view             [16]float32
	proj             [16]float32
	roomSpaceMatrix  [16]float32
	flashSpaceMatrix [16]float32
	ambientLight     float32
	flashDirY        float32
	enableShadows    int32
}

// NewShaderMain initializes and returns a new instance of ShaderMain with default parameters.
func NewShaderMain() *ShaderMain {
	return &ShaderMain{
		prg:           0,
		flashFactor:   0.0,
		enableShadows: 0,
	}
}

// Init initializes the main VAO and VBO for the shader, and allocates buffer data for dynamic drawing.
func (s *ShaderMain) Init() {
	gl.GenVertexArrays(1, &s.mainVao)
	gl.BindVertexArray(s.mainVao)
	gl.GenBuffers(1, &s.mainVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, vboMaxFloats*4, nil, gl.DYNAMIC_DRAW)
}

// SetupSamplers configures the shader program's texture samplers with specified uniform locations and texture units.
func (s *ShaderMain) SetupSamplers() {
	gl.UseProgram(s.prg)

	gl.Uniform1i(s.GetUniform(ShaderMainLocTexture), 0)
	gl.Uniform1i(s.GetUniform(ShaderMainLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(ShaderMainLocSSAO), 2)
	gl.Uniform1i(s.GetUniform(ShaderMainLocTexture), 0)
	gl.Uniform1i(s.GetUniform(ShaderMainLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(ShaderMainLocRoomShadowMap), 3)
	gl.Uniform1i(s.GetUniform(ShaderMainLocFlashShadowMap), 4)
}

// Setup initializes the width and height properties of the ShaderMain instance based on the provided dimensions.
func (s *ShaderMain) Setup(width int32, height int32) {
	s.width = width
	s.height = height
}

// GetProgram returns the OpenGL program ID associated with the ShaderMain instance.
func (s *ShaderMain) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location for the given ShaderMainLoc identifier from the predefined lookup table.
func (s *ShaderMain) GetUniform(id ShaderMainLoc) int32 {
	return s.table[id]
}

// Compile initializes, compiles, and links the shaders required for the ShaderMain program using provided assets.
func (s *ShaderMain) Compile(a IAssets) error {
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
	s.table[ShaderMainLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[ShaderMainLocProj] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[ShaderMainLocAmbientLight] = gl.GetUniformLocation(s.prg, gl.Str("u_ambient_light\x00"))
	s.table[ShaderMainLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[ShaderMainLocScreenResolution] = gl.GetUniformLocation(s.prg, gl.Str("u_screenResolution\x00"))
	s.table[ShaderMainLocFlashDir] = gl.GetUniformLocation(s.prg, gl.Str("u_flashDir\x00"))
	s.table[ShaderMainLocFlashIntensityFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_flashIntensityFactor\x00"))
	s.table[ShaderMainLocFlashConeStart] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeStart\x00"))
	s.table[ShaderMainLocFlashConeEnd] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeEnd\x00"))
	s.table[ShaderMainLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[ShaderMainLocNormalMap] = gl.GetUniformLocation(s.prg, gl.Str("u_normalMap\x00"))
	s.table[ShaderMainLocSSAO] = gl.GetUniformLocation(s.prg, gl.Str("u_ssao\x00"))
	s.table[ShaderMainLocRoomSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_roomSpaceMatrix\x00"))
	s.table[ShaderMainLocFlashSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_flashSpaceMatrix\x00"))
	s.table[ShaderMainLocRoomShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_roomShadowMap\x00"))
	s.table[ShaderMainLocFlashShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_flashShadowMap\x00"))
	s.table[ShaderMainLocEnableShadows] = gl.GetUniformLocation(s.prg, gl.Str("u_enableShadows\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}

// Prepare uploads vertex data from the given FrameVertices to the GPU buffer for rendering.
func (s *ShaderMain) Prepare(fv []float32, l int) {
	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, l*4, nil, gl.STREAM_DRAW)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, l*4, gl.Ptr(fv))
}

// UpdateUniforms updates the shader's uniform variables based on the view matrix and framebuffer dimensions.
func (s *ShaderMain) UpdateUniforms(vi *model.ViewMatrix, roomSpaceMatrix, flashSpaceMatrix [16]float32, flashFactor float32, enableShadows bool) ([16]float32, [16]float32) {
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
	return s.proj, s.view
}

// Render sets up and executes the main rendering pipeline using provided transformation matrices and texture IDs.
func (s *ShaderMain) Render(roomShadowTex, flashShadowTex, ssaoBlurTex uint32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.UseProgram(s.GetProgram())

	// Upload differito e isolato
	gl.UniformMatrix4fv(s.GetUniform(ShaderMainLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(ShaderMainLocProjection), 1, false, &s.proj[0])
	gl.Uniform1f(s.GetUniform(ShaderMainLocAmbientLight), s.ambientLight)
	gl.Uniform2f(s.GetUniform(ShaderMainLocScreenResolution), float32(s.width), float32(s.height))
	gl.Uniform3f(s.GetUniform(ShaderMainLocFlashDir), 0.0, s.flashDirY, -1.0)
	gl.Uniform1f(s.GetUniform(ShaderMainLocFlashIntensityFactor), s.flashFactor)
	gl.Uniform1f(s.GetUniform(ShaderMainLocFlashConeStart), 0.60)
	gl.Uniform1f(s.GetUniform(ShaderMainLocFlashConeEnd), 0.90)
	gl.Uniform1i(s.GetUniform(ShaderMainLocEnableShadows), s.enableShadows)
	gl.UniformMatrix4fv(s.GetUniform(ShaderMainLocRoomSpaceMatrix), 1, false, &s.roomSpaceMatrix[0])
	gl.UniformMatrix4fv(s.GetUniform(ShaderMainLocFlashSpaceMatrix), 1, false, &s.flashSpaceMatrix[0])

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
