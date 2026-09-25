package core

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/core/executor"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// WindowInput represents the input state of a window, including mouse position, buttons, scrolling, and typed text.
type WindowInput struct {
	mouse XY

	buttons [KeyLast + 1]bool

	repeat [KeyLast + 1]bool

	scroll XY

	typed string
}

// WindowPos represents the position and size of a window, defined by its x and y coordinates, width, and height.
type WindowPos struct {
	xPos int

	yPos int

	width int

	height int
}

// Window represents a graphical window with properties for input handling, display settings, and viewport bounds.
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

// currWin represents the currently active window in the application, ensuring context-specific operations.
var currWin *Window

// NewGLWindow creates and initializes a new OpenGL-based window with the specified configuration, returning a Window instance.
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
		_, _, width, height := cfg.Bounds.Bounds()
		w.window, err = glfw.CreateWindow(int(width), int(height), cfg.Title, nil, share)
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

// Destroy releases the resources associated with the window and cleans up its context.
func (w *Window) Destroy() {
	executor.Thread.Call(func() {
		w.window.Destroy()
	})
}

// ClipboardText retrieves the current text from the system clipboard.
func (w *Window) ClipboardText() string {
	return w.window.GetClipboardString()
}

// SetClipboardText sets the specified text to the system clipboard associated with the window clipboard context.
func (w *Window) SetClipboardText(text string) {
	w.window.SetClipboardString(text)
}

// SetClosed sets the closed state of the window, indicating whether it should be closed.
func (w *Window) SetClosed(closed bool) {
	executor.Thread.Call(func() {
		w.window.SetShouldClose(closed)
	})
}

// Closed checks if the window has been flagged for closure and returns true if it is.
func (w *Window) Closed() bool {
	var closed bool
	executor.Thread.Call(func() {
		closed = w.window.ShouldClose()
	})
	return closed
}

// SetTitle updates the title of the window with the specified string.
func (w *Window) SetTitle(title string) {
	executor.Thread.Call(func() {
		w.window.SetTitle(title)
	})
}

// SetBounds updates the Window's bounds and resizes the GLFW window to match the specified dimensions.
func (w *Window) SetBounds(bounds Rect) {
	w.bounds = bounds
	executor.Thread.Call(func() {
		_, _, width, height := bounds.Bounds()
		w.window.SetSize(int(width), int(height))
	})
}

// SetPos sets the position of the window on the screen using the specified XY coordinates.
func (w *Window) SetPos(pos XY) {
	executor.Thread.Call(func() {
		left, top := int(pos.X), int(pos.Y)
		w.window.SetPos(left, top)
	})
}

// GetPos retrieves the current position of the window as an XY struct.
func (w *Window) GetPos() XY {
	var v XY
	executor.Thread.Call(func() {
		x, y := w.window.GetPos()
		v = MakeVec(float64(x), float64(y))
	})
	return v
}

// Bounds returns the boundaries of the window as a Rect structure.
func (w *Window) Bounds() Rect {
	return w.bounds
}

// setFullscreen sets the window to fullscreen mode using the provided monitor. Saves the current window state for restoration.
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

// setWindowed changes the window to windowed mode using its previously stored position and size.
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

// SetMonitor sets the specified monitor for the window. If monitor is nil, the window is set to windowed mode.
func (w *Window) SetMonitor(monitor *Monitor) {
	if w.Monitor() != monitor {
		if monitor != nil {
			w.setFullscreen(monitor)
		} else {
			w.setWindowed()
		}
	}
}

// Monitor retrieves the monitor associated with the window, or nil if the window is not in fullscreen mode.
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

// Focused checks if the window is currently focused and returns true if it is.
func (w *Window) Focused() bool {
	var focused bool
	executor.Thread.Call(func() {
		focused = w.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

// SetVSync enables or disables vertical synchronization (VSync) for the window.
func (w *Window) SetVSync(vsync bool) {
	w.vsync = vsync
}

// VSync returns the current vertical sync (VSync) setting of the window.
func (w *Window) VSync() bool {
	return w.vsync
}

// SetCursorVisible sets the visibility of the cursor in the window based on the provided boolean value.
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

// SetCursorDisabled disables the cursor and locks it to the center of the window, making it invisible to the user.
func (w *Window) SetCursorDisabled() {
	w.cursorVisible = false
	executor.Thread.Call(func() {
		w.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	})
}

// CursorVisible returns whether the cursor is currently visible within the window.
func (w *Window) CursorVisible() bool {
	return w.cursorVisible
}

// Begin prepares the window for rendering by making it the current OpenGL context.
func (w *Window) Begin() {
	w.begin()
}

// GetFramebufferSize retrieves the width and height of the window's framebuffer in pixels.
func (w *Window) GetFramebufferSize() (int, int) {
	framebufferWidth, framebufferHeight := w.window.GetFramebufferSize()
	return framebufferWidth, framebufferHeight
}

// begin sets the current OpenGL context to the window instance if it's not already the active context.
func (w *Window) begin() {
	if currWin != w {
		w.window.MakeContextCurrent()
		currWin = w
	}
}

// end finalizes the current frame or operation within the window context. It is typically a no-op in this implementation.
func (w *Window) end() {
	// nothing, really
}

// Show makes the window visible if it is currently hidden or not already displayed.
func (w *Window) Show() {
	executor.Thread.Call(func() {
		w.window.Show()
	})
}

// Clipboard retrieves the current string content from the system clipboard associated with the window.
func (w *Window) Clipboard() string {
	var clipboard string
	executor.Thread.Call(func() {
		clipboard = w.window.GetClipboardString()
	})
	return clipboard
}

// SetClipboard sets the current clipboard content to the specified string.
func (w *Window) SetClipboard(str string) {
	executor.Thread.Call(func() {
		w.window.SetClipboardString(str)
	})
}

// KeysPressed returns a map indicating the current pressed state of each Button.
func (w *Window) KeysPressed() map[Button]bool {
	return w.keysPressed
}

// Pressed returns true if the specified button is currently being pressed.
func (w *Window) Pressed(button Button) bool {
	return w.currInp.buttons[button]
}

// JustPressed checks if the specified button was pressed during the current frame.
func (w *Window) JustPressed(button Button) bool {
	return w.pressEvents[button]
}

// JustReleased checks if the specified button was just released during the current frame.
func (w *Window) JustReleased(button Button) bool {
	return w.releaseEvents[button]
}

// Repeated checks if the specified button is currently being held down in a repeat state for the current frame.
func (w *Window) Repeated(button Button) bool {
	return w.currInp.repeat[button]
}

// MousePosition returns the current position of the mouse cursor relative to the window's coordinate system.
func (w *Window) MousePosition() XY {
	return w.currInp.mouse
}

// MousePreviousPosition returns the previous recorded mouse position as an XY coordinate in window space.
func (w *Window) MousePreviousPosition() XY {
	return w.prevInp.mouse
}

// SetMousePosition updates the mouse cursor position within the window bounds and adjusts internal mouse position states.
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

// MouseInsideWindow returns true if the mouse cursor is currently inside the window boundaries, false otherwise.
func (w *Window) MouseInsideWindow() bool {
	return w.cursorInsideWindow
}

// MouseScroll returns the current mouse scroll offset as an XY coordinate.
func (w *Window) MouseScroll() XY {
	return w.currInp.scroll
}

// Typed returns the string of characters typed during the current frame.
func (w *Window) Typed() string {
	return w.currInp.typed
}

// initInput initializes input handling for the window, setting up callbacks for mouse, keyboard, cursor, and scroll events.
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

// UpdateInputAndSwap updates input states, manages VSync, swaps buffers, polls events, and prepares for the next frame.
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

// UpdateInputWait processes input events and updates internal state, with an optional timeout for waiting on new events.
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

// doUpdateInput synchronizes the current input state with the temporary input state and processes joystick inputs.
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

// JoystickPresent checks if a specific joystick is currently connected.
func (w *Window) JoystickPresent(js Joystick) bool {
	return w.currJoy.connected[js]
}

// JoystickName returns the name of the specified joystick or controller.
func (w *Window) JoystickName(js Joystick) string {
	return w.currJoy.name[js]
}

// JoystickButtonCount returns the number of buttons available on the specified joystick.
func (w *Window) JoystickButtonCount(js Joystick) int {
	return len(w.currJoy.buttons[js])
}

// JoystickAxisCount returns the number of axes available for the specified joystick.
func (w *Window) JoystickAxisCount(js Joystick) int {
	return len(w.currJoy.axis[js])
}

// JoystickPressed checks if a specific button on the given joystick is currently pressed and returns true if it is.
func (w *Window) JoystickPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button))
}

// JoystickJustPressed returns true if the specified joystick button was just pressed in the current frame.
func (w *Window) JoystickJustPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button)) && !w.prevJoy.getButton(js, int(button))
}

// JoystickJustReleased returns true if the specified joystick button was released in the current frame.
func (w *Window) JoystickJustReleased(js Joystick, button GamepadButton) bool {
	return !w.currJoy.getButton(js, int(button)) && w.prevJoy.getButton(js, int(button))
}

// JoystickAxis returns the current state of the given joystick axis as a float64, ranging typically from -1.0 to 1.0.
func (w *Window) JoystickAxis(js Joystick, axis GamepadAxis) float64 {
	return w.currJoy.getAxis(js, int(axis))
}
