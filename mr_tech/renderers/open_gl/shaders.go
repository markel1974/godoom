package open_gl

import (
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"

	shaders2 "github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
)

// IShader is an interface defining methods for setting up, compiling, and configuring GPU shader programs.
type IShader interface {
	Setup(width int32, height int32)

	SetupSamplers()

	Compile(a shaders2.IAssets) error
}

// Shaders manages multiple shader programs and associated configurations used in the rendering pipeline.
type Shaders struct {
	shaderMain     *shaders2.Main
	shaderSky      *shaders2.ShaderSky
	shaderSSAO     *shaders2.SSAO
	shaderBlur     *shaders2.Blur
	shaderGeometry *shaders2.ShaderGeometry
	shaderDepth    *shaders2.Depth
	shaderPost     *shaders2.Post
	shaders        []IShader

	flashFactor   float32
	flashOffsetX  float32
	flashOffsetY  float32
	enableShadows bool
}

// NewShaders initializes and returns a new Shaders instance with default shader configurations and settings.
func NewShaders() *Shaders {
	c := &Shaders{
		shaderMain:     shaders2.NewMain(),
		shaderSky:      shaders2.NewShaderSky(),
		shaderSSAO:     shaders2.NewSSAO(),
		shaderBlur:     shaders2.NewBlur(),
		shaderGeometry: shaders2.NewShaderGeometry(),
		shaderDepth:    shaders2.NewDepth(),
		shaderPost:     shaders2.NewPost(),
		flashFactor:    3.0,
		flashOffsetX:   0.0,
		flashOffsetY:   0.0,
		enableShadows:  false,
	}
	c.shaders = append(c.shaders, c.shaderMain, c.shaderSky, c.shaderSSAO, c.shaderBlur, c.shaderGeometry, c.shaderDepth, c.shaderPost)
	c.SetShadowEnabled(true)
	return c
}

// IncreaseFlashFactor increments the flashFactor property of the Shaders instance by 1.
func (w *Shaders) IncreaseFlashFactor() {
	w.flashFactor++
}

// DecreaseFlashFactor reduces the flashFactor of the Shaders instance by 1, ensuring it does not go below 0.
func (w *Shaders) DecreaseFlashFactor() {
	if w.flashFactor > 0 {
		w.flashFactor--
	}
}

// ToggleShadows toggles the shadow rendering feature by inverting the current shadow-enabled state of the Shaders instance.
func (w *Shaders) ToggleShadows() {
	w.SetShadowEnabled(!w.enableShadows)
}

// SetShadowEnabled enables or disables shadows and updates flash offset values based on the given boolean parameter.
func (w *Shaders) SetShadowEnabled(v bool) {
	w.enableShadows = v
	if w.enableShadows {
		w.flashOffsetX = 2.0
		w.flashOffsetY = -1.0
	} else {
		w.flashOffsetX = 0.0
		w.flashOffsetY = 0.0
	}
}

// Setup initializes shaders, compiles them with assets, configures vertex attributes, establishes default states, and sets up textures.
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

	// Location 0: aPos (vec3)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Location 1: aTexCoords (vec2)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	// Location 2: aLightIntensity (float)
	gl.VertexAttribPointer(2, 1, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)
	// Location 3: aLightCenterView (vec3)
	gl.VertexAttribPointer(3, 3, gl.FLOAT, false, stride, gl.PtrOffset(6*4))
	gl.EnableVertexAttribArray(3)
	// Location 4: aNormal (vec3)
	gl.VertexAttribPointer(4, 3, gl.FLOAT, false, stride, gl.PtrOffset(9*4))
	gl.EnableVertexAttribArray(4)
	// Restore default state
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	// --- SETUP FALLBACK NORMAL MAP (TEXTURE1) ---
	var defaultNormalMap uint32
	gl.GenTextures(1, &defaultNormalMap)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, defaultNormalMap)

	// Creazione pixel indaco piatto (Z-Up) per annullare perturbazioni vettoriali
	flatNormalPixel := []uint8{128, 128, 255, 255}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(flatNormalPixel))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	// --- RIPRISTINO FLUSSO STANDARD (TEXTURE0) ---
	gl.ActiveTexture(gl.TEXTURE0)
	// I parametri qui sotto ora si applicano correttamente alla TEXTURE0 di default
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	for _, s := range w.shaders {
		s.SetupSamplers()
	}
	return nil
}

// Render executes the rendering pipeline for the scene and sky, handling input matrices, view configs, and draw commands.
func (w *Shaders) Render(vi *model.ViewMatrix, pX, pY float64, fbW int32, fbH int32, vert []float32, vertCount int, dc []*DrawCommand, skyEnabled bool, skyTexId, skyNormalTexId uint32) {
	gl.Viewport(0, 0, fbW, fbH)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	renderMain := func() {
		gl.BindVertexArray(w.shaderMain.GetVao())
		var lastTexId uint32 = math.MaxUint32
		for _, cmd := range dc {
			if cmd.vertexCount > 0 {
				if lastTexId != cmd.texId {
					gl.ActiveTexture(gl.TEXTURE0)
					gl.BindTexture(gl.TEXTURE_2D, cmd.texId)
					lastTexId = cmd.texId
				}
				gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
			}
		}
	}

	renderSky := func() {
		gl.BindVertexArray(w.shaderSky.GetVao())
		gl.Disable(gl.DEPTH_TEST)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
		gl.Enable(gl.DEPTH_TEST)
	}

	// Aggiornamento Stato (CPU)
	roomSpaceMatrix, flashSpaceMatrix := CreateSpaces(vi, pX, pY, w.flashOffsetX, w.flashOffsetY)
	proj, view := w.shaderMain.UpdateUniforms(vi, roomSpaceMatrix, flashSpaceMatrix, w.flashFactor, w.enableShadows, w.flashOffsetX, w.flashOffsetY)
	w.shaderDepth.UpdateUniforms(roomSpaceMatrix, flashSpaceMatrix)
	w.shaderGeometry.UpdateUniforms(view, proj)
	w.shaderSSAO.UpdateUniforms(view, proj)
	w.shaderSky.UpdateUniforms(view, proj)

	// Render
	w.shaderMain.Prepare(vert, vertCount)
	var roomShadowTex, flashShadowTex uint32
	if w.enableShadows {
		roomShadowTex, flashShadowTex = w.shaderDepth.GetShadowTextures()
		w.shaderDepth.Render(renderMain)
	}
	w.shaderSSAO.Prepare()
	w.shaderGeometry.Render(renderMain)
	w.shaderSSAO.Render(renderSky, w.shaderBlur.GetProgram())

	// Dirotta tutto il rendering 3D (scena + cielo) nel buffer HDR a 16-bit
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.shaderPost.GetFBO())
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	// ------------------------------

	w.shaderMain.Render(roomShadowTex, flashShadowTex, w.shaderSSAO.GetSSAOBlurTexture())

	var lastTexId uint32 = math.MaxUint32
	var lastNormId uint32 = math.MaxUint32
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
			gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
		}
	}

	if skyEnabled {
		w.shaderSky.Render(skyTexId, skyNormalTexId)
	}

	// PASS FINALE: Tonemapping & Color Grading a schermo
	w.shaderPost.Render()
}
