package open_gl

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
)

// enableAdditiveLights configures OpenGL to use additive blending for rendering by adjusting depth and blend settings.
func enableAdditiveLights() {
	gl.DepthMask(false)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DepthFunc(gl.LEQUAL)
}

// disableAdditiveLights disables blend-based additive light rendering and reconfigures the depth test to default behavior.
func disableAdditiveLights() {
	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}

// IShader defines an interface for shader operations, including setup, sampler configuration, and compilation logic.
type IShader interface {
	Setup(width int32, height int32) error
	SetupSamplers() error
	Compile(a shaders.IAssets) error
	Init() error
}

// Shaders manages multiple shader programs and related resources used in rendering, including main, sky, SSAO, and others.
type Shaders struct {
	main          *shaders.Main
	sky           *shaders.Sky
	geometry      *shaders.Geometry
	ssao          *shaders.SSAO
	blur          *shaders.Blur
	depth         *shaders.Depth
	lights        *shaders.Lights
	flashlight    *shaders.Flashlight
	post          *shaders.Post
	bloom         *shaders.Bloom
	container     []IShader
	dcRender      *DrawCommandsRender
	enableShadows bool
	metrics       *shaders.MapMetrics
}

// NewShaders initializes and returns a new instance of Shaders with default shader components and shadow settings.
func NewShaders() *Shaders {
	c := &Shaders{
		main:          nil,
		sky:           nil,
		geometry:      nil,
		ssao:          nil,
		blur:          nil,
		depth:         nil,
		lights:        nil,
		flashlight:    nil,
		post:          nil,
		bloom:         nil,
		enableShadows: false,
	}
	return c
}

// Setup initializes shaders with the provided dimensions and strides, compiles them, and sets up vertex array buffers and samplers.
func (w *Shaders) Setup(width, height, vStride, lStride int32, calibration *model.Calibration) error {
	a := &Assets{}
	w.metrics = shaders.NewMapMetrics()
	if calibration != nil {
		w.metrics.OrthoSize = calibration.OrthoSize
		w.metrics.ZNearRoom = calibration.ZNearRoom
		w.metrics.ZFarRoom = calibration.ZFarRoom + 4.0
		w.metrics.LightCamY = calibration.LightCamY + 2.0
		w.metrics.MapCenterX = calibration.MapCenterX
		w.metrics.MapCenterZ = calibration.MapCenterZ
	}
	w.dcRender = NewDrawCommandsRender()

	w.main = shaders.NewMain(vStride, w.metrics)
	w.sky = shaders.NewSky()
	w.geometry = shaders.NewGeometry()
	w.ssao = shaders.NewSSAO()
	w.blur = shaders.NewBlur()
	w.depth = shaders.NewDepth()
	w.lights = shaders.NewLights(lStride)
	w.flashlight = shaders.NewShaderFlashlight(w.metrics)
	w.post = shaders.NewPost()
	w.bloom = shaders.NewBloom()
	w.enableShadows = false
	w.container = append(w.container, w.main, w.sky, w.geometry, w.ssao, w.blur, w.depth, w.lights, w.flashlight, w.post, w.bloom)
	w.SetShadowEnabled(true)

	for _, s := range w.container {
		if err := s.Setup(width, height); err != nil {
			return err
		}
	}
	for _, s := range w.container {
		if err := s.Compile(a); err != nil {
			return err
		}
	}
	for _, s := range w.container {
		if err := s.Init(); err != nil {
			return err
		}
	}
	for _, s := range w.container {
		if err := s.SetupSamplers(); err != nil {
			return err
		}
	}
	return nil
}

// SetShadowEnabled controls the global shadow rendering state by enabling or disabling shadows for all relevant shaders.
func (w *Shaders) SetShadowEnabled(v bool) {
	w.enableShadows = v
	w.flashlight.EnableShadows(w.enableShadows)
	w.lights.EnableShadows(w.enableShadows)
	w.depth.EnableShadows(w.enableShadows)
}

// Render handles the complete rendering pipeline, including geometry, lighting, post-processing, and optional sky rendering.
func (w *Shaders) Render(vi *model.ViewMatrix, pX, pY float64, fbW int32, fbH int32, vert []float32, vertLen int32, indices []uint32, indicesLen int32, dc []*DrawCommand, skyEnabled bool, skyTexId, skyNormalTexId, skyEmissiveTexId uint32, frameLights []float32, numLights int32) {
	exec := func() { w.dcRender.Render(dc) }
	bob := vi.GetBobPhase()
	swayX := w.flashlight.GetOffsetX(bob)
	swayY := w.flashlight.GetOffsetY(bob)
	roomSpaceMatrix, flashSpaceMatrix := w.metrics.CreateSpaces(vi, pX, pY, swayX, swayY)

	proj, view, invView := w.main.UpdateUniforms(vi)
	w.depth.UpdateUniforms(roomSpaceMatrix, flashSpaceMatrix)
	w.geometry.UpdateUniforms(view, proj)
	w.ssao.UpdateUniforms(view, proj)
	w.sky.UpdateUniforms(view, proj)

	gl.Viewport(0, 0, fbW, fbH)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// MAIN PREPARE (Ora invia sia VBO che EBO)
	w.main.Prepare(vert, vertLen, indices, indicesLen)

	// LIGHTS PREPARE
	w.lights.Prepare(frameLights, numLights)

	// OMBRE
	w.depth.Render(exec, w.main.GetVAO())
	// SSAO
	w.ssao.Prepare()
	w.geometry.Render(exec)
	w.ssao.Render(w.blur.GetProgram(), w.main.GetVAO(), w.sky.GetVAO(), w.post.GetFBO(), skyEnabled)
	// MAIN
	w.main.Render(exec, w.ssao.GetSSAOBlurTexture())

	// ENABLE ADDITIVE LIGHTS
	enableAdditiveLights()
	// LIGHTS
	w.lights.Render(exec, w.depth.GetRoomShadowTextures(), view, proj, invView, roomSpaceMatrix, float32(vi.GetLightIntensity()), float32(fbW), float32(fbH))
	// FLASHLIGHTS
	w.flashlight.Render(exec, w.depth.GetFlashShadowTextures(), view, proj, invView, flashSpaceMatrix, float32(-vi.GetYaw()), swayX, swayY, float32(fbW), float32(fbH))
	// DISABLE ADDITIVE LIGHTS
	disableAdditiveLights()

	// SKYBOX
	w.sky.Render(skyTexId, skyNormalTexId, skyEnabled)
	// BLOOM
	w.bloom.Render(w.post.GetBrightBuffer())
	// POST
	w.post.Render(w.bloom.GetBloomTexture())
}

// IncreaseFlashFactor increases the flashlight's intensity factor, enhancing the brightness of the flashlight effect.
func (w *Shaders) IncreaseFlashFactor() {
	w.flashlight.IncreaseFlashFactor()
}

// DecreaseFlashFactor reduces the flashlight's intensity factor, ensuring the value does not fall below the minimum limit.
func (w *Shaders) DecreaseFlashFactor() {
	w.flashlight.DecreaseFlashFactor()
}

// ToggleShadows toggles the state of shadow rendering in the shader system.
func (w *Shaders) ToggleShadows() { w.SetShadowEnabled(!w.enableShadows) }
