package open_gl_legacy

import (
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/pixels"
	"github.com/markel1974/godoom/pixels/executor"
)

// scaleFactor defines a constant value for scaling factors used in the application.
// maxBatchVertices specifies the maximum number of vertices that can be processed in a single batch.
// maxFrameCommands sets the limit on the number of commands that can be issued per frame.
const (
	scaleFactor = 1

	maxBatchVertices = 65536 * 2

	maxFrameCommands = 4096
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
	win          *pixels.GLWindow
	screenWidth  int
	screenHeight int

	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	targetId           string

	enableClear bool
	debug       bool
	debugIdx    int

	bobPhase float64

	shaders *Shaders
	tex     *Textures
	builder *BatchBuilder
}

// NewRender initializes and returns a new instance of RenderOpenGL with default settings and prepared resources.
func NewRender() *RenderOpenGL {
	tex := NewTextures()
	r := &RenderOpenGL{
		engine:        nil,
		vi:            model.NewViewMatrix(),
		player:        nil,
		win:           nil,
		screenWidth:   0,
		screenHeight:  0,
		targetSectors: map[int]bool{0: true},
		enableClear:   false,
		debug:         false,
		debugIdx:      0,
		shaders:       NewShaders(),
		builder:       NewBatchBuilder(tex),
		tex:           tex,
		bobPhase:      0.0,
	}
	return r
}

// Setup initializes the RenderOpenGL instance by configuring essential properties based on the provided engine instance.
func (w *RenderOpenGL) Setup(en *engine.Engine) error {
	w.engine = en
	w.screenWidth = w.engine.GetWidth()
	w.screenHeight = w.engine.GetHeight()
	w.player = en.GetPlayer()
	return nil
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
		stride := w.builder.Stride()
		if err := w.shaders.Setup(int32(fbW), int32(fbH), stride); err != nil {
			return err
		}
		if err := w.tex.Setup(w.engine.GetTextures()); err != nil {
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
	// 1. Svuotamento manuale per il nuovo frame
	w.builder.Reset()

	cs, count, things, lights := w.engine.Compute(w.player, w.vi)
	w.targetLastCompiled = count
	cSky := w.builder.CreateBatch(w.vi, cs, count, things, lights)

	executor.Thread.Call(func() {
		w.win.Begin()
		pX, pY := w.player.GetPosition()
		fbW, fbH := w.win.GetFramebufferSize()
		commands := w.builder.GetDrawCommands()
		vert, vertCount := w.builder.GetFrameVertices()
		skyTexId, skyNormalTexId, emissiveTexId := uint32(0), uint32(0), uint32(0)
		skyEnabled := false
		if cSky != nil {
			skyTexId, skyNormalTexId, emissiveTexId, skyEnabled = w.tex.Get(cSky)
		}
		w.shaders.Render(w.vi, pX, pY, int32(fbW), int32(fbH), vert, vertCount, commands, skyEnabled, skyTexId, emissiveTexId, skyNormalTexId)
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
				w.shaders.IncreaseFlashFactor()
			case pixels.KeyH:
				w.shaders.DecreaseFlashFactor()
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
			w.shaders.ToggleShadows()
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

	// Modulazione della fase basata sul moto reale
	if up || down || left || right {
		w.bobPhase += impulse * 15.0 // Frequenza del passo
	} else {
		w.bobPhase *= 0.85 // Smorzamento critico verso l'origine
	}
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
