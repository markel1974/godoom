package desktop

import "github.com/go-gl/glfw/v3.3/glfw"

// WindowInput tracks per-window keyboard, mouse, and scroll input state between frames.
type WindowInput struct {
	keysPressed map[Button]bool

	mouse XY

	prevMouse XY

	distMouse XY

	pressedActive []bool

	tmpScroll XY

	activeScroll XY

	prevJoy, currJoy, tempJoy GLJoystick

	hasJoystick bool

	mouseMoved bool
}

// NewWindowInput creates and initializes a WindowInput for tracking keyboard, mouse, and scroll state.
func NewWindowInput() *WindowInput {
	return &WindowInput{
		pressedActive: make([]bool, KeyLast+1),
		keysPressed:   make(map[Button]bool),
		hasJoystick:   false,
		mouseMoved:    false,
	}
}

// SetMouse updates the current mouse position to the given x and y coordinates.
func (wi *WindowInput) SetMouse(x float64, y float64) {
	wi.mouse.X = x
	wi.mouse.Y = y
}

// SetScroll adds the given x and y scroll offsets to the pending scroll delta.
func (wi *WindowInput) SetScroll(x float64, y float64) {
	wi.tmpScroll.X += x
	wi.tmpScroll.Y += y
}

// GetScroll returns the current horizontal and vertical scroll offsets.
func (wi *WindowInput) GetScroll() (float64, float64) {
	return wi.activeScroll.X, wi.activeScroll.Y
}

// GetCurrentMouse returns the current mouse position as x and y coordinates.
func (wi *WindowInput) GetCurrentMouse() (float64, float64) {
	return wi.mouse.X, wi.mouse.Y
}

// GetPreviousMouse returns the mouse position recorded during the previous input update.
func (wi *WindowInput) GetPreviousMouse() (float64, float64) {
	return wi.prevMouse.X, wi.prevMouse.Y
}

// GetMouseDist returns the mouse movement delta on the X and Y axes since the previous input reset.
func (wi *WindowInput) GetMouseDist() (float64, float64, bool) {
	return wi.distMouse.X, wi.distMouse.Y, wi.mouseMoved
}

// RepeatButton marks the button as currently pressed without recording a new press event.
func (wi *WindowInput) RepeatButton(b Button) {
	wi.keysPressed[b] = true
}

// PressButton marks b as currently pressed and records it as a new press event.
func (wi *WindowInput) PressButton(b Button) {
	wi.keysPressed[b] = true
	wi.pressedActive[b] = true
}

// ReleaseButton marks b as no longer pressed and clears its active press state.
func (wi *WindowInput) ReleaseButton(b Button) {
	wi.pressedActive[b] = false
	delete(wi.keysPressed, b)
}

// ConsumePressedButton reports whether b was newly pressed and clears that press state if present.
func (wi *WindowInput) ConsumePressedButton(b Button) bool {
	ret := wi.pressedActive[b]
	if ret {
		wi.pressedActive[b] = false
	}
	return ret
}

// Update advances transient input state, updates mouse movement, and clears accumulated scroll for the next update.
func (wi *WindowInput) Update() {
	if wi.mouse.X != wi.prevMouse.X || wi.mouse.Y != wi.prevMouse.Y {
		wi.distMouse.X = wi.mouse.X - wi.prevMouse.X
		wi.distMouse.Y = wi.mouse.Y - wi.prevMouse.Y
		wi.prevMouse.X = wi.mouse.X
		wi.prevMouse.Y = wi.mouse.Y
		wi.mouseMoved = true
	} else {
		wi.distMouse.X = 0
		wi.distMouse.Y = 0
		wi.mouseMoved = false
	}

	wi.activeScroll.X = wi.tmpScroll.X
	wi.activeScroll.Y = wi.tmpScroll.Y
	wi.tmpScroll.X = 0
	wi.tmpScroll.Y = 0

	if wi.hasJoystick {
		for js := Joystick1; js <= JoystickLast; js++ {
			joystickPresent := glfw.Joystick(js).Present()
			wi.tempJoy.connected[js] = joystickPresent
			if joystickPresent {
				if glfw.Joystick(js).IsGamepad() {
					gamepadInputs := glfw.Joystick(js).GetGamepadState()
					wi.tempJoy.buttons[js] = gamepadInputs.Buttons[:]
					wi.tempJoy.axis[js] = gamepadInputs.Axes[:]
				} else {
					wi.tempJoy.buttons[js] = glfw.Joystick(js).GetButtons()
					wi.tempJoy.axis[js] = glfw.Joystick(js).GetAxes()
				}
				if !wi.currJoy.connected[js] {
					wi.tempJoy.name[js] = glfw.Joystick(js).GetName()
				} else {
					wi.tempJoy.name[js] = wi.currJoy.name[js]
				}
			} else {
				wi.tempJoy.buttons[js] = []glfw.Action{}
				wi.tempJoy.axis[js] = []float32{}
				wi.tempJoy.name[js] = ""
			}
		}
		wi.prevJoy = wi.currJoy
		wi.currJoy = wi.tempJoy
	}
}

// JoystickPresent checks if the specified joystick is currently connected to the system.
func (wi *WindowInput) JoystickPresent(js Joystick) bool {
	return wi.currJoy.connected[js]
}

// JoystickName returns the name of the specified joystick.
func (wi *WindowInput) JoystickName(js Joystick) string {
	return wi.currJoy.name[js]
}

// JoystickButtonCount returns the number of buttons available on the specified joystick.
func (wi *WindowInput) JoystickButtonCount(js Joystick) int {
	return len(wi.currJoy.buttons[js])
}

// JoystickAxisCount returns the number of axes available on the specified joystick.
func (wi *WindowInput) JoystickAxisCount(js Joystick) int {
	return len(wi.currJoy.axis[js])
}

// JoystickPressed checks if the specified gamepad button on the given joystick is currently being pressed.
func (wi *WindowInput) JoystickPressed(js Joystick, button GamepadButton) bool {
	return wi.currJoy.getButton(js, int(button))
}

// JoystickJustPressed checks if a specific joystick button was just pressed down during the current frame.
func (wi *WindowInput) JoystickJustPressed(js Joystick, button GamepadButton) bool {
	return wi.currJoy.getButton(js, int(button)) && !wi.prevJoy.getButton(js, int(button))
}

// JoystickJustReleased checks if a joystick button was released during the last input update. Returns true if just released.
func (wi *WindowInput) JoystickJustReleased(js Joystick, button GamepadButton) bool {
	return !wi.currJoy.getButton(js, int(button)) && wi.prevJoy.getButton(js, int(button))
}

// JoystickAxis retrieves the current value of the specified joystick axis, returning it as a floating-point value.
func (wi *WindowInput) JoystickAxis(js Joystick, axis GamepadAxis) float64 {
	return wi.currJoy.getAxis(js, int(axis))
}
