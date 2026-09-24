package core

// WindowConfig defines the configuration for a window, including appearance, behavior, and rendering properties.
type WindowConfig struct {
	Title string

	//Icon []IPicture

	Bounds Rect

	Position XY

	Monitor *Monitor

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

// CheckSampleMSAA verifies if the SamplesMSAA value is a valid multisample anti-aliasing (MSAA) setting.
func (cfg WindowConfig) CheckSampleMSAA() bool {
	flag := false
	for _, v := range []int{0, 2, 4, 8, 16} {
		if cfg.SamplesMSAA == v {
			flag = true
			break
		}
	}
	return flag
}
