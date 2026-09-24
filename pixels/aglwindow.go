package pixels

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/markel1974/godoom/pixels/executor"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// WindowConfig contains settings for configuring a window's appearance, behavior, and interaction with the system.
type WindowConfig struct {
	Title string

	//Icon []IPicture

	Bounds Rect

	Position Vec

	Monitor *GLMonitor

	Smooth bool

	Resizable bool

	Undecorated bool

	NoIconify bool

	AlwaysOnTop bool

	TransparentFramebuffer bool

	VSync bool

	Maximized bool

	Invisible bool

	SamplesMSAA int

	DisableScissorTest bool
}

// GLWindow represents an OpenGL-based window with input handling and state management capabilities.
type GLWindow struct {
	window *glfw.Window

	bounds Rect
	//canvas             *GLCanvas
	vsync              bool
	cursorVisible      bool
	cursorInsideWindow bool

	// need to save these to correctly restore a fullscreen window
	restore struct {
		xPos, yPos, width, height int
	}

	prevInp, currInp, tempInp struct {
		mouse   Vec
		buttons [KeyLast + 1]bool
		repeat  [KeyLast + 1]bool
		scroll  Vec
		typed   string
	}

	keysPressed                      map[Button]bool
	pressEvents, tempPressEvents     [KeyLast + 1]bool
	releaseEvents, tempReleaseEvents [KeyLast + 1]bool

	prevJoy, currJoy, tempJoy GLJoystick
}

// currWin represents the current active instance of a GLWindow, typically used to manage OpenGL window operations.
var currWin *GLWindow

// NewGLWindow creates a new OpenGL window based on the given WindowConfig and returns a pointer to the GLWindow or an error.
func NewGLWindow(cfg WindowConfig) (*GLWindow, error) {
	bool2int := map[bool]int{
		true:  glfw.True,
		false: glfw.False,
	}

	w := &GLWindow{bounds: cfg.Bounds, cursorVisible: true, keysPressed: make(map[Button]bool)}

	flag := false
	for _, v := range []int{0, 2, 4, 8, 16} {
		if cfg.SamplesMSAA == v {
			flag = true
			break
		}
	}
	if !flag {
		return nil, fmt.Errorf("invalid value '%v' for msaaSamples", cfg.SamplesMSAA)
	}

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
		gl.Enable(gl.MULTISAMPLE)
		w.end()

		return nil
	})
	if err != nil {
		return nil, errors.New("creating window failed")
	}

	/*
		if len(cfg.Icon) > 0 {
			imgs := make([]image.Image, len(cfg.Icon))
			for i, icon := range cfg.Icon {
				pic := NewPictureRGBAFromPicture(icon)

				fmt.Println(pic, i)
				imgs[i] = pic.Image()
			}
			executor.Thread.Call(func() {
				w.window.SetIcon(imgs)
			})
		}

	*/

	w.SetVSync(cfg.VSync)

	w.initInput()
	w.SetMonitor(cfg.Monitor)

	//w.canvas = NewGLCanvas(cfg.Bounds, cfg.Smooth)

	//w.Update()

	runtime.SetFinalizer(w, (*GLWindow).Destroy)

	return w, nil
}

// Destroy releases all resources associated with the GLWindow and invalidates it.
func (w *GLWindow) Destroy() {
	executor.Thread.Call(func() {
		w.window.Destroy()
	})
}

/*
// Update swaps buffers and polls events. Call this method at the end of each frame.
func (w *GLWindow) Update() {
	w.SwapBuffers()
	w.UpdateInput()
}

*/

// ClipboardText retrieves the current string content from the system clipboard associated with the window.
func (w *GLWindow) ClipboardText() string {
	return w.window.GetClipboardString()
}

// SetClipboardText sets the given text to the system clipboard for the associated GLWindow instance.
func (w *GLWindow) SetClipboardText(text string) {
	w.window.SetClipboardString(text)
}

/*
// SwapBuffers swaps buffers. Call this to swap buffers without polling window events.
// Note that Update invokes SwapBuffers.
func (w *GLWindow) SwapBuffers() {
	executor.Thread.Call(func() {
		_, _, oldW, oldH := intBounds(w.bounds)
		newW, newH := w.window.GetSize()
		w.bounds = w.bounds.ResizedMin(w.bounds.Size().Add(MakeVec(float64(newW-oldW), float64(newH-oldH))))
	})
	w.canvas.SetBounds(w.bounds)

	executor.Thread.Call(func() {
		w.begin()

		framebufferWidth, framebufferHeight := w.window.GetFramebufferSize()
		executor.Bounds(0, 0, framebufferWidth, framebufferHeight)

		executor.Clear(0, 0, 0, 0)
		w.canvas.gf.Frame().Begin()
		w.canvas.gf.Frame().Blit(
			nil,
			0, 0, w.canvas.Texture().Width(), w.canvas.Texture().Height(),
			0, 0, framebufferWidth, framebufferHeight,
		)
		w.canvas.gf.Frame().End()

		if w.vsync {
			glfw.SwapInterval(1)
		} else {
			glfw.SwapInterval(0)
		}
		w.window.SwapBuffers()
		w.end()
	})
}

*/

// SetClosed updates the closed state of the window, indicating whether it should be marked for closure.
func (w *GLWindow) SetClosed(closed bool) {
	executor.Thread.Call(func() {
		w.window.SetShouldClose(closed)
	})
}

// Closed checks if the window should close and returns true if it is marked to be closed, otherwise false.
func (w *GLWindow) Closed() bool {
	var closed bool
	executor.Thread.Call(func() {
		closed = w.window.ShouldClose()
	})
	return closed
}

// SetTitle sets the window's title to the specified string.
func (w *GLWindow) SetTitle(title string) {
	executor.Thread.Call(func() {
		w.window.SetTitle(title)
	})
}

// SetBounds updates the window's bounding rectangle and adjusts its size based on the specified dimensions.
func (w *GLWindow) SetBounds(bounds Rect) {
	w.bounds = bounds
	executor.Thread.Call(func() {
		_, _, width, height := intBounds(bounds)
		w.window.SetSize(width, height)
	})
}

// SetPos sets the position of the window to the specified coordinates defined by the Vec parameter.
func (w *GLWindow) SetPos(pos Vec) {
	executor.Thread.Call(func() {
		left, top := int(pos.X), int(pos.Y)
		w.window.SetPos(left, top)
	})
}

// GetPos retrieves the current position of the GLWindow and returns it as a Vec containing the x and y coordinates.
func (w *GLWindow) GetPos() Vec {
	var v Vec
	executor.Thread.Call(func() {
		x, y := w.window.GetPos()
		v = MakeVec(float64(x), float64(y))
	})
	return v
}

// Bounds retrieves the rectangular dimensions of the current window.
func (w *GLWindow) Bounds() Rect {
	return w.bounds
}

// setFullscreen sets the window to fullscreen mode on the specified monitor.
func (w *GLWindow) setFullscreen(monitor *GLMonitor) {
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
func (w *GLWindow) setWindowed() {
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
func (w *GLWindow) SetMonitor(monitor *GLMonitor) {
	if w.Monitor() != monitor {
		if monitor != nil {
			w.setFullscreen(monitor)
		} else {
			w.setWindowed()
		}
	}
}

// Monitor returns the GLMonitor currently associated with the GLWindow or nil if no monitor is connected.
func (w *GLWindow) Monitor() *GLMonitor {
	var monitor *glfw.Monitor
	executor.Thread.Call(func() {
		monitor = w.window.GetMonitor()
	})
	if monitor == nil {
		return nil
	}
	return &GLMonitor{
		monitor: monitor,
	}
}

// Focused returns true if the window is currently focused, indicating it has input control.
func (w *GLWindow) Focused() bool {
	var focused bool
	executor.Thread.Call(func() {
		focused = w.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

// SetVSync enables or disables vertical synchronization (VSync) for the window based on the provided boolean parameter.
func (w *GLWindow) SetVSync(vsync bool) {
	w.vsync = vsync
}

// VSync returns the current state of vertical synchronization (VSync) for the GLWindow instance.
func (w *GLWindow) VSync() bool {
	return w.vsync
}

// SetCursorVisible toggles the visibility of the cursor within the window based on the provided boolean parameter.
func (w *GLWindow) SetCursorVisible(visible bool) {
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
func (w *GLWindow) SetCursorDisabled() {
	w.cursorVisible = false
	executor.Thread.Call(func() {
		w.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	})
}

// CursorVisible returns the visibility state of the cursor for the GLWindow.
func (w *GLWindow) CursorVisible() bool {
	return w.cursorVisible
}

// Begin initializes the beginning of a rendering operation for the GLWindow instance.
func (w *GLWindow) Begin() {
	w.begin()
}

// GetFramebufferSize retrieves the current width and height of the framebuffer for the GLWindow in pixels.
func (w *GLWindow) GetFramebufferSize() (int, int) {
	framebufferWidth, framebufferHeight := w.window.GetFramebufferSize()
	return framebufferWidth, framebufferHeight
}

// begin sets the current OpenGL context to this window if it is not already set as the current context.
func (w *GLWindow) begin() {
	if currWin != w {
		w.window.MakeContextCurrent()
		currWin = w
	}
}

// end terminates the current operation or context associated with the GLWindow instance.
func (w *GLWindow) end() {
	// nothing, really
}

// Show makes the GLWindow visible by invoking the platform-specific show functionality on the main thread.
func (w *GLWindow) Show() {
	executor.Thread.Call(func() {
		w.window.Show()
	})
}

// Clipboard retrieves the current clipboard content as a string from the window context.
func (w *GLWindow) Clipboard() string {
	var clipboard string
	executor.Thread.Call(func() {
		clipboard = w.window.GetClipboardString()
	})
	return clipboard
}

// SetClipboard sets the specified string to the system clipboard for the current window.
func (w *GLWindow) SetClipboard(str string) {
	executor.Thread.Call(func() {
		w.window.SetClipboardString(str)
	})
}

// KeysPressed returns a map indicating the current state of keys, where the key is the Button and the value is its pressed state.
func (w *GLWindow) KeysPressed() map[Button]bool {
	return w.keysPressed
}

// Pressed checks if the specified button is currently pressed, returning true if it is and false otherwise.
func (w *GLWindow) Pressed(button Button) bool {
	return w.currInp.buttons[button]
}

// JustPressed checks if the specified button was pressed during the current frame and returns true if pressed, false otherwise.
func (w *GLWindow) JustPressed(button Button) bool {
	return w.pressEvents[button]
}

// JustReleased checks if the specified button was just released during the current frame and returns true if so.
func (w *GLWindow) JustReleased(button Button) bool {
	return w.releaseEvents[button]
}

// Repeated checks if the specified button is currently being held down and repeatedly triggered as input.
func (w *GLWindow) Repeated(button Button) bool {
	return w.currInp.repeat[button]
}

// MousePosition returns the current position of the mouse cursor as a Vec relative to the GLWindow.
func (w *GLWindow) MousePosition() Vec {
	return w.currInp.mouse
}

// MousePreviousPosition retrieves the previous mouse position as a Vec relative to the GLWindow instance.
func (w *GLWindow) MousePreviousPosition() Vec {
	return w.prevInp.mouse
}

// SetMousePosition updates the mouse cursor's position within the window bounds and synchronizes the internal state.
func (w *GLWindow) SetMousePosition(v Vec) {
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
func (w *GLWindow) MouseInsideWindow() bool {
	return w.cursorInsideWindow
}

// MouseScroll retrieves the current scroll offset as a Vec. It reflects the accumulated mouse scroll input.
func (w *GLWindow) MouseScroll() Vec {
	return w.currInp.scroll
}

// Typed returns the string representation of the most recently typed input within the GLWindow instance.
func (w *GLWindow) Typed() string {
	return w.currInp.typed
}

// initInput initializes input-handling callbacks for mouse, keyboard, cursor, scrolling, and character input events.
func (w *GLWindow) initInput() {
	executor.Thread.Call(func() {
		w.window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
			switch action {
			case glfw.Press:
				w.tempPressEvents[Button(button)] = true
				w.tempInp.buttons[Button(button)] = true
			case glfw.Release:
				w.tempReleaseEvents[Button(button)] = true
				w.tempInp.buttons[Button(button)] = false
			}
		})

		w.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
			if key == glfw.KeyUnknown {
				return
			}
			switch action {
			case glfw.Press:
				w.keysPressed[Button(key)] = true
				w.tempPressEvents[Button(key)] = true
				w.tempInp.buttons[Button(key)] = true
			case glfw.Release:
				delete(w.keysPressed, Button(key))
				w.tempReleaseEvents[Button(key)] = true
				w.tempInp.buttons[Button(key)] = false
			case glfw.Repeat:
				w.keysPressed[Button(key)] = true
				w.tempInp.repeat[Button(key)] = true
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

		w.window.SetScrollCallback(func(_ *glfw.Window, xoff, yoff float64) {
			w.tempInp.scroll.X += xoff
			w.tempInp.scroll.Y += yoff
		})

		w.window.SetCharCallback(func(_ *glfw.Window, r rune) {
			w.tempInp.typed += string(r)
		})
	})
}

// UpdateInput processes input events for the window by polling and executing the necessary update logic.
func (w *GLWindow) UpdateInput() {
	executor.Thread.Call(func() { glfw.PollEvents() })
	w.doUpdateInput()
}

// UpdateInputAndSwap updates the input state, swaps the buffers, and polls for window events in the current thread context.
func (w *GLWindow) UpdateInputAndSwap() {
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
func (w *GLWindow) UpdateInputWait(timeout time.Duration) {
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
func (w *GLWindow) doUpdateInput() {
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

	w.updateJoystickInput()
}

// JoystickPresent checks if the specified joystick is currently connected to the system.
func (w *GLWindow) JoystickPresent(js Joystick) bool {
	return w.currJoy.connected[js]
}

// JoystickName retrieves the name of the specified joystick.
func (w *GLWindow) JoystickName(js Joystick) string {
	return w.currJoy.name[js]
}

// JoystickButtonCount returns the number of buttons present on the specified joystick.
func (w *GLWindow) JoystickButtonCount(js Joystick) int {
	return len(w.currJoy.buttons[js])
}

// JoystickAxisCount returns the number of axes available for the specified joystick.
func (w *GLWindow) JoystickAxisCount(js Joystick) int {
	return len(w.currJoy.axis[js])
}

// JoystickPressed checks if a specific button on the given joystick is currently pressed and returns true if pressed.
func (w *GLWindow) JoystickPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button))
}

// JoystickJustPressed checks if a specific joystick button was just pressed during the current frame.
func (w *GLWindow) JoystickJustPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button)) && !w.prevJoy.getButton(js, int(button))
}

// JoystickJustReleased checks if a specific joystick button was just released in the current frame.
func (w *GLWindow) JoystickJustReleased(js Joystick, button GamepadButton) bool {
	return !w.currJoy.getButton(js, int(button)) && w.prevJoy.getButton(js, int(button))
}

// JoystickAxis retrieves the current value of the specified axis on the given joystick.
func (w *GLWindow) JoystickAxis(js Joystick, axis GamepadAxis) float64 {
	return w.currJoy.getAxis(js, int(axis))
}

// updateJoystickInput updates the state of all connected joysticks and gamepads, including buttons, axes, and connection status.
func (w *GLWindow) updateJoystickInput() {
	for js := Joystick1; js <= JoystickLast; js++ {
		// Determine and store if the joystick was connected
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
				// The joystick was recently connected, we get the name
				w.tempJoy.name[js] = glfw.Joystick(js).GetName()
			} else {
				// Use the name from the previous one
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

// intBounds computes the integer bounds of a rectangle by flooring and ceiling its min and max values respectively.
func intBounds(bounds Rect) (x, y, w, h int) {
	x0 := int(math.Floor(bounds.Min.X))
	y0 := int(math.Floor(bounds.Min.Y))
	x1 := int(math.Ceil(bounds.Max.X))
	y1 := int(math.Ceil(bounds.Max.Y))
	return x0, y0, x1 - x0, y1 - y0
}
