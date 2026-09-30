package q3

import (
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// _q3LightStyle0 is the default light style array with a constant intensity value of 1.0.
var _q3LightStyle0 = []float64{1.0}

// _q3LightStyle1 defines a sequence of brightness levels for a specific light style pattern.
var _q3LightStyle1 = []float64{
	1.0, 1.0, 1.08, 1.0, 1.0, 1.17, 1.0, 1.0, 1.17, 1.0,
	1.0, 1.08, 1.17, 1.08, 1.0, 1.0, 1.17, 1.08, 1.33, 1.08,
	1.0, 1.0, 1.17,
}

// _q3LightStyle2 represents a predefined light intensity pattern with values oscillating between 0.0 and 2.08 in a sine-like manner.
var _q3LightStyle2 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50, 1.58,
	1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83, 1.75,
	1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0, 0.92,
	0.83, 0.75, 0.67, 0.58, 0.50, 0.42, 0.33, 0.25, 0.17, 0.08,
	0.0,
}

// _q3LightStyle3 defines a sequence of light intensity values representing a predefined light flicker style pattern.
var _q3LightStyle3 = []float64{
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.0, 0.08, 0.17,
	0.25, 0.33, 0.42, 0.50,
}

// _q3LightStyle4 represents an alternating pattern of high (1.0) and low (0.0) light intensity values.
var _q3LightStyle4 = []float64{
	1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0,
}

// _q3LightStyle5 defines a sequence of floating-point values representing light intensity variations in a wave-like pattern.
var _q3LightStyle5 = []float64{
	0.75, 0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50,
	1.58, 1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83,
	1.75, 1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0,
	0.92, 0.83, 0.75,
}

// _q3LightStyle6 defines a light intensity pattern with varying brightness values over a sequence of 17 floats.
var _q3LightStyle6 = []float64{
	1.08, 1.0, 1.17, 1.08, 1.33, 1.08, 1.0, 1.17, 1.0, 1.08,
	1.0, 1.17, 1.0, 1.17, 1.0, 1.08, 1.17,
}

// _q3LightStyle7 defines a sequence of light intensity values used to represent a specific light flicker pattern.
var _q3LightStyle7 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.08, 0.17, 0.25,
	0.33, 0.42, 0.50, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0,
	0.0, 1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
}

// _q3LightStyle8 defines a light flicker pattern with alternating intensity changes, including sustained and gradual fades.
var _q3LightStyle8 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 0.0,
	0.0, 0.0, 1.0, 1.0, 1.0, 0.0, 0.08, 0.17, 0.25, 0.33,
	0.42, 0.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 1.0,
}

// _q3LightStyle9 defines a light pattern with initial zero intensity and a steady increase to a uniform intensity of 2.08.
var _q3LightStyle9 = []float64{
	0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08,
}

// _q3LightStyle10 represents a light style with specific intensity changes over time for rendering effects.
var _q3LightStyle10 = []float64{
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0,
	0.0, 1.0, 1.0, 1.0, 0.0,
}

// _q3LightStyle11 represents a sine wave light pattern with values gradually increasing and then decreasing.
var _q3LightStyle11 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.42, 1.33,
	1.25, 1.17, 1.08, 1.0, 0.92, 0.83, 0.75, 0.67, 0.58, 0.50,
	0.42, 0.33, 0.25, 0.17, 0.08, 0.0,
}

// _q3LightStyles defines a collection of light intensity patterns used to vary lighting styles in the system.
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

// lightTargetName defines the property key for identifying an entity's target in a mapping or lighting context.
const lightTargetName = "targetname"

// Lights is a type that manages a collection of entities and their relationships to produce light configurations.
// It maps target names to entities and provides methods for creating and configuring light objects.
type Lights struct {
	entities []*lumps.Entity
	targets  map[string]*lumps.Entity
}

// NewLights initializes a Lights instance by mapping entities to their target names, if present.
func NewLights(entities []*lumps.Entity) *Lights {
	l := &Lights{
		entities: entities,
		targets:  make(map[string]*lumps.Entity),
	}
	for _, ent := range entities {
		if targetName, ok := ent.Properties[lightTargetName]; ok {
			l.targets[targetName] = ent
		}
	}
	return l
}

// Create initializes and returns a Light object based on the provided entity,
// position, and Quake III light properties.
func (l *Lights) Create(ent *lumps.Entity, pos geometry.XYZ) (*config.Light, error) {
	mangleStr, _ := ent.GetProperty("mangle")
	colorStr, _ := ent.GetProperty("_color")
	angleStr, _ := ent.GetProperty("angle")
	targetStr, hasTarget := ent.GetProperty("target")
	lightStr, _ := ent.GetProperty("light")
	angle, _ := lumps.ParseFloat(angleStr)
	kind := config.LightKindAmbient
	//dirX, dirY, dirZ := 0.0, -1.0, 0.0 // Q3 world coordinates: default direction is down.
	dirX, dirY, dirZ := 0.0, 0.0, -1.0
	r, g, b := 1.0, 1.0, 1.0 // White light.
	style := _q3LightStyle0

	baseIntensity := 300.0 // Quake III default light intensity.
	intensity := 0.0
	falloff := 0.0

	if len(lightStr) > 0 {
		if v, ok := lumps.ParseFloat(lightStr); ok {
			baseIntensity = v
		}
	}

	if sIndex, ok := ent.GetProperty("style"); ok {
		if index, err := strconv.Atoi(sIndex); err == nil &&
			index >= 0 && index < len(_q3LightStyles) {
			style = _q3LightStyles[index]
		}
	}

	if len(colorStr) > 0 {
		if cr, cg, cb, valid := lumps.ParseVector(colorStr); valid {
			if cr > 1.0 || cg > 1.0 || cb > 1.0 {
				r = cr / 255.0
				g = cg / 255.0
				b = cb / 255.0
			} else {
				r = cr
				g = cg
				b = cb
			}
		}
	}

	// -------------------------------------------------------------------------
	// DIRECTION / SPOTLIGHT
	// -------------------------------------------------------------------------
	//
	// In Quake III, a light with a target is a spotlight.
	// The direction is the normalized vector:
	//
	//     target.origin - light.origin
	//
	// Keep this as a direct vector calculation instead of converting
	// target -> yaw/pitch -> direction.
	//
	// mangle is retained as an optional toolchain-specific orientation
	// property.
	// -------------------------------------------------------------------------

	if hasTarget && len(targetStr) > 0 {
		kind = config.LightKindSpot
		if otherEnt, ok := l.targets[targetStr]; ok {
			if originStr, ok := otherEnt.GetProperty("origin"); ok {
				if tx, ty, tz, valid := lumps.ParseVector(originStr); valid {
					dx := tx - pos.X
					dy := ty - pos.Y
					dz := tz - pos.Z
					dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
					if dist > 0.0 {
						dirX = dx / dist
						dirY = dy / dist
						dirZ = dz / dist
					}
				}
			}
		}
	} else if len(mangleStr) > 0 {
		// Toolchain-specific mangle support.
		kind = config.LightKindSpot
		if yaw, pitch, _, valid := lumps.ParseVector(mangleStr); valid {
			dirX, dirY, dirZ = lumps.CalcDirection(yaw, pitch)
		}
	} else if angleStr != "" {
		// Optional angle-based orientation used by the map/toolchain.
		kind = config.LightKindSpot
		switch angle {
		case -1:
			//dirX, dirY, dirZ = 0.0, 1.0, 0.0 // Look up.
			dirX, dirY, dirZ = 0.0, 0.0, 1.0 // Look up.
		case -2:
			//dirX, dirY, dirZ = 0.0, -1.0, 0.0 // Look down.
			dirX, dirY, dirZ = 0.0, 0.0, -1.0 // Look down.
		default:
			dirX, dirY, dirZ = lumps.CalcDirection(angle, 0)
		}
	}
	if kind == config.LightKindSpot {
		falloff = baseIntensity * 0.08
		intensity = baseIntensity * 0.08
	} else {
		falloff = baseIntensity * 0.04
		intensity = baseIntensity * 0.4
	}
	cl := config.NewConfigLight(pos, intensity, kind, falloff)
	cl.R = r
	cl.G = g
	cl.B = b
	cl.DirX = dirX
	cl.DirY = dirY
	cl.DirZ = dirZ
	cl.Style = style

	return cl, nil
}

/*
// Create initializes and returns a Light object based on the provided entity, angle, position, and properties.
func (l *Lights) Create(ent *lumps.Entity, pos geometry.XYZ) (*config.Light, error) {
	mangleStr2, _ := ent.GetProperty("mangle")
	colorStr, _ := ent.GetProperty("_color")
	angleStr, _ := ent.GetProperty("angle")
	targetStr, hasTarget := ent.GetProperty("target")
	lightStr, _ := ent.GetProperty("light")
	angle, _ := lumps.ParseFloat(angleStr)
	kind := config.LightKindAmbient
	dirX, dirY, dirZ := 0.0, -1.0, 0.0 // Default: look down
	r, g, b := 1.0, 1.0, 1.0           // white
	style := _q3LightStyle0
	baseIntensity := 300.0 // Typical Quake default fallback
	intensity := 0.0
	falloff := 0.0

	if len(lightStr) > 0 {
		if v, ok := lumps.ParseFloat(lightStr); ok {
			baseIntensity = v
		}
	}

	if sIndex, ok := ent.GetProperty("style"); ok {
		if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 && index < len(_q3LightStyles) {
			style = _q3LightStyles[index]
		}
	}

	if len(colorStr) > 0 {
		if cr, cg, cb, valid := lumps.ParseVector(colorStr); valid {
			if cr > 1.0 || cg > 1.0 || cb > 1.0 {
				r, g, b = cr/255.0, cg/255.0, cb/255.0
			} else {
				r, g, b = cr, cg, cb
			}
		}
	}

	// Detect spotlight
	if hasTarget && len(targetStr) > 0 {
		kind = config.LightKindSpot
		if otherEnt, ok := l.targets[targetStr]; ok {
			if otherEnt.Properties[lightTargetName] == targetStr {
				if originStr, ok := otherEnt.Properties["origin"]; ok {
					if tx, ty, tz, valid := lumps.ParseVector(originStr); valid {
						dx := tx - pos.X
						dy := ty - pos.Y
						dz := tz - pos.Z
						yaw := math.Atan2(dy, dx) * 180 / math.Pi
						pitch := math.Atan2(dz, math.Sqrt(dx*dx+dy*dy)) * 180 / math.Pi
						mangleStr2 = fmt.Sprintf("%f %f 0", yaw, pitch)
					}
				}
			}
		}
	} else if len(mangleStr2) > 0 {
		kind = config.LightKindSpot
	}

	// SPOTLIGHT DIRECTION
	if kind == config.LightKindSpot {
		falloff = baseIntensity * 0.01
		intensity = baseIntensity * 0.1

		if len(mangleStr2) > 0 {
			if yaw, pitch, _, valid := lumps.ParseVector(mangleStr2); valid {
				dirX, dirY, dirZ = lumps.CalcDirection(yaw, pitch)
			}
		} else {
			if angle == -1 {
				dirX, dirY, dirZ = 0.0, 1.0, 0.0 // Look up
			} else if angle == -2 {
				dirX, dirY, dirZ = 0.0, -1.0, 0.0 // Look down
			} else {
				dirX, dirY, dirZ = lumps.CalcDirection(angle, 0)
			}
		}
	} else {
		falloff = baseIntensity * 0.05
		intensity = baseIntensity * 0.5
	}

	// CONFIGURATION CREATION
	cl := config.NewConfigLight(pos, intensity, kind, falloff)
	cl.R = r
	cl.G = g
	cl.B = b
	cl.DirX = dirX
	cl.DirY = dirY
	cl.DirZ = dirZ
	cl.Style = style

	return cl, nil
}


*/
