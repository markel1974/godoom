//go:build js && wasm

package web

import (
	"syscall/js"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// WindowConfig defines the configuration options for initializing a window, including size, title, fullscreen, and VSync.
type WindowConfig struct {
	Title      string
	Width      int
	Height     int
	Fullscreen bool
	VSync      bool
}

// XY represents a 2D point or vector with X and Y coordinates as float64 values.
type XY struct {
	X float64
	Y float64
}

// Rect represents a rectangle defined by its top-left corner (X, Y) and its dimensions (W, H).
type Rect struct {
	X float64
	Y float64
	W float64
	H float64
}

// Button represents an input device button or key, such as a keyboard key or mouse button.
type Button int

// Button constants mirroring the desktop API
const (
	KeyEscape    = Button(256)
	KeyW         = Button(87)
	KeyS         = Button(83)
	KeyA         = Button(65)
	KeyD         = Button(68)
	KeyUp        = Button(265)
	KeyDown      = Button(264)
	KeyLeft      = Button(263)
	KeyRight     = Button(262)
	KeyL         = Button(76)
	KeyK         = Button(75)
	KeyO         = Button(79)
	KeyP         = Button(80)
	KeyC         = Button(67)
	KeyTab       = Button(258)
	KeySpace     = Button(32)
	KeyM         = Button(77)
	KeyN         = Button(78)
	MouseButton1 = Button(0)
	MouseButton2 = Button(1)
)

var codeToButton = map[string]Button{
	"Escape":     KeyEscape,
	"KeyW":       KeyW,
	"KeyS":       KeyS,
	"KeyA":       KeyA,
	"KeyD":       KeyD,
	"ArrowUp":    KeyUp,
	"ArrowDown":  KeyDown,
	"ArrowLeft":  KeyLeft,
	"ArrowRight": KeyRight,
	"KeyL":       KeyL,
	"KeyK":       KeyK,
	"KeyO":       KeyO,
	"KeyP":       KeyP,
	"KeyC":       KeyC,
	"Tab":        KeyTab,
	"Space":      KeySpace,
	"KeyM":       KeyM,
	"KeyN":       KeyN,
}

// Window represents a browser-based rendering window using WebGL for graphics rendering. It includes input handling capabilities.
type Window struct {
	canvas js.Value
	gl     js.Value
	closed bool
	width  int
	height int

	keysDown     map[Button]bool
	keysPressed  map[Button]bool
	keysReleased map[Button]bool

	mouseX                float64
	mouseY                float64
	prevMouseX            float64
	prevMouseY            float64
	scrollX               float64
	scrollY               float64
	renderPrepareFn       func() error
	renderStartFn         func(int, int)
	playerMouseMoveFn     func(float64, float64)
	playerMovesFn         func(float64, bool, bool, bool, bool)
	playerThrowFn         func()
	playerFireFn          func()
	playerDuckingToggleFn func()
	playerJumpFn          func(multi bool)
	toggleShadowsFn       func()
	enableClearFn         func()
	decreaseFlashFactorFn func()
	increaseFlashFactorFn func()
}

// NewGLWindow creates a new OpenGL window with the specified configuration and initializes WebGL2 context.
func NewGLWindow(ctx *Context, cfg WindowConfig) *Window {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "canvas")
	if canvas.IsUndefined() || canvas.IsNull() {
		canvas = doc.Call("createElement", "canvas")
		canvas.Set("id", "canvas")
		doc.Get("body").Call("appendChild", canvas)
	}

	canvas.Set("width", cfg.Width)
	canvas.Set("height", cfg.Height)

	// WebGL2 initialization
	glArgs := js.Global().Get("Object").New()
	glArgs.Set("alpha", false)
	glArgs.Set("antialias", false)
	glArgs.Set("depth", true)
	glArgs.Set("stencil", true)
	glArgs.Set("powerPreference", "high-performance")
	glArgs.Set("desynchronized", true)

	gl := canvas.Call("getContext", "webgl2", glArgs)

	// Enable extensions
	gl.Call("getExtension", "EXT_color_buffer_float")
	gl.Call("getExtension", "OES_texture_float_linear")
	gl.Call("getExtension", "EXT_color_buffer_half_float")

	w := &Window{
		canvas:       canvas,
		gl:           gl,
		width:        cfg.Width,
		height:       cfg.Height,
		keysDown:     make(map[Button]bool),
		keysPressed:  make(map[Button]bool),
		keysReleased: make(map[Button]bool),
	}

	w.bindEvents()

	ctx.gl = gl
	return w
}

// bindEvents binds event listeners for key and mouse events, updating internal state based on user interactions.
func (w *Window) bindEvents() {
	js.Global().Set("onkeydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		code := e.Get("code").String()
		if btn, ok := codeToButton[code]; ok {
			if !w.keysDown[btn] {
				w.keysPressed[btn] = true
			}
			w.keysDown[btn] = true
		}
		return nil
	}))

	js.Global().Set("onkeyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		code := e.Get("code").String()
		if btn, ok := codeToButton[code]; ok {
			w.keysDown[btn] = false
			w.keysReleased[btn] = true
		}
		return nil
	}))

	w.canvas.Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.canvas.Call("requestPointerLock")
		return nil
	}))

	js.Global().Set("mouseTracker", js.Global().Get("Object").New())
	js.Global().Get("mouseTracker").Set("x", 0)
	js.Global().Get("mouseTracker").Set("y", 0)
	js.Global().Get("window").Call("eval", `
		window.gameMouseTracker = { x: 0, y: 0 };
		document.getElementById('canvas').onmousemove = function(e) {
			if (e.movementX !== undefined) {
				window.gameMouseTracker.x += e.movementX;
				window.gameMouseTracker.y -= e.movementY;
			} else {
				window.gameMouseTracker.x = e.clientX;
				window.gameMouseTracker.y = e.clientY;
			}
		};
	`)

	w.canvas.Set("onmousedown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		btnIdx := e.Get("button").Int()
		btn := MouseButton1
		if btnIdx == 2 {
			btn = MouseButton2
		}
		if !w.keysDown[btn] {
			w.keysPressed[btn] = true
		}
		w.keysDown[btn] = true
		return nil
	}))

	w.canvas.Set("onmouseup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		btnIdx := e.Get("button").Int()
		btn := MouseButton1
		if btnIdx == 2 {
			btn = MouseButton2
		}
		w.keysDown[btn] = false
		w.keysReleased[btn] = true
		return nil
	}))

	w.canvas.Set("oncontextmenu", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		args[0].Call("preventDefault")
		return nil
	}))

	w.canvas.Set("onwheel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		w.scrollX += e.Get("deltaX").Float()
		w.scrollY += e.Get("deltaY").Float()
		return nil
	}))
}

func (w *Window) Begin() {
	w.scrollX = 0
	w.scrollY = 0
}

// UpdateInputAndSwap resets the state of keysPressed and keysReleased maps at the end of each frame.
func (w *Window) UpdateInputAndSwap() {
	// Reset per-frame state
	for k := range w.keysPressed {
		delete(w.keysPressed, k)
	}
	for k := range w.keysReleased {
		delete(w.keysReleased, k)
	}
}

// Closed returns the closed state of the window as a boolean.
func (w *Window) Closed() bool {
	return w.closed
}

// SetClosed sets the closed state of the Window. Use true to mark the window as closed, false to mark it as open.
func (w *Window) SetClosed(closed bool) {
	w.closed = closed
}

// GetFramebufferSize returns the width and height of the window's framebuffer in pixels.
func (w *Window) GetFramebufferSize() (int, int) {
	return w.width, w.height
}

// Pressed checks if the specified button is currently held down and returns true if it is, otherwise false.
func (w *Window) Pressed(button Button) bool {
	return w.keysDown[button]
}

// JustPressed checks if the specified button was pressed during the current frame and returns true if it was.
func (w *Window) JustPressed(button Button) bool {
	return w.keysPressed[button]
}

// MousePosition returns the current mouse cursor position relative to the window as an XY struct.
func (w *Window) MousePosition() XY {
	return XY{X: w.mouseX, Y: w.mouseY}
}

// Destroy marks the window as closed, releasing its associated resources and flagging it for cleanup.
func (w *Window) Destroy() {
	w.closed = true
}

// SetTitle sets the title of the window to the specified string.
func (w *Window) SetTitle(title string) {}

// SetCursorVisible sets the visibility of the cursor. Pass true to make it visible, false to hide it.
func (w *Window) SetCursorVisible(v bool) {}

// SetCursorDisabled disables the cursor, typically used for capturing mouse input in games or applications.
func (w *Window) SetCursorDisabled() {}

// MouseScroll returns the current mouse scroll offset as an XY struct containing horizontal (X) and vertical (Y) values.
func (w *Window) MouseScroll() XY { return XY{X: w.scrollX, Y: w.scrollY} }

func (w *Window) Setup(r api.IRender) error {
	w.renderPrepareFn = r.RenderPrepare
	w.renderStartFn = r.RenderStart
	w.playerMouseMoveFn = r.RenderPlayerMouseMove
	w.playerMovesFn = r.RenderPlayerMoves
	w.playerThrowFn = r.RenderPlayerThrow
	w.playerFireFn = r.RenderPlayerFire
	w.playerDuckingToggleFn = r.RenderPlayerDuckingToggle
	w.playerJumpFn = r.RenderPlayerJump
	w.toggleShadowsFn = r.RenderToggleShadows
	w.enableClearFn = r.RenderEnableClear
	w.decreaseFlashFactorFn = r.RenderDecreaseFlashFactor
	w.increaseFlashFactorFn = r.RenderIncreaseFlashFactor
	return nil
}

// Start begins the main loop via requestAnimationFrame.
func (w *Window) Start() {
	if err := w.renderPrepareFn(); err != nil {
		panic(err)
	}

	var renderFrame js.Func
	renderFrame = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.Begin()

		tracker := js.Global().Get("window").Get("gameMouseTracker")
		w.mouseX += tracker.Get("x").Float()
		w.mouseY += tracker.Get("y").Float()
		// Reset in JS
		tracker.Set("x", 0)
		tracker.Set("y", 0)

		if w.mouseX != w.prevMouseX || w.mouseY != w.prevMouseY {
			w.playerMouseMoveFn(w.mouseX-w.prevMouseX, w.mouseY-w.prevMouseY)
			w.prevMouseX = w.mouseX
			w.prevMouseY = w.mouseY
		}
		w.renderStartFn(w.width, w.height)
		var up, down, left, right bool

		if w.scrollX != 0 || w.scrollY != 0 {
			if w.scrollY < 0 { // Web wheel delta negative means scroll up
				up = true
			} else if w.scrollY > 0 {
				down = true
			}
		}

		impulse := 0.06
		for v, isDown := range w.keysDown {
			if !isDown {
				continue
			}
			switch v {
			case KeyW:
				up = true
				impulse = 0.01
			case KeyUp:
				up = true
			case KeyS:
				down = true
				impulse = 0.01
			case KeyDown:
				down = true
			case KeyA:
				left = true
			case KeyLeft:
				left = true
			case KeyD:
				right = true
			case KeyRight:
				right = true
			case KeyL:
				w.increaseFlashFactorFn()
			case KeyK:
				w.decreaseFlashFactorFn()
			}
		}

		w.playerMovesFn(impulse, up, down, left, right)

		if w.JustPressed(KeyO) {
			w.playerThrowFn()
		}
		if w.JustPressed(KeyP) {
			w.playerFireFn()
		}
		if w.JustPressed(KeyC) {
			w.enableClearFn()
		}
		if w.JustPressed(KeyTab) || w.Pressed(MouseButton2) {
			w.playerDuckingToggleFn()
		}
		if w.JustPressed(KeySpace) {
			w.playerJumpFn(false)
		}
		if w.Pressed(MouseButton1) {
			w.playerJumpFn(true)
		}
		if w.JustPressed(KeyN) {
			w.toggleShadowsFn()
		}

		w.UpdateInputAndSwap()

		if !w.Closed() {
			js.Global().Call("requestAnimationFrame", renderFrame)
		} else {
			renderFrame.Release()
		}
		return nil
	})
	js.Global().Call("requestAnimationFrame", renderFrame)
}
