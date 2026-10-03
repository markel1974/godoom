package desktop

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop/executor"
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
	th      *executor.MainThread
	monitor *glfw.Monitor
}

// NewPrimaryMonitor retrieves the primary monitor and wraps it into a Monitor structure for further usage.
func NewPrimaryMonitor(th *executor.MainThread) *Monitor {
	var monitor *glfw.Monitor
	th.Call(func() {
		monitor = glfw.GetPrimaryMonitor()
	})
	return &Monitor{
		th:      th,
		monitor: monitor,
	}
}

// NewMonitors retrieves a list of all connected monitors and returns them as []*Monitor.
func NewMonitors(th *executor.MainThread) []*Monitor {
	var monitors []*Monitor
	th.Call(func() {
		for _, monitor := range glfw.GetMonitors() {
			monitors = append(monitors, NewyMonitor(th, monitor))
		}
	})
	return monitors
}

// NewyMonitor creates and returns a new Monitor instance, associating it with the provided MainThread and GLFW Monitor.
func NewyMonitor(th *executor.MainThread, monitor *glfw.Monitor) *Monitor {
	return &Monitor{
		th:      th,
		monitor: monitor,
	}
}

// Name retrieves the name of the monitor associated with the Monitor instance. Uses thread-safe execution.
func (m *Monitor) Name() string {
	var name string
	m.th.Call(func() { name = m.monitor.GetName() })
	return name
}

// PhysicalSize retrieves the physical dimensions of the monitor in millimeters as width and height.
func (m *Monitor) PhysicalSize() (float64, float64) {
	var width, height float64
	var wi, hi int
	m.th.Call(func() {
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
	m.th.Call(func() {
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
	m.th.Call(func() { mode = m.monitor.GetVideoMode() })
	width = float64(mode.Width)
	height = float64(mode.Height)
	return width, height
}

// BitDepth retrieves the bit depth of the red, green, and blue color channels for the current video mode.
func (m *Monitor) BitDepth() (int, int, int) {
	var red, green, blue int
	var mode *glfw.VidMode
	m.th.Call(func() { mode = m.monitor.GetVideoMode() })
	red = mode.RedBits
	green = mode.GreenBits
	blue = mode.BlueBits
	return red, green, blue
}

// RefreshRate retrieves the monitor's current refresh rate in Hertz as a floating-point value.
func (m *Monitor) RefreshRate() float64 {
	var rate float64
	var mode *glfw.VidMode
	m.th.Call(func() { mode = m.monitor.GetVideoMode() })
	rate = float64(mode.RefreshRate)
	return rate
}

// VideoModes retrieves all available video modes for the monitor, including their dimensions and refresh rates.
func (m *Monitor) VideoModes() []VideoMode {
	var vModes []VideoMode
	var modes []*glfw.VidMode
	m.th.Call(func() {
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
