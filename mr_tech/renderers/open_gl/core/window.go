package core

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/core/executor"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type WindowInput struct {
	mouse XY

	buttons [KeyLast + 1]bool

	repeat [KeyLast + 1]bool

	scroll XY

	typed string
}

type WindowPos struct {
	xPos   int
	yPos   int
	width  int
	height int
}

// Window represents an OpenGL-based window with input handling and state management capabilities.
type Window struct {
	window                           *glfw.Window
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
}

// currWin represents the current active instance of a Window, typically used to manage OpenGL window operations.
var currWin *Window

// NewGLWindow creates a new OpenGL window based on the given WindowConfig and returns a pointer to the Window or an error.
func NewGLWindow(cfg WindowConfig) (*Window, error) {
	bool2int := map[bool]int{
		true:  glfw.True,
		false: glfw.False,
	}

	if !cfg.CheckSampleMSAA() {
		return nil, fmt.Errorf("invalid value '%v' for SamplesMSAA", cfg.SamplesMSAA)
	}

	w := &Window{bounds: cfg.Bounds, cursorVisible: true, keysPressed: make(map[Button]bool)}

	err := executor.Thread.CallErr(func() error {
		var err error
		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.Resizable, bool2int[cfg.Resizable])
		glfw.WindowHint(glfw.Decorated, bool2int[!cfg.Undecorated])
		glfw.WindowHint(glfw.Floating, bool2int[cfg.AlwaysOnTop])
		glfw.WindowHint(glfw.AutoIconify, bool2int[!cfg.NoIconify])
		glfw.WindowHint(glfw.TransparentFramebuffer, bool2int[cfg.TransparentFramebuffer])
		glfw.WindowHint(glfw.Maximized, bool2int[cfg.Maximized])
		glfw.WindowHint(glfw.Visible, bool2int[!cfg.Invisible])
		glfw.WindowHint(glfw.Samples, cfg.SamplesMSAA)
		if cfg.Position.X != 0 || cfg.Position.Y != 0 {
			glfw.WindowHint(glfw.Visible, glfw.False)
		}
		var share *glfw.Window
		if currWin != nil {
			share = currWin.window
		}
		_, _, width, height := intBounds(cfg.Bounds)
		w.window, err = glfw.CreateWindow(width, height, cfg.Title, nil, share)
		if err != nil {
			return err
		}
		if cfg.Position.X != 0 || cfg.Position.Y != 0 {
			w.window.SetPos(int(cfg.Position.X), int(cfg.Position.Y))
			w.window.Show()
		}
		// enter the OpenGL context
		w.begin()
		executor.Thread.Init(cfg.DisableScissorTest)
		w.end()

		return nil
	})
	if err != nil {
		return nil, errors.New("creating window failed")
	}

	//if len(cfg.Icon) > 0 {
	//	images := make([]image.Image, len(cfg.Icon))
	//	for i, icon := range cfg.Icon {
	//		pic := NewPictureRGBAFromPicture(icon)
	//		fmt.Println(pic, i)
	//		images[i] = pic.Image()
	//	}
	//	executor.Thread.Call(func() {
	//		w.window.SetIcon(images)
	//	})
	//}

	w.SetVSync(cfg.VSync)

	w.initInput()
	w.SetMonitor(cfg.Monitor)

	runtime.SetFinalizer(w, (*Window).Destroy)

	return w, nil
}

// Destroy releases all resources associated with the Window and invalidates it.
func (w *Window) Destroy() {
	executor.Thread.Call(func() {
		w.window.Destroy()
	})
}

// ClipboardText retrieves the current string content from the system clipboard associated with the window.
func (w *Window) ClipboardText() string {
	return w.window.GetClipboardString()
}

// SetClipboardText sets the given text to the system clipboard for the associated Window instance.
func (w *Window) SetClipboardText(text string) {
	w.window.SetClipboardString(text)
}

// SetClosed updates the closed state of the window, indicating whether it should be marked for closure.
func (w *Window) SetClosed(closed bool) {
	executor.Thread.Call(func() {
		w.window.SetShouldClose(closed)
	})
}

// Closed checks if the window should close and returns true if it is marked to be closed, otherwise false.
func (w *Window) Closed() bool {
	var closed bool
	executor.Thread.Call(func() {
		closed = w.window.ShouldClose()
	})
	return closed
}

// SetTitle sets the window's title to the specified string.
func (w *Window) SetTitle(title string) {
	executor.Thread.Call(func() {
		w.window.SetTitle(title)
	})
}

// SetBounds updates the window's bounding rectangle and adjusts its size based on the specified dimensions.
func (w *Window) SetBounds(bounds Rect) {
	w.bounds = bounds
	executor.Thread.Call(func() {
		_, _, width, height := intBounds(bounds)
		w.window.SetSize(width, height)
	})
}

// SetPos sets the position of the window to the specified coordinates defined by the XY parameter.
func (w *Window) SetPos(pos XY) {
	executor.Thread.Call(func() {
		left, top := int(pos.X), int(pos.Y)
		w.window.SetPos(left, top)
	})
}

// GetPos retrieves the current position of the Window and returns it as a XY containing the x and y coordinates.
func (w *Window) GetPos() XY {
	var v XY
	executor.Thread.Call(func() {
		x, y := w.window.GetPos()
		v = MakeVec(float64(x), float64(y))
	})
	return v
}

// Bounds retrieves the rectangular dimensions of the current window.
func (w *Window) Bounds() Rect {
	return w.bounds
}

// setFullscreen sets the window to fullscreen mode on the specified monitor.
func (w *Window) setFullscreen(monitor *Monitor) {
	executor.Thread.Call(func() {
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

// setWindowed configures the window to operate in windowed mode with its last known position and size.
func (w *Window) setWindowed() {
	executor.Thread.Call(func() {
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

// SetMonitor sets the monitor for the window, switching between fullscreen and windowed mode as necessary.
func (w *Window) SetMonitor(monitor *Monitor) {
	if w.Monitor() != monitor {
		if monitor != nil {
			w.setFullscreen(monitor)
		} else {
			w.setWindowed()
		}
	}
}

// Monitor returns the Monitor currently associated with the Window or nil if no monitor is connected.
func (w *Window) Monitor() *Monitor {
	var monitor *glfw.Monitor
	executor.Thread.Call(func() {
		monitor = w.window.GetMonitor()
	})
	if monitor == nil {
		return nil
	}
	return &Monitor{
		monitor: monitor,
	}
}

// Focused returns true if the window is currently focused, indicating it has input control.
func (w *Window) Focused() bool {
	var focused bool
	executor.Thread.Call(func() {
		focused = w.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

// SetVSync enables or disables vertical synchronization (VSync) for the window based on the provided boolean parameter.
func (w *Window) SetVSync(vsync bool) {
	w.vsync = vsync
}

// VSync returns the current state of vertical synchronization (VSync) for the Window instance.
func (w *Window) VSync() bool {
	return w.vsync
}

// SetCursorVisible toggles the visibility of the cursor within the window based on the provided boolean parameter.
func (w *Window) SetCursorVisible(visible bool) {
	w.cursorVisible = visible
	executor.Thread.Call(func() {
		if visible {
			w.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		} else {
			w.window.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
		}
	})
}

// SetCursorDisabled disables the visibility and movement of the cursor within the window, placing it in a disabled state.
func (w *Window) SetCursorDisabled() {
	w.cursorVisible = false
	executor.Thread.Call(func() {
		w.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	})
}

// CursorVisible returns the visibility state of the cursor for the Window.
func (w *Window) CursorVisible() bool {
	return w.cursorVisible
}

// Begin initializes the beginning of a rendering operation for the Window instance.
func (w *Window) Begin() {
	w.begin()
}

// GetFramebufferSize retrieves the current width and height of the framebuffer for the Window in pixels.
func (w *Window) GetFramebufferSize() (int, int) {
	framebufferWidth, framebufferHeight := w.window.GetFramebufferSize()
	return framebufferWidth, framebufferHeight
}

// begin sets the current OpenGL context to this window if it is not already set as the current context.
func (w *Window) begin() {
	if currWin != w {
		w.window.MakeContextCurrent()
		currWin = w
	}
}

// end terminates the current operation or context associated with the Window instance.
func (w *Window) end() {
	// nothing, really
}

// Show makes the Window visible by invoking the platform-specific show functionality on the main thread.
func (w *Window) Show() {
	executor.Thread.Call(func() {
		w.window.Show()
	})
}

// Clipboard retrieves the current clipboard content as a string from the window context.
func (w *Window) Clipboard() string {
	var clipboard string
	executor.Thread.Call(func() {
		clipboard = w.window.GetClipboardString()
	})
	return clipboard
}

// SetClipboard sets the specified string to the system clipboard for the current window.
func (w *Window) SetClipboard(str string) {
	executor.Thread.Call(func() {
		w.window.SetClipboardString(str)
	})
}

// KeysPressed returns a map indicating the current state of keys, where the key is the Button and the value is its pressed state.
func (w *Window) KeysPressed() map[Button]bool {
	return w.keysPressed
}

// Pressed checks if the specified button is currently pressed, returning true if it is and false otherwise.
func (w *Window) Pressed(button Button) bool {
	return w.currInp.buttons[button]
}

// JustPressed checks if the specified button was pressed during the current frame and returns true if pressed, false otherwise.
func (w *Window) JustPressed(button Button) bool {
	return w.pressEvents[button]
}

// JustReleased checks if the specified button was just released during the current frame and returns true if so.
func (w *Window) JustReleased(button Button) bool {
	return w.releaseEvents[button]
}

// Repeated checks if the specified button is currently being held down and repeatedly triggered as input.
func (w *Window) Repeated(button Button) bool {
	return w.currInp.repeat[button]
}

// MousePosition returns the current position of the mouse cursor as a XY relative to the Window.
func (w *Window) MousePosition() XY {
	return w.currInp.mouse
}

// MousePreviousPosition retrieves the previous mouse position as a XY relative to the Window instance.
func (w *Window) MousePreviousPosition() XY {
	return w.prevInp.mouse
}

// SetMousePosition updates the mouse cursor's position within the window bounds and synchronizes the internal state.
func (w *Window) SetMousePosition(v XY) {
	executor.Thread.Call(func() {
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

// MouseInsideWindow checks if the mouse cursor is currently inside the window boundary and returns true if it is.
func (w *Window) MouseInsideWindow() bool {
	return w.cursorInsideWindow
}

// MouseScroll retrieves the current scroll offset as a XY. It reflects the accumulated mouse scroll input.
func (w *Window) MouseScroll() XY {
	return w.currInp.scroll
}

// Typed returns the string representation of the most recently typed input within the Window instance.
func (w *Window) Typed() string {
	return w.currInp.typed
}

// initInput initializes input-handling callbacks for mouse, keyboard, cursor, scrolling, and character input events.
func (w *Window) initInput() {
	executor.Thread.Call(func() {
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

// UpdateInputAndSwap updates the input state, swaps the buffers, and polls for window events in the current thread context.
func (w *Window) UpdateInputAndSwap() {
	executor.Thread.Call(func() {
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

// UpdateInputWait processes input events with an optional timeout to wait for new input before proceeding.
func (w *Window) UpdateInputWait(timeout time.Duration) {
	executor.Thread.Call(func() {
		if timeout <= 0 {
			glfw.WaitEvents()
		} else {
			glfw.WaitEventsTimeout(timeout.Seconds())
		}
	})
	w.doUpdateInput()
}

// doUpdateInput synchronizes the current and previous input states, processes input events, and clears temporary input data.
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

// JoystickName retrieves the name of the specified joystick.
func (w *Window) JoystickName(js Joystick) string {
	return w.currJoy.name[js]
}

// JoystickButtonCount returns the number of buttons present on the specified joystick.
func (w *Window) JoystickButtonCount(js Joystick) int {
	return len(w.currJoy.buttons[js])
}

// JoystickAxisCount returns the number of axes available for the specified joystick.
func (w *Window) JoystickAxisCount(js Joystick) int {
	return len(w.currJoy.axis[js])
}

// JoystickPressed checks if a specific button on the given joystick is currently pressed and returns true if pressed.
func (w *Window) JoystickPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button))
}

// JoystickJustPressed checks if a specific joystick button was just pressed during the current frame.
func (w *Window) JoystickJustPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button)) && !w.prevJoy.getButton(js, int(button))
}

// JoystickJustReleased checks if a specific joystick button was just released in the current frame.
func (w *Window) JoystickJustReleased(js Joystick, button GamepadButton) bool {
	return !w.currJoy.getButton(js, int(button)) && w.prevJoy.getButton(js, int(button))
}

// JoystickAxis retrieves the current value of the specified axis on the given joystick.
func (w *Window) JoystickAxis(js Joystick, axis GamepadAxis) float64 {
	return w.currJoy.getAxis(js, int(axis))
}

// intBounds computes the integer bounds of a rectangle by flooring and ceiling its min and max values respectively.
func intBounds(bounds Rect) (int, int, int, int) {
	x0 := int(math.Floor(bounds.Min.X))
	y0 := int(math.Floor(bounds.Min.Y))
	x1 := int(math.Ceil(bounds.Max.X))
	y1 := int(math.Ceil(bounds.Max.Y))
	return x0, y0, x1 - x0, y1 - y0
}
