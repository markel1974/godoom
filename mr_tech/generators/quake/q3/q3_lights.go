package q3

// _q3LightStyle0 defines a default constant light style with uniform intensity throughout.
var _q3LightStyle0 = []float64{1.0}

// _q3LightStyle1 represents a predefined light intensity pattern for rendering light styles.
var _q3LightStyle1 = []float64{
	1.0, 1.0, 1.08, 1.0, 1.0, 1.17, 1.0, 1.0, 1.17, 1.0,
	1.0, 1.08, 1.17, 1.08, 1.0, 1.0, 1.17, 1.08, 1.33, 1.08,
	1.0, 1.0, 1.17,
}

// _q3LightStyle2 represents a dynamic light style pattern using a sinusoidal-like modulation of intensity levels.
var _q3LightStyle2 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50, 1.58,
	1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83, 1.75,
	1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0, 0.92,
	0.83, 0.75, 0.67, 0.58, 0.50, 0.42, 0.33, 0.25, 0.17, 0.08,
	0.0,
}

// _q3LightStyle3 defines a light style with oscillating brightness patterns using a mix of steady and gradual transitions.
var _q3LightStyle3 = []float64{
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.0, 0.08, 0.17,
	0.25, 0.33, 0.42, 0.50,
}

// _q3LightStyle4 represents a light style pattern alternating between full intensity (1.0) and zero intensity (0.0).
var _q3LightStyle4 = []float64{
	1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0,
}

// _q3LightStyle5 represents a sinusoidal-like sequence of brightness values primarily used for dynamic light simulation.
var _q3LightStyle5 = []float64{
	0.75, 0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50,
	1.58, 1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83,
	1.75, 1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0,
	0.92, 0.83, 0.75,
}

// _q3LightStyle6 represents a light style pattern with varying intensity values for illumination effects.
var _q3LightStyle6 = []float64{
	1.08, 1.0, 1.17, 1.08, 1.33, 1.08, 1.0, 1.17, 1.0, 1.08,
	1.0, 1.17, 1.0, 1.17, 1.0, 1.08, 1.17,
}

// _q3LightStyle7 represents a light style pattern defined by a sequence of float64 intensity values.
var _q3LightStyle7 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.08, 0.17, 0.25,
	0.33, 0.42, 0.50, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0,
	0.0, 1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
}

// _q3LightStyle8 defines a specific light intensity pattern as a sequence of normalized float values.
var _q3LightStyle8 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 0.0,
	0.0, 0.0, 1.0, 1.0, 1.0, 0.0, 0.08, 0.17, 0.25, 0.33,
	0.42, 0.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 1.0,
}

// _q3LightStyle9 defines a light style pattern with an initial sequence of zeros followed by constant intensity values.
var _q3LightStyle9 = []float64{
	0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08,
}

// _q3LightStyle10 defines a light intensity pattern as a sequence of floating-point values for visual effects.
var _q3LightStyle10 = []float64{
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0,
	0.0, 1.0, 1.0, 1.0, 0.0,
}

// _q3LightStyle11 defines a light intensity pattern with a wave-like progression, starting and ending at zero intensity.
var _q3LightStyle11 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.42, 1.33,
	1.25, 1.17, 1.08, 1.0, 0.92, 0.83, 0.75, 0.67, 0.58, 0.50,
	0.42, 0.33, 0.25, 0.17, 0.08, 0.0,
}

// _q3LightStyles holds predefined light animation styles used to simulate various light effects in the game.
var _q3LightStyles = [][]float64{
	_q3LightStyle0,
	_q3LightStyle1,
	_q3LightStyle2,
	_q3LightStyle3,
	_q3LightStyle4,
	_q3LightStyle5,
	_q3LightStyle6,
	_q3LightStyle7,
	_q3LightStyle8,
	_q3LightStyle9,
	_q3LightStyle10,
	_q3LightStyle11,
}
