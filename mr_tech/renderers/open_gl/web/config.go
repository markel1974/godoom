package web

// WindowConfig defines the configuration options for initializing a window, including size, title, fullscreen, and VSync.
type WindowConfig struct {
	Title      string
	Fullscreen bool
	VSync      bool
	Width      int
	Height     int
	Fps        int
}
