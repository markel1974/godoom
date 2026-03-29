package open_gl

import (
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
)

// IShader represents an interface for managing shader programs, including setup, sampler configuration, and compilation.
type IShader interface {
	Setup(width int32, height int32)
	SetupSamplers()
	Compile(a shaders.IAssets) error
}

// Shaders manages a collection of shader programs and related state for rendering, including lighting and post-processing.
type Shaders struct {
	main       *shaders.Main
	sky        *shaders.ShaderSky
	ssao       *shaders.SSAO
	blur       *shaders.Blur
	depth      *shaders.Depth
	lights     *shaders.Lights
	flashlight *shaders.Flashlight
	post       *shaders.Post
	bloom      *shaders.Bloom
	container  []IShader

	enableShadows bool
}

// NewShaders initializes and returns a new instance of Shaders with default container and settings.
func NewShaders() *Shaders {
	c := &Shaders{
		main:          shaders.NewMain(),
		sky:           shaders.NewShaderSky(),
		ssao:          shaders.NewSSAO(),
		blur:          shaders.NewBlur(),
		depth:         shaders.NewDepth(),
		lights:        shaders.NewLights(),
		flashlight:    shaders.NewShaderFlashlight(),
		post:          shaders.NewPost(),
		bloom:         shaders.NewBloom(),
		enableShadows: false,
	}
	c.container = append(c.container, c.main, c.sky, c.ssao, c.blur /*c.shaderGeometry,*/, c.depth, c.lights, c.flashlight, c.post, c.bloom)
	c.SetShadowEnabled(true)
	return c
}

// IncreaseFlashFactor increments the flashFactor field of the Shaders instance by 1.
func (w *Shaders) IncreaseFlashFactor() {
	w.flashlight.IncreaseFlashFactor()
}

// DecreaseFlashFactor reduces the flashFactor value by 1, ensuring it does not drop below 0.
func (w *Shaders) DecreaseFlashFactor() {
	w.flashlight.DecreaseFlashFactor()
}

// ToggleShadows toggles the shadow rendering state by inverting the current shadow-enabled flag.
func (w *Shaders) ToggleShadows() { w.SetShadowEnabled(!w.enableShadows) }

// SetShadowEnabled toggles the shadow rendering feature on or off based on the provided boolean value.
func (w *Shaders) SetShadowEnabled(v bool) {
	w.enableShadows = v
	w.flashlight.EnableShadows(w.enableShadows)
	w.lights.EnableShadows(w.enableShadows)
	w.depth.EnableShadows(w.enableShadows)
}

// Setup initializes and configures all container, VBOs, VAOs, and UBOs for rendering, and handles shader compilation and linking.
func (w *Shaders) Setup(width, height, vStride, lStride int32) error {
	a := &Assets{}
	for _, s := range w.container {
		s.Setup(width, height)
	}
	for _, s := range w.container {
		if err := s.Compile(a); err != nil {
			return err
		}
	}
	w.main.Init(vStride)

	w.lights.Init(lStride)

	for _, s := range w.container {
		s.SetupSamplers()
	}
	return nil
}

// Render renders the scene using the specified view matrix, framebuffer dimensions, vertex data, draw commands, and lighting settings.
func (w *Shaders) Render(vi *model.ViewMatrix, pX, pY float64, fbW int32, fbH int32, vert []float32, vertCount int32, dc []*DrawCommand, skyEnabled bool, skyTexId, skyNormalTexId, skyEmissiveTexId uint32, frameLights []float32, numLights int32) {
	renderGeometry := func() {
		var lastTexId, lastNormId, lastEmissiveId uint32 = math.MaxUint32, math.MaxUint32, math.MaxUint32
		for _, cmd := range dc {
			if cmd.vertexCount > 0 {
				if lastTexId != cmd.texId {
					gl.ActiveTexture(gl.TEXTURE0)
					gl.BindTexture(gl.TEXTURE_2D, cmd.texId)
					lastTexId = cmd.texId
				}
				if lastNormId != cmd.normTexId {
					gl.ActiveTexture(gl.TEXTURE1)
					gl.BindTexture(gl.TEXTURE_2D, cmd.normTexId)
					lastNormId = cmd.normTexId
				}
				if lastEmissiveId != cmd.emissiveTexId {
					gl.ActiveTexture(gl.TEXTURE5)
					gl.BindTexture(gl.TEXTURE_2D, cmd.emissiveTexId)
					lastEmissiveId = cmd.emissiveTexId
				}
				gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
			}
		}
	}

	bob := vi.GetBobPhase()
	swayX := w.flashlight.GetOffsetX(bob)
	swayY := w.flashlight.GetOffsetY(bob)
	roomSpaceMatrix, flashSpaceMatrix := shaders.CreateSpaces(vi, pX, pY, swayX, swayY)

	proj, view, invView := w.main.UpdateUniforms(vi)
	w.depth.UpdateUniforms(roomSpaceMatrix, flashSpaceMatrix)
	w.ssao.UpdateUniforms(view, proj)
	w.sky.UpdateUniforms(view, proj)

	gl.Viewport(0, 0, fbW, fbH)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	w.lights.Prepare(frameLights, numLights)

	w.main.Prepare(vert, vertCount)
	// OMBRE
	w.depth.Render(renderGeometry, w.main.GetVAO())
	// SSAO
	w.ssao.Render(w.blur.GetProgram(), w.main.GetVAO(), w.sky.GetVAO(), w.post.GetFBO(), skyEnabled)
	// MAIN
	w.main.Render(renderGeometry, w.ssao.GetSSAOBlurTexture())

	// PREPARE FOR ADDITIVE
	gl.DepthMask(false)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DepthFunc(gl.LEQUAL)

	// LIGHTS
	w.lights.Render(renderGeometry, w.depth.GetRoomShadowTextures(), view, proj, invView, roomSpaceMatrix, float32(vi.GetLightIntensity()), float32(fbW), float32(fbH))
	// FLASHLIGHTS
	w.flashlight.Render(renderGeometry, w.depth.GetFlashShadowTextures(), view, proj, invView, flashSpaceMatrix, float32(-vi.GetYaw()), swayX, swayY, float32(fbW), float32(fbH))

	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)

	// 4. SKYBOX
	w.sky.Render(skyTexId, skyNormalTexId, skyEnabled)

	// 5. POST
	bloomTex := w.bloom.Render(w.post.GetBrightBuffer())
	w.post.Render(bloomTex)
}
