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
	w.shaderFlashlight.EnableFlash(w.enableShadows)
}

// Setup initializes and configures all shaders, VBOs, VAOs, and UBOs for rendering, and handles shader compilation and linking.
func (w *Shaders) Setup(width, height, stride int32) error {
	a := &Assets{}
	for _, s := range w.shaders {
		s.Setup(width, height)
	}
	for _, s := range w.shaders {
		if err := s.Compile(a); err != nil {
			return err
		}
	}
	w.shaderMain.Init()

	// VBO A 8 FLOAT
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	w.shaderLights.Init()

	for _, s := range w.shaders {
		s.SetupSamplers()
	}
	return nil
}

// Render renders the scene using the specified view matrix, framebuffer dimensions, vertex data, draw commands, and lighting settings.
func (w *Shaders) Render(vi *model.ViewMatrix, pX, pY float64, fbW int32, fbH int32, vert []float32, vertCount int32, dc []*DrawCommand, skyEnabled bool, skyTexId, skyNormalTexId, skyEmissiveTexId uint32, frameLights []float32, numLights int32) {
	gl.Viewport(0, 0, fbW, fbH)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	drawScreenQuad := func() {
		gl.BindVertexArray(w.shaderSky.GetVao())
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	}

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

	// UPLOAD UBO LIGHTS
	w.shaderLights.Prepare(frameLights, numLights)

	bob := vi.GetBobPhase()
	swayX := w.shaderFlashlight.GetOffsetX()
	swayY := w.shaderFlashlight.GetOffsetY()

	if w.enableShadows {
		swayX += float32(math.Cos(bob*0.5) * 1.1)
		swayY -= float32(math.Abs(math.Sin(bob)) * 1.2)
	}

	roomSpaceMatrix, flashSpaceMatrix := shaders.CreateSpaces(vi, pX, pY, swayX, swayY)

	// Per il main update usiamo la vecchia firma temporanea, in seguito ripuliremo shaders/main.go per non prendere parametri torcia.
	proj, view := w.shaderMain.UpdateUniforms(vi)

	var invView [16]float32
	if inv, ok := shaders.MatrixInverse4x4(view); ok {
		invView = inv
	}

	w.shaderDepth.UpdateUniforms(roomSpaceMatrix, flashSpaceMatrix)
	//w.shaderGeometry.UpdateUniforms(view, proj)
	w.shaderSSAO.UpdateUniforms(view, proj)
	w.shaderSky.UpdateUniforms(view, proj)

	w.shaderMain.Prepare(vert, vertCount)

	// 1. OMBRE
	var roomShadowTex, flashShadowTex uint32
	if w.enableShadows {
		roomShadowTex, flashShadowTex = w.shaderDepth.GetShadowTextures()
		gl.BindVertexArray(w.shaderMain.GetVao())
		w.shaderDepth.Render(renderGeometry)
	}

	// 2. SSAO PRE-PASS
	w.shaderSSAO.Prepare()
	gl.BindVertexArray(w.shaderMain.GetVao())
	//w.shaderGeometry.Render(renderGeometry)
	w.shaderSSAO.Render(drawScreenQuad, w.shaderBlur.GetProgram())

	// 3. FORWARD MULTI-PASS
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.shaderPost.GetFBO())
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.BindVertexArray(w.shaderMain.GetVao())

	// PASS A: BASE
	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(true)
	w.shaderMain.Render(w.shaderSSAO.GetSSAOBlurTexture())
	renderGeometry()

	// PREPARE FOR ADDITIVE
	gl.DepthMask(false)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DepthFunc(gl.LEQUAL)

	ambientLight := float32(vi.GetLightIntensity())
	enableShadowsInt := int32(0)
	if w.enableShadows {
		enableShadowsInt = 1
	}

	// PASS B: LUCI AMBIENTALI E UBO
	w.shaderLights.Render(view, proj, invView, roomSpaceMatrix, ambientLight, enableShadowsInt, float32(fbW), float32(fbH))
	if w.enableShadows {
		gl.ActiveTexture(gl.TEXTURE3)
		gl.BindTexture(gl.TEXTURE_2D, roomShadowTex)
	}
	renderGeometry()

	// PASS C: TORCIA
	if w.shaderFlashlight.GetFactor() > 0 {
		pitchShear := float32(-vi.GetYaw())

		w.shaderFlashlight.Render(view, proj, invView, flashSpaceMatrix, pitchShear, swayX, swayY, float32(fbW), float32(fbH))
		if w.enableShadows {
			gl.ActiveTexture(gl.TEXTURE4)
			gl.BindTexture(gl.TEXTURE_2D, flashShadowTex)
		}
		renderGeometry()
	}

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
