package shaders

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
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
	view              [16]float32
	proj              [16]float32
	emissiveIntensity float32
	aoFactor          float32
	stride            int32
	w                 int32
	h                 int32
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
		s.vboBytesCap[i] = vboBytesSize
		s.eboBytesCap[i] = eboBytesSize

		gl.BindVertexArray(s.mainVAO[i])

		gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVBO[i])
		gl.BufferData(gl.ARRAY_BUFFER, vboBytesSize, nil, gl.DYNAMIC_DRAW)

		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, s.mainEBO[i])
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, eboBytesSize, nil, gl.DYNAMIC_DRAW)

		// Il valore s.stride ora deve essere 40 (10 float * 4 byte)
		strideBytes := s.stride

		// Location 0: aPos (x, y, z) - 3 float
		gl.VertexAttribPointer(0, 3, gl.FLOAT, false, strideBytes, gl.PtrOffset(0))
		gl.EnableVertexAttribArray(0)

		// Location 1: aTexCoords (u, v, layer) - 3 float
		gl.VertexAttribPointer(1, 3, gl.FLOAT, false, strideBytes, gl.PtrOffset(3*4))
		gl.EnableVertexAttribArray(1)

		// NUOVO - Location 2: aOrigin (worldX, worldY, worldZ) - 3 float
		gl.VertexAttribPointer(2, 3, gl.FLOAT, false, strideBytes, gl.PtrOffset(6*4))
		gl.EnableVertexAttribArray(2)

		// NUOVO - Location 3: aIsBillboard (flag) - 1 float
		gl.VertexAttribPointer(3, 1, gl.FLOAT, false, strideBytes, gl.PtrOffset(9*4))
		gl.EnableVertexAttribArray(3)
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
func (s *Main) Prepare(vertices []float32, verticesLen int32, indices []uint32, indicesLen int32, fbW, fbH int32) {
	//if fbW != s.w || fbH != s.h {
	//	s.w = fbW
	//	s.h = fbH
	//	s.scaleX, s.scaleY = s.metrics.GetScale(fbW, fbH)
	//}

	gl.Viewport(0, 0, fbW, fbH)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

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

// UpdateUniforms3d calculates and updates the projection, view, and inverse view matrices based on the given ViewMatrix.
func (s *Main) UpdateUniforms3d(vi *model.ViewMatrix, scaleX float32, scaleY float32) ([16]float32, [16]float32, [16]float32) {
	// Acquire angles
	sinY, cosY := vi.GetAngle()
	pitch := -vi.GetPitch()
	roll := vi.GetRoll()
	// Sine and Cosine of Pitch and Roll
	sinP, cosP := math.Sin(pitch), math.Cos(pitch)
	sinR, cosR := math.Sin(roll), math.Cos(roll)
	// Calculate camera base vectors (Quake-style True 3D)
	// Start from pure Yaw orientation mapped for OpenGL, and apply Pitch (up/down)
	// Forward vector (The direction the camera is looking)
	fX := float32(cosY * cosP)
	fY := float32(sinP)
	fZ := float32(-sinY * cosP)
	// Temporary Up vector (Tilted by Pitch, but without Roll)
	upX := float32(-cosY * sinP)
	upY := float32(cosP)
	upZ := float32(sinY * sinP)
	// Temporary Right vector (Always parallel to the floor before Roll)
	rightX := float32(sinY)
	rightY := float32(0)
	rightZ := float32(cosY)
	// Apply Roll (Bobbing/Tilt), rotate Right and Up vectors around the Forward axis
	rX := rightX*float32(cosR) + upX*float32(sinR)
	rY := rightY*float32(cosR) + upY*float32(sinR)
	rZ := rightZ*float32(cosR) + upZ*float32(sinR)

	uX := upX*float32(cosR) - rightX*float32(sinR)
	uY := upY*float32(cosR) - rightY*float32(sinR)
	uZ := upZ*float32(cosR) - rightZ*float32(sinR)

	// Spatial Mapping and Translation
	// Transform position from Model (Z-Up) to OpenGL space (Y-Up)
	viX, viY := vi.GetXY()
	viZ := vi.GetZ()
	ex := float32(viX)
	ey := float32(viZ)
	ez := float32(-viY)
	// View Matrix Translation (Inverse dot product)
	tx := -(rX*ex + rY*ey + rZ*ez)
	ty := -(uX*ex + uY*ey + uZ*ez)
	tz := fX*ex + fY*ey + fZ*ez // Note: positive because OpenGL looks toward -F
	// Pure Projection Matrix (No Pitch Shearing)
	zFarRoom := s.metrics.GetRoomZFar()
	zNearRoom := s.metrics.GetRoomZNear()
	s.proj = [16]float32{
		scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, 0, (zFarRoom + zNearRoom) / (zNearRoom - zFarRoom), -1,
		0, 0, (2 * zFarRoom * zNearRoom) / (zNearRoom - zFarRoom), 0,
	}
	// View Matrix (Standard OpenGL Column-Major Layout)
	s.view = [16]float32{
		rX, uX, -fX, 0, // Column 0 (Screen X vector)
		rY, uY, -fY, 0, // Column 1 (Screen Y vector)
		rZ, uZ, -fZ, 0, // Column 2 (Screen Z vector)
		tx, ty, tz, 1, // Column 3 (Positional translation)
	}
	// Matrix inversion (Useful for dynamic Skyboxes or advanced Frustum Culling)
	var invView [16]float32
	if inv, ok := MatrixInverse4x4(s.view); ok {
		invView = inv
	}
	return s.proj, s.view, invView
}

func (s *Main) UpdateUniforms2d(vi *model.ViewMatrix, scaleX float32, scaleY float32) ([16]float32, [16]float32, [16]float32) {
	pitchShear := float32(-vi.GetPitch())
	sinA, cosA := vi.GetAngle()
	// Base Forward (Z) and Right (X) vectors from Yaw only
	fX, fZ := float32(cosA), float32(-sinA)
	rX, rZ := float32(sinA), float32(cosA)
	// Roll
	roll := float32(vi.GetRoll())
	sinR, cosR := float32(math.Sin(float64(roll))), float32(math.Cos(float64(roll)))
	// Rotate Right (X) and Up (Y) vectors around the Forward (Z) axis
	// Original local Up was (0, 1, 0)
	// Original local Right was (rX, 0, rZ)
	// New Right vector (X)
	newRx := rX * cosR
	newRy := sinR
	newRz := rZ * cosR
	// New Up vector (Y)
	newUx := -rX * sinR
	newUy := cosR
	newUz := -rZ * sinR

	viX, viY := vi.GetXY()
	viZ := vi.GetZ()
	ex, ey, ez := float32(viX), float32(viZ), float32(-viY)
	// Translation uses the new oriented vectors to shift the world
	tx := -(newRx*ex + newRy*ey + newRz*ez)
	ty := -(newUx*ex + newUy*ey + newUz*ez)
	// Up/Right rotate around Forward, Dir is unchanged
	tz := -((-fX)*ex + (-fZ)*ez)
	zFarRoom := s.metrics.GetRoomZFar()
	zNearRoom := s.metrics.GetRoomZNear()
	s.proj = [16]float32{
		-scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, pitchShear, (zFarRoom + zNearRoom) / (zNearRoom - zFarRoom), -1,
		0, 0, (2 * zFarRoom * zNearRoom) / (zNearRoom - zFarRoom), 0,
	}
	// Updated View Matrix (Column-Major)
	s.view = [16]float32{
		newRx, newUx, -fX, 0, // Col 0
		newRy, newUy, 0, 0, // Col 1
		newRz, newUz, -fZ, 0, // Col 2
		tx, ty, tz, 1, // Col 3
	}
	var invView [16]float32
	if inv, ok := MatrixInverse4x4(s.view); ok {
		invView = inv
	}
	return s.proj, s.view, invView
}

// Render prepares and executes the rendering pipeline using the provided geometry and SSAO texture.
func (s *Main) Render(renderGeometry func(), ssaoBlurTex uint32, targetFbo uint32, fbW, fbH int32) {
	// target FBO preparation
	gl.BindFramebuffer(gl.FRAMEBUFFER, targetFbo)
	gl.Viewport(0, 0, fbW, fbH)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	gl.UseProgram(s.GetProgram())

	gl.UniformMatrix4fv(s.GetUniform(MainLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(MainLocProjection), 1, false, &s.proj[0])
	gl.Uniform2f(s.GetUniform(MainLocScreenResolution), float32(fbW), float32(fbH))
	gl.Uniform1f(s.GetUniform(MainLocEmissiveIntensity), s.emissiveIntensity)
	gl.Uniform1f(s.GetUniform(MainLocAoFactor), s.aoFactor)

	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)

	gl.BindVertexArray(s.mainVAO[s.frameIdx])

	gl.ActiveTexture(gl.TEXTURE2)
	gl.BindTexture(gl.TEXTURE_2D, ssaoBlurTex)

	// enable Alpha To Coverage only for the main geometry
	gl.Enable(gl.SAMPLE_ALPHA_TO_COVERAGE)
	renderGeometry()
	// disable it immediately to not destroy light passes
	gl.Disable(gl.SAMPLE_ALPHA_TO_COVERAGE)
}
