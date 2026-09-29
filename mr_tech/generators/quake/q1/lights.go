package q1

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// _q1LightStyle0 defines a constant light animation pattern with uniform intensity throughout.
var _q1LightStyle0 = []float64{1.0}

// _q1LightStyle1 defines a lighting style represented as a sequence of float64 intensity values.
var _q1LightStyle1 = []float64{
	1.0, 1.0, 1.08, 1.0, 1.0, 1.17, 1.0, 1.0, 1.17, 1.0,
	1.0, 1.08, 1.17, 1.08, 1.0, 1.0, 1.17, 1.08, 1.33, 1.08,
	1.0, 1.0, 1.17,
}

// _q1LightStyle2 defines a waveform pattern with a gradual increase and decrease in intensity values in a symmetrical manner.
var _q1LightStyle2 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50, 1.58,
	1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83, 1.75,
	1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0, 0.92,
	0.83, 0.75, 0.67, 0.58, 0.50, 0.42, 0.33, 0.25, 0.17, 0.08,
	0.0,
}

// _q1LightStyle3 defines a series of float64 values representing a specific lighting style configuration pattern.
var _q1LightStyle3 = []float64{
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.0, 0.08, 0.17,
	0.25, 0.33, 0.42, 0.50,
}

// _q1LightStyle4 represents a repeating pattern of alternating high (1.0) and low (0.0) light intensity values.
var _q1LightStyle4 = []float64{
	1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0,
}

// _q1LightStyle5 defines a sequence of light intensity values forming a sinusoidal-like pattern with peaks and troughs.
var _q1LightStyle5 = []float64{
	0.75, 0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.50,
	1.58, 1.67, 1.75, 1.83, 1.92, 2.0, 2.08, 2.0, 1.92, 1.83,
	1.75, 1.67, 1.58, 1.50, 1.42, 1.33, 1.25, 1.17, 1.08, 1.0,
	0.92, 0.83, 0.75,
}

// _q1LightStyle6 represents a sequence of light intensity values oscillating around a normalized brightness level of 1.0.
var _q1LightStyle6 = []float64{
	1.08, 1.0, 1.17, 1.08, 1.33, 1.08, 1.0, 1.17, 1.0, 1.08,
	1.0, 1.17, 1.0, 1.17, 1.0, 1.08, 1.17,
}

// _q1LightStyle7 represents a predefined light animation pattern using a sequence of float64 values.
var _q1LightStyle7 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.08, 0.17, 0.25,
	0.33, 0.42, 0.50, 1.0, 1.0, 1.0, 1.0, 0.0, 0.0, 0.0,
	0.0, 1.0, 1.0, 1.0, 0.0, 0.0, 1.0, 1.0,
}

// _q1LightStyle8 defines a sequence of floating-point values representing a custom lighting pattern or animation style.
var _q1LightStyle8 = []float64{
	1.0, 1.0, 1.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 0.0,
	0.0, 0.0, 1.0, 1.0, 1.0, 0.0, 0.08, 0.17, 0.25, 0.33,
	0.42, 0.0, 0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 1.0,
}

// _q1LightStyle9 represents a light style pattern consisting of an initial series of zeroes followed by repeated 2.08 values.
var _q1LightStyle9 = []float64{
	0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08, 2.08,
}

// _q1LightStyle10 defines a sequence of light intensity variations represented as a slice of float64 values.
var _q1LightStyle10 = []float64{
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0,
	0.0, 1.0, 1.0, 1.0, 0.0,
}

// _q1LightStyle11 defines a sinusoidal light intensity pattern that peaks near the center and tapers symmetrically.
var _q1LightStyle11 = []float64{
	0.0, 0.08, 0.17, 0.25, 0.33, 0.42, 0.50, 0.58, 0.67, 0.75,
	0.83, 0.92, 1.0, 1.08, 1.17, 1.25, 1.33, 1.42, 1.42, 1.33,
	1.25, 1.17, 1.08, 1.0, 0.92, 0.83, 0.75, 0.67, 0.58, 0.50,
	0.42, 0.33, 0.25, 0.17, 0.08, 0.0,
}

// _q1LightStyles contains predefined light style sequences represented as a 2D slice of float64 values.
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

type Lights struct {
	targetEntities map[string]*lumps.Entity
}

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

func (l *Lights) CreateLight(ent *lumps.Entity, pos geometry.XYZ, subClass string) *config.Light {
	kind := config.LightKindAmbient
	q1Intensity := 300.0
	// Default direction: down.
	dirX, dirY, dirZ := 0.0, 0.0, -1.0
	r, g, b := 1.0, 1.0, 1.0
	coneAngle := 40.0 // Quake default
	lightStr, _ := ent.GetProperty("light")
	targetStr, _ := ent.GetProperty("target")
	mangleStr, _ := ent.GetProperty("mangle")
	angleStr, _ := ent.GetProperty("angle")
	colorStr, _ := ent.GetProperty("_color")
	styleStr, _ := ent.GetProperty("style")

	if lightStr != "" {
		if value, err := strconv.ParseFloat(lightStr, 64); err == nil {
			q1Intensity = value
		}
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
		if len(originStr) == 0 {
			fmt.Println("origin vector not found")
			return nil
		}
		var tx, ty, tz float64
		if _, err := fmt.Sscanf(originStr, "%f %f %f", &tx, &ty, &tz); err != nil {
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
		//dirZ = 0
	} else if mangleStr != "" {
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

	if colorStr != "" {
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

	style := []float64{1.0}
	if styleStr != "" {
		if index, err := strconv.Atoi(styleStr); err == nil {
			if index >= 0 && index < len(_q1LightStyles) {
				style = _q1LightStyles[index]
			} else if index >= 32 {
				// Switchable Quake light styles.
				// The runtime currently has no separate representation
				// for the trigger/switch state, so default to steady ON.
				style = []float64{1.0}
			}
		}
	}

	// Runtime normalization
	var falloff float64
	var intensity float64
	if kind == config.LightKindSpot {
		falloff = q1Intensity * 0.1
		intensity = q1Intensity * 0.1
		//intensity *= 0.05
		//falloff = intensity * 1.0
		if len(angleStr) > 0 {
			if angleStr != "" {
				if value, err := strconv.ParseFloat(angleStr, 64); err == nil {
					coneAngle = value
				}
			}
		}
	} else {
		//falloff = q1Intensity * 0.03
		//intensity = q1Intensity * 0.05
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
	light.Style = style
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
