package q1

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// _q1LightStyle0 defines a lighting style with constant intensity of 1.0.
var _q1LightStyle0 = []float64{1.0}

// _q1LightStyle1 defines a sequence of float64 values representing one of the predefined light intensity patterns.
var _q1LightStyle1 = []float64{
	1.0, 1.0, 1.08, 1.0, 1.0, 1.17, 1.0, 1.0, 1.17, 1.0,
	1.0, 1.08, 1.17, 1.08, 1.0, 1.0, 1.17, 1.08, 1.33, 1.08,
	1.0, 1.0, 1.17,
}

// _q1LightStyle2 defines a light intensity pattern with a smooth increase and decrease sequence in a loop-like structure.
var _q1LightStyle2 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50, 1.58,
	1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83, 1.75,
	1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0, 0.92,
	0.83, 0.75, 0.67, 0.58, 0.50, 0.42, 0.33, 0.25, 0.17, 0.08,
	0.0,
}

// _q1LightStyle3 defines a sequence of float64 values representing a specific light intensity pattern.
var _q1LightStyle3 = []float64{
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.0, 0.08, 0.17,
	0.25, 0.33, 0.42, 0.50,
}

// _q1LightStyle4 represents a light style pattern alternating between 1.0 and 0.0 at equal intervals.
var _q1LightStyle4 = []float64{
	1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0,
}

// _q1LightStyle5 defines a sequence of light intensity values, creating a smooth fluctuating brightness pattern.
var _q1LightStyle5 = []float64{
	0.75, 0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50,
	1.58, 1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83,
	1.75, 1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0,
	0.92, 0.83, 0.75,
}

// _q1LightStyle6 defines a sequence of float values representing a light style pattern in a specific configuration.
var _q1LightStyle6 = []float64{
	1.08, 1.0, 1.17, 1.08, 1.33, 1.08, 1.0, 1.17, 1.0, 1.08,
	1.0, 1.17, 1.0, 1.17, 1.0, 1.08, 1.17,
}

// _q1LightStyle7 defines a sequence of light intensity values representing a distinct light style pattern.
var _q1LightStyle7 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.08, 0.17, 0.25,
	0.33, 0.42, 0.50, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0,
	0.0, 1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
}

// _q1LightStyle8 represents an array of float values defining a specific light style pattern for rendering effects.
var _q1LightStyle8 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 0.0,
	0.0, 0.0, 1.0, 1.0, 1.0, 0.0, 0.08, 0.17, 0.25, 0.33,
	0.42, 0.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 1.0,
}

// _q1LightStyle9 represents a sequence of float64 values for a specific light style pattern with two distinct intensity levels.
var _q1LightStyle9 = []float64{
	0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08,
}

// _q1LightStyle10 represents a predefined sequence of light intensity values, alternating between high and low states.
var _q1LightStyle10 = []float64{
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0,
	0.0, 1.0, 1.0, 1.0, 0.0,
}

// _q1LightStyle11 defines a sequence of float64 values representing a light intensity pattern with a symmetric rise and fall.
var _q1LightStyle11 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.42, 1.33,
	1.25, 1.17, 1.08, 1.0, 0.92, 0.83, 0.75, 0.67, 0.58, 0.50,
	0.42, 0.33, 0.25, 0.17, 0.08, 0.0,
}

// _q1LightStyles defines a collection of predefined light intensity patterns to simulate various lighting effects.
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

// LightStyle parses a style string and returns a corresponding array of light intensity values from predefined styles.
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

// Lights manages a map of target entities, enabling the creation and manipulation of light configurations in the system.
type Lights struct {
	targetEntities map[string]*lumps.Entity
}

// NewLights initializes and returns a Lights instance using the provided slice of Entity objects.
// It maps entities with a "targetname" property to facilitate light creation.
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

func (l *Lights) computeColor(colorStr string) (float64, float64, float64, bool) {
	if len(colorStr) == 0 {
		return 0, 0, 0, false
	}
	cr, cg, cb, valid := lumps.ParseVector(colorStr)
	if !valid {
		return 0, 0, 0, false
	}
	var r, g, b float64
	if cr > 1.0 || cg > 1.0 || cb > 1.0 {
		r = cr / 255.0
		g = cg / 255.0
		b = cb / 255.0
	} else {
		r = cr
		g = cg
		b = cb
	}
	return r, g, b, true
}

// CreateLight generates a new light source based on the given entity, position, and subclass parameters.
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
		return nil
		kind = config.LightKindSpot
		yaw, pitch, _, valid := lumps.ParseVector(mangleStr)
		if !valid {
			fmt.Printf("Invalid mangle vector: %s\n", mangleStr)
			return nil
		}
		dirX, dirY, dirZ = lumps.CalcDirection(yaw, pitch)

		dirZ = dirZ
		dirY = -dirY
		dirX = -dirX
	}

	r, g, b, ok := l.computeColor(colorStr)
	if !ok {
		r, g, b = 1.0, 1.0, 1.0
	}

	var falloff float64
	var intensity float64
	if kind == config.LightKindSpot {
		falloff = q1Intensity * 0.1
		intensity = q1Intensity * 0.1
		if c, valid := lumps.ParseFloat(angleStr); valid {
			coneAngle = c
		}
	} else {
		falloff = q1Intensity * 0.01
		intensity = q1Intensity * 0.1
	}

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

/*

// createLight creates a new Light instance based on entity properties and position, returning an error if invalid or missing data.
func (l *Lights) createLightOLD(ent *lumps.Entity, pos geometry.XYZ, subClass string, allEntities []*lumps.Entity) *config.Light {
	kind := config.LightKindAmbient
	intensity := 300.0 // Typical Quake default fallback
	falloff := 0.0
	dirX, dirY, dirZ := 0.0, -1.0, 0.0 // Default: look down
	r, g, b := 1.0, 1.0, 1.0           // Default White
	mangleStr, _ := ent.Properties["mangle"]
	colorStr, _ := ent.Properties["_color"]
	targetStr, _ := ent.Properties["target"]
	lightStr, _ := ent.Properties["light"]
	angleStr, _ := ent.Properties["angle"]
	var angle float64

	if len(lightStr) > 0 {
		intensity, _ = strconv.ParseFloat(lightStr, 64)
	}

	if len(targetStr) > 0 {
		kind = config.LightKindSpot
		for _, targetEnt := range allEntities {
			if targetEnt.Properties["targetname"] == targetStr {
				if tOrigin, ok := targetEnt.Properties["origin"]; ok {
					var tx, ty, tz float64
					_, _ = fmt.Sscanf(tOrigin, "%f %f %f", &tx, &ty, &tz)
					targetPos := lumps.CreateXYZ(tx, ty, tz)
					// Compute directional vector
					dx := targetPos.X - pos.X
					dy := targetPos.Y - pos.Y
					dz := targetPos.Z - pos.Z
					// Normalize vector
					if length := math.Sqrt(dx*dx + dy*dy + dz*dz); length > 0 {
						dirX, dirY, dirZ = dx/length, dy/length, dz/length
					}
					break
				}
			}
		}
	} else if len(angleStr) > 0 {
		kind = config.LightKindSpot
		if len(mangleStr) > 0 {
			if yaw, pitch, _, valid := lumps.ParseVector(mangleStr); valid {
				dirX, dirY, dirZ = lumps.CalcDirection(yaw, pitch)
			}
		} else {
			angle, _ = strconv.ParseFloat(angleStr, 64)
			if angle == -1 {
				dirX, dirY, dirZ = 0.0, 1.0, 0.0 // Look up
			} else if angle == -2 {
				dirX, dirY, dirZ = 0.0, -1.0, 0.0 // Look down
			} else {
				dirX, dirY, dirZ = lumps.CalcDirection(angle, 0)
			}
		}
	}

	style := _q1LightStyle0
	if sIndex, ok := ent.Properties["style"]; ok {
		// handles light, light_fluoro, light_fluorospark
		if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 {
			if index < len(_q1LightStyles) {
				style = _q1LightStyles[index]
			} else if index >= 32 {
				// Quake 1 uses styles 32-63 for switchable (trigger) lights. Default to steady ON.
				style = _q1LightStyle0
			} else {
				fmt.Println("invalid light style index:", sIndex)
			}
		} else {
			fmt.Println("invalid light style index:", sIndex)
		}
	}

	// COLOR (Standard Quake 2 / Modern Quake 1)
	if len(colorStr) > 0 {
		if cr, cg, cb, valid := lumps.ParseVector(colorStr); valid {
			if cr > 1.0 || cg > 1.0 || cb > 1.0 {
				r, g, b = cr/255.0, cg/255.0, cb/255.0
			} else {
				r, g, b = cr, cg, cb
			}
		}
	}

	if kind == config.LightKindSpot {
		intensity = intensity * 0.9
		falloff = intensity * 10
	} else {
		falloff = intensity * 0.03
		intensity = intensity * 0.003
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

	return cl
}

*/
