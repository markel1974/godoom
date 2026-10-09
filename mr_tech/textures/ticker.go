package textures

// _tickIntervalRef defines the reference interval used for calculating time-based ratios in the system.
const tickIntervalRef = uint64(120)

// _monotonicTick is an internal counter used to track monotonically increasing ticks in the system.
var _monotonicTick uint64

// _globalTick is a globally incrementing counter used for tracking elapsed ticks across the system.
var _globalTick uint64

// _tickIntervalTarget defines the target interval duration in ticks, used for ratio calculations in timing operations.
var _tickIntervalTarget uint64

// _tickRatio represents the scaling factor between the reference tick interval and the target tick interval.
var _tickRatio float64

func init() {
	SetFps(tickIntervalRef)
}

// SetFps sets the target tick interval and adjusts the tick ratio for timing calculations.
func SetFps(t uint64) {
	_tickIntervalTarget = t
	_tickRatio = float64(tickIntervalRef / _tickIntervalTarget)
}

// Tick updates the monotonic tick counter, computes the global tick, and calculates the current tick based on the interval.
func Tick() {
	_monotonicTick++
	_globalTick = uint64(float64(_monotonicTick) * _tickRatio)
}

// GlobalTick returns the current global tick value as a uint64.
func GlobalTick() uint64 {
	return _globalTick
}

// TickGrouped computes the floating-point frame index by dividing the tick by the group size.
func TickGrouped(tick uint64, groupSize int) float64 {
	frameFloat := float64(tick) / float64(groupSize)
	return frameFloat
}
