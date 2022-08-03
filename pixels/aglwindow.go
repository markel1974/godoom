package pixels

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"runtime"
	"time"

	"github.com/markel1974/godoom/pixels/executor"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type WindowConfig struct {
	Title string

	Icon []IPicture

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
}

type GLWindow struct {
	window *glfw.Window

	bounds             Rect
	canvas             *GLCanvas
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

var currWin *GLWindow

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
		executor.Init()
		gl.Enable(gl.MULTISAMPLE)
		w.end()

		return nil
	})
	if err != nil {
		return nil, errors.New("creating window failed")
	}

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

	w.SetVSync(cfg.VSync)

	w.initInput()
	w.SetMonitor(cfg.Monitor)

	w.canvas = NewGLCanvas(cfg.Bounds, cfg.Smooth)

	w.Update()

	runtime.SetFinalizer(w, (*GLWindow).Destroy)

	return w, nil
}

// Destroy destroys the GLWindow. The GLWindow can't be used any further.
func (w *GLWindow) Destroy() {
	executor.Thread.Call(func() {
		w.window.Destroy()
	})
}

// Update swaps buffers and polls events. Call this method at the end of each frame.
func (w *GLWindow) Update() {
	w.SwapBuffers()
	w.UpdateInput()
}

// ClipboardText returns the current value of the systems clipboard.
func (w *GLWindow) ClipboardText() string {
	return w.window.GetClipboardString()
}

// SetClipboardText passes the given string to the underlying glfw window to set the systems clipboard.
func (w *GLWindow) SetClipboardText(text string) {
	w.window.SetClipboardString(text)
}

// SwapBuffers swaps buffers. Call this to swap buffers without polling window events.
// Note that Update invokes SwapBuffers.
func (w *GLWindow) SwapBuffers() {
	executor.Thread.Call(func() {
		_, _, oldW, oldH := intBounds(w.bounds)
		newW, newH := w.window.GetSize()
		w.bounds = w.bounds.ResizedMin(w.bounds.Size().Add(MakeVec(
			float64(newW-oldW),
			float64(newH-oldH),
		)))
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

// SetClosed sets the closed flag of the GLWindow.
//
// This is useful when overriding the user's attempt to close the GLWindow, or just to close the
// GLWindow from within the program.
func (w *GLWindow) SetClosed(closed bool) {
	executor.Thread.Call(func() {
		w.window.SetShouldClose(closed)
	})
}

// Closed returns the closed flag of the GLWindow, which reports whether the GLWindow should be closed.
//
// The closed flag is automatically set when a user attempts to close the GLWindow.
func (w *GLWindow) Closed() bool {
	var closed bool
	executor.Thread.Call(func() {
		closed = w.window.ShouldClose()
	})
	return closed
}

// SetTitle changes the title of the GLWindow.
func (w *GLWindow) SetTitle(title string) {
	executor.Thread.Call(func() {
		w.window.SetTitle(title)
	})
}

// SetBounds sets the bounds of the GLWindow in pixels. Bounds can be fractional, but the actual size
// of the window will be rounded to integers.
func (w *GLWindow) SetBounds(bounds Rect) {
	w.bounds = bounds
	executor.Thread.Call(func() {
		_, _, width, height := intBounds(bounds)
		w.window.SetSize(width, height)
	})
}

// SetPos sets the position, in screen coordinates, of the upper-left corner
// of the client area of the window. Position can be fractional, but the actual position
// of the window will be rounded to integers.
//
// If it is a full screen window, this function does nothing.
func (w *GLWindow) SetPos(pos Vec) {
	executor.Thread.Call(func() {
		left, top := int(pos.X), int(pos.Y)
		w.window.SetPos(left, top)
	})
}

// GetPos gets the position, in screen coordinates, of the upper-left corner
// of the client area of the window. The position is rounded to integers.
func (w *GLWindow) GetPos() Vec {
	var v Vec
	executor.Thread.Call(func() {
		x, y := w.window.GetPos()
		v = MakeVec(float64(x), float64(y))
	})
	return v
}

// Bounds returns the current bounds of the GLWindow.
func (w *GLWindow) Bounds() Rect {
	return w.bounds
}

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

// SetMonitor sets the GLWindow fullscreen on the given GLMonitor. If the GLMonitor is nil, the GLWindow
// will be restored to windowed state instead.
//
// The GLWindow will be automatically set to the GLMonitor's resolution. If you want a different
// resolution, you will need to set it manually with SetBounds method.
func (w *GLWindow) SetMonitor(monitor *GLMonitor) {
	if w.Monitor() != monitor {
		if monitor != nil {
			w.setFullscreen(monitor)
		} else {
			w.setWindowed()
		}
	}
}

// GLMonitor returns a monitor the GLWindow is fullscreen on. If the GLWindow is not fullscreen, this
// function returns nil.
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

// Focused returns true if the GLWindow has input focus.
func (w *GLWindow) Focused() bool {
	var focused bool
	executor.Thread.Call(func() {
		focused = w.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

// SetVSync sets whether the GLWindow's Update should synchronize with the monitor refresh rate.
func (w *GLWindow) SetVSync(vsync bool) {
	w.vsync = vsync
}

// VSync returns whether the GLWindow is set to synchronize with the monitor refresh rate.
func (w *GLWindow) VSync() bool {
	return w.vsync
}

// SetCursorVisible sets the visibility of the mouse cursor inside the GLWindow client area.
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

// SetCursorDisabled hides the cursor and provides unlimited virtual cursor movement
// make cursor visible using SetCursorVisible
func (w *GLWindow) SetCursorDisabled() {
	w.cursorVisible = false
	executor.Thread.Call(func() {
		w.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	})
}

// CursorVisible returns the visibility status of the mouse cursor.
func (w *GLWindow) CursorVisible() bool {
	return w.cursorVisible
}

// Note: must be called inside the main thread.
func (w *GLWindow) begin() {
	if currWin != w {
		w.window.MakeContextCurrent()
		currWin = w
	}
}

// Note: must be called inside the main thread.
func (w *GLWindow) end() {
	// nothing, really
}

// MakeTriangles generates a specialized copy of the supplied ITriangles that will draw onto this
// GLWindow.
//
// GLWindow supports ITrianglesPosition, ITrianglesColor and ITrianglesPicture.
func (w *GLWindow) MakeTriangles(t ITriangles) ITargetTriangles {
	return w.canvas.MakeTriangles(t)
}

// MakePicture generates a specialized copy of the supplied IPicture that will draw onto this GLWindow.
//
// GLWindow supports IPictureColor.
func (w *GLWindow) MakePicture(p IPicture) ITargetPicture {
	return w.canvas.MakePicture(p)
}

// SetMatrix sets a Matrix that every point will be projected by.
func (w *GLWindow) SetMatrix(m Matrix) {
	w.canvas.SetMatrix(m)
}

// SetColorMask sets a global color mask for the GLWindow.
func (w *GLWindow) SetColorMask(c color.Color) {
	w.canvas.SetColorMask(c)
}

// SetComposeMethod sets a Porter-Duff composition method to be used in the following draws onto
// this GLWindow.
func (w *GLWindow) SetComposeMethod(cmp ComposeMethod) {
	w.canvas.SetComposeMethod(cmp)
}

// SetSmooth sets whether the stretched Pictures drawn onto this GLWindow should be drawn smooth or pixelated.
func (w *GLWindow) SetSmooth(smooth bool) {
	w.canvas.SetSmooth(smooth)
}

// Smooth returns whether the stretched Pictures drawn onto this GLWindow are set to be drawn smooth or pixelated.
func (w *GLWindow) Smooth() bool {
	return w.canvas.Smooth()
}

// Clear clears the GLWindow with a single color.
func (w *GLWindow) Clear(c color.Color) {
	w.canvas.Clear(c)
}

// Color returns the color of the pixel over the given position inside the GLWindow.
func (w *GLWindow) Color(at Vec) RGBA {
	return w.canvas.Color(at)
}

// GLCanvas returns the window's underlying GLCanvas
func (w *GLWindow) Canvas() *GLCanvas {
	return w.canvas
}

// Show makes the window visible, if it was previously hidden. If the window is already visible or is in full screen mode, this function does nothing.
func (w *GLWindow) Show() {
	executor.Thread.Call(func() {
		w.window.Show()
	})
}

// Clipboard returns the contents of the system clipboard.
func (w *GLWindow) Clipboard() string {
	var clipboard string
	executor.Thread.Call(func() {
		clipboard = w.window.GetClipboardString()
	})
	return clipboard
}

// SetClipboardString sets the system clipboard to the specified UTF-8 encoded string.
func (w *GLWindow) SetClipboard(str string) {
	executor.Thread.Call(func() {
		w.window.SetClipboardString(str)
	})
}

func (w *GLWindow) KeysPressed() map[Button]bool {
	return w.keysPressed
}

func (w *GLWindow) Pressed(button Button) bool {
	return w.currInp.buttons[button]
}

// JustPressed returns whether the Button has been pressed in the last frame.
func (w *GLWindow) JustPressed(button Button) bool {
	return w.pressEvents[button]
}

// JustReleased returns whether the Button has been released in the last frame.
func (w *GLWindow) JustReleased(button Button) bool {
	return w.releaseEvents[button]
}

// Repeated returns whether a repeat event has been triggered on button.
//
// Repeat event occurs repeatedly when a button is held down for some time.
func (w *GLWindow) Repeated(button Button) bool {
	return w.currInp.repeat[button]
}

// MousePosition returns the current mouse position in the GLWindow's Bounds.
func (w *GLWindow) MousePosition() Vec {
	return w.currInp.mouse
}

// MousePreviousPosition returns the previous mouse position in the GLWindow's Bounds.
func (w *GLWindow) MousePreviousPosition() Vec {
	return w.prevInp.mouse
}

// SetMousePosition positions the mouse cursor anywhere within the GLWindow's Bounds.
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

// MouseInsideWindow returns true if the mouse position is within the GLWindow's Bounds.
func (w *GLWindow) MouseInsideWindow() bool {
	return w.cursorInsideWindow
}

// MouseScroll returns the mouse scroll amount (in both axes) since the last call to GLWindow.Update.
func (w *GLWindow) MouseScroll() Vec {
	return w.currInp.scroll
}

// Typed returns the text typed on the keyboard since the last call to GLWindow.Update.
func (w *GLWindow) Typed() string {
	return w.currInp.typed
}

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

// UpdateInput polls window events. Call this function to poll window events without swapping buffers. Note that the Update method invokes UpdateInput.
func (w *GLWindow) UpdateInput() {
	executor.Thread.Call(func() { glfw.PollEvents() })
	w.doUpdateInput()
}

// UpdateInputWait blocks until an event is received or a timeout. If timeout is 0
// then it will wait indefinitely
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

// internal input bookkeeping
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

// JoystickPresent returns if the joystick is currently connected.
//
// This API is experimental.
func (w *GLWindow) JoystickPresent(js Joystick) bool {
	return w.currJoy.connected[js]
}

// JoystickName returns the name of the joystick. A disconnected joystick will return an
// empty string.
//
// This API is experimental.
func (w *GLWindow) JoystickName(js Joystick) string {
	return w.currJoy.name[js]
}

// JoystickButtonCount returns the number of buttons a connected joystick has.
//
// This API is experimental.
func (w *GLWindow) JoystickButtonCount(js Joystick) int {
	return len(w.currJoy.buttons[js])
}

// JoystickAxisCount returns the number of axes a connected joystick has.
//
// This API is experimental.
func (w *GLWindow) JoystickAxisCount(js Joystick) int {
	return len(w.currJoy.axis[js])
}

// JoystickPressed returns whether the joystick Button is currently pressed down.
// If the button index is out of range, this will return false.
//
// This API is experimental.
func (w *GLWindow) JoystickPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button))
}

// JoystickJustPressed returns whether the joystick Button has just been pressed down.
// If the button index is out of range, this will return false.
//
// This API is experimental.
func (w *GLWindow) JoystickJustPressed(js Joystick, button GamepadButton) bool {
	return w.currJoy.getButton(js, int(button)) && !w.prevJoy.getButton(js, int(button))
}

// JoystickJustReleased returns whether the joystick Button has just been released up.
// If the button index is out of range, this will return false.
//
// This API is experimental.
func (w *GLWindow) JoystickJustReleased(js Joystick, button GamepadButton) bool {
	return !w.currJoy.getButton(js, int(button)) && w.prevJoy.getButton(js, int(button))
}

// JoystickAxis returns the value of a joystick axis at the last call to GLWindow.Update.
// If the axis index is out of range, this will return 0.
//
// This API is experimental.
func (w *GLWindow) JoystickAxis(js Joystick, axis GamepadAxis) float64 {
	return w.currJoy.getAxis(js, int(axis))
}

// Used internally during GLWindow.UpdateInput to update the state of the joysticks.
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
