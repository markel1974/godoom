package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
)

// MainLoc represents an identifier for uniform variables in the Main shader program.
type MainLoc int

const (
	MainLocView = MainLoc(iota)
	MainLocProjection
	MainLocTexture
	MainLocSSAO
	MainLocScreenResolution
	MainLocEmissiveMap
	MainLocEmissiveIntensity
	MainLocAoFactor
	MainLocLast
)

// Main represents the main OpenGL shader and associated resources for rendering.
type Main struct {
	prg               uint32
	table             [MainLocLast]int32
	mainVao           uint32
	mainVbo           uint32
	width             int32
	height            int32
	view              [16]float32
	proj              [16]float32
	emissiveIntensity float32
	aoFactor          float32
}

// NewMain initializes and returns a new instance of Main with default parameters.
func NewMain() *Main {
	return &Main{
		prg:               0,
		emissiveIntensity: 4.0,
		aoFactor:          0.8,
	}
}

// Init initializes the main VAO and VBO for the shader.
func (s *Main) Init() {
	const vboMaxFloats = 1024 * 1024 * 4

	gl.GenVertexArrays(1, &s.mainVao)
	gl.BindVertexArray(s.mainVao)
	gl.GenBuffers(1, &s.mainVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, vboMaxFloats*4, nil, gl.DYNAMIC_DRAW)
}

// SetupSamplers configures the shader program's texture samplers.
func (s *Main) SetupSamplers() {
	gl.UseProgram(s.prg)

	gl.Uniform1i(s.GetUniform(MainLocTexture), 0)
	gl.Uniform1i(s.GetUniform(MainLocSSAO), 2)
	gl.Uniform1i(s.GetUniform(MainLocEmissiveMap), 5)
}

// Setup initializes the width and height properties of the Main instance.
func (s *Main) Setup(width int32, height int32) {
	s.width = width
	s.height = height
}

// GetProgram returns the OpenGL program ID associated with the Main instance.
func (s *Main) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location for the given MainLoc identifier.
func (s *Main) GetUniform(id MainLoc) int32 {
	return s.table[id]
}

// GetVao returns the ID of the main Vertex Array Object (VAO).
func (s *Main) GetVao() uint32 {
	return s.mainVao
}

// Compile initializes, compiles, and links the shaders.
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
	s.table[MainLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[MainLocScreenResolution] = gl.GetUniformLocation(s.prg, gl.Str("u_screenResolution\x00"))
	s.table[MainLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[MainLocSSAO] = gl.GetUniformLocation(s.prg, gl.Str("u_ssao\x00"))
	s.table[MainLocEmissiveMap] = gl.GetUniformLocation(s.prg, gl.Str("u_emissiveMap\x00"))
	s.table[MainLocEmissiveIntensity] = gl.GetUniformLocation(s.prg, gl.Str("u_emissiveIntensity\x00"))
	s.table[MainLocAoFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_aoFactor\x00"))

	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in main: %d", idx)
		}
	}
	return nil
}

// Prepare uploads vertex data from the given FrameVertices to the GPU buffer.
func (s *Main) Prepare(vertices []float32, verticesLen int32) {
	total := int(verticesLen * 4)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, total, nil, gl.STREAM_DRAW)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, total, gl.Ptr(vertices))
}

// UpdateUniforms updates the shader's uniform variables for base rendering.
func (s *Main) UpdateUniforms(vi *model.ViewMatrix) ([16]float32, [16]float32) {
	aspect := float32(s.width) / float32(s.height)
	scaleX := -(fovScaleFactor / aspect) * float32(model.HFov)
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

	s.proj = [16]float32{
		scaleX, 0, 0, 0,
		0, fovScaleY, 0, 0,
		0, pitchShear, (zFarRoom + zNearRoom) / (zNearRoom - zFarRoom), -1,
		0, 0, (2 * zFarRoom * zNearRoom) / (zNearRoom - zFarRoom), 0,
	}
	s.view = [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}

	return s.proj, s.view
}

// Render sets up and executes the main rendering pipeline for the base pass.
func (s *Main) Render(ssaoBlurTex uint32) {
	gl.UseProgram(s.GetProgram())

	gl.UniformMatrix4fv(s.GetUniform(MainLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(MainLocProjection), 1, false, &s.proj[0])
	gl.Uniform2f(s.GetUniform(MainLocScreenResolution), float32(s.width), float32(s.height))
	gl.Uniform1f(s.GetUniform(MainLocEmissiveIntensity), s.emissiveIntensity)
	gl.Uniform1f(s.GetUniform(MainLocAoFactor), s.aoFactor)

	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
	gl.BindVertexArray(s.mainVao)

	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, ssaoBlurTex)
}
