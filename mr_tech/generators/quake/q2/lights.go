package q2

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const (
	lightTargetName = "targetname"
)

// _q2LightStyle0 defines a static light style with a constant intensity of 1.0.
var _q2LightStyle0 = []float64{1.0}

// _q2LightStyle1 defines a sequence of float64 values representing a light style pattern or intensity transitions.
var _q2LightStyle1 = []float64{
	1.0, 1.0, 1.08, 1.0, 1.0, 1.17, 1.0, 1.0, 1.17, 1.0,
	1.0, 1.08, 1.17, 1.08, 1.0, 1.0, 1.17, 1.08, 1.33, 1.08,
	1.0, 1.0, 1.17,
}

// _q2LightStyle2 defines a light animation pattern with smooth brightness transitions forming a bell curve-like shape.
var _q2LightStyle2 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50, 1.58,
	1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83, 1.75,
	1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0, 0.92,
	0.83, 0.75, 0.67, 0.58, 0.50, 0.42, 0.33, 0.25, 0.17, 0.08,
	0.0,
}

// _q2LightStyle3 defines a sequence of values representing a specific light intensity pattern for visual effects.
var _q2LightStyle3 = []float64{
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.0, 0.08, 0.17,
	0.25, 0.33, 0.42, 0.50,
}

// _q2LightStyle4 represents a binary light pattern alternating between full light (1.0) and no light (0.0).
var _q2LightStyle4 = []float64{
	1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0,
}

// _q2LightStyle5 represents a light intensity style with smooth oscillations peaking at 2.08 and tapering symmetrically.
var _q2LightStyle5 = []float64{
	0.75, 0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50,
	1.58, 1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83,
	1.75, 1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0,
	0.92, 0.83, 0.75,
}

// _q2LightStyle6 defines a sequence of normalized light intensity variations applied as a specific light style pattern.
var _q2LightStyle6 = []float64{
	1.08, 1.0, 1.17, 1.08, 1.33, 1.08, 1.0, 1.17, 1.0, 1.08,
	1.0, 1.17, 1.0, 1.17, 1.0, 1.08, 1.17,
}

// _q2LightStyle7 defines a specific light animation profile using an array of float64 values to represent intensity over time.
var _q2LightStyle7 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.08, 0.17, 0.25,
	0.33, 0.42, 0.50, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0,
	0.0, 1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
}

// _q2LightStyle8 defines a light intensity pattern as a sequence of floating-point values representing transitions.
var _q2LightStyle8 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 0.0,
	0.0, 0.0, 1.0, 1.0, 1.0, 0.0, 0.08, 0.17, 0.25, 0.33,
	0.42, 0.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 1.0,
}

// _q2LightStyle9 defines a specific light intensity pattern transitioning from all zero values to a constant value of 2.08.
var _q2LightStyle9 = []float64{
	0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08,
}

// _q2LightStyle10 defines a custom light style pattern represented by a sequence of float64 intensity values.
var _q2LightStyle10 = []float64{
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0,
	0.0, 1.0, 1.0, 1.0, 0.0,
}

// _q2LightStyle11 defines a sequence of brightness values for creating a pulsating light effect with capped oscillations.
var _q2LightStyle11 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.42, 1.33,
	1.25, 1.17, 1.08, 1.0, 0.92, 0.83, 0.75, 0.67, 0.58, 0.50,
	0.42, 0.33, 0.25, 0.17, 0.08, 0.0,
}

// _q2LightStyles is a collection of predefined light intensity patterns represented as slices of float64.
var _q2LightStyles = [][]float64{
	_q2LightStyle0,
	_q2LightStyle1,
	_q2LightStyle2,
	_q2LightStyle3,
	_q2LightStyle4,
	_q2LightStyle5,
	_q2LightStyle6,
	_q2LightStyle7,
	_q2LightStyle8,
	_q2LightStyle9,
	_q2LightStyle10,
	_q2LightStyle11,
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
	if index >= 0 && index < len(_q2LightStyles) {
		return _q2LightStyles[index]
	}
	// Switchable Quake light styles.
	// The runtime currently has no separate representation
	// for the trigger/switch state, so default to steady ON.
	return defaultStyle
}

// Lights manages Quake 2 light entities and their target relationships.
type Lights struct {
	targets map[string]*lumps.Entity
}

// NewLights initializes a Lights structure by mapping entity targetnames
// to their corresponding entities.
func NewLights(entities []*lumps.Entity) *Lights {
	l := &Lights{targets: make(map[string]*lumps.Entity)}
	//targetname = identifier
	//target = destination
	for _, ent := range entities {
		targetName, ok := ent.GetProperty("targetname")
		if !ok || len(targetName) == 0 {
			continue
		}
		if _, exists := l.targets[targetName]; exists {
			continue
		}
		l.targets[targetName] = ent
	}
	return l
}

// CreateLight generates a light source based on a Quake 2 entity's
// properties, position, and subclass.
func (l *Lights) CreateLight(ent *lumps.Entity, pos geometry.XYZ) *config.Light {
	const maxIntensity = 230.0
	kind := config.LightKindAmbient
	q2Intensity := maxIntensity
	dirX, dirY, dirZ := 0.0, 0.0, -1.0 // Default Q2 light direction: down.
	coneAngle := 10.0                  // qrad3/Q2 default spotlight cone.

	lightStr, _ := ent.GetProperty("light")
	lightAltStr, _ := ent.GetProperty("_light")
	targetStr, _ := ent.GetProperty("target")
	classname, _ := ent.GetProperty("classname")
	colorStr, _ := ent.GetProperty("_color")
	styleStr, _ := ent.GetProperty("style")
	styleAltStr, _ := ent.GetProperty("_style")
	coneStr, _ := ent.GetProperty("_cone")

	// Intensity
	// Q2 uses "light" as the normal intensity key.
	// "_light" is accepted as an alternative.
	if v, ok := lumps.ParseFloat(lightStr); ok {
		q2Intensity = v
	} else if v, ok := lumps.ParseFloat(lightAltStr); ok {
		q2Intensity = v
	}

	// Spotlight
	// In Q2 a spotlight can be represented by a light having a "target".
	// The target is NOT a group: it identifies the entity used to determine
	// the direction. If multiple entities have the same targetname, the
	// target lookup uses the first one.
	// targetname itself has no directional meaning for a light.
	if len(targetStr) > 0 {
		kind = config.LightKindSpot
		targetEnt := l.targets[targetStr]
		if targetEnt == nil {
			fmt.Printf("warning: light target entity %q not found\n", targetStr)
			return nil
		}
		originStr, _ := targetEnt.GetProperty("origin")
		tx, ty, tz, ok := lumps.ParseVector(originStr)
		if !ok {
			fmt.Printf("warning: invalid origin for light target %q: %q\n", targetStr, originStr)
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
	}

	// Optional light_spot compatibility
	// The retail Q2 levels examined so far use "light", not "light_spot",
	// and no such entity was found in levels 12 and 18.
	// Keep support here for maps/tools using the extended light_spot
	// representation, but do not confuse it with targetname semantics.
	if kind != config.LightKindSpot && classname == "light_spot" {
		kind = config.LightKindSpot
	}

	// Color
	r, g, b, ok := lumps.ParseColorVector(colorStr)
	if !ok {
		r, g, b = 1.0, 1.0, 1.0
	}

	// Cone
	// Q2/qrad3 uses "_cone" for the spotlight cone.
	// Default is 10 degrees.
	if kind == config.LightKindSpot {
		if c, valid := lumps.ParseFloat(coneStr); valid && c > 0 {
			coneAngle = c
		}
	}

	// Radius / brightness conversion
	const engineDecayConstant = 4.605
	const q2RadiusQuadScalePoint = 0.004605
	const q2RadiusQuadScaleSpot = 0.0115

	var desiredRadius float64
	var desiredBrightness float64
	if kind == config.LightKindSpot {
		desiredRadius = q2RadiusQuadScaleSpot * (q2Intensity * q2Intensity)
		desiredBrightness = q2Intensity * 1.1
	} else {
		if q2Intensity > maxIntensity {
			q2Intensity = maxIntensity
		}
		desiredRadius = q2RadiusQuadScalePoint * (q2Intensity * q2Intensity)
		desiredBrightness = q2Intensity * 0.5
	}

	// Engine rule: radius = falloff * decayConstant * intensity
	// Therefore: falloff = radius / (decayConstant * intensity)
	intensity := desiredBrightness
	falloff := 0.0
	if intensity > 0 {
		falloff = desiredRadius / (engineDecayConstant * intensity)
	}

	light := config.NewConfigLight(pos, intensity, kind, falloff)
	light.R = r
	light.G = g
	light.B = b
	light.DirX = dirX
	light.DirY = dirY
	light.DirZ = dirZ

	// style
	if len(styleStr) > 0 {
		light.Style = LightStyle(styleStr)
	} else if len(styleAltStr) > 0 {
		light.Style = LightStyle(styleAltStr)
	}

	// Spotlight cone
	light.CutOff = coneAngle
	light.OuterCutOff = coneAngle + 5.0
	if light.Intensity <= 0 {
		fmt.Println("warning: light intensity is zero")
	}

	return light
}
