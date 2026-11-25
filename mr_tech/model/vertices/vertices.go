package vertices

import "sync/atomic"

const (
	RenderModeOpaque             = 0.0
	RenderModeLiquid             = 0.5
	RenderModeBillboard          = 1.0
	RenderModeBillboardSpherical = 1.1
	RenderModeModel3D            = 2.0
	RenderModeInterface          = 3.0
)

var globalStateEpoch atomic.Uint64

// StateEpoch returns a monotonic counter incremented when renderable animation/action/deform state changes.
func StateEpoch() uint64 {
	return globalStateEpoch.Load()
}

// BumpStateEpoch increments and returns the global renderable state epoch.
func BumpStateEpoch() uint64 {
	return globalStateEpoch.Add(1)
}
