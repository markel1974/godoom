package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// ShaderSkyLoc represents the location index of shader uniforms specific to the Sky renderer.
type ShaderSkyLoc int

// ShaderSkyLocProjection represents the projection matrix location for the sky shader.
// ShaderSkyLocView represents the view matrix location for the sky shader.
// ShaderSkyLocSky represents the sky texture location for the sky shader.
// ShaderSkyLocLast represents the total count of sky shader locations.
const (
	ShaderSkyLocProjection = ShaderSkyLoc(iota)
	ShaderSkyLocView
	ShaderSkyLocSky
	ShaderSkyLocLast
)

// Sky is a type that manages the state and rendering of the sky shader in a graphics application.
type Sky struct {
	prg    uint32
	table  [ShaderSkyLocLast]int32
	skyVAO uint32
	skyVBO uint32
	view   [16]float32
	proj   [16]float32
	width  int32
	height int32
}

// NewSky creates and returns a new instance of Sky with default uninitialized properties.
func NewSky() *Sky {
	return &Sky{
		prg: 0,
	}
}

// SetupSamplers initializes the sky vertex array and buffer objects and prepares the sampler for rendering sky elements.
func (s *Sky) SetupSamplers() error {
	// sky
	gl.GenVertexArrays(1, &s.skyVAO)
	gl.BindVertexArray(s.skyVAO)
	gl.GenBuffers(1, &s.skyVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.skyVBO)
	skyQuadVertices := []float32{-1.0, -1.0, 1.0, -1.0, -1.0, 1.0, 1.0, 1.0}
	gl.BufferData(gl.ARRAY_BUFFER, len(skyQuadVertices)*4, gl.Ptr(skyQuadVertices), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Restore default state
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	return nil
}

// Setup initializes the Sky dimensions by assigning the provided width and height values.
func (s *Sky) Setup(width int32, height int32) error {
	s.width = width
	s.height = height
	return nil
}

// Init initializes the Sky instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *Sky) Init() error {
	return nil
}

// GetProgram returns the OpenGL program identifier associated with the Sky instance.
func (s *Sky) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the location of a shader uniform variable by its ID.
func (s *Sky) GetUniform(id ShaderSkyLoc) int32 {
	return s.table[id]
}

// UpdateUniforms updates the view and projection matrices for the shader.
func (s *Sky) UpdateUniforms(view, proj [16]float32) {
	s.view = view
	s.proj = proj
}

// GetVAO retrieves the vertex array object (VAO) associated with the Sky instance.
func (s *Sky) GetVAO() uint32 {
	return s.skyVAO
}

// Compile compiles the shader program for rendering the sky using provided vertex and fragment shaders from assets.
func (s *Sky) Compile(a IAssets) error {
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
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in sky: %d", idx)
		}
	}
	return nil
}

// Render handles the rendering of the sky by setting up shaders, texture bindings, and drawing the vertex array.
func (s *Sky) Render(texId uint32, normTexId uint32, skyEnabled bool) {
	if !skyEnabled {
		return
	}
	gl.UseProgram(s.GetProgram())

	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(false)

	gl.UniformMatrix4fv(s.GetUniform(ShaderSkyLocProjection), 1, false, &s.proj[0])
	gl.UniformMatrix4fv(s.GetUniform(ShaderSkyLocView), 1, false, &s.view[0])

	gl.BindVertexArray(s.skyVAO)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texId)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, normTexId)
	gl.Uniform1i(s.GetUniform(ShaderSkyLocSky), 0)

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}
