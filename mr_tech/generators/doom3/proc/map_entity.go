package proc

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// MapEntity represents a parsed entity from a Doom 3 .map file.
type MapEntity struct {
	Properties map[string]string
}

// NewMapEntity initializes an empty MapEntity.
func NewMapEntity() *MapEntity {
	return &MapEntity{
		Properties: make(map[string]string),
	}
}

// Parse reads tokens from the lexer to populate the MapEntity's properties and skips sub-blocks if encountered.
func (e *MapEntity) Parse(lexer *Lexer) error {
	for {
		tok, tErr := lexer.NextToken()
		if tErr != nil {
			return tErr
		}
		if tok == "}" {
			break
		}
		// Sub-blocks such as brushDef3 { ... } or patchDef2 { ... }
		if tok == "{" {
			if err := lexer.SkipBlock(); err != nil {
				return err
			}
			continue
		}
		// Property key
		key := tok
		val, err := lexer.NextToken()
		if err != nil {
			return err
		}
		e.Properties[strings.ToLower(key)] = val
	}
	return nil
}

// GetProperty retrieves a property value by key.
func (e *MapEntity) GetProperty(key string) (string, bool) {
	val, ok := e.Properties[key]
	return val, ok
}

// GetOrigin parses and returns the 3D position vector from the "origin" property.
func (e *MapEntity) GetOrigin() (geometry.XYZ, error) {
	val, ok := e.GetProperty("origin")
	if !ok {
		return geometry.XYZ{}, fmt.Errorf("entity has no origin")
	}
	parts := strings.Fields(val)
	if len(parts) < 3 {
		return geometry.XYZ{}, fmt.Errorf("invalid origin string: %s", val)
	}
	x, err1 := strconv.ParseFloat(parts[0], 64)
	y, err2 := strconv.ParseFloat(parts[1], 64)
	z, err3 := strconv.ParseFloat(parts[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return geometry.XYZ{}, fmt.Errorf("failed to parse origin coordinates")
	}
	return geometry.XYZ{X: x, Y: y, Z: z}, nil
}

// GetColor parses and returns RGB color components from "_color" property, defaulting to white (1, 1, 1).
func (e *MapEntity) GetColor() (float64, float64, float64) {
	val, ok := e.GetProperty("_color")
	if !ok {
		return 1.0, 1.0, 1.0
	}
	parts := strings.Fields(val)
	if len(parts) < 3 {
		return 1.0, 1.0, 1.0
	}
	r, err1 := strconv.ParseFloat(parts[0], 64)
	g, err2 := strconv.ParseFloat(parts[1], 64)
	b, err3 := strconv.ParseFloat(parts[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 1.0, 1.0, 1.0
	}
	return r, g, b
}

// GetLightRadius parses and returns the maximum radius of the light.
func (e *MapEntity) GetLightRadius() float64 {
	val, ok := e.GetProperty("light_radius")
	if !ok {
		val, ok = e.GetProperty("radius")
	}
	if !ok {
		return 300.0
	}
	parts := strings.Fields(val)
	if len(parts) == 0 {
		return 300.0
	}
	var maxR float64 = 0
	for _, p := range parts {
		r, err := strconv.ParseFloat(p, 64)
		if err == nil {
			if r < 0 {
				r = -r
			}
			if r > maxR {
				maxR = r
			}
		}
	}
	if maxR <= 0 {
		return 300.0
	}
	return maxR
}

// GetAngle parses and returns the yaw angle from the "angle", "angles", or "rotation" property.
func (e *MapEntity) GetAngle() float64 {
	val, ok := e.GetProperty("angle")
	if ok {
		if a, err := strconv.ParseFloat(val, 64); err == nil {
			return a
		}
	}

	if anglesStr, ok := e.GetProperty("angles"); ok {
		parts := strings.Fields(anglesStr)
		if len(parts) >= 2 {
			if yaw, err := strconv.ParseFloat(parts[1], 64); err == nil {
				return yaw
			}
		}
	}

	// In Doom 3, entities often use a 3x3 rotation matrix "m00 m01 m02 m10 m11 m12 m20 m21 m22"
	if rot, ok := e.GetProperty("rotation"); ok {
		parts := strings.Fields(rot)
		if len(parts) >= 4 {
			m00, err1 := strconv.ParseFloat(parts[0], 64)
			m01, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil {
				rad := math.Atan2(m01, m00)
				deg := rad * 180.0 / math.Pi
				if deg < 0 {
					deg += 360.0
				}
				return deg
			}
		}
	}

	return 0.0
}

func (e *MapEntity) isPropEnabled(key string) bool {
	val, ok := e.GetProperty(key)
	if !ok {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(val))
	return v != "" && v != "0" && v != "false"
}

// IsDormant checks if the entity is dormant, hidden, or waiting for a trigger/teleport/cinematic/spawner event in Doom 3.
func (e *MapEntity) IsDormant() bool {
	if e.isPropEnabled("teleport") {
		return true
	}
	if val, ok := e.GetProperty("hide"); ok && (val == "1" || val == "2" || strings.ToLower(val) == "true") {
		return true
	}
	if e.isPropEnabled("trigger") {
		return true
	}
	if val, ok := e.GetProperty("bind"); ok && val != "" {
		return true
	}
	if val, ok := e.GetProperty("trigger_anim"); ok && val != "" {
		return true
	}
	if e.isPropEnabled("num_cinematics") {
		return true
	}
	if e.isPropEnabled("spawner") {
		return true
	}
	return false
}

// IsLevelAvailable determines whether the entity should spawn at the given difficulty level
// (1-2: Easy, 3: Medium, 4: Hard, 5: Nightmare) using Doom 3's not_easy/not_medium/not_hard keys.
func (e *MapEntity) IsLevelAvailable(level int) bool {
	switch level {
	case 1, 2:
		return !e.isPropEnabled("not_easy")
	case 3:
		return !e.isPropEnabled("not_medium")
	case 4:
		return !e.isPropEnabled("not_hard")
	case 5:
		return !e.isPropEnabled("not_hard") && !e.isPropEnabled("not_nightmare")
	}
	return true
}

// GetLightKind determines whether the light entity is a spotlight, ambient, or diffuse point light.
func (e *MapEntity) GetLightKind() (string, string, string) {
	tex, _ := e.GetProperty("texture")
	ambient, _ := e.GetProperty("ambient")
	target, _ := e.GetProperty("light_target")
	return tex, ambient, target
}

// GetSpotlightParams computes the direction vector and cone cut-off angles for a spotlight.
func (e *MapEntity) GetSpotlightParams() (geometry.XYZ, float64, float64) {
	dir := geometry.XYZ{X: 0, Y: 0, Z: -1}
	cutOff := 45.0

	if targetStr, ok := e.GetProperty("light_target"); ok {
		parts := strings.Fields(targetStr)
		if len(parts) == 3 {
			x, err1 := strconv.ParseFloat(parts[0], 64)
			y, err2 := strconv.ParseFloat(parts[1], 64)
			z, err3 := strconv.ParseFloat(parts[2], 64)
			if err1 == nil && err2 == nil && err3 == nil {
				length := math.Sqrt(x*x + y*y + z*z)
				if length > 0 {
					dir = geometry.XYZ{X: x / length, Y: y / length, Z: z / length}
					if rightStr, ok := e.GetProperty("light_right"); ok {
						rParts := strings.Fields(rightStr)
						if len(rParts) == 3 {
							rx, rErr1 := strconv.ParseFloat(rParts[0], 64)
							ry, rErr2 := strconv.ParseFloat(rParts[1], 64)
							rz, rErr3 := strconv.ParseFloat(rParts[2], 64)
							if rErr1 == nil && rErr2 == nil && rErr3 == nil {
								rLen := math.Sqrt(rx*rx + ry*ry + rz*rz)
								if rLen > 0 {
									cutOff = math.Atan2(rLen, length) * 180.0 / math.Pi
								}
							}
						}
					}
				}
			}
		}
	} else if rotStr, ok := e.GetProperty("rotation"); ok {
		parts := strings.Fields(rotStr)
		if len(parts) == 9 {
			r02, err1 := strconv.ParseFloat(parts[2], 64)
			r12, err2 := strconv.ParseFloat(parts[5], 64)
			r22, err3 := strconv.ParseFloat(parts[8], 64)
			if err1 == nil && err2 == nil && err3 == nil {
				dx, dy, dz := -r02, -r12, -r22
				length := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if length > 0 {
					dir = geometry.XYZ{X: dx / length, Y: dy / length, Z: dz / length}
				}
			}
		}
	}

	outerCutOff := cutOff + 10.0
	return dir, cutOff, outerCutOff
}

// TransformPoint transforms a local coordinate point into world space using the entity's rotation and origin.
func (e *MapEntity) TransformPoint(p geometry.XYZ) geometry.XYZ {
	origin, _ := e.GetOrigin()
	rot := e.TransformVector(p)
	return geometry.XYZ{X: origin.X + rot.X, Y: origin.Y + rot.Y, Z: origin.Z + rot.Z}
}

// TransformVector rotates a direction/offset vector using the entity's rotation, angles, or angle properties without translation.
func (e *MapEntity) TransformVector(p geometry.XYZ) geometry.XYZ {
	if rotStr, ok := e.GetProperty("rotation"); ok {
		parts := strings.Fields(rotStr)
		if len(parts) == 9 {
			r0, e0 := strconv.ParseFloat(parts[0], 64)
			r1, e1 := strconv.ParseFloat(parts[1], 64)
			r2, e2 := strconv.ParseFloat(parts[2], 64)
			r3, e3 := strconv.ParseFloat(parts[3], 64)
			r4, e4 := strconv.ParseFloat(parts[4], 64)
			r5, e5 := strconv.ParseFloat(parts[5], 64)
			r6, e6 := strconv.ParseFloat(parts[6], 64)
			r7, e7 := strconv.ParseFloat(parts[7], 64)
			r8, e8 := strconv.ParseFloat(parts[8], 64)
			if e0 == nil && e1 == nil && e2 == nil && e3 == nil && e4 == nil && e5 == nil && e6 == nil && e7 == nil && e8 == nil {
				rx := p.X*r0 + p.Y*r3 + p.Z*r6
				ry := p.X*r1 + p.Y*r4 + p.Z*r7
				rz := p.X*r2 + p.Y*r5 + p.Z*r8
				return geometry.XYZ{X: rx, Y: ry, Z: rz}
			}
		}
	}

	if anglesStr, ok := e.GetProperty("angles"); ok {
		parts := strings.Fields(anglesStr)
		if len(parts) >= 3 {
			pitch, errP := strconv.ParseFloat(parts[0], 64)
			yaw, errY := strconv.ParseFloat(parts[1], 64)
			roll, errR := strconv.ParseFloat(parts[2], 64)
			if errP == nil && errY == nil && errR == nil {
				radYaw := yaw * math.Pi / 180.0
				radPitch := pitch * math.Pi / 180.0
				radRoll := roll * math.Pi / 180.0
				sy, cy := math.Sin(radYaw), math.Cos(radYaw)
				sp, cp := math.Sin(radPitch), math.Cos(radPitch)
				sr, cr := math.Sin(radRoll), math.Cos(radRoll)

				m00 := cp * cy
				m01 := cp * sy
				m02 := -sp
				m10 := sr*sp*cy - cr*sy
				m11 := sr*sp*sy + cr*cy
				m12 := sr * cp
				m20 := cr*sp*cy + sr*sy
				m21 := cr*sp*sy - sr*cy
				m22 := cr * cp

				rx := p.X*m00 + p.Y*m10 + p.Z*m20
				ry := p.X*m01 + p.Y*m11 + p.Z*m21
				rz := p.X*m02 + p.Y*m12 + p.Z*m22
				return geometry.XYZ{X: rx, Y: ry, Z: rz}
			}
		}
	}

	if angleStr, ok := e.GetProperty("angle"); ok {
		if a, err := strconv.ParseFloat(angleStr, 64); err == nil && a != 0 {
			rad := a * math.Pi / 180.0
			cosA := math.Cos(rad)
			sinA := math.Sin(rad)
			rx := p.X*cosA - p.Y*sinA
			ry := p.X*sinA + p.Y*cosA
			return geometry.XYZ{X: rx, Y: ry, Z: p.Z}
		}
	}

	return p
}
