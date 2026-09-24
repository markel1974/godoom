package core

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/core/executor"
)

// VideoMode represents a video mode with specific resolution and refresh rate.
type VideoMode struct {
	// Width is the width of the vide mode in pixels.
	Width int
	// Height is the height of the video mode in pixels.
	Height int
	// RefreshRate holds the refresh rate of the associated monitor in Hz.
	RefreshRate int
}

// Monitor represents a wrapper around a GLFW monitor providing additional utility methods for monitor information.
type Monitor struct {
	monitor *glfw.Monitor
}

// PrimaryMonitor retrieves the primary monitor and wraps it into a Monitor structure for further usage.
func PrimaryMonitor() *Monitor {
	var monitor *glfw.Monitor
	executor.Thread.Call(func() {
		monitor = glfw.GetPrimaryMonitor()
	})
	return &Monitor{
		monitor: monitor,
	}
}

// Monitors retrieves a list of all connected monitors and returns them as []*Monitor.
func Monitors() []*Monitor {
	var monitors []*Monitor
	executor.Thread.Call(func() {
		for _, monitor := range glfw.GetMonitors() {
			monitors = append(monitors, &Monitor{monitor: monitor})
		}
	})
	return monitors
}

// Name retrieves the name of the monitor associated with the Monitor instance. Uses thread-safe execution.
func (m *Monitor) Name() string {
	var name string
	executor.Thread.Call(func() { name = m.monitor.GetName() })
	return name
}

// PhysicalSize retrieves the physical dimensions of the monitor in millimeters as width and height.
func (m *Monitor) PhysicalSize() (float64, float64) {
	var width, height float64
	var wi, hi int
	executor.Thread.Call(func() {
		wi, hi = m.monitor.GetPhysicalSize()
	})
	width = float64(wi)
	height = float64(hi)
	return width, height
}

// Position retrieves the current x and y position of the monitor in screen coordinates as float64 values.
func (m *Monitor) Position() (float64, float64) {
	var x, y float64
	var xi, yi int
	executor.Thread.Call(func() {
		xi, yi = m.monitor.GetPos()
	})
	x = float64(xi)
	y = float64(yi)
	return x, y
}

// Size retrieves the current width and height of the monitor in pixels as reported by its video mode.
func (m *Monitor) Size() (float64, float64) {
	var width, height float64
	var mode *glfw.VidMode
	executor.Thread.Call(func() { mode = m.monitor.GetVideoMode() })
	width = float64(mode.Width)
	height = float64(mode.Height)
	return width, height
}

// BitDepth retrieves the bit depth of the red, green, and blue color channels for the current video mode.
func (m *Monitor) BitDepth() (int, int, int) {
	var red, green, blue int
	var mode *glfw.VidMode
	executor.Thread.Call(func() { mode = m.monitor.GetVideoMode() })
	red = mode.RedBits
	green = mode.GreenBits
	blue = mode.BlueBits
	return red, green, blue
}

// RefreshRate retrieves the monitor's current refresh rate in Hertz as a floating-point value.
func (m *Monitor) RefreshRate() float64 {
	var rate float64
	var mode *glfw.VidMode
	executor.Thread.Call(func() { mode = m.monitor.GetVideoMode() })
	rate = float64(mode.RefreshRate)
	return rate
}

// VideoModes retrieves all available video modes for the monitor, including their dimensions and refresh rates.
func (m *Monitor) VideoModes() []VideoMode {
	var vModes []VideoMode
	var modes []*glfw.VidMode
	executor.Thread.Call(func() {
		modes = m.monitor.GetVideoModes()
	})
	for _, mode := range modes {
		vModes = append(vModes, VideoMode{
			Width:       mode.Width,
			Height:      mode.Height,
			RefreshRate: mode.RefreshRate,
		})
	}
	return vModes
}
