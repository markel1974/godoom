package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl_legacy/shaders"
)

// MainLoc represents an enumerated type for identifying shader uniform locations.
type MainLoc int

// mainDoubleBuffer defines the number of buffers used in double buffering for efficient rendering operations.
const (
	mainDoubleBuffer = 2
)

// MainLocView represents the location index for the view matrix.
// MainLocProjection represents the location index for the projection matrix.
// MainLocTexture represents the location index for the texture sampler.
// MainLocSSAO represents the location index for the SSAO effect.
// MainLocScreenResolution represents the location index for the screen resolution.
// MainLocEmissiveMap represents the location index for the emissive map texture.
// MainLocEmissiveIntensity represents the location index for the emissive light intensity.
// MainLocAoFactor represents the location index for the ambient occlusion factor.
// MainLocLast represents the last index in the MainLoc enumeration.
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

// Main represents the primary rendering configuration and state for a graphics pipeline.
type Main struct {
	prg               uint32
	table             [MainLocLast]int32
	mainVAO           [mainDoubleBuffer]uint32
	mainVBO           [mainDoubleBuffer]uint32
	mainEBO           [mainDoubleBuffer]uint32
	vboBytesCap       [mainDoubleBuffer]int
	eboBytesCap       [mainDoubleBuffer]int
	frameIdx          int
	width             int32
	height            int32
	scaleX            float32
	scaleY            float32
	view              [16]float32
	proj              [16]float32
	emissiveIntensity float32
	aoFactor          float32
	stride            int32
	metrics           *MapMetrics
}

// NewMain creates and initializes a new instance of Main with the provided vertex stride value.
func NewMain(stride int32, metrics *MapMetrics) *Main {
	return &Main{
		prg:               0,
		emissiveIntensity: 4.0,
		aoFactor:          0.8,
		stride:            stride,
		metrics:           metrics,
	}
}

// Init initializes OpenGL buffers and vertex array objects, configures memory layout, and enables depth testing.
func (s *Main) Init() error {
	vboBytesSize := 131072 * int(s.stride)
	eboBytesSize := 262144 * 4

	gl.GenVertexArrays(mainDoubleBuffer, &s.mainVAO[0])
	gl.GenBuffers(mainDoubleBuffer, &s.mainVBO[0])
	gl.GenBuffers(mainDoubleBuffer, &s.mainEBO[0])

	for i := 0; i < mainDoubleBuffer; i++ {
		// Salva la capacità iniziale
		s.vboBytesCap[i] = vboBytesSize
		s.eboBytesCap[i] = eboBytesSize

		gl.BindVertexArray(s.mainVAO[i])

		gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVBO[i])
		gl.BufferData(gl.ARRAY_BUFFER, vboBytesSize, nil, gl.DYNAMIC_DRAW)

		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, s.mainEBO[i])
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, eboBytesSize, nil, gl.DYNAMIC_DRAW)

		strideBytes := s.stride
		gl.VertexAttribPointer(0, 3, gl.FLOAT, false, strideBytes, gl.PtrOffset(0))
		gl.EnableVertexAttribArray(0)
		gl.VertexAttribPointer(1, 3, gl.FLOAT, false, strideBytes, gl.PtrOffset(3*4))
		gl.EnableVertexAttribArray(1)
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	return nil
}

// SetupSamplers initializes and binds sampler uniforms for texture, SSAO, and emissive maps to the shader program.
func (s *Main) SetupSamplers() error {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(MainLocTexture), 0)
	gl.Uniform1i(s.GetUniform(MainLocSSAO), 2)
	gl.Uniform1i(s.GetUniform(MainLocEmissiveMap), 5)
	return nil
}

// Setup initializes the width and height values for the Main instance and returns an error if any issues occur.
func (s *Main) Setup(width int32, height int32) error {
	s.width = width
	s.height = height
	s.scaleX, s.scaleY = s.metrics.GetScale(s.width, s.height)
	return nil
}

// GetProgram returns the program ID associated with the Main instance.
func (s *Main) GetProgram() uint32 {
	return s.prg
}

// GetUniform returns the location of the specified uniform variable from the internal table.
func (s *Main) GetUniform(id MainLoc) int32 {
	return s.table[id]
}

// GetVAO returns the vertex array object identifier for the current frame buffer.
func (s *Main) GetVAO() uint32 {
	return s.mainVAO[s.frameIdx]
}

// Compile loads, compiles, and links the vertex and fragment shaders into a program, and sets up uniform locations.
func (s *Main) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "main.frag"

	vertexSrc, fragmentSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}
	vertexShader, err := ShaderCompile(vertId, string(vertexSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(fragId, string(fragmentSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram("main", vertexShader, fragmentShader)
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

// Prepare updates vertex and index buffer data for the current frame using double buffering.
func (s *Main) Prepare(vertices []float32, verticesLen int32, indices []uint32, indicesLen int32) {
	s.frameIdx = (s.frameIdx + 1) % mainDoubleBuffer

	vTotal := int(verticesLen) * 4
	iTotal := int(indicesLen) * 4

	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVBO[s.frameIdx])
	if vTotal > s.vboBytesCap[s.frameIdx] {
		newCap := vTotal * 2
		gl.BufferData(gl.ARRAY_BUFFER, newCap, nil, gl.DYNAMIC_DRAW)
		s.vboBytesCap[s.frameIdx] = newCap
	}
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, vTotal, gl.Ptr(vertices))

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, s.mainEBO[s.frameIdx])
	if iTotal > s.eboBytesCap[s.frameIdx] {
		newCap := iTotal * 2
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, newCap, nil, gl.DYNAMIC_DRAW)
		s.eboBytesCap[s.frameIdx] = newCap
	}
	gl.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, iTotal, gl.Ptr(indices))
}

// UpdateUniforms calculates and updates the projection, view, and inverse view matrices based on the given ViewMatrix.
func (s *Main) UpdateUniforms(vi *model.ViewMatrix) ([16]float32, [16]float32, [16]float32) {
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

	zFarRoom := s.metrics.GetRoomZFar()
	zNearRoom := s.metrics.GetRoomZNear()
	s.proj = [16]float32{
		-s.scaleX, 0, 0, 0,
		0, s.scaleY, 0, 0,
		0, pitchShear, (zFarRoom + zNearRoom) / (zNearRoom - zFarRoom), -1,
		0, 0, (2 * zFarRoom * zNearRoom) / (zNearRoom - zFarRoom), 0,
	}
	s.view = [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}

	var invView [16]float32
	if inv, ok := shaders.MatrixInverse4x4(s.view); ok {
		invView = inv
	}

	return s.proj, s.view, invView
}

// Render prepares and executes the rendering pipeline using the provided geometry and SSAO texture.
func (s *Main) Render(renderGeometry func(), ssaoBlurTex uint32) {
	gl.UseProgram(s.GetProgram())

	gl.UniformMatrix4fv(s.GetUniform(MainLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(MainLocProjection), 1, false, &s.proj[0])
	gl.Uniform2f(s.GetUniform(MainLocScreenResolution), float32(s.width), float32(s.height))
	gl.Uniform1f(s.GetUniform(MainLocEmissiveIntensity), s.emissiveIntensity)
	gl.Uniform1f(s.GetUniform(MainLocAoFactor), s.aoFactor)

	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)

	// Il Bind del VAO riattiva automaticamente lo stato del VBO e dell'EBO associati
	gl.BindVertexArray(s.mainVAO[s.frameIdx])

	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, ssaoBlurTex)

	renderGeometry()
}
