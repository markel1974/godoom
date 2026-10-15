package shaders

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

// Handler manages rendering components, graphical settings, metrics, and the overall rendering pipeline configuration.
type Handler struct {
	ctx api.IContext
	cal *model.Calibration

	fbW int32
	fbH int32

	mapMetrics *metrics.Map

	flash               *model.Flash
	flashlightMetrics   *metrics.Flashlight
	shadowLightsMetrics *metrics.ShadowLights

	main        *Main
	sky         *Sky
	geometry    *Geometry
	ssao        *SSAO
	blur        *Blur
	depth       *Depth
	lights      *Lights
	shadowLight *ShadowLight
	post        *Post
	bloom       *Bloom
	occlusion   *Occlusion
	additive    *Additive
	liquid      *Liquid
	deferred    *Deferred

	emissiveIntensity float32
	aoFactor          float32
	exposure          float32
	contrast          float32
	saturation        float32
	bloomIntensity    float32
	bloomBlur         int32
	kernelSize        int32
	noiseTextureSize  int32
	radius            float32
	bias              float32
	hvPassages        int32
	internalPassages  int32
}

// NewHandler initializes and returns a new instance of Handler with the specified vertex and light stride values.
func NewHandler(vStride, lStride int32) *Handler {
	return &Handler{
		//TODO Dynamic
		emissiveIntensity: 4.0,
		aoFactor:          0.8,

		exposure:       0.1,
		contrast:       1.05,
		saturation:     1.0,
		bloomIntensity: 0.05,
		bloomBlur:      1,

		kernelSize:       64, //GPU ITERATION
		noiseTextureSize: 4 * 4,
		radius:           12.0,
		bias:             0.025,

		hvPassages:       5,
		internalPassages: 3, //GPU ITERATION

		additive:    NewAdditive(),
		bloom:       NewBloom(), //TODO passaggi orizzontali e verticali dalla configurazione
		blur:        NewBlur(),
		deferred:    NewDeferred(lStride),
		depth:       NewDepth(8), //TODO numero di shadowLights dalla configurazione
		geometry:    NewGeometry(),
		lights:      NewLights(lStride),
		liquid:      NewLiquid(),
		main:        NewMain(vStride),
		occlusion:   NewOcclusion(),
		post:        NewPost(),
		shadowLight: NewShadowLight(),
		sky:         NewSky(),
		ssao:        NewSSAO(),
	}
}

// Setup initializes the Handler by configuring context, calibration, flash, and various metrics and components.
func (h *Handler) Setup(ctx api.IContext, cal *model.Calibration, flash *model.Flash, fbW, fbH int32) error {
	h.ctx = ctx
	h.cal = cal
	h.flash = flash
	h.fbW = fbW
	h.fbH = fbH

	h.mapMetrics = metrics.NewMap()
	h.mapMetrics.SetOrthoSize(float32(h.cal.OrthoSize), float32(h.cal.ZNearRoom), float32(h.cal.ZFarRoom)+4.0)
	h.mapMetrics.SetCenter(float32(h.cal.MapCenterX), float32(h.cal.MapCenterZ), float32(h.cal.LightCamY)+2.0)
	h.flashlightMetrics = metrics.NewFlashlight(h.flash)
	h.shadowLightsMetrics = metrics.NewShadowLights()

	if err := h.additive.Setup(h); err != nil {
		return err
	}
	if err := h.bloom.Setup(h); err != nil {
		return err
	}
	if err := h.blur.Setup(h); err != nil {
		return err
	}
	if err := h.deferred.Setup(h); err != nil {
		return err
	}
	if err := h.depth.Setup(h); err != nil {
		return err
	}
	if err := h.geometry.Setup(h); err != nil {
		return err
	}
	if err := h.lights.Setup(h); err != nil {
		return err
	}
	if err := h.liquid.Setup(h); err != nil {
		return err
	}
	if err := h.main.Setup(h); err != nil {
		return err
	}
	if err := h.occlusion.Setup(h); err != nil {
		return err
	}
	if err := h.post.Setup(h); err != nil {
		return err
	}
	if err := h.shadowLight.Setup(h); err != nil {
		return err
	}
	if err := h.sky.Setup(h); err != nil {
		return err
	}
	if err := h.ssao.Setup(h); err != nil {
		return err
	}
	return nil
}

// GetContext retrieves the IContext instance associated with the Handler.
func (h *Handler) GetContext() api.IContext {
	return h.ctx
}

// GetCalibration retrieves the calibration settings associated with the Handler, returning a pointer to a Calibration object.
func (h *Handler) GetCalibration() *model.Calibration {
	return h.cal
}

// GetMapMetrics returns the map metrics associated with the Handler.
func (h *Handler) GetMapMetrics() *metrics.Map {
	return h.mapMetrics
}

// GetFlashlightMetrics retrieves the shadow metrics associated with the Handler.
func (h *Handler) GetFlashlightMetrics() *metrics.Flashlight {
	return h.flashlightMetrics
}

// GetShadowLightsMetrics retrieves the shadow light metrics associated with the Handler.
func (h *Handler) GetShadowLightsMetrics() *metrics.ShadowLights {
	return h.shadowLightsMetrics
}

// GetFlash retrieves the Flash instance associated with the Handler, providing access to its rendering and shading parameters.
func (h *Handler) GetFlash() *model.Flash {
	return h.flash
}

// GetMain returns the Main instance associated with the Handler.
func (h *Handler) GetMain() *Main {
	return h.main
}

// GetSky retrieves the Sky instance associated with the Handler.
func (h *Handler) GetSky() *Sky {
	return h.sky
}

// GetGeometry retrieves the Geometry instance configured for the Handler, used for geometry rendering in the graphics pipeline.
func (h *Handler) GetGeometry() *Geometry {
	return h.geometry
}

// GetSSAO retrieves the SSAO (Screen Space Ambient Occlusion) instance associated with the handler.
func (h *Handler) GetSSAO() *SSAO {
	return h.ssao
}

// GetBlur retrieves the Blur instance used for performing blur effects in the rendering pipeline.
func (h *Handler) GetBlur() *Blur {
	return h.blur
}

// GetDepth retrieves the depth object managed by the Handler.
func (h *Handler) GetDepth() *Depth {
	return h.depth
}

// GetLights retrieves the Lights instance associated with the Handler.
func (h *Handler) GetLights() *Lights {
	return h.lights
}

// GetShadowLight returns the ShadowLight instance associated with the Handler.
func (h *Handler) GetShadowLight() *ShadowLight {
	return h.shadowLight
}

// GetPost retrieves the Post object associated with the Handler.
func (h *Handler) GetPost() *Post {
	return h.post
}

// GetBloom returns the Bloom instance associated with the Handler.
func (h *Handler) GetBloom() *Bloom {
	return h.bloom
}

// GetOcclusion retrieves the Occlusion instance associated with the Handler.
func (h *Handler) GetOcclusion() *Occlusion {
	return h.occlusion
}

// GetAdditive returns the Additive instance associated with the Handler. It handles additive blending in rendering.
func (h *Handler) GetAdditive() *Additive {
	return h.additive
}

// GetLiquid returns the Liquid instance associated with the Handler, used for rendering liquid effects in a 3D context.
func (h *Handler) GetLiquid() *Liquid {
	return h.liquid
}

// GetDeferred returns the Deferred instance used for managing deferred rendering operations in the Handler.
func (h *Handler) GetDeferred() *Deferred {
	return h.deferred
}

// GetEmissiveIntensity returns the current emissive intensity value of the Handler.
func (h *Handler) GetEmissiveIntensity() float32 {
	return h.emissiveIntensity
}

// GetAoFactor returns the ambient occlusion factor, which influences the intensity of ambient occlusion effects.
func (h *Handler) GetAoFactor() float32 {
	return h.aoFactor
}

// GetExposure returns the current exposure value used in rendering calculations.
func (h *Handler) GetExposure() float32 {
	return h.exposure
}

// GetContrast returns the contrast value of the Handler.
func (h *Handler) GetContrast() float32 {
	return h.contrast
}

// GetSaturation retrieves the current saturation value used in the rendering process.
func (h *Handler) GetSaturation() float32 {
	return h.saturation
}

// GetBloomIntensity returns the current bloom intensity value as a float32.
func (h *Handler) GetBloomIntensity() float32 {
	return h.bloomIntensity
}

// GetBloomBlur retrieves the current level of blur applied to bloom effects as an integer value.
func (h *Handler) GetBloomBlur() int32 {
	return h.bloomBlur
}

// GetKernelSize returns the kernel size used for rendering operations within the handler.
func (h *Handler) GetKernelSize() int32 {
	return h.kernelSize
}

// GetNoiseTextureSize returns the size of the noise texture used in rendering.
func (h *Handler) GetNoiseTextureSize() int32 {
	return h.noiseTextureSize
}

// GetRadius returns the radius value associated with the Handler instance.
func (h *Handler) GetRadius() float32 {
	return h.radius
}

// GetBias returns the bias value used for certain rendering computations in the Handler.
func (h *Handler) GetBias() float32 {
	return h.bias
}

// GetHvPassages retrieves the hvPassage value associated with the Handler instance.
func (h *Handler) GetHvPassages() int32 {
	return h.hvPassages
}

// GetInternalPassages returns the value of the internalPassage field from the Handler.
func (h *Handler) GetInternalPassages() int32 {
	return h.internalPassages
}

// GetSize retrieves the framebuffer width and height as two int32 values.
func (h *Handler) GetSize() (int32, int32) {
	return h.fbW, h.fbH
}

// SetSize updates the framebuffer width and height if they differ from the current values and returns true on change.
func (h *Handler) SetSize(fbW, fbH int32) bool {
	if fbW != h.fbW || fbH != h.fbH {
		h.fbW = fbW
		h.fbH = fbH
		return true
	}
	return false
}
