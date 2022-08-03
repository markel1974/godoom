package pixels

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/markel1974/godoom/pixels/executor"
)

// VideoMode represents all properties of a video mode and is
// associated with a monitor if it is used in fullscreen mode.
type VideoMode struct {
	// Width is the width of the vide mode in pixels.
	Width int
	// Height is the height of the video mode in pixels.
	Height int
	// RefreshRate holds the refresh rate of the associated monitor in Hz.
	RefreshRate int
}

// GLMonitor represents a physical display attached to your computer.
type GLMonitor struct {
	monitor *glfw.Monitor
}

// PrimaryMonitor returns the main monitor (usually the one with the taskbar and stuff).
func PrimaryMonitor() *GLMonitor {
	var monitor *glfw.Monitor
	executor.Thread.Call(func() {
		monitor = glfw.GetPrimaryMonitor()
	})
	return &GLMonitor{
		monitor: monitor,
	}
}

// Monitors returns a slice of all currently available monitors.
func Monitors() []*GLMonitor {
	var monitors []*GLMonitor
	executor.Thread.Call(func() {
		for _, monitor := range glfw.GetMonitors() {
			monitors = append(monitors, &GLMonitor{monitor: monitor})
		}
	})
	return monitors
}

// Name returns a human-readable name of the GLMonitor.
func (m *GLMonitor) Name() string {
	var name string
	executor.Thread.Call(func() { name = m.monitor.GetName() })
	return name
}

// PhysicalSize returns the size of the display area of the GLMonitor in millimeters.
func (m *GLMonitor) PhysicalSize() (width, height float64) {
	var wi, hi int
	executor.Thread.Call(func() {
		wi, hi = m.monitor.GetPhysicalSize()
	})
	width = float64(wi)
	height = float64(hi)
	return
}

// Position returns the position of the upper-left corner of the GLMonitor in screen coordinates.
func (m *GLMonitor) Position() (x, y float64) {
	var xi, yi int
	executor.Thread.Call(func() {
		xi, yi = m.monitor.GetPos()
	})
	x = float64(xi)
	y = float64(yi)
	return
}

// Size returns the resolution of the GLMonitor in pixels.
func (m *GLMonitor) Size() (width, height float64) {
	var mode *glfw.VidMode
	executor.Thread.Call(func() { mode = m.monitor.GetVideoMode() })
	width = float64(mode.Width)
	height = float64(mode.Height)
	return
}

// BitDepth returns the number of bits per color of the GLMonitor.
func (m *GLMonitor) BitDepth() (red, green, blue int) {
	var mode *glfw.VidMode
	executor.Thread.Call(func() { mode = m.monitor.GetVideoMode() })
	red = mode.RedBits
	green = mode.GreenBits
	blue = mode.BlueBits
	return
}

// RefreshRate returns the refresh frequency of the GLMonitor in Hz (refreshes/second).
func (m *GLMonitor) RefreshRate() (rate float64) {
	var mode *glfw.VidMode
	executor.Thread.Call(func() { mode = m.monitor.GetVideoMode() })
	rate = float64(mode.RefreshRate)
	return
}

// VideoModes returns all available video modes for the monitor.
func (m *GLMonitor) VideoModes() (vmodes []VideoMode) {
	var modes []*glfw.VidMode
	executor.Thread.Call(func() {
		modes = m.monitor.GetVideoModes()
	})
	for _, mode := range modes {
		vmodes = append(vmodes, VideoMode{
			Width:       mode.Width,
			Height:      mode.Height,
			RefreshRate: mode.RefreshRate,
		})
	}
	return
}
