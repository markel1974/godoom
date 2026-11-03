package doom3

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
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

// GetAngle parses and returns the yaw angle from the "angle" property.
func (e *MapEntity) GetAngle() float64 {
	val, ok := e.GetProperty("angle")
	if !ok {
		return 0.0
	}
	a, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0
	}
	return a
}

// MapParser parses Doom 3 .map ASCII files.
type MapParser struct {
	lexer *proc.Lexer
}

// NewMapParser initializes a new MapParser with the given .map file bytes.
func NewMapParser(data []byte) *MapParser {
	return &MapParser{
		lexer: proc.NewLexer(data),
	}
}

// Parse extracts all entities defined in the .map file.
func (p *MapParser) Parse() ([]*MapEntity, error) {
	var entities []*MapEntity

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if tok == "{" {
			ent, err := p.parseEntity()
			if err != nil {
				return nil, err
			}
			if len(ent.Properties) > 0 {
				entities = append(entities, ent)
			}
		}
	}

	return entities, nil
}

func (p *MapParser) parseEntity() (*MapEntity, error) {
	ent := NewMapEntity()

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}

		if tok == "}" {
			break
		}

		// Sub-blocks such as brushDef3 { ... } or patchDef2 { ... }
		if tok == "{" {
			if err := p.lexer.SkipBlock(); err != nil {
				return nil, err
			}
			continue
		}

		// Property key
		key := tok
		val, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}

		ent.Properties[strings.ToLower(key)] = val
	}

	return ent, nil
}
