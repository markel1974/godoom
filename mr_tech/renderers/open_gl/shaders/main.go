package shaders

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// MainLoc represents an integer-based enumerator used as an identifier for uniform locations within shaders.
type MainLoc int

// mainDoubleBuffer defines the number of buffer sets used for double buffering operations in rendering pipelines.
const (
	mainDoubleBuffer = 2
)

// MainLocView represents the location of the view matrix.
// MainLocProjection represents the location of the projection matrix.
// MainLocTexture represents the location of the texture data.
// MainLocSSAO represents the location of the screen space ambient occlusion (SSAO) data.
// MainLocScreenResolution represents the location of the screen resolution data.
// MainLocEmissiveMap represents the location of the emissive map data.
// MainLocEmissiveIntensity represents the location of the emissive intensity value.
// MainLocAoFactor represents the location of the ambient occlusion factor.
// MainLocTime represents the location of the time data.
// MainLocNear represents the location of the near plane distance in the projection matrix.
// MainLocFar represents the location of the far plane distance in the projection matrix.
// MainLocLast is the last location identifier, used as a sentinel or limit.
const (
	MainLocView = MainLoc(iota)
	MainLocProjection
	MainLocTexture
	MainLocSSAO
	MainLocScreenResolution
	MainLocEmissiveMap
	MainLocEmissiveIntensity
	MainLocAoFactor
	MainLocTime
	MainLocNear
	MainLocFar
	MainLocLast
)

// Main represents the central structure for managing rendering states and operations.
type Main struct {
	ctx               api.IContext
	prgOpaque         uint32
	prgAdditive       uint32
	prgLiquid         uint32
	tableOpaque       [MainLocLast]int32
	tableAdditive     [MainLocLast]int32
	tableLiquid       [MainLocLast]int32
	mainVAO           [mainDoubleBuffer]uint32
	mainVBO           [mainDoubleBuffer]uint32
	mainEBO           [mainDoubleBuffer]uint32
	vboBytesCap       [mainDoubleBuffer]int
	eboBytesCap       [mainDoubleBuffer]int
	frameIdx          int
	view              [16]float32
	proj              [16]float32
	invView           [16]float32
	viewPtr           *float32
	projPtr           *float32
	invViewPtr        *float32
	emissiveIntensity float32
	aoFactor          float32
	stride            int32
	w                 int32
	h                 int32
	metrics           *MapMetrics
}

// NewMain initializes and returns a new instance of Main with the provided context, stride, and metrics.
func NewMain(ctx api.IContext, stride int32, metrics *MapMetrics) *Main {
	m := &Main{
		ctx:               ctx,
		prgOpaque:         0,
		prgAdditive:       0,
		emissiveIntensity: 4.0,
		aoFactor:          0.8,
		stride:            stride,
		metrics:           metrics,
	}
	m.viewPtr = &m.view[0]
	m.projPtr = &m.proj[0]
	m.invViewPtr = &m.invView[0]
	return m
}

// Init initializes the necessary OpenGL resources, such as VAOs, VBOs, and EBOs, and configures the vertex attributes.
func (s *Main) Init() error {
	vboBytesSize := 131072 * int(s.stride)
	eboBytesSize := 262144 * 4

	s.ctx.GenVertexArrays(mainDoubleBuffer, &s.mainVAO[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.mainVBO[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.mainEBO[0])

	for i := 0; i < mainDoubleBuffer; i++ {
		s.vboBytesCap[i] = vboBytesSize
		s.eboBytesCap[i] = eboBytesSize

		s.ctx.BindVertexArray(s.mainVAO[i])

		s.ctx.BindBuffer(api.ARRAY_BUFFER, s.mainVBO[i])
		s.ctx.BufferData(api.ARRAY_BUFFER, vboBytesSize, nil, api.DYNAMIC_DRAW)

		s.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, s.mainEBO[i])
		s.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, eboBytesSize, nil, api.DYNAMIC_DRAW)

		// Il valore s.stride ora deve essere 40 (10 float * 4 byte)
		strideBytes := s.stride

		//  aPos (x, y, z) - 3 float
		s.ctx.VertexAttribPointer(0, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(0))
		s.ctx.EnableVertexAttribArray(0)

		//  aTexCoords (u, v, layer) - 3 float
		s.ctx.VertexAttribPointer(1, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(3*4))
		s.ctx.EnableVertexAttribArray(1)

		// Location 2: aOrigin (worldX, worldY, worldZ) - 3 float
		s.ctx.VertexAttribPointer(2, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(6*4))
		s.ctx.EnableVertexAttribArray(2)

		// Location 3: aRenderMode (flag) - 1 float
		s.ctx.VertexAttribPointer(3, 1, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(9*4))
		s.ctx.EnableVertexAttribArray(3)

		// Location 4: nextPos (x, y, z) - 3 float
		s.ctx.VertexAttribPointer(4, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(10*4))
		s.ctx.EnableVertexAttribArray(4)

		//  Location 5: aLerp (t) - 1 float
		s.ctx.VertexAttribPointer(5, 1, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(13*4))
		s.ctx.EnableVertexAttribArray(5)

		// NUOVO - Location 6: aYaw (angolo) - 1 float
		s.ctx.VertexAttribPointer(6, 1, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(14*4))
		s.ctx.EnableVertexAttribArray(6)
	}

	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	return nil
}

// SetupSamplers initializes sampler uniforms for opaque and additive shaders, binding texture units for diffuse and emissive maps.
func (s *Main) SetupSamplers() error {
	diffuseUnits := []int32{0, 1, 2, 3}
	emissiveUnits := []int32{8, 9, 10, 11}

	// Setup Opaque Samplers
	s.ctx.UseProgram(s.prgOpaque)
	s.ctx.Uniform1iv(s.GetUniformOpaque(MainLocTexture), 4, &diffuseUnits[0])
	s.ctx.Uniform1iv(s.GetUniformOpaque(MainLocEmissiveMap), 4, &emissiveUnits[0])
	s.ctx.Uniform1i(s.GetUniformOpaque(MainLocSSAO), 14)

	// Setup Additive Samplers
	s.ctx.UseProgram(s.prgAdditive)
	s.ctx.Uniform1iv(s.GetUniformAdditive(MainLocTexture), 4, &diffuseUnits[0])

	return nil
}

// GetProgramOpaque returns the program ID used for rendering opaque objects.
func (s *Main) GetProgramOpaque() uint32 {
	return s.prgOpaque
}

// GetProgramAdditive returns the program ID associated with additive rendering.
func (s *Main) GetProgramAdditive() uint32 {
	return s.prgAdditive
}

// GetUniformOpaque retrieves the uniform location for the opaque shader program using the given MainLoc identifier.
func (s *Main) GetUniformOpaque(id MainLoc) int32 {
	return s.tableOpaque[id]
}

// GetUniformAdditive returns the additive uniform location associated with the given MainLoc identifier.
func (s *Main) GetUniformAdditive(id MainLoc) int32 {
	return s.tableAdditive[id]
}

// GetVAO returns the Vertex Array Object (VAO) associated with the current frame index.
func (s *Main) GetVAO() uint32 {
	return s.mainVAO[s.frameIdx]
}

func (s *Main) GetLocView() int32 { return s.tableOpaque[MainLocView] }

func (s *Main) GetLocProj() int32 { return s.tableOpaque[MainLocProjection] }

// Compile initializes and compiles shader programs for opaque, additive, and liquid rendering using provided asset sources.
func (s *Main) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragOpaqueId = "main_opaque.frag"
	const fragAdditiveId = "main_additive.frag"
	const fragLiquidId = "main_liquid.frag"

	vertexSrc, fragmentOpaqueSrc, err := a.ReadMulti(vertId, fragOpaqueId)
	if err != nil {
		return err
	}

	vertexShader, err := ShaderCompile(s.ctx, vertId, string(vertexSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}

	// Compile Opaque Program
	fragOpaqueShader, err := ShaderCompile(s.ctx, fragOpaqueId, string(fragmentOpaqueSrc), api.FRAGMENT_SHADER)
	if err != nil {
		//s.ctx.DeleteShader(vertexShader)
		return err
	}
	s.prgOpaque, err = ShaderCreateProgram(s.ctx, "main_opaque", vertexShader, fragOpaqueShader)
	if err != nil {
		return err
	}
	// Compile Additive Program

	fragmentAdditiveSrc, err := a.Read(fragAdditiveId)
	if err != nil {
		return err
	}
	fragAdditiveShader, err := ShaderCompile(s.ctx, fragAdditiveId, string(fragmentAdditiveSrc), api.FRAGMENT_SHADER)
	if err != nil {
		return err
	}
	s.prgAdditive, err = ShaderCreateProgram(s.ctx, "main_additive", vertexShader, fragAdditiveShader)
	if err != nil {
		return err
	}
	// Compile Liquid Program
	fragmentLiquidSrc, err := a.Read(fragLiquidId)
	if err != nil {
		return err
	}
	fragLiquidShader, err := ShaderCompile(s.ctx, fragLiquidId, string(fragmentLiquidSrc), api.FRAGMENT_SHADER)
	if err != nil {
		return err
	}
	s.prgLiquid, err = ShaderCreateProgram(s.ctx, "main_liquid", vertexShader, fragLiquidShader)
	if err != nil {
		return err
	}

	// Setup Uniforms Opaque
	s.tableOpaque[MainLocView] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_view\x00"))
	s.tableOpaque[MainLocProjection] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_projection\x00"))
	s.tableOpaque[MainLocScreenResolution] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_screenResolution\x00"))
	s.tableOpaque[MainLocTexture] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_texture\x00"))
	s.tableOpaque[MainLocSSAO] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_ssao\x00"))
	s.tableOpaque[MainLocEmissiveMap] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_emissiveMap\x00"))
	s.tableOpaque[MainLocEmissiveIntensity] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_emissiveIntensity\x00"))
	s.tableOpaque[MainLocAoFactor] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_aoFactor\x00"))
	s.tableOpaque[MainLocTime] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_time\x00"))

	// Setup Uniforms Additive
	s.tableAdditive[MainLocView] = s.ctx.GetUniformLocation(s.prgAdditive, s.ctx.Str("u_view\x00"))
	s.tableAdditive[MainLocProjection] = s.ctx.GetUniformLocation(s.prgAdditive, s.ctx.Str("u_projection\x00"))
	s.tableAdditive[MainLocTexture] = s.ctx.GetUniformLocation(s.prgAdditive, s.ctx.Str("u_texture\x00"))
	s.tableAdditive[MainLocTime] = s.ctx.GetUniformLocation(s.prgAdditive, s.ctx.Str("u_time\x00"))

	s.tableLiquid[MainLocView] = s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_view\x00"))
	s.tableLiquid[MainLocProjection] = s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_projection\x00"))
	s.tableLiquid[MainLocScreenResolution] = s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_resolution\x00"))
	s.tableLiquid[MainLocTexture] = s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_texture\x00"))
	s.tableLiquid[MainLocTime] = s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_time\x00"))
	s.tableLiquid[MainLocNear] = s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_near\x00"))
	s.tableLiquid[MainLocFar] = s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_far\x00"))

	//println("DEBUG LOCATIONS! View:", s.tableLiquid[MainLocView], "Proj:", s.tableLiquid[MainLocProjection], "Time:", s.tableLiquid[MainLocTime])

	s.ctx.UseProgram(s.prgLiquid)
	texUnits := []int32{0, 1, 2, 3}
	s.ctx.Uniform1iv(s.tableLiquid[MainLocTexture], 4, &texUnits[0])
	s.ctx.Uniform1i(s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_refractionTex\x00")), 12)
	s.ctx.Uniform1i(s.ctx.GetUniformLocation(s.prgLiquid, s.ctx.Str("u_depthTex\x00")), 13)
	s.ctx.UseProgram(0)

	return nil
}

// Prepare configures the rendering pipeline, updates buffer data, and manages double buffering for rendering operations.
func (s *Main) Prepare(vertices []float32, verticesLen int32, indices []uint32, indicesLen int32, fbW, fbH int32) {
	//if fbW != s.w || fbH != s.h {
	//	s.w = fbW
	//	s.h = fbH
	//	s.scaleX, s.scaleY = s.metrics.GetScale(fbW, fbH)
	//}

	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)

	s.frameIdx = (s.frameIdx + 1) % mainDoubleBuffer

	vTotal := int(verticesLen) * 4
	iTotal := int(indicesLen) * 4

	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.mainVBO[s.frameIdx])
	if vTotal > s.vboBytesCap[s.frameIdx] {
		newCap := vTotal * 2
		s.ctx.BufferData(api.ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		s.vboBytesCap[s.frameIdx] = newCap
	}
	s.ctx.BufferSubData(api.ARRAY_BUFFER, 0, vTotal, s.ctx.Ptr(vertices))

	s.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, s.mainEBO[s.frameIdx])
	if iTotal > s.eboBytesCap[s.frameIdx] {
		newCap := iTotal * 2
		s.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		s.eboBytesCap[s.frameIdx] = newCap
	}
	s.ctx.BufferSubData(api.ELEMENT_ARRAY_BUFFER, 0, iTotal, s.ctx.Ptr(indices))
}

// UpdateUniforms3d computes and updates projection, view, and inverse view matrices for 3D rendering.
// It uses the provided view matrix and scaling factors for calculations and returns the matrices.
func (s *Main) UpdateUniforms3d(vi *model.ViewMatrix, scaleX float32, scaleY float32) (*float32, *float32, *float32) {
	// Acquire angles
	sinY, cosY := vi.GetAngleFull()
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
	viX, viY, viZ := vi.GetView()
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
	proj := [16]float32{
		scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, 0, (zFarRoom + zNearRoom) / (zNearRoom - zFarRoom), -1,
		0, 0, (2 * zFarRoom * zNearRoom) / (zNearRoom - zFarRoom), 0,
	}
	copy(s.proj[:], proj[:])
	// View Matrix (Standard OpenGL Column-Major Layout)
	view := [16]float32{
		rX, uX, -fX, 0, // Column 0 (Screen X vector)
		rY, uY, -fY, 0, // Column 1 (Screen Y vector)
		rZ, uZ, -fZ, 0, // Column 2 (Screen Z vector)
		tx, ty, tz, 1, // Column 3 (Positional translation)
	}
	copy(s.view[:], view[:])
	// Matrix inversion (Useful for dynamic Skyboxes or advanced Frustum Culling)
	if inv, ok := MatrixInverse4x4(s.view); ok {
		copy(s.invView[:], inv[:])
	}
	return s.projPtr, s.viewPtr, s.invViewPtr
}

// UpdateUniforms2d updates the 2D projection and view matrices based on the provided view matrix and scaling factors.
// Returns the updated projection matrix, view matrix, and the inverse view matrix.
func (s *Main) UpdateUniforms2d(vi *model.ViewMatrix, scaleX float32, scaleY float32) ([16]float32, [16]float32, [16]float32) {
	pitchShear := float32(-vi.GetPitch())
	sinA, cosA := vi.GetAngleFull()
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

	viX, viY, viZ := vi.GetView()
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

// RenderOpaque executes the main rendering pipeline, applying geometries, shaders, and SSAO textures to the target framebuffer.
func (s *Main) RenderOpaque(renderGeometry func(), ssaoBlurTex uint32, targetFbo uint32, fbW, fbH int32) {
	interval := float32(0) //unused
	// target FBO preparation
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, targetFbo)
	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)

	s.ctx.UseProgram(s.GetProgramOpaque())

	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocView), 1, false, &s.view[0])
	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocProjection), 1, false, &s.proj[0])
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocTime), interval)

	s.ctx.Uniform2f(s.GetUniformOpaque(MainLocScreenResolution), float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocEmissiveIntensity), s.emissiveIntensity)
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocAoFactor), s.aoFactor)

	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)

	s.ctx.BindVertexArray(s.mainVAO[s.frameIdx])

	s.ctx.ActiveTexture(api.TEXTURE14)
	s.ctx.BindTexture(api.TEXTURE_2D, ssaoBlurTex)

	// enable Alpha To Coverage only for the main geometry
	s.ctx.Enable(api.SAMPLE_ALPHA_TO_COVERAGE)
	renderGeometry()
	// disable it immediately to not destroy light passes
	s.ctx.Disable(api.SAMPLE_ALPHA_TO_COVERAGE)
}

// RenderAdditive configures and executes additive rendering by applying blending settings and invoking the provided geometry rendering function.
func (s *Main) RenderAdditive(renderGeometry func()) {
	s.ctx.UseProgram(s.GetProgramAdditive())

	interval := float32(0) //unused
	s.ctx.UniformMatrix4fv(s.GetUniformAdditive(MainLocView), 1, false, &s.view[0])
	s.ctx.UniformMatrix4fv(s.GetUniformAdditive(MainLocProjection), 1, false, &s.proj[0])
	s.ctx.Uniform1f(s.GetUniformAdditive(MainLocTime), interval)

	s.ctx.DepthMask(false)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	s.ctx.Enable(api.BLEND)
	s.ctx.BlendFunc(api.ONE, api.ONE)

	s.ctx.BindVertexArray(s.mainVAO[s.frameIdx])

	renderGeometry()

	s.ctx.Disable(api.BLEND)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LESS)
	s.ctx.DepthMask(true)
}

// RenderLiquid executes the rendering of liquid effects using the provided geometry and textures.
func (s *Main) RenderLiquid(renderGeometry func(), refractionTex, depthTex uint32, transparent bool, fbW, fbH int32) {
	s.ctx.UseProgram(s.prgLiquid)

	//TODO from config
	const liquidTimeScale = 0.02
	interval := float32(textures.GlobalTick()) * liquidTimeScale
	//TODO from config
	if transparent {
		s.ctx.Enable(api.BLEND)
		s.ctx.BlendFunc(api.SRC_ALPHA, api.ONE_MINUS_SRC_ALPHA)
		s.ctx.DepthMask(false) // Do not write to depth buffer
	} else {
		s.ctx.Disable(api.BLEND)
		s.ctx.DepthMask(true)
	}
	s.ctx.UniformMatrix4fv(s.tableLiquid[MainLocView], 1, false, &s.view[0])
	s.ctx.UniformMatrix4fv(s.tableLiquid[MainLocProjection], 1, false, &s.proj[0])
	s.ctx.Uniform1f(s.tableLiquid[MainLocTime], interval)
	s.ctx.Uniform2f(s.tableLiquid[MainLocScreenResolution], float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.tableLiquid[MainLocNear], 0.1)
	s.ctx.Uniform1f(s.tableLiquid[MainLocFar], 1000.0)

	s.ctx.BindVertexArray(s.mainVAO[s.frameIdx])

	s.ctx.ActiveTexture(api.TEXTURE12)
	s.ctx.BindTexture(api.TEXTURE_2D, refractionTex)
	s.ctx.ActiveTexture(api.TEXTURE13)
	s.ctx.BindTexture(api.TEXTURE_2D, depthTex)

	renderGeometry()

	// UNBIND TEXTURES TO PREVENT FEEDBACK LOOPS IN POST RESOLVE
	s.ctx.ActiveTexture(api.TEXTURE12)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.ActiveTexture(api.TEXTURE13)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)

	s.ctx.Disable(api.BLEND)
	s.ctx.DepthMask(true)
}
