package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IBuilder provides methods for constructing and managing rendering data, such as vertices, lights, textures, and draw commands.
type IBuilder interface {
	Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine)

	GetSkyTexture() *textures.Texture

	GetSkyUV() (float64, float64)

	GetVerticesStride() int32

	GetLightsStride() int32

	GetDrawCommands() *DrawCommandsRender

	GetDrawCommandsAdditive() *DrawCommandsRender

	GetDrawCommandsLiquid() *DrawCommandsRender

	GetHWOcclusion() *OcclusionHW

	GetVertices() ([]float32, int32, []uint32, int32)

	GetLights() ([]float32, int32)

	GetShadowLights() ([8]*metrics.Light, int32)
}

// RenderOpenGL encapsulates rendering logic using the OpenGL API.
type RenderOpenGL struct {
	ctx    api.IContext
	engine *engine.Engine
	vi     *model.ViewMatrix
	player *model.ThingPlayer
	tex    *Textures
	//builder     IBuilder
	builder     *BuilderVolume
	enableClear bool

	flash         *model.Flash
	container     []IShader
	enableShadows bool

	mapMetrics          *metrics.Map
	flashlightMetrics   *metrics.Flashlight
	shadowLightsMetrics *metrics.ShadowLights

	handler    *shaders.Handler
	main       *shaders.Main
	sky        *shaders.Sky
	geometry   *shaders.Geometry
	ssao       *shaders.SSAO
	blur       *shaders.Blur
	depth      *shaders.Depth
	lights     *shaders.Lights
	flashlight *shaders.ShadowLight
	post       *shaders.Post
	bloom      *shaders.Bloom
	occlusion  *shaders.Occlusion
	additive   *shaders.Additive
	liquid     *shaders.Liquid
	deferred   *shaders.Deferred

	cal    *model.Calibration
	width  int32
	height int32
	scaleX float32
	scaleY float32

	dcOpaque   *DrawCommandsRender
	dcAdditive *DrawCommandsRender
	dcLiquid   *DrawCommandsRender
	hwOcc      *OcclusionHW
}

// NewRender initializes a new OpenGL rendering instance with the given context and dimensions.
func NewRender(ctx api.IContext) *RenderOpenGL {
	r := &RenderOpenGL{
		ctx:         ctx,
		engine:      nil,
		vi:          model.NewViewMatrix(),
		player:      nil,
		enableClear: false,
	}
	return r
}

// Setup initializes the RenderOpenGL instance by linking it to the provided engine and setting up the rendering context.
func (w *RenderOpenGL) Setup(en *engine.Engine) error {
	w.engine = en
	w.player = en.GetPlayer()
	if err := w.ctx.Setup(w); err != nil {
		return err
	}
	return nil
}

// Start initializes the rendering process by invoking the context's Start method with RenderStart.
func (w *RenderOpenGL) Start() {
	w.ctx.Start()
}

// RenderPrepare initializes shaders, textures, and builder components required for rendering and configures global settings.
func (w *RenderOpenGL) RenderPrepare() error {
	cal := w.engine.GetCalibration()
	w.tex = NewTextures(w.ctx)
	w.player.SetPitchOptions(-1.5, 1.5, 0.01)
	w.builder = NewBuilderVolume(w.ctx, w.tex, cal)
	w.dcOpaque = w.builder.GetDrawCommandsOpaque()
	w.dcAdditive = w.builder.GetDrawCommandsAdditive()
	w.dcLiquid = w.builder.GetDrawCommandsLiquid()
	w.hwOcc = w.builder.GetHWOcclusion()
	w.flash = w.player.GetFlash()
	w.cal = cal
	vStride := w.builder.GetVerticesStride()
	lStride := w.builder.GetLightsStride()
	a := &Assets{}
	w.ctx.Enable(api.MULTISAMPLE)
	//w.ctx.Enable(gl_api.SAMPLE_ALPHA_TO_COVERAGE)
	w.handler = shaders.NewHandler(vStride, lStride)
	if err := w.handler.Setup(w.ctx, cal, w.flash, 0, 0); err != nil {
		return err
	}
	w.mapMetrics = w.handler.GetMapMetrics()
	w.flashlightMetrics = w.handler.GetFlashlightMetrics()
	w.shadowLightsMetrics = w.handler.GetShadowLightsMetrics()
	w.main = w.handler.GetMain()
	w.sky = w.handler.GetSky()
	w.geometry = w.handler.GetGeometry()
	w.ssao = w.handler.GetSSAO()
	w.blur = w.handler.GetBlur()
	w.depth = w.handler.GetDepth()
	w.lights = w.handler.GetLights()
	w.flashlight = w.handler.GetShadowLight()
	w.post = w.handler.GetPost()
	w.bloom = w.handler.GetBloom()
	w.occlusion = w.handler.GetOcclusion()
	w.additive = w.handler.GetAdditive()
	w.liquid = w.handler.GetLiquid()
	w.enableShadows = false
	w.container = append(w.container, w.main, w.sky, w.geometry, w.ssao, w.blur, w.depth, w.lights, w.flashlight, w.post, w.bloom, w.occlusion, w.additive, w.liquid)
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
	if err := w.tex.Setup(w.engine.GetTextures()); err != nil {
		return err
	}
	return nil
}

// RenderAdvance triggers the execution of the engine's computation pipeline with the current player and view matrix state.
func (w *RenderOpenGL) RenderAdvance() {
	w.engine.Compute(w.player, w.vi)
}

// RenderStart initializes and executes the rendering process for the current frame with specified framebuffer dimensions.
func (w *RenderOpenGL) RenderStart(fbW1 int, fbH1 int, winW int, winH int) {
	fbW, fbH := int32(fbW1), int32(fbH1)
	w.builder.Compute(fbW, fbH, w.vi, w.engine)

	vert, vertLen, indices, indicesLen := w.builder.GetVertices()
	lights, lightsCount := w.builder.GetLights()
	shadowLights, shadowLightsCount := w.builder.GetShadowLights()

	skyTex, skyEnabled := float32(-1.0), false
	if cSky := w.builder.GetSkyTexture(); cSky != nil {
		skyTex, skyEnabled = w.tex.Get(cSky)
	}

	if w.handler.SetSize(fbW, fbH) {
		w.scaleX, w.scaleY = w.mapMetrics.GetScale3d(fbW, fbH, float32(w.cal.AspectRatio), float32(w.cal.FovVerticalDegrees))
		w.flashlightMetrics.Rebuild(fbW, fbH)
	}

	w.bindTextureBuckets()

	px, _, pz := w.vi.GetView()
	w.mapMetrics.SetMapCenter(float32(px), float32(pz), w.mapMetrics.GetLightCamY())
	w.mapMetrics.Update(w.vi, w.scaleX, w.scaleY)

	flashX, flashY, flashSensitivity := float32(0), float32(0), float32(0)
	if w.flashlight.HasShadow() {
		swayX, swayY, swaySensitivity := w.vi.GetSway()
		flashX, flashY, flashSensitivity = float32(swayX), float32(swayY), float32(swaySensitivity)
	}

	w.flashlightMetrics.Update(w.mapMetrics.GetMainViewPtr(), flashX, flashY)

	w.shadowLightsMetrics.Update(shadowLights, uint32(shadowLightsCount))

	//RENDER START HERE
	w.depth.Prepare()
	// MAIN PREPARE (VBO che EBO)
	w.main.Prepare(vert, vertLen, indices, indicesLen)
	// LIGHTS PREPARE
	w.lights.Prepare(lights, lightsCount)
	// BIND TEXTURE
	//w.bindTextureBuckets()
	// OMBRE
	w.depth.Render(w.dcOpaque.Render)
	// SSAO PREPARE
	w.ssao.Prepare()
	// GEOMETRY
	w.geometry.Render(w.dcOpaque.Render)
	// SSAO
	w.ssao.Render(skyEnabled)
	// MAIN OPAQUE
	w.main.Render(w.dcOpaque.Render)
	// HW OCCLUSION
	w.occlusion.Render(w.hwOcc.Render)

	// MAIN ADDITIVE
	if w.dcAdditive.HasCommands() {
		w.additive.RenderAdditive(w.dcAdditive.Render)
	}
	// MAIN LIQUID (Refraction & Depth Fog)
	if w.dcLiquid.HasCommands() {
		// OPTIMIZATION: We DO NOT call w.post.ResolveMSAA here.
		// Sampling the PREVIOUS FRAME's resolved FBO for refraction and depth fog
		// saves a massive 3x fullscreen MSAA blit (cutting ResolveMSAA time in half).
		// The 1-frame lag (16ms) is totally imperceptible through the distortion.
		// w.post.ResolveMSAA(fbW, fbH)
		w.liquid.Render(w.dcLiquid.Render, true)
	}
	// ENABLE ADDITIVE LIGHTS
	EnableAdditiveLights(w.ctx)
	// FLASHLIGHTS
	w.flashlight.Render(
		w.dcOpaque.Render, -1,
		flashX, flashY, 0.0,
		float32(w.flash.GetOffsetX())-(flashX*flashSensitivity),
		float32(w.flash.GetOffsetY())+(flashY*flashSensitivity),
		-1.0,
		float32(w.flash.GetIntensity()), float32(w.flash.GetFalloff()),
		float32(w.flash.GetConeStart()), float32(w.flash.GetConeEnd()))
	// SHADOW LIGHTS
	for lightIdx := int32(0); lightIdx < shadowLightsCount; lightIdx++ {
		light := shadowLights[lightIdx]
		w.flashlight.Render(w.dcOpaque.Render, lightIdx,
			light.X, light.Y, light.Z,
			light.DirX, light.DirY, light.DirZ,
			light.Intensity, light.Falloff, light.OuterCutOff, light.CutOff)
	}
	// AMBIENT LIGHTS
	w.lights.Render(w.dcOpaque.Render, float32(w.vi.GetLightIntensity()))
	// DISABLE ADDITIVE LIGHTS
	DisableAdditiveLights(w.ctx)
	//w.bindTextureBuckets() // Restore texture arrays!
	// SKYBOX
	if skyEnabled {
		u, v := w.builder.GetSkyUV()
		tick := float32(textures.TickF())
		w.sky.Render(skyTex, float32(u)*tick, float32(v)*tick)
	}
	// MSAA resolution
	w.post.Prepare()
	// BLOOM
	w.bloom.Render()
	// POST
	w.post.Render(int32(winW), int32(winH))
}

// RenderToggleShadows toggles the shadow rendering state in the shaders, enabling or disabling shadows dynamically.
func (w *RenderOpenGL) RenderToggleShadows() {
	w.SetShadowEnabled(!w.enableShadows)
}

// RenderEnableClear sets the enableClear flag to true, indicating that the rendering system should clear buffers.
func (w *RenderOpenGL) RenderEnableClear() {
	w.enableClear = true
}

// RenderPlayerThrow triggers the player to throw a predefined object with a set velocity and default throwable index.
func (w *RenderOpenGL) RenderPlayerThrow() {
	const throwableIndex = 2
	w.player.Throw(throwableIndex, 300)
}

// RenderPlayerFire triggers the player to fire their weapon, using the "gun" identifier as the weapon type.
func (w *RenderOpenGL) RenderPlayerFire() {
	w.player.Fire("gun")
}

// RenderPlayerDuckingToggle toggles the player's ducking state through the player's SetDucking method.
func (w *RenderOpenGL) RenderPlayerDuckingToggle() { w.player.SetDucking() }

// RenderPlayerJump triggers the player's jump action with the option for a multi-jump when the multi flag is true.
func (w *RenderOpenGL) RenderPlayerJump(multi bool) { w.player.SetJump(multi) }

// RenderPlayerMoves updates the player's movement based on impulse and directional input flags (up, down, left, right).
func (w *RenderOpenGL) RenderPlayerMoves(impulse float64, up bool, down bool, left bool, right bool) {
	w.player.Move(impulse, up, down, right, left)
}

// RenderPlayerMouseMove adjusts the player's view angle and pitch based on mouse movement values within defined thresholds.
func (w *RenderOpenGL) RenderPlayerMouseMove(mouseX float64, mouseY float64) {
	const offset = 10
	if mouseX > offset {
		mouseX = offset
	} else if mouseX < -offset {
		mouseX = -offset
	}
	if mouseY > offset {
		mouseY = offset
	} else if mouseY < -offset {
		mouseY = -offset
	}
	w.player.AddAngle(-mouseX * 0.03)
	w.player.SetPitch(mouseY)
	//w.player.MoveApply(0, 0, 0)
}

func (w *RenderOpenGL) RenderIncreaseFlashFactor() {
	w.player.GetFlash().IncreaseFlashFactor()
}
func (w *RenderOpenGL) RenderDecreaseFlashFactor() {
	w.player.GetFlash().DecreaseFlashFactor()
}

// SetShadowEnabled controls the global shadow rendering state by enabling or disabling shadows for all relevant shaders.
func (w *RenderOpenGL) SetShadowEnabled(v bool) {
	w.enableShadows = v
	w.flashlight.EnableShadows(w.enableShadows)
	w.lights.EnableShadows(w.enableShadows)
	w.depth.EnableShadows(w.enableShadows)
}

// bindTextureBuckets binds texture buckets to specific OpenGL texture units for rendering diffuse, normal, and emissive maps.
func (w *RenderOpenGL) bindTextureBuckets() {
	// Unità 0-3: Diffuse | 4-7: Normal | 8-11: Emissive
	for i := 0; i < w.tex.GetBucketsLen(); i++ {
		diffuse, normal, emissive := w.tex.GetBucket(i)

		w.ctx.ActiveTexture(api.TEXTURE0 + uint32(i))
		w.ctx.BindTexture(api.TEXTURE_2D_ARRAY, diffuse)

		w.ctx.ActiveTexture(api.TEXTURE4 + uint32(i))
		w.ctx.BindTexture(api.TEXTURE_2D_ARRAY, normal)

		w.ctx.ActiveTexture(api.TEXTURE8 + uint32(i))
		w.ctx.BindTexture(api.TEXTURE_2D_ARRAY, emissive)
	}
}
