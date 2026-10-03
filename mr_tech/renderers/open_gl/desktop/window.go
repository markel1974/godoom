package desktop

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop/executor"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// _thread is a global variable holding a pointer to the singleton MainThread instance for managing serialized execution.
var _thread *executor.MainThread

// init initializes the main thread executor and assigns it to the global _thread variable.
func init() {
	_thread = executor.NewMainThread()
}

// WindowInput encapsulates data related to user input for a window, including mouse position, key states, scrolling, and text input.
type WindowInput struct {
	mouse XY

	buttons [KeyLast + 1]bool

	repeat [KeyLast + 1]bool

	scroll XY

	typed string
}

// WindowPos represents the position and size of a window with x, y coordinates and width, height dimensions.
type WindowPos struct {
	xPos int

	yPos int

	width int

	height int
}

// Window represents a graphical application window and manages its configuration, state, input, and rendering behavior.
type Window struct {
	th                               *executor.MainThread
	window                           *glfw.Window
	cfg                              WindowConfig
	bounds                           Rect
	vsync                            bool
	cursorVisible                    bool
	cursorInsideWindow               bool
	restore                          WindowPos
	prevInp, currInp, tempInp        WindowInput
	keysPressed                      map[Button]bool
	pressEvents, tempPressEvents     [KeyLast + 1]bool
	releaseEvents, tempReleaseEvents [KeyLast + 1]bool
	prevJoy, currJoy, tempJoy        GLJoystick

	renderSetupFn         func() error
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

// currWin holds a pointer to the currently active Window instance, ensuring only one context is active at a time.
var currWin *Window

// NewGLWindow creates and initializes a new OpenGL window using the specified context and window configuration.
func NewGLWindow(ctx api.IContext, cfg WindowConfig) *Window {
	w := &Window{
		th:            _thread,
		cfg:           cfg,
		cursorVisible: true,
		keysPressed:   make(map[Button]bool),
	}
	w.th.SetContext(ctx)
	return w
}

// Start initializes and begins the primary execution loop for the window's main thread.
func (w *Window) Start() {
	w.th.Run(w.doRun)
}

// Setup initializes rendering functions for the Window object using the provided IRender instance.
func (w *Window) Setup(r api.IRender) error {
	w.renderSetupFn = r.RenderSetup
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

// Destroy gracefully shuts down and releases any resources held by the window object.
func (w *Window) Destroy() {
	w.th.Call(func() {
		w.window.Destroy()
	})
}

// ClipboardText retrieves the current text content from the system clipboard associated with the window.
func (w *Window) ClipboardText() string {
	return w.window.GetClipboardString()
}

// SetClipboardText sets the system clipboard to the specified text.
func (w *Window) SetClipboardText(text string) {
	w.window.SetClipboardString(text)
}

// SetClosed sets the closed state of the window to the specified value, triggering the window's close behavior if true.
func (w *Window) SetClosed(closed bool) {
	w.th.Call(func() {
		w.window.SetShouldClose(closed)
	})
}

// Closed checks if the window should be closed and returns a boolean value indicating its state.
func (w *Window) Closed() bool {
	var closed bool
	w.th.Call(func() {
		closed = w.window.ShouldClose()
	})
	return closed
}

// SetTitle sets the window's title to the specified string.
func (w *Window) SetTitle(title string) {
	w.th.Call(func() {
		w.window.SetTitle(title)
	})
}

// SetBounds updates the window's bounding rectangle and resizes it to match the specified dimensions.
func (w *Window) SetBounds(bounds Rect) {
	w.bounds = bounds
	w.th.Call(func() {
		_, _, width, height := bounds.Bounds()
		w.window.SetSize(int(width), int(height))
	})
}

// SetPos sets the position of the window to the specified coordinates (X, Y) using a thread-safe call.
func (w *Window) SetPos(pos XY) {
	w.th.Call(func() {
		left, top := int(pos.X), int(pos.Y)
		w.window.SetPos(left, top)
	})
}

// GetPos returns the current position of the window as an XY coordinate in screen space.
func (w *Window) GetPos() XY {
	var v XY
	w.th.Call(func() {
		x, y := w.window.GetPos()
		v = MakeVec(float64(x), float64(y))
	})
	return v
}

// Bounds retrieves the rectangular boundaries of the window.
func (w *Window) Bounds() Rect {
	return w.bounds
}

// setFullscreen sets the window to fullscreen mode on the specified monitor. Stores current position and size for restoration.
func (w *Window) setFullscreen(monitor *Monitor) {
	w.th.Call(func() {
		w.restore.xPos, w.restore.yPos = w.window.GetPos()
		w.restore.width, w.restore.height = w.window.GetSize()
		mode := monitor.monitor.GetVideoMode()
		w.window.SetMonitor(
			monitor.monitor,
			0,
			0,
			mode.Width,
			mode.Height,
			mode.RefreshRate,
		)
	})
}

// setWindowed transitions the window to windowed mode using the previously stored position and dimensions.
func (w *Window) setWindowed() {
	w.th.Call(func() {
		w.window.SetMonitor(
			nil,
			w.restore.xPos,
			w.restore.yPos,
			w.restore.width,
			w.restore.height,
			0,
		)
	})
}

// SetMonitor changes the window's monitor to enable fullscreen or windowed mode based on the provided monitor parameter.
func (w *Window) SetMonitor(monitor *Monitor) {
	if w.Monitor() != monitor {
		if monitor != nil {
			w.setFullscreen(monitor)
		} else {
			w.setWindowed()
		}
	}
}

// Monitor returns the monitor associated with the window or nil if no monitor is found.
func (w *Window) Monitor() *Monitor {
	var monitor *glfw.Monitor
	w.th.Call(func() {
		monitor = w.window.GetMonitor()
	})
	if monitor == nil {
		return nil
	}
	return &Monitor{
		monitor: monitor,
	}
}

// Focused checks if the window currently has input focus and returns true if focused, otherwise false.
func (w *Window) Focused() bool {
	var focused bool
	w.th.Call(func() {
		focused = w.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

// SetVSync enables or disables vertical synchronization (VSync) for the window.
func (w *Window) SetVSync(vsync bool) {
	w.vsync = vsync
}

// VSync returns the current vertical synchronization (VSync) setting for the window.
func (w *Window) VSync() bool {
	return w.vsync
}

// SetCursorVisible sets the visibility of the cursor for the current window based on the provided boolean value.
func (w *Window) SetCursorVisible(visible bool) {
	w.cursorVisible = visible
	w.th.Call(func() {
		if visible {
			w.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		} else {
			w.window.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
		}
	})
}

// SetCursorDisabled disables the mouse cursor and sets it to "disabled" mode, typically used for first-person camera controls.
func (w *Window) SetCursorDisabled() {
	w.cursorVisible = false
	w.th.Call(func() {
		w.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	})
}

// CursorVisible returns a boolean indicating whether the cursor is currently visible in the window.
func (w *Window) CursorVisible() bool {
	return w.cursorVisible
}

// Begin activates the context for the current window to prepare for rendering operations.
func (w *Window) Begin() {
	w.begin()
}

// GetFramebufferSize retrieves the width and height of the framebuffer in pixels.
func (w *Window) GetFramebufferSize() (int, int) {
	framebufferWidth, framebufferHeight := w.window.GetFramebufferSize()
	return framebufferWidth, framebufferHeight
}

// begin sets the current OpenGL context to the window if it is not already active.
func (w *Window) begin() {
	if currWin != w {
		w.window.MakeContextCurrent()
		currWin = w
	}
}

// end finalizes the current operation or state for the Window instance.
func (w *Window) end() {
	// nothing, really
}

// Show makes the window visible on the screen, ensuring the operation runs on the appropriate thread.
func (w *Window) Show() {
	w.th.Call(func() {
		w.window.Show()
	})
}

// Clipboard retrieves the current text content of the system clipboard and returns it as a string.
func (w *Window) Clipboard() string {
	var clipboard string
	w.th.Call(func() {
		clipboard = w.window.GetClipboardString()
	})
	return clipboard
}

// SetClipboard sets the string content of the system clipboard.
func (w *Window) SetClipboard(str string) {
	w.th.Call(func() {
		w.window.SetClipboardString(str)
	})
}

// KeysPressed returns a map indicating the state of each key, where the key is the Button and the value is a boolean.
func (w *Window) KeysPressed() map[Button]bool {
	return w.keysPressed
}

// Pressed checks if the specified button is currently pressed and returns true if it is, otherwise false.
func (w *Window) Pressed(button Button) bool {
	return w.currInp.buttons[button]
}

// JustPressed returns true if the specified button was just pressed during the most recent input update cycle.
func (w *Window) JustPressed(button Button) bool {
	return w.pressEvents[button]
}

// JustReleased checks if the specified button was released during the last input update.
func (w *Window) JustReleased(button Button) bool {
	return w.releaseEvents[button]
}

// Repeated checks if the given button is currently being held in a repeated state.
func (w *Window) Repeated(button Button) bool {
	return w.currInp.repeat[button]
}

// MousePosition returns the current position of the mouse cursor relative to the window as an XY struct.
func (w *Window) MousePosition() XY {
	return w.currInp.mouse
}

// MousePreviousPosition returns the mouse position recorded during the previous input update cycle.
func (w *Window) MousePreviousPosition() XY {
	return w.prevInp.mouse
}

// SetMousePosition sets the mouse position within the window bounds based on the provided XY coordinates.
func (w *Window) SetMousePosition(v XY) {
	w.th.Call(func() {
		if (v.X >= 0 && v.X <= w.bounds.W()) &&
			(v.Y >= 0 && v.Y <= w.bounds.H()) {
			w.window.SetCursorPos(
				v.X+w.bounds.Min.X,
				(w.bounds.H()-v.Y)+w.bounds.Min.Y,
			)
			w.prevInp.mouse = v
			w.currInp.mouse = v
			w.tempInp.mouse = v
		}
	})
}

// MouseInsideWindow returns true if the mouse cursor is inside the bounds of the window, otherwise false.
func (w *Window) MouseInsideWindow() bool {
	return w.cursorInsideWindow
}

// MouseScroll returns the accumulated mouse scroll offset as an XY vector.
func (w *Window) MouseScroll() XY {
	return w.currInp.scroll
}

// Typed returns the current string input typed by the user in the window.
func (w *Window) Typed() string {
	return w.currInp.typed
}

// initInput initializes input handling by setting up GLFW input callbacks for mouse, keyboard, cursor, scroll, and character events.
func (w *Window) initInput() {
	w.th.Call(func() {
		w.window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
			switch action {
			case glfw.Press:
				w.tempPressEvents[button] = true
				w.tempInp.buttons[button] = true
			case glfw.Release:
				w.tempReleaseEvents[button] = true
				w.tempInp.buttons[button] = false
			}
		})

		w.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
			if key == glfw.KeyUnknown {
				return
			}
			switch action {
			case glfw.Press:
				w.keysPressed[Button(key)] = true
				w.tempPressEvents[key] = true
				w.tempInp.buttons[key] = true
			case glfw.Release:
				delete(w.keysPressed, Button(key))
				w.tempReleaseEvents[key] = true
				w.tempInp.buttons[key] = false
			case glfw.Repeat:
				w.keysPressed[Button(key)] = true
				w.tempInp.repeat[key] = true
			}
		})

		//TODO cursorInsideWindow
		//w.cursorInsideWindow
		//x, y := w.window.GetCursorPos()
		//w.window.GetPos()
		//w.window.GetSize()

		w.window.SetCursorEnterCallback(func(_ *glfw.Window, entered bool) {
			w.cursorInsideWindow = entered
		})

		w.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
			w.tempInp.mouse = MakeVec(
				x+w.bounds.Min.X,
				(w.bounds.H()-y)+w.bounds.Min.Y,
			)
		})

		w.window.SetScrollCallback(func(_ *glfw.Window, xOff, yOff float64) {
			w.tempInp.scroll.X += xOff
			w.tempInp.scroll.Y += yOff
		})

		w.window.SetCharCallback(func(_ *glfw.Window, r rune) {
			w.tempInp.typed += string(r)
		})
	})
}

// UpdateInputAndSwap updates input state, swaps buffers, and polls events for a window, with optional vertical sync control.
func (w *Window) UpdateInputAndSwap() {
	w.th.Call(func() {
		w.begin()
		if w.vsync {
			glfw.SwapInterval(1)
		} else {
			glfw.SwapInterval(0)
		}
		w.window.SwapBuffers()
		glfw.PollEvents()
	})
	w.doUpdateInput()
}

// UpdateInputWait handles input event processing with an optional timeout duration to control wait behavior.
func (w *Window) UpdateInputWait(timeout time.Duration) {
	w.th.Call(func() {
		if timeout <= 0 {
			glfw.WaitEvents()
		} else {
			glfw.WaitEventsTimeout(timeout.Seconds())
		}
	})
	w.doUpdateInput()
}

// doUpdateInput updates the current input state for the window, including keyboard, mouse, and joystick data.
func (w *Window) doUpdateInput() {
	//keyboard
	w.prevInp = w.currInp
	w.currInp = w.tempInp
	//w.keysPressed = w.tempKeysPressed
	w.pressEvents = w.tempPressEvents
	w.releaseEvents = w.tempReleaseEvents
	// Clear last frame's temporary status
	//w.tempKeysPressed = []Button{}
	w.tempPressEvents = [KeyLast + 1]bool{}
	w.tempReleaseEvents = [KeyLast + 1]bool{}
	w.tempInp.repeat = [KeyLast + 1]bool{}
	w.tempInp.scroll = ZV
	w.tempInp.typed = ""
	//joysticks
	for js := Joystick1; js <= JoystickLast; js++ {
		joystickPresent := glfw.Joystick(js).Present()
		w.tempJoy.connected[js] = joystickPresent
		if joystickPresent {
			if glfw.Joystick(js).IsGamepad() {
				gamepadInputs := glfw.Joystick(js).GetGamepadState()
				w.tempJoy.buttons[js] = gamepadInputs.Buttons[:]
				w.tempJoy.axis[js] = gamepadInputs.Axes[:]
			} else {
				w.tempJoy.buttons[js] = glfw.Joystick(js).GetButtons()
				w.tempJoy.axis[js] = glfw.Joystick(js).GetAxes()
			}
			if !w.currJoy.connected[js] {
				w.tempJoy.name[js] = glfw.Joystick(js).GetName()
			} else {
				w.tempJoy.name[js] = w.currJoy.name[js]
			}
		} else {
			w.tempJoy.buttons[js] = []glfw.Action{}
			w.tempJoy.axis[js] = []float32{}
			w.tempJoy.name[js] = ""
		}
	}
	w.prevJoy = w.currJoy
	w.currJoy = w.tempJoy
}

// JoystickPresent checks if the specified joystick is currently connected to the system.
func (w *Window) JoystickPresent(js Joystick) bool {
	return w.currJoy.connected[js]
}

// JoystickName returns the name of the specified joystick.
func (w *Window) JoystickName(js Joystick) string {
	return w.currJoy.name[js]
}

// JoystickButtonCount returns the number of buttons available on the specified joystick.
func (w *Window) JoystickButtonCount(js Joystick) int {
	return len(w.currJoy.buttons[js])
}

// JoystickAxisCount returns the number of axes available on the specified joystick.
func (w *Window) JoystickAxisCount(js Joystick) int {
	return len(w.currJoy.axis[js])
}

// JoystickPressed checks if the specified gamepad button on the given joystick is currently being pressed.
func (w *Window) JoystickPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button))
}

// JoystickJustPressed checks if a specific joystick button was just pressed down during the current frame.
func (w *Window) JoystickJustPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button)) && !w.prevJoy.getButton(js, int(button))
}

// JoystickJustReleased checks if a joystick button was released during the last input update. Returns true if just released.
func (w *Window) JoystickJustReleased(js Joystick, button GamepadButton) bool {
	return !w.currJoy.getButton(js, int(button)) && w.prevJoy.getButton(js, int(button))
}

// JoystickAxis retrieves the current value of the specified joystick axis, returning it as a floating-point value.
func (w *Window) JoystickAxis(js Joystick, axis GamepadAxis) float64 {
	return w.currJoy.getAxis(js, int(axis))
}

// doPrepare initializes and configures the GLFW window based on the provided configuration settings.
func (w *Window) doPrepare() error {
	bool2int := map[bool]int{true: glfw.True, false: glfw.False}

	if !w.cfg.CheckSampleMSAA() {
		return fmt.Errorf("invalid value '%v' for SamplesMSAA", w.cfg.SamplesMSAA)
	}

	w.bounds = w.cfg.Bounds

	err := w.th.CallErr(func() error {
		var err error
		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.Resizable, bool2int[w.cfg.Resizable])
		glfw.WindowHint(glfw.Decorated, bool2int[!w.cfg.Undecorated])
		glfw.WindowHint(glfw.Floating, bool2int[w.cfg.AlwaysOnTop])
		glfw.WindowHint(glfw.AutoIconify, bool2int[!w.cfg.NoIconify])
		glfw.WindowHint(glfw.TransparentFramebuffer, bool2int[w.cfg.TransparentFramebuffer])
		glfw.WindowHint(glfw.Maximized, bool2int[w.cfg.Maximized])
		glfw.WindowHint(glfw.Visible, bool2int[!w.cfg.Invisible])
		glfw.WindowHint(glfw.Samples, w.cfg.SamplesMSAA)
		if w.cfg.Position.X != 0 || w.cfg.Position.Y != 0 {
			glfw.WindowHint(glfw.Visible, glfw.False)
		}
		var share *glfw.Window
		if currWin != nil {
			share = currWin.window
		}
		_, _, width, height := w.cfg.Bounds.Bounds()
		w.window, err = glfw.CreateWindow(int(width), int(height), w.cfg.Title, nil, share)
		if err != nil {
			return err
		}
		if w.cfg.Position.X != 0 || w.cfg.Position.Y != 0 {
			w.window.SetPos(int(w.cfg.Position.X), int(w.cfg.Position.Y))
			w.window.Show()
		}
		// enter the OpenGL context
		w.begin()
		w.th.Init(w.cfg.DisableScissorTest)
		w.end()

		return nil
	})
	if err != nil {
		return errors.New("creating window failed")
	}

	//if len(cfg.Icon) > 0 {
	//	images := make([]image.Image, len(cfg.Icon))
	//	for i, icon := range cfg.Icon {
	//		pic := NewPictureRGBAFromPicture(icon)
	//		fmt.Println(pic, i)
	//		images[i] = pic.Image()
	//	}
	//	w.th.Call(func() {
	//		w.window.SetIcon(images)
	//	})
	//}

	w.SetVSync(w.cfg.VSync)

	w.initInput()

	w.SetMonitor(w.cfg.Monitor)

	runtime.SetFinalizer(w, (*Window).Destroy)

	return nil
}

// doRun executes the main loop, updating input, handling user interactions, and rendering frames until the window is closed.
func (w *Window) doRun() {
	if err := w.doPrepare(); err != nil {
		panic(err)
		return
	}

	if err := w.th.CallErr(func() error {
		return w.renderSetupFn()
	}); err != nil {
		panic(err)
		return
	}

	mouseConnected := true
	for !w.Closed() {
		w.th.Call(func() {
			w.Begin()
			fbW, fbH := w.GetFramebufferSize()
			w.renderStartFn(fbW, fbH)
		})

		if mouseConnected && w.MouseInsideWindow() {
			mousePos := w.MousePosition()
			mousePrevPos := w.MousePreviousPosition()
			if mousePos.X != mousePrevPos.X || mousePos.Y != mousePrevPos.Y {
				mouseX := mousePos.X - mousePrevPos.X
				mouseY := mousePos.Y - mousePrevPos.Y
				w.playerMouseMoveFn(mouseX, mouseY)
			}
		}

		var up, down, left, right bool

		if scroll := w.MouseScroll(); scroll.Y != 0 {
			if scroll.Y > 0 {
				up = true
			} else {
				down = true
			}
		}

		var impulse = 0.06
		for v := range w.KeysPressed() {
			switch v {
			case KeyEscape:
				return
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
			case KeyLeft:
				left = true
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
		if w.JustPressed(KeyM) {
			mouseConnected = !mouseConnected
		}
		if w.JustPressed(KeyN) {
			w.toggleShadowsFn()
		}
		//	if d.win.JustPressed(KeyT) {
		//	d.BuildersUpdate()
		//}
		w.UpdateInputAndSwap()
	}
}
