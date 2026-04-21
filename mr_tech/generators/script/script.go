package script

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// scaleW defines the horizontal scaling factor.
// scaleH defines the vertical scaling factor.
const (
	scaleW = 10.0
	scaleH = 50.0
)

// Parser is a type responsible for parsing and constructing game configuration data from text inputs.
type Parser struct {
}

// NewParser creates and returns a new instance of Parser, used for parsing and managing level configuration data.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses the given script data, initializing and returning the game's configuration or an error if parsing fails.
func (p *Parser) Parse(id string) (*config.Root, error) {
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	t, tErr := NewTextures(basePath)
	if tErr != nil {
		return nil, tErr
	}

	player := config.NewConfigPlayer(geometry.XYZ{}, 0, 10, 3, 20)
	player.Speed = 60

	cal := config.NewConfigCalibration(false, 0, 0, 0, 0, 0, 0, true)
	cfg := config.NewConfigRoot(cal, nil, player, nil, 1.0, t)

	oldData := strings.Split(id, "\n")
	configSectorIdx := 0

	for _, data := range oldData {
		var verb string
		r := strings.NewReader(data)
		if _, err := fmt.Fscanf(r, "%s", &verb); err != nil {
			continue
		}
		switch verb {
		case "#":
			//comment
		case "vertex":
			vertex, err := p.parseVertex(r)
			if err != nil {
				return nil, err
			}
			cfg.Vertices = append(cfg.Vertices, vertex...)
		case "sector":
			cs, err := p.parseSector(r, cfg.Vertices, configSectorIdx)
			if err != nil {
				return nil, err
			}
			configSectorIdx++
			cfg.Sectors = append(cfg.Sectors, cs)
		case "player":
			if err := p.parsePlayer(r, cfg.Player); err != nil {
				return nil, err
			}
		default:
			continue
		}
	}

	if cfg.Vertices == nil {
		return nil, errors.New("nil vertices")
	}

	p.finalize(cfg)

	return cfg, nil
}

// finalize processes and finalizes the sector definitions by scanning for reversed segments and updating their properties.
func (p *Parser) finalize(cfg *config.Root) {
	// 2. Fase di Sigillatura (Rescan Topologico)
	type edgeKey struct{ p1, p2 geometry.XY }
	lineDefsCache := make(map[edgeKey]*config.Sector)
	for _, sector := range cfg.Sectors {
		for _, s := range sector.Segments {
			lineDefsCache[edgeKey{s.Start, s.End}] = sector
		}
	}
	for _, sector := range cfg.Sectors {
		for _, s := range sector.Segments {
			if s.Kind != config.SegmentWall {
				// Cerchiamo il segmento invertito (il "lato B" della linea)
				revKey := edgeKey{p1: s.End, p2: s.Start}
				if neighborSector, ok := lineDefsCache[revKey]; ok {
					s.Tag = neighborSector.Id
					s.Kind = config.SegmentUnknown
				} else {
					// Nessun segmento corrispondente trovato: la linea deve essere un muro
					s.Kind = config.SegmentWall
					s.Tag = "unknown"
				}
			}
		}
	}
}

// parseVertex reads vertex data from the provided reader and returns a slice of geometry.XY or an error on failure.
func (p *Parser) parseVertex(r io.Reader) ([]geometry.XY, error) {
	var cfgVertices []geometry.XY
	for {
		var vertexY float64
		if _, err := fmt.Fscanf(r, "%f", &vertexY); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		for {
			xy := geometry.XY{Y: vertexY}
			if _, err := fmt.Fscanf(r, "%f", &xy.X); err != nil {
				if err != io.EOF {
					return nil, err
				}
				break
			}
			cfgVertices = append(cfgVertices, xy)
		}
	}
	return cfgVertices, nil
}

// parseSector parses sector data from the provided reader, using vertices and a sector index, and constructs a Sector.
// Returns the created Sector or an error if parsing fails.
func (p *Parser) parseSector(r io.Reader, cfgVertices []geometry.XY, configSectorIdx int) (*config.Sector, error) {
	if cfgVertices == nil {
		return nil, errors.New("nil vertices")
	}
	const falloff = 10.0
	const lightIntensity = 1.5
	cs := config.NewConfigSector(strconv.Itoa(configSectorIdx), lightIntensity, config.LightKindAmbient, falloff)
	if _, err := fmt.Fscanf(r, "%f%f", &cs.FloorY, &cs.CeilY); err != nil {
		return nil, err
	}
	type data struct {
		Val  int
		Kind int
	}
	var numbers []data
	var word string
	for {
		if _, err := fmt.Fscanf(r, "%32s", &word); err != nil {
			break
		}
		var d data
		d.Val = config.SegmentUnknown
		d.Kind = config.SegmentUnknown
		if word[0] != 'x' {
			if val, err := strconv.Atoi(word); err == nil {
				d.Val = val
			}
		}
		numbers = append(numbers, d)
	}

	if len(numbers) == 0 || len(numbers)%2 > 0 {
		return nil, errors.New("empty or invalid sector definition")
	}

	m := len(numbers) / 2
	for idx := 0; idx < m; idx++ {
		v1Idx := numbers[idx]
		v2Idx := numbers[(idx+1)%m] // Chiusura topologica
		neighborId := numbers[idx+m]

		if v1Idx.Val < 0 || v1Idx.Val >= len(cfgVertices) || v2Idx.Val < 0 || v2Idx.Val >= len(cfgVertices) {
			return nil, fmt.Errorf("invalid vertex index, max: %d", len(cfgVertices))
		}

		start := cfgVertices[v1Idx.Val]
		end := cfgVertices[v2Idx.Val]

		seg := config.NewConfigSegment("", neighborId.Kind, start, end)
		seg.Middle = config.NewConfigAnimation([]string{"wall2.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
		seg.Lower = config.NewConfigAnimation([]string{"wall.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
		seg.Upper = config.NewConfigAnimation([]string{"wall3.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
		cs.Segments = append(cs.Segments, seg)
	}
	cs.Floor = config.NewConfigAnimation([]string{"floor.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
	cs.Ceil = config.NewConfigAnimation([]string{"ceil.ppm"}, config.AnimationKindLoop, scaleW, scaleH)

	return cs, nil
}

// parsePlayer parses player position, angle, and updates the provided Player instance from the given io.Reader input.
func (p *Parser) parsePlayer(r io.Reader, player *config.Player) error {
	if _, err := fmt.Fscanf(r, "%f %f %f", &player.Position.X, &player.Position.Y, &player.Angle); err != nil {
		return err
	}
	return nil
}
