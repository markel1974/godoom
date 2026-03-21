package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/pixels"
	"github.com/markel1974/godoom/pixels/executor"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// scaleFactor defines a constant value for scaling factors used in the application.
// maxBatchVertices specifies the maximum number of vertices that can be processed in a single batch.
// maxFrameCommands sets the limit on the number of commands that can be issued per frame.
// vboMaxFloats determines the maximum number of floats that can be stored in the vertex buffer object.
const (
	scaleFactor = 1

	maxBatchVertices = 65536 * 2

	maxFrameCommands = 4096

	vboMaxFloats = 1024 * 1024 * 4
)

// SpriteNode represents a renderable entity in a scene, including its associated model and squared distance from the camera.
type SpriteNode struct {
	Thing  model.IThing
	DistSq float64
}

// RenderOpenGL is responsible for managing and executing OpenGL rendering operations for the game environment.
type RenderOpenGL struct {
	engine       *engine.Engine
	vi           *model.ViewMatrix
	player       *model.ThingPlayer
	textures     textures.ITextures
	win          *pixels.GLWindow
	screenWidth  int
	screenHeight int

	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	targetId           string

	enableClear   bool
	debug         bool
	debugIdx      int
	flashFactor   float32
	flashOffsetX  float32
	flashOffsetY  float32
	enableShadows bool

	compiler *Compiler
	builder  *BatchBuilder
}

// NewOpenGLRender initializes and returns a new instance of RenderOpenGL with default settings and prepared resources.
func NewOpenGLRender() *RenderOpenGL {
	compiler := NewCompiler()
	r := &RenderOpenGL{
		engine:        nil,
		vi:            model.NewViewMatrix(),
		textures:      nil,
		player:        nil,
		win:           nil,
		screenWidth:   0,
		screenHeight:  0,
		targetSectors: map[int]bool{0: true},
		enableClear:   false,
		debug:         false,
		debugIdx:      0,
		flashFactor:   3.0,
		flashOffsetX:  0.0,
		flashOffsetY:  0.0,
		enableShadows: false,
		compiler:      compiler,
		builder:       NewBatchBuilder(compiler),
	}
	return r
}

// Setup initializes the RenderOpenGL instance by configuring essential properties based on the provided engine instance.
func (w *RenderOpenGL) Setup(en *engine.Engine) error {
	w.engine = en
	w.screenWidth = w.engine.GetWidth()
	w.screenHeight = w.engine.GetHeight()
	w.player = en.GetPlayer()
	w.textures = en.GetTextures()
	return nil
}

// glInit initializes OpenGL state, buffers, shaders, and samplers required for rendering and SSAO processing.
func (w *RenderOpenGL) glInit() error {
	stride := w.builder.Stride()
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

	return nil
}

// renderScene renders the current scene by iterating over draw commands and issuing OpenGL draw calls.
func (w *RenderOpenGL) glRenderScene() {
	gl.BindVertexArray(w.compiler.shaderMain.mainVao)
	var lastTexId uint32 = math.MaxUint32

	commands := w.builder.GetDrawCommands()
	for _, cmd := range commands {
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

// drawScreenQuad renders a full-screen quad using the sky vertex array and disables depth testing during the draw operation.
func (w *RenderOpenGL) glRenderSky() {
	gl.BindVertexArray(w.compiler.shaderSky.skyVao)
	gl.Disable(gl.DEPTH_TEST)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.Enable(gl.DEPTH_TEST)
}

// doInitialize initializes the OpenGL rendering environment and compiles shaders and textures for the renderer.
func (w *RenderOpenGL) doInitialize() error {
	cfg := pixels.WindowConfig{
		Bounds:      pixels.R(0, 0, float64(w.screenWidth)*scaleFactor, float64(w.screenHeight)*scaleFactor),
		VSync:       true,
		Undecorated: false,
		Smooth:      false,
	}
	var winErr error
	w.win, winErr = pixels.NewGLWindow(cfg)
	if winErr != nil {
		return winErr
	}

	thErr := executor.Thread.CallErr(func() error {
		w.win.Begin()

		fbW, fbH := w.win.GetFramebufferSize()

		w.compiler.Setup(int32(fbW), int32(fbH))
		if err := w.compiler.CompileShaders(); err != nil {
			return err
		}

		w.compiler.shaderMain.Init()

		if err := w.glInit(); err != nil {
			return err
		}

		w.compiler.SetupSamplers()

		if err := w.compiler.CompileTextures(w.textures); err != nil {
			return err
		}
		return nil
	})

	if thErr != nil {
		return thErr
	}
	return nil
}

// Start initializes and starts the OpenGL rendering loop by invoking the provided rendering function.
func (w *RenderOpenGL) Start() {
	pixels.GLRun(w.doRun)
}

// doRender performs the rendering process by computing the scene, creating rendering batches, and issuing draw commands.
func (w *RenderOpenGL) doRender() {
	cs, count, things := w.engine.Compute(w.player, w.vi)
	w.targetLastCompiled = count
	cSky := w.builder.CreateBatch(w.vi, cs, count, things)

	executor.Thread.Call(func() {
		w.win.Begin()
		fbW, fbH := w.win.GetFramebufferSize()
		gl.Viewport(0, 0, int32(fbW), int32(fbH))
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Aggiornamento Stato (CPU)
		pX, pY := w.player.GetPosition()
		roomSpaceMatrix, flashSpaceMatrix := CreateSpaces(w.vi, pX, pY, w.flashOffsetX, w.flashOffsetY)
		proj, view := w.compiler.shaderMain.UpdateUniforms(w.vi, roomSpaceMatrix, flashSpaceMatrix, w.flashFactor, w.enableShadows, w.flashOffsetX, w.flashOffsetY)
		w.compiler.shaderDepth.UpdateUniforms(roomSpaceMatrix, flashSpaceMatrix)
		w.compiler.shaderGeometry.UpdateUniforms(view, proj)
		w.compiler.shaderSSAO.UpdateUniforms(view, proj)
		w.compiler.shaderSky.UpdateUniforms(view, proj)

		// Render
		w.compiler.shaderMain.Prepare(w.builder.GetFrameVertices())
		var roomShadowTex, flashShadowTex uint32
		if w.enableShadows {
			roomShadowTex, flashShadowTex = w.compiler.shaderDepth.GetShadowTextures()
			w.compiler.shaderDepth.Render(w.glRenderScene)
		}
		w.compiler.shaderSSAO.Prepare()
		w.compiler.shaderGeometry.Render(w.glRenderScene)
		w.compiler.shaderSSAO.Render(w.glRenderSky, w.compiler.shaderBlur.GetProgram())
		blurTex := w.compiler.shaderSSAO.GetSSAOBlurTexture()
		w.compiler.shaderMain.Render(roomShadowTex, flashShadowTex, blurTex)

		var lastTexId uint32 = math.MaxUint32
		var lastNormId uint32 = math.MaxUint32
		for _, cmd := range w.builder.GetDrawCommands() {
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

		if cSky != nil {
			texId, normTexId, ok := w.compiler.GetTexture(cSky)
			w.compiler.shaderSky.Render(texId, normTexId, ok)
		}
	})
}

// doRun executes the main rendering and input handling loop for the RenderOpenGL instance.
func (w *RenderOpenGL) doRun() {
	if err := w.doInitialize(); err != nil {
		panic(err)
	}
	mouseConnected := true
	for !w.win.Closed() {
		w.doRender()

		if mouseConnected && w.win.MouseInsideWindow() {
			mousePos := w.win.MousePosition()
			mousePrevPos := w.win.MousePreviousPosition()
			if mousePos.X != mousePrevPos.X || mousePos.Y != mousePrevPos.Y {
				mouseX := mousePos.X - mousePrevPos.X
				mouseY := mousePos.Y - mousePrevPos.Y
				w.doPlayerMouseMove(mouseX, mouseY)
			}
		}

		var up, down, left, right bool

		if scroll := w.win.MouseScroll(); scroll.Y != 0 {
			if scroll.Y > 0 {
				up = true
			} else {
				down = true
			}
		}

		var impulse = 0.06
		for v := range w.win.KeysPressed() {
			switch v {
			case pixels.KeyEscape:
				return
			case pixels.KeyW:
				up = true
				impulse = 0.01
			case pixels.KeyUp:
				up = true
			case pixels.KeyS:
				down = true
				impulse = 0.01
			case pixels.KeyDown:
				down = true
			case pixels.KeyLeft:
				left = true
			case pixels.KeyRight:
				right = true
			case pixels.KeyV:
				w.doDebugMoveSector(true)
			case pixels.KeyB:
				w.doDebugMoveSector(false)
			case pixels.KeyL:
				w.flashFactor++
			case pixels.KeyH:
				if w.flashFactor > 0 {
					w.flashFactor--
				}
			}
		}

		w.doPlayerMoves(impulse, up, down, left, right)

		if w.win.JustPressed(pixels.KeyF) {
			w.doPlayerFire()
		}
		if w.win.JustPressed(pixels.KeyC) {
			w.enableClear = true
			w.doDebugMoveSectorToggle()
		}
		if w.win.JustPressed(pixels.KeyZ) {
			w.doDebugMoveSector(true)
		}
		if w.win.JustPressed(pixels.KeyX) {
			w.doDebugMoveSector(false)
		}
		if w.win.JustPressed(pixels.KeyTab) || w.win.Pressed(pixels.MouseButton2) {
			w.doPlayerDuckingToggle()
		}
		if w.win.JustPressed(pixels.KeySpace) || w.win.Pressed(pixels.MouseButton1) {
			w.doPlayerJump()
		}
		if w.win.JustPressed(pixels.Key8) {
			w.doDebug(0)
		}
		if w.win.JustPressed(pixels.Key0) {
			w.doDebug(1)
		}
		if w.win.JustPressed(pixels.Key9) {
			w.doDebug(-1)
		}
		if w.win.JustPressed(pixels.KeyM) {
			mouseConnected = !mouseConnected
		}
		if w.win.JustPressed(pixels.KeyN) {
			w.enableShadows = !w.enableShadows
			if w.enableShadows {
				w.flashOffsetX = 4.0
				w.flashOffsetY = -2.0
			} else {
				w.flashOffsetX = 0.0
				w.flashOffsetY = 0.0
			}
		}

		w.win.UpdateInputAndSwap()
	}
}

// doPlayerFire triggers the player's fire action by retrieving position, angle, and sector, and invoking the engine's fire logic.
func (w *RenderOpenGL) doPlayerFire() {
	x, y := w.player.GetPosition()
	w.engine.Fire(w.player.GetSector(), x, y, w.player.GetRealAngle())
}

// doPlayerDuckingToggle toggles the player's ducking state by invoking the SetDucking method on the player instance.
func (w *RenderOpenGL) doPlayerDuckingToggle() { w.player.SetDucking() }

// doPlayerJump triggers the player's jump action by invoking the SetJump method on the player instance.
func (w *RenderOpenGL) doPlayerJump() { w.player.SetJump() }

// doPlayerMoves moves the player based on the provided impulse and directional flags (up, down, left, right).
func (w *RenderOpenGL) doPlayerMoves(impulse float64, up bool, down bool, left bool, right bool) {
	w.player.Move(impulse, up, down, left, right)
}

// doPlayerMouseMove adjusts the player's angle and yaw based on mouse movement, clamping the values within a defined offset range.
func (w *RenderOpenGL) doPlayerMouseMove(mouseX float64, mouseY float64) {
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
	w.player.AddAngle(mouseX * 0.03)
	w.player.SetYaw(mouseY)
	w.player.MoveApply(0, 0)
}

// doDebug toggles debugging or sets the debug mode to a specific sector depending on the provided next parameter.
func (w *RenderOpenGL) doDebug(next int) {
	if next == 0 {
		w.debug = !w.debug
		return
	}
	w.debug = true
	idx := w.debugIdx + next
	if idx < 0 || idx >= w.engine.Len() {
		return
	}
	w.debugIdx = idx
	sector := w.engine.SectorAt(idx)
	const offset = 5
	x := sector.Segments[0].Start.X + offset
	y := sector.Segments[0].Start.Y + offset
	w.player.SetSector(sector)
	w.player.SetXY(x, y)
}

// doDebugMoveSectorToggle toggles the `targetEnabled` state, enabling or disabling the debug move sector functionality.
func (w *RenderOpenGL) doDebugMoveSectorToggle() { w.targetEnabled = !w.targetEnabled }

// doDebugMoveSector adjusts the target sector index and updates the target sectors map based on the direction provided.
func (w *RenderOpenGL) doDebugMoveSector(forward bool) {
	if forward {
		if w.targetIdx < w.targetLastCompiled {
			w.targetIdx++
		}
	} else {
		if w.targetIdx > 0 {
			w.targetIdx--
		}
	}
	for k := 0; k < w.targetLastCompiled; k++ {
		w.targetSectors[k] = k == w.targetIdx
	}
}
