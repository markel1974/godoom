package open_gl

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
)

const full3d = true

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
	Init() error
	SetupSamplers() error
	Compile(a shaders.IAssets) error
}

// Shaders manages multiple shader programs and related resources used in rendering, including main, sky, SSAO, and others.
type Shaders struct {
	tex           *Textures
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
	enableShadows bool
	metrics       *shaders.MapMetrics
	cal           *model.Calibration
	w             int32
	h             int32
	scaleX        float32
	scaleY        float32
}

// NewShaders initializes and returns a new instance of Shaders with default shader components and shadow settings.
func NewShaders() *Shaders {
	c := &Shaders{
		tex:           nil,
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
func (w *Shaders) Setup(vStride, lStride int32, cal *model.Calibration, tex *Textures) error {
	gl.Enable(gl.MULTISAMPLE)
	//gl.Enable(gl.SAMPLE_ALPHA_TO_COVERAGE)
	a := &Assets{}
	w.tex = tex
	w.cal = cal
	w.metrics = shaders.NewMapMetrics()
	w.metrics.SetOrthoSize(float32(w.cal.OrthoSize), float32(w.cal.ZNearRoom), float32(w.cal.ZFarRoom)+4.0)
	w.metrics.SetMapCenter(float32(w.cal.MapCenterX), float32(w.cal.MapCenterZ), float32(w.cal.LightCamY)+2.0)

	w.main = shaders.NewMain(vStride, w.metrics)
	w.sky = shaders.NewSky()
	w.geometry = shaders.NewGeometry()
	w.ssao = shaders.NewSSAO()
	w.blur = shaders.NewBlur()
	w.depth = shaders.NewDepth(w.metrics)
	w.lights = shaders.NewLights(lStride, w.cal)
	w.flashlight = shaders.NewShaderFlashlight(w.metrics, w.cal)
	w.post = shaders.NewPost()
	w.bloom = shaders.NewBloom()
	w.enableShadows = false
	w.container = append(w.container, w.main, w.sky, w.geometry, w.ssao, w.blur, w.depth, w.lights, w.flashlight, w.post, w.bloom)
	w.SetShadowEnabled(true)

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
func (w *Shaders) Render(vi *model.ViewMatrix, fbW int32, fbH int32, vert []float32, vertLen int32, indices []uint32, indicesLen int32, dc *DrawCommandsRender, skyEnabled bool, skyLayer float32, frameLights []float32, numLights int32) {
	if (w.w != fbW) || (w.h != fbH) {
		w.w = fbW
		w.h = fbH
		shadowW := fbW
		shadowH := fbW
		w.metrics.SetFlash(float32(w.cal.FlashFovDeg), float32(w.cal.FlashFalloff), float32(w.cal.ZNearFlash), float32(w.cal.ZFarFlash), shadowW, shadowH)
		if full3d {
			w.scaleX, w.scaleY = w.metrics.GetScale3d(fbW, fbH, float32(w.cal.ScaleFactor), float32(w.cal.FovVerticalDegrees))
		} else {
			w.scaleX, w.scaleY = w.metrics.GetScale2d(fbW, fbH)
		}

	}
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, w.tex.GetDiffuseArray())
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, w.tex.GetNormalArray())
	gl.ActiveTexture(gl.TEXTURE5)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, w.tex.GetEmissiveArray())

	px, _, pz := vi.GetXYZ()
	w.metrics.SetMapCenter(float32(px), float32(pz), w.metrics.GetLightCamY())

	//dirX, dirY, dirZ := vi.GetForwardVector()
	flashX, flashY, flashSensitivity := float32(0), float32(0), float32(0)
	if w.flashlight.HasShadow() {
		swayX, swayY, swaySensitivity := vi.GetSway()
		flashX, flashY, flashSensitivity = float32(swayX), float32(swayY), float32(swaySensitivity)
	}
	roomSpaceMatrix, flashSpaceMatrix, mainView := w.metrics.CreateSpaces(vi, flashX, flashY)
	var proj, view, invView = [16]float32{}, [16]float32{}, [16]float32{}
	if full3d {
		proj, view, invView = w.main.UpdateUniforms3d(vi, w.scaleX, w.scaleY)
	} else {
		proj, view, invView = w.main.UpdateUniforms2d(vi, w.scaleX, w.scaleY)
	}

	w.depth.UpdateUniforms(roomSpaceMatrix, flashSpaceMatrix, mainView)
	w.geometry.UpdateUniforms(view, proj)
	w.ssao.UpdateUniforms(view, proj)
	w.sky.UpdateUniforms(view, proj)

	// MAIN PREPARE (VBO che EBO)
	w.main.Prepare(vert, vertLen, indices, indicesLen, fbW, fbH)
	// LIGHTS PREPARE
	w.lights.Prepare(frameLights, numLights)
	// OMBRE
	w.depth.Render(dc.Render, w.main.GetVAO(), fbW, fbH)
	// SSAO PREPARE
	w.ssao.Prepare(fbW, fbH)
	// GEOMETRY
	w.geometry.Render(dc.Render)
	// SSAO
	w.ssao.Render(w.blur.GetProgram(), w.main.GetVAO(), w.sky.GetVAO(), w.post.GetFBO(), skyEnabled)
	// MAIN
	w.main.Render(dc.Render, w.ssao.GetSSAOBlurTexture(), w.post.GetFBO(), fbW, fbH)
	// ENABLE ADDITIVE LIGHTS
	enableAdditiveLights()
	// LIGHTS
	w.lights.Render(dc.Render, w.depth.GetRoomShadowTextures(), view, proj, invView, roomSpaceMatrix, float32(vi.GetLightIntensity()), float32(fbW), float32(fbH))
	// FLASHLIGHTS
	w.flashlight.Render(dc.Render, w.depth.GetFlashShadowTextures(), view, proj, invView, flashSpaceMatrix, flashX, flashY, flashSensitivity, float32(fbW), float32(fbH))
	// DISABLE ADDITIVE LIGHTS
	disableAdditiveLights()
	// SKYBOX
	w.sky.Render(skyLayer, skyEnabled)
	// MSAA resolution
	w.post.Prepare(fbW, fbH)
	// BLOOM
	w.bloom.Render(w.post.GetBrightBuffer(), fbW, fbH)
	// POST
	w.post.Render(w.bloom.GetBloomTexture(), fbW, fbH)
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
