package open_gl

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// ShaderSkyLoc represents an enumerated type for locating shader uniform variables in a ShaderSky instance.
type ShaderSkyLoc int

// ShaderSkyLocProjection specifies the location index for the projection matrix in the shader.
// ShaderSkyLocView specifies the location index for the view matrix in the shader.
// ShaderSkyLocSky specifies the location index for the sky texture in the shader.
// ShaderSkyLocLast marks the last valid index for shader locations.
const (
	ShaderSkyLocProjection = ShaderSkyLoc(iota)
	ShaderSkyLocView
	ShaderSkyLocSky
	ShaderSkyLocLast
)

// ShaderSky represents a GPU shader program used for rendering sky environments with custom vertex and fragment workflows.
type ShaderSky struct {
	prg    uint32
	table  [ShaderSkyLocLast]int32
	skyVao uint32
	skyVbo uint32
	view   [16]float32
	proj   [16]float32
	width  int32
	height int32
}

// NewShaderSky initializes and returns a new instance of ShaderSky with default uninitialized properties.
func NewShaderSky() *ShaderSky {
	return &ShaderSky{
		prg: 0,
	}
}

// SetupSamplers initializes and configures the vertex array and buffer objects for rendering the sky geometry.
func (s *ShaderSky) SetupSamplers() {
	// sky
	gl.GenVertexArrays(1, &s.skyVao)
	gl.BindVertexArray(s.skyVao)
	gl.GenBuffers(1, &s.skyVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.skyVbo)
	skyQuadVertices := []float32{-1.0, -1.0, 1.0, -1.0, -1.0, 1.0, 1.0, 1.0}
	gl.BufferData(gl.ARRAY_BUFFER, len(skyQuadVertices)*4, gl.Ptr(skyQuadVertices), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Restore default state
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
}

// Setup initializes the ShaderSky instance with the provided width and height values.
func (s *ShaderSky) Setup(width int32, height int32) {
	s.width = width
	s.height = height
}

// GetProgram returns the OpenGL shader program ID associated with the ShaderSky instance.
func (s *ShaderSky) GetProgram() uint32 {
	return s.prg
}

// GetUniform returns the OpenGL uniform location for the specified ShaderSkyLoc identifier.
func (s *ShaderSky) GetUniform(id ShaderSkyLoc) int32 {
	return s.table[id]
}

// UpdateUniforms updates the view and projection matrices used by the shader with the given values.
func (s *ShaderSky) UpdateUniforms(view, proj [16]float32) {
	s.view = view
	s.proj = proj
}

// Compile loads and compiles the vertex and fragment shaders, creates a shader program, and initializes uniform locations.
func (s *ShaderSky) Compile(a IAssets) error {
	vertexSrc, fragmentSrc, err := a.ReadMulti("sky.vert", "sky.frag")
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
	s.table[ShaderSkyLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[ShaderSkyLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[ShaderSkyLocSky] = gl.GetUniformLocation(s.prg, gl.Str("u_sky\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}

// Render renders the skybox using the specified textures and projection/view matrices. Handles depth testing and texture bindings.
func (s *ShaderSky) Render(texId uint32, normTexId uint32, validTex bool) {
	gl.UseProgram(s.GetProgram())

	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(false)

	gl.UniformMatrix4fv(s.GetUniform(ShaderSkyLocProjection), 1, false, &s.proj[0])
	gl.UniformMatrix4fv(s.GetUniform(ShaderSkyLocView), 1, false, &s.view[0])

	gl.BindVertexArray(s.skyVao)

	if validTex {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texId)
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, normTexId)
		gl.Uniform1i(s.GetUniform(ShaderSkyLocSky), 0)
	}

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}
