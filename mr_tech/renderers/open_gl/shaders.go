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
	shaderMain *shaders.Main
	shaderSky  *shaders.ShaderSky
	shaderSSAO *shaders.SSAO
	shaderBlur *shaders.Blur
	//shaderGeometry   *shaders.ShaderGeometry
	shaderDepth      *shaders.Depth
	shaderLights     *shaders.Lights
	shaderFlashlight *shaders.Flashlight
	shaderPost       *shaders.Post
	shaderBloom      *shaders.Bloom
	shaders          []IShader

	enableShadows bool
}

// NewShaders initializes and returns a new instance of Shaders with default shaders and settings.
func NewShaders() *Shaders {
	c := &Shaders{
		shaderMain: shaders.NewMain(),
		shaderSky:  shaders.NewShaderSky(),
		shaderSSAO: shaders.NewSSAO(),
		shaderBlur: shaders.NewBlur(),
		//shaderGeometry:   shaders.NewShaderGeometry(),
		shaderDepth:      shaders.NewDepth(),
		shaderLights:     shaders.NewLights(),
		shaderFlashlight: shaders.NewShaderFlashlight(),
		shaderPost:       shaders.NewPost(),
		shaderBloom:      shaders.NewBloom(),
		enableShadows:    false,
	}
	c.shaders = append(c.shaders, c.shaderMain, c.shaderSky, c.shaderSSAO, c.shaderBlur /*c.shaderGeometry,*/, c.shaderDepth, c.shaderLights, c.shaderFlashlight, c.shaderPost, c.shaderBloom)
	c.SetShadowEnabled(true)
	return c
}

// IncreaseFlashFactor increments the flashFactor field of the Shaders instance by 1.
func (w *Shaders) IncreaseFlashFactor() {
	w.shaderFlashlight.IncreaseFlashFactor()
}

// DecreaseFlashFactor reduces the flashFactor value by 1, ensuring it does not drop below 0.
func (w *Shaders) DecreaseFlashFactor() {
	w.shaderFlashlight.DecreaseFlashFactor()
}

// ToggleShadows toggles the shadow rendering state by inverting the current shadow-enabled flag.
func (w *Shaders) ToggleShadows() { w.SetShadowEnabled(!w.enableShadows) }

// SetShadowEnabled toggles the shadow rendering feature on or off based on the provided boolean value.
func (w *Shaders) SetShadowEnabled(v bool) {
	w.enableShadows = v
	w.shaderFlashlight.EnableShadows(w.enableShadows)
	w.shaderLights.EnableShadows(w.enableShadows)
	w.shaderDepth.EnableShadows(w.enableShadows)
}

// Setup initializes and configures all shaders, VBOs, VAOs, and UBOs for rendering, and handles shader compilation and linking.
func (w *Shaders) Setup(width, height, vStride, lStride int32) error {
	a := &Assets{}
	for _, s := range w.shaders {
		s.Setup(width, height)
	}
	for _, s := range w.shaders {
		if err := s.Compile(a); err != nil {
			return err
		}
	}
	w.shaderMain.Init(vStride)

	w.shaderLights.Init(lStride)

	for _, s := range w.shaders {
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
	swayX := w.shaderFlashlight.GetOffsetX(bob)
	swayY := w.shaderFlashlight.GetOffsetY(bob)
	roomSpaceMatrix, flashSpaceMatrix := shaders.CreateSpaces(vi, pX, pY, swayX, swayY)

	proj, view, invView := w.shaderMain.UpdateUniforms(vi)
	w.shaderDepth.UpdateUniforms(roomSpaceMatrix, flashSpaceMatrix)
	w.shaderSSAO.UpdateUniforms(view, proj)
	w.shaderSky.UpdateUniforms(view, proj)

	gl.Viewport(0, 0, fbW, fbH)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	w.shaderLights.Prepare(frameLights, numLights)

	w.shaderMain.Prepare(vert, vertCount)

	// 1. OMBRE
	w.shaderDepth.Render(renderGeometry, w.shaderMain.GetVAO())

	// 2. SSAO
	w.shaderSSAO.Render(w.shaderBlur.GetProgram(), w.shaderMain.GetVAO(), w.shaderSky.GetVAO(), w.shaderPost.GetFBO())

	// 3. MAIN
	w.shaderMain.Render(renderGeometry, w.shaderSSAO.GetSSAOBlurTexture())

	// PREPARE FOR ADDITIVE
	gl.DepthMask(false)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DepthFunc(gl.LEQUAL)

	// PASS B: LUCI AMBIENTALI E UBO
	ambientLight := float32(vi.GetLightIntensity())
	roomShadowTex := w.shaderDepth.GetRoomShadowTextures()
	w.shaderLights.Render(renderGeometry, roomShadowTex, view, proj, invView, roomSpaceMatrix, ambientLight, float32(fbW), float32(fbH))

	// PASS C: TORCIA
	pitchShear := float32(-vi.GetYaw())
	flashShadowTex := w.shaderDepth.GetFlashShadowTextures()
	w.shaderFlashlight.Render(renderGeometry, flashShadowTex, view, proj, invView, flashSpaceMatrix, pitchShear, swayX, swayY, float32(fbW), float32(fbH))

	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)

	// 4. SKYBOX
	if skyEnabled {
		w.shaderSky.Render(skyTexId, skyNormalTexId)
	}

	// 5. POST
	bloomTex := w.shaderBloom.Render(w.shaderPost.GetBrightBuffer())
	w.shaderPost.Render(bloomTex)
}
