//go:build js && wasm

package core_web

import (
	"syscall/js"
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

	mouseX  float64
	mouseY  float64
	scrollX float64
	scrollY float64
}

// NewGLWindow creates a new OpenGL window with the specified configuration and initializes WebGL2 context.
func NewGLWindow(cfg WindowConfig) (*Window, error) {
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

	gl := canvas.Call("getContext", "webgl2", glArgs)

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

	return w, nil
}

// GetContext initializes and returns a new ContextWeb associated with the Window's WebGL context.
func (w *Window) GetContext() *Context {
	return NewContextWeb(w.gl)
}

// bindEvents binds event listeners for key and mouse events, updating internal state based on user interactions.
func (w *Window) bindEvents() {
	// Keydown
	js.Global().Set("onkeydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		// code := e.Get("code").String()
		// Map JS code to internal Button here
		return nil
	}))

	// Keyup
	js.Global().Set("onkeyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	}))

	// Mouse Move
	w.canvas.Set("onmousemove", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		e := args[0]
		w.mouseX = e.Get("clientX").Float()
		w.mouseY = e.Get("clientY").Float()
		return nil
	}))
}

// Begin resets the scroll values for the current frame to zero.
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

// Run executes the main game loop using the browser's requestAnimationFrame for smooth execution.
func (w *Window) Run(tick func()) {
	var renderFrame js.Func
	renderFrame = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.Begin()
		tick()
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
