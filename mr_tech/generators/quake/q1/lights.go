package q1

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// _q1LightStyle0 represents a light style with a constant intensity value of 1.0 throughout.
var _q1LightStyle0 = []float64{1.0}

// _q1LightStyle1 defines a sequence of brightness levels for a light style animation pattern in a floating-point array.
var _q1LightStyle1 = []float64{
	1.0, 1.0, 1.08, 1.0, 1.0, 1.17, 1.0, 1.0, 1.17, 1.0,
	1.0, 1.08, 1.17, 1.08, 1.0, 1.0, 1.17, 1.08, 1.33, 1.08,
	1.0, 1.0, 1.17,
}

// _q1LightStyle2 represents a smooth wave-like pattern of light intensity values, cycling back to its starting point.
var _q1LightStyle2 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50, 1.58,
	1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83, 1.75,
	1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0, 0.92,
	0.83, 0.75, 0.67, 0.58, 0.50, 0.42, 0.33, 0.25, 0.17, 0.08,
	0.0,
}

// _q1LightStyle3 defines a sequence of float64 values representing a specific lighting intensity pattern or style.
var _q1LightStyle3 = []float64{
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.0, 0.08, 0.17,
	0.25, 0.33, 0.42, 0.50,
}

// _q1LightStyle4 represents a light style sequence with alternating values of 1.0 and 0.0 in a repeated pattern.
var _q1LightStyle4 = []float64{
	1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0,
}

// _q1LightStyle5 represents a light style sequence with symmetric intensity transitions from low to high and back to low.
var _q1LightStyle5 = []float64{
	0.75, 0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50,
	1.58, 1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83,
	1.75, 1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0,
	0.92, 0.83, 0.75,
}

// _q1LightStyle6 defines a pattern of light intensity values used in dynamic lighting calculations.
var _q1LightStyle6 = []float64{
	1.08, 1.0, 1.17, 1.08, 1.33, 1.08, 1.0, 1.17, 1.0, 1.08,
	1.0, 1.17, 1.0, 1.17, 1.0, 1.08, 1.17,
}

// _q1LightStyle7 defines a sequence of float64 values representing a lighting pattern with periodic changes in intensity.
var _q1LightStyle7 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.08, 0.17, 0.25,
	0.33, 0.42, 0.50, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0,
	0.0, 1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
}

// _q1LightStyle8 represents a specific light style pattern defined as a sequence of float64 values.
var _q1LightStyle8 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 0.0,
	0.0, 0.0, 1.0, 1.0, 1.0, 0.0, 0.08, 0.17, 0.25, 0.33,
	0.42, 0.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 1.0,
}

// _q1LightStyle9 defines a light style pattern with an initial sequence of zeros followed by repeated 2.08 values.
var _q1LightStyle9 = []float64{
	0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08,
}

// _q1LightStyle10 represents a specific light intensity pattern, defined as a sequence of float64 values.
var _q1LightStyle10 = []float64{
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0,
	0.0, 1.0, 1.0, 1.0, 0.0,
}

// _q1LightStyle11 defines a light style pattern with gradual intensity fluctuation and symmetry around a peak value.
var _q1LightStyle11 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.42, 1.33,
	1.25, 1.17, 1.08, 1.0, 0.92, 0.83, 0.75, 0.67, 0.58, 0.50,
	0.42, 0.33, 0.25, 0.17, 0.08, 0.0,
}

// _q1LightStyles contains predefined sequences of light intensity levels represented as nested slices of float64 values.
var _q1LightStyles = [][]float64{
	_q1LightStyle0,
	_q1LightStyle1,
	_q1LightStyle2,
	_q1LightStyle3,
	_q1LightStyle4,
	_q1LightStyle5,
	_q1LightStyle6,
	_q1LightStyle7,
	_q1LightStyle8,
	_q1LightStyle9,
	_q1LightStyle10,
	_q1LightStyle11,
}

// LightStyle returns a light style pattern as a slice of float64 values based on the input string identifier.
// If the input is empty or invalid, it returns a default light style with a steady intensity of 1.0.
func LightStyle(styleStr string) []float64 {
	defaultStyle := []float64{1.0}
	if len(styleStr) == 0 {
		return defaultStyle
	}
	index, err := strconv.Atoi(styleStr)
	if err != nil {
		return defaultStyle
	}
	if index >= 0 && index < len(_q1LightStyles) {
		return _q1LightStyles[index]
	}
	// Switchable Quake light styles.
	// The runtime currently has no separate representation
	// for the trigger/switch state, so default to steady ON.
	return defaultStyle
}

// Lights represents a collection of entities mapped by their target names, used for managing and creating light sources.
type Lights struct {
	targetEntities map[string]*lumps.Entity
}

// NewLights initializes a Lights structure by mapping entity targetnames to their corresponding entities.
func NewLights(entities []*lumps.Entity) *Lights {
	targetEntities := make(map[string]*lumps.Entity)
	for _, ent := range entities {
		if targetStr, _ := ent.GetProperty("targetname"); len(targetStr) > 0 {
			targetEntities[targetStr] = ent
		}
	}
	return &Lights{
		targetEntities: targetEntities,
	}
}

// CreateLight generates a light source based on an entity's properties, position, and subclass, returning the configured light.
func (l *Lights) CreateLight(ent *lumps.Entity, pos geometry.XYZ, subClass string) *config.Light {
	kind := config.LightKindAmbient
	q1Intensity := 300.0
	dirX, dirY, dirZ := 0.0, 0.0, -1.0 // Default direction: down.
	coneAngle := 40.0                  // Quake default
	lightStr, _ := ent.GetProperty("light")
	targetStr, _ := ent.GetProperty("target")
	mangleStr, _ := ent.GetProperty("mangle")
	angleStr, _ := ent.GetProperty("angle")
	colorStr, _ := ent.GetProperty("_color")
	styleStr, _ := ent.GetProperty("style")

	if v, ok := lumps.ParseFloat(lightStr); ok {
		q1Intensity = v
	}

	//target: trasforma la light in spotlight + determina direzione
	//mangle: trasforma la light in spotlight + determina direzione
	if len(targetStr) > 0 {
		kind = config.LightKindSpot
		targetEnt := l.targetEntities[targetStr]
		if targetEnt == nil {
			fmt.Println("target entity not found")
			return nil
		}
		originStr, _ := targetEnt.GetProperty("origin")
		tx, ty, tz, ok := lumps.ParseVector(originStr)
		if !ok {
			fmt.Println("invalid origin vector")
			return nil
		}
		dx := tx - pos.X
		dy := ty - pos.Y
		dz := tz - pos.Z
		if length := math.Sqrt(dx*dx + dy*dy + dz*dz); length > 0 {
			dirX = dx / length
			dirY = dy / length
			dirZ = dz / length
		}
	} else if len(mangleStr) > 0 {
		//TODO DISABLED FOR THE MOMENT
		//return nil
		kind = config.LightKindSpot
		yaw, pitch, _, valid := lumps.ParseVector(mangleStr)
		if !valid {
			fmt.Printf("Invalid mangle vector: %s\n", mangleStr)
			return nil
		}
		dirX, dirY, dirZ = lumps.CalcDirection(yaw, pitch)
		//dirZ = dirZ
		//dirY = -dirY
		//dirX = -dirX
	}

	r, g, b, ok := lumps.ParseColorVector(colorStr)
	if !ok {
		r, g, b = 1.0, 1.0, 1.0
	}
	const engineDecayConstant = 4.605
	const q1RadiusQuadScalePoint = 0.004605
	const q1RadiusQuadScaleSpot = 0.0115

	var desiredRadius float64
	var desiredBrightness float64
	if kind == config.LightKindSpot {
		desiredRadius = q1RadiusQuadScaleSpot * (q1Intensity * q1Intensity)
		desiredBrightness = q1Intensity * 0.008
		if c, valid := lumps.ParseFloat(angleStr); valid {
			coneAngle = c
		}
	} else {
		desiredRadius = q1RadiusQuadScalePoint * (q1Intensity * q1Intensity)
		desiredBrightness = q1Intensity * 0.1
	}

	// Engine Rule
	intensity := desiredBrightness
	falloff := desiredRadius / (engineDecayConstant * intensity)

	fmt.Printf("Light intensity: %f, falloff: %f\n", intensity, falloff)

	light := config.NewConfigLight(pos, intensity, kind, falloff)
	light.R = r
	light.G = g
	light.B = b
	light.DirX = dirX
	light.DirY = dirY
	light.DirZ = dirZ
	light.Style = LightStyle(styleStr)
	light.CutOff = coneAngle
	light.OuterCutOff = coneAngle + 5.0
	if light.Intensity <= 0 {
		fmt.Println("warning light intensity is zero")
	}
	return light
}
