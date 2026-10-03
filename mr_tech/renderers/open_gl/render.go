package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop/executor"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IBuilder is an interface for constructing and managing draw commands, vertex data, and lighting in a rendering pipeline.
type IBuilder interface {
	Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine)

	GetSkyTexture() *textures.Texture

	GetSkyUV() (float64, float64)

	GetVerticesStride() int32

	GetLightsStride() int32

	GetDrawCommands() *DrawCommandsRender

	GetDrawCommandsAdditive() *DrawCommandsRender

	GetVertices() ([]float32, int32, []uint32, int32)

	GetLights() ([]float32, int32)

	GetShadowLights() ([8]*Light, int32)
}

// SpriteNode represents a renderable entity in a scene, including its associated model and squared distance from the camera.
type SpriteNode struct {
	Thing  model.IThing
	DistSq float64
}

// RenderOpenGL is responsible for managing and executing OpenGL rendering operations for the game environment.
type RenderOpenGL struct {
	th              *executor.MainThread
	ctx             api.IContext
	engine          *engine.Engine
	vi              *model.ViewMatrix
	player          *model.ThingPlayer
	win             *desktop.Window
	shaders         *Shaders
	tex             *Textures
	builder         IBuilder
	enableClear     bool
	builders        []IBuilder
	startWidth      int32
	startHeight     int32
	buildersCounter int
}

// NewRender initializes and returns a new instance of RenderOpenGL with default settings and prepared resources.
func NewRender(ctx api.IContext, th *executor.MainThread, w, h int32) *RenderOpenGL {
	r := &RenderOpenGL{
		ctx:         ctx,
		th:          th,
		engine:      nil,
		vi:          model.NewViewMatrix(),
		player:      nil,
		win:         nil,
		enableClear: false,
		shaders:     nil,
		startWidth:  w,
		startHeight: h,
	}
	return r
}

// Setup initializes the RenderOpenGL instance by configuring essential properties based on the provided engine instance.
func (w *RenderOpenGL) Setup(en *engine.Engine) error {
	w.engine = en
	w.player = en.GetPlayer()
	return nil
}

// doInitialize initializes the OpenGL rendering environment and compiles shaders and textures for the renderer.
func (w *RenderOpenGL) doInitialize() error {
	bounds := desktop.R(0, 0, float64(w.startWidth), float64(w.startHeight))
	cfg := desktop.WindowConfig{
		Bounds:             bounds,
		VSync:              true,
		Undecorated:        false,
		Smooth:             false,
		Resizable:          true,
		DisableScissorTest: true,
	}
	var winErr error
	w.win, winErr = desktop.NewGLWindow(w.th, cfg)
	if winErr != nil {
		return winErr
	}
	thErr := w.th.CallErr(func() error {
		w.win.Begin()
		cal := w.engine.GetCalibration()
		w.tex = NewTextures(w.ctx)
		w.buildersCounter = 0
		//if cal.Full3d {
		w.player.SetPitchOptions(-1.5, 1.5, 0.01)
		w.builder = NewBuilderVolume(w.ctx, w.tex, cal)
		w.builders = append(w.builders, w.builder)
		//} else {
		//	w.builder = NewBuilderTraverse(w.tex, cal)
		//	w.builders = append(w.builders, w.builder, NewBuilderScene(w.tex))
		//}
		vStride := w.builder.GetVerticesStride()
		lStride := w.builder.GetLightsStride()
		w.shaders = NewShaders(w.ctx)
		if err := w.shaders.Setup(vStride, lStride, w.player, cal, w.tex); err != nil {
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
	w.th.Run(w.doRun)
}

// doRender performs the rendering process by computing the scene, creating rendering batches, and issuing draw commands.
func (w *RenderOpenGL) doRender() {
	w.th.Call(func() {
		w.win.Begin()
		fbW, fbH := w.win.GetFramebufferSize()
		w.engine.Compute(w.player, w.vi)
		w.builder.Compute(int32(fbW), int32(fbH), w.vi, w.engine)
		cSky := w.builder.GetSkyTexture()
		commands := w.builder.GetDrawCommands()
		commandsAdditive := w.builder.GetDrawCommandsAdditive()
		vert, vertLen, indices, indicesLen := w.builder.GetVertices()
		light, lightsCount := w.builder.GetLights()

		shadowLights, shadowLightsCount := w.builder.GetShadowLights()
		skyLayer := float32(-1.0)
		skyEnabled := false
		skyU, skyV := float32(0), float32(0)
		if cSky != nil {
			skyLayer, skyEnabled = w.tex.Get(cSky)
			u, v := w.builder.GetSkyUV()
			tick := float32(textures.GlobalTick()) / 60.0
			skyU = float32(u) * tick
			skyV = float32(v) * tick
			// DEBUG
			//if textures.GlobalTick()%60 == 0 {
			//	fmt.Println("SKY DEBUG: Name=", cSky.GetName(), " Layer=", skyLayer, " Enabled=", skyEnabled, " U=", skyU, " V=", skyV)
			//}
		}
		w.shaders.Render(w.vi, int32(fbW), int32(fbH), vert, vertLen, indices, indicesLen, commands, commandsAdditive, skyEnabled, skyLayer, skyU, skyV, light, lightsCount, shadowLights, shadowLightsCount)
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
			case desktop.KeyEscape:
				return
			case desktop.KeyW:
				up = true
				impulse = 0.01
			case desktop.KeyUp:
				up = true
			case desktop.KeyS:
				down = true
				impulse = 0.01
			case desktop.KeyDown:
				down = true
			case desktop.KeyLeft:
				left = true
			case desktop.KeyRight:
				right = true
			case desktop.KeyL:
				w.player.GetFlash().IncreaseFlashFactor()
			case desktop.KeyK:
				w.player.GetFlash().DecreaseFlashFactor()
			}
		}

		w.doPlayerMoves(impulse, up, down, left, right)

		if w.win.JustPressed(desktop.KeyO) {
			w.doPlayerThrow()
		}
		if w.win.JustPressed(desktop.KeyP) {
			w.doPlayerFire()
		}
		if w.win.JustPressed(desktop.KeyC) {
			w.enableClear = true
		}
		if w.win.JustPressed(desktop.KeyTab) || w.win.Pressed(desktop.MouseButton2) {
			w.doPlayerDuckingToggle()
		}
		if w.win.JustPressed(desktop.KeySpace) {
			w.doPlayerJump(false)
		}
		if w.win.Pressed(desktop.MouseButton1) {
			w.doPlayerJump(true)
		}
		if w.win.JustPressed(desktop.KeyM) {
			mouseConnected = !mouseConnected
		}
		if w.win.JustPressed(desktop.KeyN) {
			w.shaders.ToggleShadows()
		}
		if w.win.JustPressed(desktop.KeyT) {
			w.buildersCounter++
			index := w.buildersCounter % (len(w.builders))
			w.builder = w.builders[index]
		}
		w.win.UpdateInputAndSwap()
	}
}

// doPlayerFire triggers the player's fire action by retrieving position, angle, and sector, and invoking the engine's fire logic.
func (w *RenderOpenGL) doPlayerThrow() {
	const throwableIndex = 2
	w.player.Throw(throwableIndex, 300)
}

// doPlayerFire triggers the player's fire action by retrieving position, angle, and sector, and invoking the engine's fire logic.
func (w *RenderOpenGL) doPlayerFire() {
	w.player.Fire("gun")
}

// doPlayerDuckingToggle toggles the player's ducking state by invoking the SetDucking method on the player instance.
func (w *RenderOpenGL) doPlayerDuckingToggle() { w.player.SetDucking() }

// doPlayerJump triggers the player's jump action by invoking the SetJump method on the player instance.
func (w *RenderOpenGL) doPlayerJump(multi bool) { w.player.SetJump(multi) }

// doPlayerMoves moves the player based on the provided impulse and directional flags (up, down, left, right).
func (w *RenderOpenGL) doPlayerMoves(impulse float64, up bool, down bool, left bool, right bool) {
	w.player.Move(impulse, up, down, right, left)
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
	w.player.AddAngle(-mouseX * 0.03)
	w.player.SetPitch(mouseY)
	//w.player.MoveApply(0, 0, 0)
}
