package shaders

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	metrics2 "github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
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
	tableOpaque       [MainLocLast]int32
	mainVAO           [mainDoubleBuffer]uint32
	mainVBO           [mainDoubleBuffer]uint32
	mainEBO           [mainDoubleBuffer]uint32
	vboBytesCap       [mainDoubleBuffer]int
	eboBytesCap       [mainDoubleBuffer]int
	frameIdx          int
	emissiveIntensity float32
	aoFactor          float32
	stride            int32
	w                 int32
	h                 int32
	metrics           *metrics2.Map
}

// NewMain initializes and returns a new instance of Main with the provided context, stride, and metrics.
func NewMain(ctx api.IContext, stride int32, metrics *metrics2.Map) *Main {
	m := &Main{
		ctx:               ctx,
		prgOpaque:         0,
		emissiveIntensity: 4.0,
		aoFactor:          0.8,
		stride:            stride,
		metrics:           metrics,
	}

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

	return nil
}

// GetProgramOpaque returns the program ID used for rendering opaque objects.
func (s *Main) GetProgramOpaque() uint32 {
	return s.prgOpaque
}

// GetUniformOpaque retrieves the uniform location for the opaque shader program using the given MainLoc identifier.
func (s *Main) GetUniformOpaque(id MainLoc) int32 {
	return s.tableOpaque[id]
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

// Render executes the main rendering pipeline, applying geometries, shaders, and SSAO textures to the target framebuffer.
func (s *Main) Render(renderGeometry func(), view, proj *float32, ssaoBlurTex uint32, targetFbo uint32, fbW, fbH int32) {
	interval := float32(0) //unused
	// target FBO preparation
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, targetFbo)
	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)

	s.ctx.UseProgram(s.GetProgramOpaque())

	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocProjection), 1, false, proj)
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
