package desktop

import "github.com/go-gl/glfw/v3.3/glfw"

// WindowInput tracks per-window keyboard, mouse, and scroll input state between frames.
type WindowInput struct {
	keysPressed map[Button]bool

	mouse XY

	prevMouse XY

	distMouse XY

	pressedTemp []bool

	pressedActive []bool

	hasPressed bool

	empty []bool

	tmpScroll XY

	activeScroll XY

	prevJoy, currJoy, tempJoy GLJoystick

	hasJoystick bool

	mouseMoved bool

	//releasedTemp []bool

	//released []bool

	//hasReleased bool

	//repeat []bool

	//typed string
}

// NewWindowInput creates and initializes a WindowInput for tracking keyboard, mouse, and scroll state.
func NewWindowInput() *WindowInput {
	return &WindowInput{
		pressedTemp:   make([]bool, KeyLast+1),
		pressedActive: make([]bool, KeyLast+1),
		empty:         make([]bool, KeyLast+1),
		keysPressed:   make(map[Button]bool),
		hasPressed:    false,
		hasJoystick:   false,
		mouseMoved:    false,
		//releasedTemp:    make([]bool, KeyLast+1),
		//released: make([]bool, KeyLast+1),
		//hasReleased: false,
		//repeat:         make([]bool, KeyLast+1),
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

// AddRepeatedButton marks the button as currently pressed without recording a new press event.
func (wi *WindowInput) AddRepeatedButton(b Button) {
	wi.keysPressed[b] = true
}

// AddPressedButton records b as currently pressed and marks it as newly pressed for the next input update.
func (wi *WindowInput) AddPressedButton(b Button) {
	wi.keysPressed[b] = true
	wi.pressedTemp[b] = true
	wi.hasPressed = true
}

// HasPressedButton reports whether b is marked as pressed in the active input state.
func (wi *WindowInput) HasPressedButton(b Button) bool {
	return wi.pressedActive[b]
}

// AddReleasedButton records b as released and removes it from the currently pressed buttons.
func (wi *WindowInput) AddReleasedButton(b Button) {
	wi.pressedActive[b] = false
	delete(wi.keysPressed, b)
	//wi.releasedTemp[b] = true
	//wi.hasReleased = true
}

// HasReleasedButton reports whether b was released during the last input update.
//func (wi *WindowInput) HasReleasedButton(b Button) bool {
//	return wi.released[b]
//}

// Update advances transient input state, updates mouse movement, and clears accumulated scroll for the next update.
func (wi *WindowInput) Update() {
	if wi.hasPressed {
		copy(wi.pressedActive, wi.pressedTemp)
		copy(wi.pressedTemp, wi.empty)
		wi.hasPressed = false
	}
	//if wi.hasReleased {
	//	copy(wi.released, wi.releasedTemp)
	//	copy(wi.releasedTemp, wi.empty)
	//	wi.hasReleased = false
	//}

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
