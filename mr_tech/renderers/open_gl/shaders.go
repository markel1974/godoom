package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
)

//const full3d = true

// enableAdditiveLights configures OpenGL to use additive blending for rendering by adjusting depth and blend settings.
func enableAdditiveLights(ctx api.IContext) {
	ctx.DepthMask(false)
	ctx.Enable(api.BLEND)
	ctx.BlendFunc(api.ONE, api.ONE)
	ctx.DepthFunc(api.LEQUAL)
}

// disableAdditiveLights disables blend-based additive light rendering and reconfigures the depth test to default behavior.
func disableAdditiveLights(ctx api.IContext) {
	ctx.Disable(api.BLEND)
	ctx.DepthMask(true)
	ctx.DepthFunc(api.LESS)
}

// IShader defines an interface for shader operations, including setup, sampler configuration, and compilation logic.
type IShader interface {
	Init() error
	SetupSamplers() error
	Compile(a shaders.IAssets) error
}

// Shaders manages multiple shader programs and related resources used in rendering, including main, sky, SSAO, and others.
type Shaders struct {
	ctx           api.IContext
	tex           *Textures
	flash         *model.Flash
	container     []IShader
	enableShadows bool

	handler       *shaders.Handler
	mapMetrics    *metrics.Map
	shadowMetrics *metrics.Shadows
	main          *shaders.Main
	sky           *shaders.Sky
	geometry      *shaders.Geometry
	ssao          *shaders.SSAO
	blur          *shaders.Blur
	depth         *shaders.Depth
	lights        *shaders.Lights
	shadowLight   *shaders.ShadowLight
	post          *shaders.Post
	bloom         *shaders.Bloom
	occlusion     *shaders.Occlusion
	additive      *shaders.Additive
	liquid        *shaders.Liquid
	deferred      *shaders.Deferred

	cal               *model.Calibration
	width             int32
	height            int32
	scaleX            float32
	scaleY            float32
	dynaLightMatrices []*float32
	spotlightMetrics  []*metrics.Spotlights
	dcOpaque          *DrawCommandsRender
	dcAdditive        *DrawCommandsRender
	dcLiquid          *DrawCommandsRender
	hwOcc             *OcclusionHW
}

// NewShaders initializes and returns a new instance of Shaders with default shader components and shadow settings.
func NewShaders(ctx api.IContext) *Shaders {
	c := &Shaders{
		ctx:               ctx,
		tex:               nil,
		main:              nil,
		sky:               nil,
		geometry:          nil,
		ssao:              nil,
		blur:              nil,
		depth:             nil,
		lights:            nil,
		shadowLight:       nil,
		post:              nil,
		bloom:             nil,
		enableShadows:     false,
		dynaLightMatrices: nil,
		spotlightMetrics:  nil,
	}
	return c
}

// Setup initializes shaders with the provided dimensions and strides, compiles them, and sets up vertex array buffers and samplers.
func (w *Shaders) Setup(vStride, lStride int32, p *model.ThingPlayer, cal *model.Calibration, tex *Textures, dcOpaque *DrawCommandsRender, dcAdditive *DrawCommandsRender, dcLiquid *DrawCommandsRender, hwOcc *OcclusionHW) error {
	w.ctx.Enable(api.MULTISAMPLE)
	//w.ctx.Enable(gl_api.SAMPLE_ALPHA_TO_COVERAGE)
	a := &Assets{}
	w.flash = p.GetFlash()
	w.tex = tex
	w.cal = cal
	w.dcOpaque = dcOpaque
	w.dcAdditive = dcAdditive
	w.dcLiquid = dcLiquid
	w.hwOcc = hwOcc

	w.handler = shaders.NewHandler(vStride, lStride)
	if err := w.handler.Setup(w.ctx, cal, p.GetFlash(), 0, 0); err != nil {
		return err
	}
	w.mapMetrics = w.handler.GetMapMetrics()
	w.shadowMetrics = w.handler.GetShadowMetrics()

	w.main = w.handler.GetMain()
	w.sky = w.handler.GetSky()
	w.geometry = w.handler.GetGeometry()
	w.ssao = w.handler.GetSSAO()
	w.blur = w.handler.GetBlur()
	w.depth = w.handler.GetDepth()
	w.lights = w.handler.GetLights()
	w.shadowLight = w.handler.GetShadowLight()
	w.post = w.handler.GetPost()
	w.bloom = w.handler.GetBloom()
	w.occlusion = w.handler.GetOcclusion()
	w.additive = w.handler.GetAdditive()
	w.liquid = w.handler.GetLiquid()

	w.enableShadows = false
	w.container = append(w.container, w.main, w.sky, w.geometry, w.ssao, w.blur, w.depth, w.lights, w.shadowLight, w.post, w.bloom, w.occlusion, w.additive, w.liquid)

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
	w.shadowLight.EnableShadows(w.enableShadows)
	w.lights.EnableShadows(w.enableShadows)
	w.depth.EnableShadows(w.enableShadows)
}

// Render handles the complete rendering pipeline, including geometry, lighting, post-processing, and optional sky rendering.
func (w *Shaders) Render(vi *model.ViewMatrix, fbW int32, fbH int32, winW int32, winH int32, vert []float32, vertLen int32, indices []uint32, indicesLen int32, skyEnabled bool, skyLayer, skyU, skyV float32, lights []float32, lightsNum int32, shadowLights [8]*Light, shadowLightsNum int32) {
	if w.handler.SetSize(fbW, fbH) {
		w.shadowMetrics.Rebuild(fbW, fbH)
		w.scaleX, w.scaleY = w.mapMetrics.GetScale3d(fbW, fbH, float32(w.cal.AspectRatio), float32(w.cal.FovVerticalDegrees))
	}

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

	px, _, pz := vi.GetView()
	w.mapMetrics.SetMapCenter(float32(px), float32(pz), w.mapMetrics.GetLightCamY())
	w.mapMetrics.Update(vi, w.scaleX, w.scaleY)

	//dirX, dirY, dirZ := vi.GetForwardVector()
	flashX, flashY, flashSensitivity := float32(0), float32(0), float32(0)
	if w.shadowLight.HasShadow() {
		swayX, swayY, swaySensitivity := vi.GetSway()
		flashX, flashY, flashSensitivity = float32(swayX), float32(swayY), float32(swaySensitivity)
	}

	w.shadowMetrics.Update(w.mapMetrics.GetMainViewPtr(), flashX, flashY)

	if int(shadowLightsNum) >= len(w.dynaLightMatrices) {
		dynaLightsLen := shadowLightsNum * 2
		w.dynaLightMatrices = make([]*float32, dynaLightsLen)
		w.spotlightMetrics = make([]*metrics.Spotlights, dynaLightsLen)
		for lx := int32(0); lx < dynaLightsLen; lx++ {
			w.spotlightMetrics[lx] = metrics.NewSpotlight()
		}
	}

	for lx := int32(0); lx < shadowLightsNum; lx++ {
		light := shadowLights[lx]
		pX, pY, pZ := light.X, light.Y, light.Z
		dX, dY, dZ := light.DirX, light.DirY, light.DirZ
		// Il FOV dell'ombra deve abbracciare interamente il CutOff del faretto
		// Se il faretto ha un outer cutoff di 40°, il FOV deve essere circa 80-90°
		//TODO FROM CONFIG
		fovDeg := float32(90.0)
		near := float32(0.1)
		factor := light.Intensity
		falloff := light.Falloff
		//TODO FROM CONFIG
		effectiveRadius := float32(4.605) * falloff * factor
		dynaMetrics := w.spotlightMetrics[lx]
		dynaMetrics.CreateSpace(pX, pY, pZ, dX, dY, dZ, fovDeg, near, effectiveRadius)
		w.dynaLightMatrices[lx] = dynaMetrics.GetSpacePtr()
	}

	//RENDER START HERE
	w.depth.Prepare(w.dynaLightMatrices, uint32(shadowLightsNum))
	// MAIN PREPARE (VBO che EBO)
	w.main.Prepare(vert, vertLen, indices, indicesLen)
	// LIGHTS PREPARE
	w.lights.Prepare(lights, lightsNum)
	// BIND TEXTURE
	w.bindTextureBuckets()
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
	if w.hwOcc != nil {
		w.occlusion.Render(func() { w.hwOcc.RenderQueries() })
	}
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
		w.liquid.Render(w.dcLiquid.Render, true)
	}
	// ENABLE ADDITIVE LIGHTS
	enableAdditiveLights(w.ctx)
	// FLASHLIGHTS
	w.shadowLight.Render(
		w.dcOpaque.Render, -1,
		flashX, flashY, 0.0,
		float32(w.flash.GetOffsetX())-(flashX*flashSensitivity),
		float32(w.flash.GetOffsetY())+(flashY*flashSensitivity),
		-1.0,
		float32(w.flash.GetIntensity()), float32(w.flash.GetFalloff()),
		float32(w.flash.GetConeStart()), float32(w.flash.GetConeEnd()))
	// DYNAMIC LIGHTS
	for lightIdx := int32(0); lightIdx < shadowLightsNum; lightIdx++ {
		light := shadowLights[lightIdx]
		w.shadowLight.Render(w.dcOpaque.Render, lightIdx,
			light.X, light.Y, light.Z,
			light.DirX, light.DirY, light.DirZ,
			light.Intensity, light.Falloff, light.OuterCutOff, light.CutOff)
	}
	// AMBIENT LIGHTS
	w.lights.Render(w.dcOpaque.Render, float32(vi.GetLightIntensity()))
	// DISABLE ADDITIVE LIGHTS
	disableAdditiveLights(w.ctx)
	w.bindTextureBuckets() // Restore texture arrays!
	// SKYBOX
	if skyEnabled {
		w.sky.Render(skyLayer, skyU, skyV)
	}
	// MSAA resolution
	w.post.Prepare()
	// BLOOM
	w.bloom.Render()
	// POST
	w.post.Render(winW, winH)
}

// ToggleShadows toggles the state of shadow rendering in the shader system.
func (w *Shaders) ToggleShadows() { w.SetShadowEnabled(!w.enableShadows) }

func (w *Shaders) bindTextureBuckets() {
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
