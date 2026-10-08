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
	ctx               api.IContext
	tex               *Textures
	flash             *model.Flash
	main              *shaders.Main
	sky               *shaders.Sky
	geometry          *shaders.Geometry
	ssao              *shaders.SSAO
	blur              *shaders.Blur
	depth             *shaders.Depth
	lights            *shaders.Lights
	shadowLight       *shaders.ShadowLight
	post              *shaders.Post
	bloom             *shaders.Bloom
	occlusion         *shaders.Occlusion
	additive          *shaders.Additive
	liquid            *shaders.Liquid
	container         []IShader
	enableShadows     bool
	mapMetrics        *metrics.Map
	shadowMetrics     *metrics.Shadows
	cal               *model.Calibration
	w                 int32
	h                 int32
	scaleX            float32
	scaleY            float32
	dynaLightMatrices []*float32
	dynaLightMetrics  []*metrics.Spotlights
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
		dynaLightMetrics:  nil,
	}
	return c
}

// Setup initializes shaders with the provided dimensions and strides, compiles them, and sets up vertex array buffers and samplers.
func (w *Shaders) Setup(vStride, lStride int32, p *model.ThingPlayer, cal *model.Calibration, tex *Textures) error {
	w.ctx.Enable(api.MULTISAMPLE)
	//w.ctx.Enable(gl_api.SAMPLE_ALPHA_TO_COVERAGE)
	a := &Assets{}
	w.flash = p.GetFlash()
	w.tex = tex
	w.cal = cal
	w.mapMetrics = metrics.NewMap()
	w.mapMetrics.SetOrthoSize(float32(w.cal.OrthoSize), float32(w.cal.ZNearRoom), float32(w.cal.ZFarRoom)+4.0)
	w.mapMetrics.SetMapCenter(float32(w.cal.MapCenterX), float32(w.cal.MapCenterZ), float32(w.cal.LightCamY)+2.0)
	w.shadowMetrics = metrics.NewShadows(w.flash)

	w.main = shaders.NewMain(w.ctx, vStride, w.mapMetrics)
	w.sky = shaders.NewSky(w.ctx)
	w.geometry = shaders.NewGeometry(w.ctx)
	w.ssao = shaders.NewSSAO(w.ctx)
	w.blur = shaders.NewBlur(w.ctx)
	w.depth = shaders.NewDepth(w.ctx, w.mapMetrics, w.shadowMetrics, 8)
	w.lights = shaders.NewLights(w.ctx, lStride, w.cal)
	w.shadowLight = shaders.NewShaderShadowLight(w.ctx, w.cal)
	w.post = shaders.NewPost(w.ctx)
	w.bloom = shaders.NewBloom(w.ctx)
	w.occlusion = shaders.NewOcclusion(w.ctx)
	w.additive = shaders.NewAdditive(w.ctx)
	w.liquid = shaders.NewLiquid(w.ctx)
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
func (w *Shaders) Render(dcOpaque *DrawCommandsRender, dcAdditive *DrawCommandsRender, dcLiquid *DrawCommandsRender, hwOcc *OcclusionHW, vi *model.ViewMatrix, fbW int32, fbH int32, vert []float32, vertLen int32, indices []uint32, indicesLen int32, skyEnabled bool, skyLayer, skyU, skyV float32, lights []float32, lightsNum int32, shadowLights [8]*Light, shadowLightsNum int32) {
	if (w.w != fbW) || (w.h != fbH) {
		w.w = fbW
		w.h = fbH
		w.shadowMetrics.Rebuild(w.w, w.h)
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
	projPtr := w.mapMetrics.GetProjPtr()
	viewPtr := w.mapMetrics.GetViewPtr()
	invViewPtr := w.mapMetrics.GetInvViewPtr()

	//dirX, dirY, dirZ := vi.GetForwardVector()
	flashX, flashY, flashSensitivity := float32(0), float32(0), float32(0)
	if w.shadowLight.HasShadow() {
		swayX, swayY, swaySensitivity := vi.GetSway()
		flashX, flashY, flashSensitivity = float32(swayX), float32(swayY), float32(swaySensitivity)
	}
	flashDirX := float32(w.flash.GetOffsetX()) - (flashX * flashSensitivity)
	flashDirY := float32(w.flash.GetOffsetY()) + (flashY * flashSensitivity)
	flashTex := w.depth.GetFlashShadowTextures()
	roomTex := w.depth.GetRoomShadowTextures()

	w.shadowMetrics.Update(w.mapMetrics.GetMainViewPtr(), flashX, flashY)

	if int(shadowLightsNum) >= len(w.dynaLightMatrices) {
		dynaLightsLen := shadowLightsNum * 2
		w.dynaLightMatrices = make([]*float32, dynaLightsLen)
		w.dynaLightMetrics = make([]*metrics.Spotlights, dynaLightsLen)
		for lx := int32(0); lx < dynaLightsLen; lx++ {
			w.dynaLightMetrics[lx] = metrics.NewSpotlight()
		}
	}

	for lx := int32(0); lx < shadowLightsNum; lx++ {
		light := shadowLights[lx]
		pX, pY, pZ := light.X, light.Y, light.Z
		dX, dY, dZ := light.DirX, light.DirY, light.DirZ
		// Il FOV dell'ombra deve abbracciare interamente il CutOff del faretto
		// Se il faretto ha un outer cutoff di 40°, il FOV deve essere circa 80-90°
		fovDeg := float32(90.0)
		near := float32(0.1)
		factor := light.Intensity
		falloff := light.Falloff
		//TODO FROM CONFIG
		effectiveRadius := float32(4.605) * falloff * factor
		//TODO WRONG
		if effectiveRadius < 256.0 {
			effectiveRadius = 256.0
		}
		dynaMetrics := w.dynaLightMetrics[lx]
		dynaMetrics.CreateSpotLightSpace(pX, pY, pZ, dX, dY, dZ, fovDeg, near, effectiveRadius)
		w.dynaLightMatrices[lx] = dynaMetrics.GetSpotLightSpacePtr()
	}

	w.depth.UpdateUniforms(w.mapMetrics.GetRoomSpacePtr(), w.shadowMetrics.GetShadowSpacePtr(), w.mapMetrics.GetMainViewPtr(), w.dynaLightMatrices, uint32(shadowLightsNum))
	w.geometry.UpdateUniforms(viewPtr, projPtr)
	w.ssao.UpdateUniforms(viewPtr, projPtr)
	w.sky.UpdateUniforms(viewPtr, projPtr)

	// MAIN PREPARE (VBO che EBO)
	w.main.Prepare(vert, vertLen, indices, indicesLen, fbW, fbH)
	// LIGHTS PREPARE
	w.lights.Prepare(lights, lightsNum)

	w.bindTextureBuckets()

	// OMBRE
	w.depth.Render(dcOpaque.Render, w.main.GetVAO(), fbW, fbH)
	// SSAO PREPARE
	w.ssao.Prepare(fbW, fbH)
	// GEOMETRY
	w.geometry.Render(dcOpaque.Render)
	// SSAO
	w.ssao.Render(w.blur.GetProgram(), w.main.GetVAO(), w.sky.GetVAO(), w.post.GetFBO(), skyEnabled)
	// MAIN OPAQUE
	w.main.Render(dcOpaque.Render, viewPtr, projPtr, w.ssao.GetSSAOBlurTexture(), w.post.GetFBO(), fbW, fbH)

	// HW OCCLUSION
	if hwOcc != nil {
		w.occlusion.Render(func() { hwOcc.RenderQueries() }, viewPtr, projPtr)
	}
	// MAIN ADDITIVE
	if dcAdditive.HasCommands() {
		w.additive.RenderAdditive(dcAdditive.Render, w.main.GetVAO(), viewPtr, projPtr)
	}

	// MAIN LIQUID (Refraction & Depth Fog)
	if dcLiquid.HasCommands() {
		// OPTIMIZATION: We DO NOT call w.post.ResolveMSAA here.
		// Sampling the PREVIOUS FRAME's resolved FBO for refraction and depth fog
		// saves a massive 3x fullscreen MSAA blit (cutting ResolveMSAA time in half).
		// The 1-frame lag (16ms) is totally imperceptible through the distortion.
		w.liquid.Render(dcLiquid.Render, w.main.GetVAO(), w.post.GetColorBuffer(), w.post.GetDepthBuffer(), true, fbW, fbH, viewPtr, projPtr)
	}
	// ENABLE ADDITIVE LIGHTS
	enableAdditiveLights(w.ctx)

	// FLASHLIGHTS
	fConeStart := float32(w.flash.GetConeStart())
	fConeEnd := float32(w.flash.GetConeEnd())
	w.shadowLight.Render(
		dcOpaque.Render, w.main.GetVAO(), flashTex, viewPtr, projPtr, invViewPtr, w.shadowMetrics.GetShadowSpacePtr(),
		0, flashX, flashY, 0.0,
		flashDirX, flashDirY, -1.0,
		float32(w.flash.GetIntensity()), float32(w.flash.GetFalloff()), fConeStart, fConeEnd, float32(fbW), float32(fbH))

	// DYNAMIC LIGHTS
	for lx := int32(0); lx < shadowLightsNum; lx++ {
		light := shadowLights[lx]
		lTex, _, lMatrix := w.depth.GetShadowLightTextures(uint32(lx))
		factor := light.Intensity
		falloff := light.Falloff
		wPosX, wPosY, wPosZ := light.X, light.Y, light.Z
		wDirX, wDirY, wDirZ := light.DirX, light.DirY, light.DirZ
		//CutOff 0.7, OuterCutOff 0.9
		w.shadowLight.Render(
			dcOpaque.Render, w.main.GetVAO(), lTex, viewPtr, projPtr, invViewPtr, lMatrix,
			1, wPosX, wPosY, wPosZ,
			wDirX, wDirY, wDirZ,
			factor, falloff, light.OuterCutOff, light.CutOff, float32(fbW), float32(fbH),
		)
	}
	// LIGHTS
	w.lights.Render(dcOpaque.Render, w.main.GetVAO(), roomTex, viewPtr, projPtr, invViewPtr, w.mapMetrics.GetRoomSpacePtr(), float32(vi.GetLightIntensity()), float32(fbW), float32(fbH))

	// DISABLE ADDITIVE LIGHTS
	disableAdditiveLights(w.ctx)

	w.bindTextureBuckets() // Restore texture arrays!

	// SKYBOX
	w.sky.Render(skyLayer, skyEnabled, skyU, skyV)
	// MSAA resolution
	w.post.Prepare(fbW, fbH)
	// BLOOM
	w.bloom.Render(w.post.GetBrightBuffer(), fbW, fbH)
	// POST
	w.post.Render(w.bloom.GetBloomTexture(), fbW, fbH)
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
