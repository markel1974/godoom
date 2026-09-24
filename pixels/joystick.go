package pixels

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

// Joystick is a joystick or controller (gamepad).
type Joystick int

// List all of the joysticks.
const (
	Joystick1  = Joystick(glfw.Joystick1)
	Joystick2  = Joystick(glfw.Joystick2)
	Joystick3  = Joystick(glfw.Joystick3)
	Joystick4  = Joystick(glfw.Joystick4)
	Joystick5  = Joystick(glfw.Joystick5)
	Joystick6  = Joystick(glfw.Joystick6)
	Joystick7  = Joystick(glfw.Joystick7)
	Joystick8  = Joystick(glfw.Joystick8)
	Joystick9  = Joystick(glfw.Joystick9)
	Joystick10 = Joystick(glfw.Joystick10)
	Joystick11 = Joystick(glfw.Joystick11)
	Joystick12 = Joystick(glfw.Joystick12)
	Joystick13 = Joystick(glfw.Joystick13)
	Joystick14 = Joystick(glfw.Joystick14)
	Joystick15 = Joystick(glfw.Joystick15)
	Joystick16 = Joystick(glfw.Joystick16)

	JoystickLast = Joystick(glfw.JoystickLast)
)

// GamepadAxis corresponds to a gamepad axis.
type GamepadAxis int

// Gamepad axis IDs.
const (
	AxisLeftX        = GamepadAxis(glfw.AxisLeftX)
	AxisLeftY        = GamepadAxis(glfw.AxisLeftY)
	AxisRightX       = GamepadAxis(glfw.AxisRightX)
	AxisRightY       = GamepadAxis(glfw.AxisRightY)
	AxisLeftTrigger  = GamepadAxis(glfw.AxisLeftTrigger)
	AxisRightTrigger = GamepadAxis(glfw.AxisRightTrigger)
	AxisLast         = GamepadAxis(glfw.AxisLast)
)

// GamepadButton corresponds to a gamepad button.
type GamepadButton int

// Gamepad button IDs.
const (
	ButtonA           = GamepadButton(glfw.ButtonA)
	ButtonB           = GamepadButton(glfw.ButtonB)
	ButtonX           = GamepadButton(glfw.ButtonX)
	ButtonY           = GamepadButton(glfw.ButtonY)
	ButtonLeftBumper  = GamepadButton(glfw.ButtonLeftBumper)
	ButtonRightBumper = GamepadButton(glfw.ButtonRightBumper)
	ButtonBack        = GamepadButton(glfw.ButtonBack)
	ButtonStart       = GamepadButton(glfw.ButtonStart)
	ButtonGuide       = GamepadButton(glfw.ButtonGuide)
	ButtonLeftThumb   = GamepadButton(glfw.ButtonLeftThumb)
	ButtonRightThumb  = GamepadButton(glfw.ButtonRightThumb)
	ButtonDpadUp      = GamepadButton(glfw.ButtonDpadUp)
	ButtonDpadRight   = GamepadButton(glfw.ButtonDpadRight)
	ButtonDpadDown    = GamepadButton(glfw.ButtonDpadDown)
	ButtonDpadLeft    = GamepadButton(glfw.ButtonDpadLeft)
	ButtonLast        = GamepadButton(glfw.ButtonLast)
	ButtonCross       = GamepadButton(glfw.ButtonCross)
	ButtonCircle      = GamepadButton(glfw.ButtonCircle)
	ButtonSquare      = GamepadButton(glfw.ButtonSquare)
	ButtonTriangle    = GamepadButton(glfw.ButtonTriangle)
)

type GLJoystick struct {
	connected [JoystickLast + 1]bool
	name      [JoystickLast + 1]string
	buttons   [JoystickLast + 1][]glfw.Action
	axis      [JoystickLast + 1][]float32
}

// Returns if a button on a joystick is down, returning false if the button or joystick is invalid.
func (js *GLJoystick) getButton(joystick Joystick, button int) bool {
	// Check that the joystick and button is valid, return false by default
	if js.buttons[joystick] == nil || button >= len(js.buttons[joystick]) || button < 0 {
		return false
	}
	return js.buttons[joystick][byte(button)] == glfw.Press
}

// Returns the value of a joystick axis, returning 0 if the button or joystick is invalid.
func (js *GLJoystick) getAxis(joystick Joystick, axis int) float64 {
	// Check that the joystick and axis is valid, return 0 by default.
	if js.axis[joystick] == nil || axis >= len(js.axis[joystick]) || axis < 0 {
		return 0
	}
	return float64(js.axis[joystick][axis])
}
