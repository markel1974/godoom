package open_gl

import (
	"fmt"
	"math"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/markel1974/godoom/mr_tech/model"
)

// IShader is an interface defining methods for setting up, compiling, and configuring GPU shader programs.
type IShader interface {
	Setup(width int32, height int32)

	SetupSamplers()

	Compile(a IAssets) error
}

// ShaderCompile compiles a shader from source code and returns the shader ID or an error if compilation fails.
func ShaderCompile(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	cSources, free := gl.Strs(source + "\x00")
	gl.ShaderSource(shader, 1, cSources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}
	return shader, nil
}

// ShaderCreateProgram links a vertex and fragment shader into a shader program, validates it, and returns the program ID.
func ShaderCreateProgram(vertexShader uint32, fragmentShader uint32) (uint32, error) {
	shaderProgram := gl.CreateProgram()
	gl.AttachShader(shaderProgram, vertexShader)
	gl.AttachShader(shaderProgram, fragmentShader)
	gl.LinkProgram(shaderProgram)
	var status int32
	gl.GetProgramiv(shaderProgram, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		return 0, fmt.Errorf("failed to link shader prg")
	}
	gl.UseProgram(shaderProgram)
	gl.DeleteShader(fragmentShader)
	gl.DeleteShader(vertexShader)

	return shaderProgram, nil
}

// Shaders manages multiple shader programs and associated configurations used in the rendering pipeline.
type Shaders struct {
	shaderMain     *ShaderMain
	shaderSky      *ShaderSky
	shaderSSAO     *ShaderSSAO
	shaderBlur     *ShaderBlur
	shaderGeometry *ShaderGeometry
	shaderDepth    *ShaderDepth
	shaderPost     *ShaderPost
	shaders        []IShader

	flashFactor   float32
	flashOffsetX  float32
	flashOffsetY  float32
	enableShadows bool
}

// NewShaders initializes and returns a new Shaders instance with default shader configurations and settings.
func NewShaders() *Shaders {
	c := &Shaders{
		shaderMain:     NewShaderMain(),
		shaderSky:      NewShaderSky(),
		shaderSSAO:     NewShaderSSAO(),
		shaderBlur:     NewShaderBlur(),
		shaderGeometry: NewShaderGeometry(),
		shaderDepth:    NewShaderDepth(),
		shaderPost:     NewShaderPost(),
		flashFactor:    3.0,
		flashOffsetX:   0.0,
		flashOffsetY:   0.0,
		enableShadows:  false,
	}
	c.shaders = append(c.shaders, c.shaderMain, c.shaderSky, c.shaderSSAO, c.shaderBlur, c.shaderGeometry, c.shaderDepth, c.shaderPost)
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

// EnableShadows toggles the shadow rendering state and updates related flash offset values accordingly.
func (w *Shaders) EnableShadows() {
	w.enableShadows = !w.enableShadows
	if w.enableShadows {
		w.flashOffsetX = 1.0
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

	renderScene := func() {
		gl.BindVertexArray(w.shaderMain.mainVao)
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
		gl.BindVertexArray(w.shaderSky.skyVao)
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
		w.shaderDepth.Render(renderScene)
	}
	w.shaderSSAO.Prepare()
	w.shaderGeometry.Render(renderScene)
	w.shaderSSAO.Render(renderSky, w.shaderBlur.GetProgram())
	blurTex := w.shaderSSAO.GetSSAOBlurTexture()

	// Dirotta tutto il rendering 3D (scena + cielo) nel buffer HDR a 16-bit
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.shaderPost.GetFBO())
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	// ------------------------------

	w.shaderMain.Render(roomShadowTex, flashShadowTex, blurTex)

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
