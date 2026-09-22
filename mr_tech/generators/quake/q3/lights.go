package q3

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

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

const lightTargetName = "targetname"

type Lights struct {
	entities []*lumps.Entity
	targets  map[string]*lumps.Entity
}

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

func (l *Lights) Create(ent *lumps.Entity, angle float64, pos geometry.XYZ) *config.Light {
	mangleStr, _ := ent.Properties["mangle"]
	colorStr, _ := ent.Properties["_color"]
	targetStr, hasTarget := ent.Properties["target"]

	// Detect spotlight
	isSpot := false
	if hasTarget && len(targetStr) > 0 {
		isSpot = true
		if otherEnt, ok := l.targets[targetStr]; ok {
			if otherEnt.Properties[lightTargetName] == targetStr {
				if originStr, ok := otherEnt.Properties["origin"]; ok {
					if tx, ty, tz, valid := lumps.ParseVector(originStr); valid {
						dx := tx - pos.X
						dy := ty - pos.Y
						dz := tz - pos.Z
						yaw := math.Atan2(dy, dx) * 180 / math.Pi
						pitch := math.Atan2(dz, math.Sqrt(dx*dx+dy*dy)) * 180 / math.Pi
						mangleStr = fmt.Sprintf("%f %f 0", yaw, pitch)
					}
				}
			}
		}
	} else if len(mangleStr) > 0 {
		isSpot = true
	}
	style := _q3LightStyle0
	if sIndex, ok := ent.Properties["style"]; ok {
		if index, err := strconv.Atoi(sIndex); err == nil && index >= 0 && index < len(_q3LightStyles) {
			style = _q3LightStyles[index]
		}
	}
	light := l.doCreate(ent, angle, mangleStr, colorStr, pos, style, isSpot)
	return light
}

// createLight creates a light configuration based on entity properties, position, type, intensity, and direction.
func (l *Lights) doCreate(entity *lumps.Entity, angle float64, mangleStr, colorStr string, pos geometry.XYZ, style []float64, isSpot bool) *config.Light {
	intensity := 0.0
	falloff := 0.0
	var kind config.LightKind

	// BASE INTENSITY
	if l, ok := entity.Properties["light"]; ok {
		intensity, _ = strconv.ParseFloat(l, 64)
		//intensity *= 0.3
	} else {
		intensity = 300 // Typical Quake default fallback
	}

	// COLOR (Standard Quake 2 / Modern Quake 1)
	r, g, b := 1.0, 1.0, 1.0 // Default White
	if len(colorStr) > 0 {
		if cr, cg, cb, valid := lumps.ParseVector(colorStr); valid {
			if cr > 1.0 || cg > 1.0 || cb > 1.0 {
				r, g, b = cr/255.0, cg/255.0, cb/255.0
			} else {
				r, g, b = cr, cg, cb
			}
		}
	}

	// SPOTLIGHT DIRECTION
	dirX, dirY, dirZ := 0.0, -1.0, 0.0 // Default: look down
	if isSpot {
		kind = config.LightKindSpot
		intensity = intensity * 0.9
		falloff = intensity * 10
		if len(mangleStr) > 0 {
			if yaw, pitch, _, valid := lumps.ParseVector(mangleStr); valid {
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
		kind = config.LightKindAmbient
		intensity = intensity * 0.02
		falloff = intensity
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
